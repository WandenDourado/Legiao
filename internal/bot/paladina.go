package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// paladinaBrain holds the front line: stands between the nearest threat and
// the party's frailest member (monsters chase whoever is closest, so this
// is geometry, not a threat stat), shields under pressure, swings whenever
// something is in reach, and calls Avatar Divino when the group is falling.
//
// E, so no corredor do mapa 6, derruba os canhoes: ver huntCannon.
type paladinaBrain struct {
	targetID   string
	decideIn   float32
	retreating bool
	// huntFor e quanto tempo ainda resta da missao dos canhoes depois que ela
	// levantou o Avatar dos Deuses. Ver CannonHuntWindow.
	huntFor float32
}

func (b *paladinaBrain) Think(v View) Intent {
	if b.huntFor > 0 {
		b.huntFor -= v.Dt
	}
	// A CACADA AOS CANHOES VEM ANTES DE TUDO.
	//
	// Nao e prioridade por ser mais perigoso: e prioridade porque e a unica
	// coisa que so ela faz. Enquanto os canhoes estao de pe, o corredor cobra
	// 45 de dano a cada 2,2 s de todo o grupo e nenhuma outra classe tem como
	// calar — nem escoltar, nem segurar a linha, nem recuar resolve. Se ela
	// ficar fazendo o trabalho de front line aqui, ninguem termina a fase.
	if intent, ok := b.huntCannon(v); ok {
		return intent
	}

	if b.decideIn <= 0 {
		b.decideIn = decideEvery
		if foe, ok := mostThreateningFoe(v.Self, v.Allies, engageableFoes(v)); ok {
			b.targetID = foe.ID
		} else {
			b.targetID = ""
		}
	} else {
		b.decideIn -= v.Dt
	}

	target, hasTarget := findFoe(v.Foes, b.targetID)
	// Gated on Shield already being spent (v.PrimaryReady false): a front
	// line that retreats before trying to mitigate abandons the group
	// (plan §A4).
	retreating := paladinaRetreatHysteresis(&b.retreating, healthFrac(v.Self), v.PrimaryReady)

	intent := Intent{}

	// Position: between the target and the frailest ally, at sword range.
	dest := followDest(v)
	usingTravel := false
	if td, ok := travelDest(v); ok {
		dest, usingTravel = td, true
	}
	if hasTarget {
		frailest := frailestAlly(v.Self, v.Allies)
		mid := rl.Vector2Lerp(frailest, target.Pos, 0.6)
		dest = mid
		usingTravel = false
	}
	if retreating {
		nearest, hasNearest := nearestFoe(v.Self.Pos, v.Foes)
		dest = retreatDest(v, nearest.Pos, hasNearest)
		usingTravel = false
	}
	finishMove(&intent, v, dest, usingTravel)

	// Sword sweep whenever something is within reach — never while
	// retreating, or she would keep swinging with her back turned instead
	// of actually opening distance (plan §A4).
	if !retreating && hasTarget && rl.Vector2Distance(v.Self.Pos, target.Pos) <= frontRing {
		aim := target.Pos
		intent.Attack = &aim
	}

	// Shield: surrounded, or badly hurt.
	inMeleeRange := 0
	for _, f := range v.Foes {
		if rl.Vector2Distance(v.Self.Pos, f.Pos) <= frontRing {
			inMeleeRange++
		}
	}
	lowHealth := healthFrac(v.Self) < 0.5
	if v.PrimaryReady && (inMeleeRange >= shieldFoes || lowHealth) {
		intent.Skill = &Cast{SkillID: "shield", Aim: v.Self.Pos}
		return intent
	}

	// Avatar Divino: the group is falling apart.
	if v.UltimateReady && groupIsFalling(v.Self, v.Allies, v.EnemiesLeft) {
		intent.Skill = &Cast{SkillID: "divine_avatar", Aim: v.Self.Pos}
	}

	return intent
}

// huntCannon e a travessia do corredor final: levantar o Avatar dos Deuses,
// andar ate o canhao mais perto sob fogo, e bater ate ele cair.
//
// Devolve ok=false quando nao ha nada a cacar — que e o caso em todos os mapas
// menos o 6, e no 6 depois que os dois postos cairam. Aí a Paladina volta a ser
// a Paladina de sempre.
//
// TRES DECISOES, e cada uma existe por causa de um jeito de isto dar errado:
//
//  1. Ela so assume a missao com a suprema NA MAO (v.UltimateReady) ou ja
//     dentro da janela dela (b.huntFor). Sem essa porta, uma Paladina com o
//     Avatar em recarga marcharia para dentro do bombardeio e morreria a
//     meio caminho, de graca, toda vez.
//  2. Ela levanta a suprema so a CannonAvatarRange, e nao ao avistar o posto:
//     conjurada cedo demais, a imunidade acaba antes de ela chegar.
//  3. Enquanto b.huntFor corre, ela IGNORA recuo, alvo e escolta. Ferida no
//     meio do corredor, a resposta certa nao e voltar — voltar tambem custa a
//     travessia inteira sob fogo. E terminar.
func (b *paladinaBrain) huntCannon(v View) (Intent, bool) {
	cannon, ok := nearestCannon(v.Self.Pos, v.Cannons)
	if !ok {
		b.huntFor = 0
		return Intent{}, false
	}
	dist := rl.Vector2Distance(v.Self.Pos, cannon.HitCentre)
	committed := b.huntFor > 0
	if !committed {
		if dist > CannonHuntRange || !v.UltimateReady {
			return Intent{}, false
		}
	}

	intent := Intent{}
	if !committed {
		if dist > CannonAvatarRange {
			// Ainda longe demais para conjurar: aproxima com a suprema
			// guardada e o Escudo Sagrado de pe. Este trecho e o unico da
			// caçada em que ela leva dano, e o escudo agora paga duas vezes
			// nele — absorve e cura (skill.ShieldCastHeal). Sem isto ela
			// atravessaria a metade perigosa do corredor no peito.
			moveTo(&intent, v.Self.Pos, cannon.HitCentre, v.Allies)
			if v.PrimaryReady {
				intent.Skill = &Cast{SkillID: "shield", Aim: v.Self.Pos}
			}
			return intent, true
		}
		// Perto o bastante para a caminhada caber na suprema: levanta o
		// Avatar dos Deuses e a partir daqui ela nao volta mais.
		b.huntFor = CannonHuntWindow
		intent.Skill = &Cast{SkillID: "divine_avatar", Aim: v.Self.Pos}
		moveTo(&intent, v.Self.Pos, cannon.HitCentre, v.Allies)
		return intent, true
	}
	if dist > CannonStrikeRange {
		moveTo(&intent, v.Self.Pos, cannon.HitCentre, v.Allies)
		return intent, true
	}
	// Chegou: bate, e PARA de andar. O golpe mira o MEIO DO PLINTO (o ponto
	// de navegacao do bot, network.cannonPost.hitAnchor) — o acerto real e
	// decidido contra a caixa inteira da peca (network.cannonPost.hitBox),
	// entao mirar perto do centro so precisa ficar dentro dela. Sem destino
	// nenhum aqui de proposito: continuar pedindo para andar ate la so faria
	// ela empurrar a pedra pelo resto da cena.
	aim := cannon.HitCentre
	intent.Attack = &aim
	return intent, true
}

// nearestCannon devolve o posto de pe mais proximo de pos.
func nearestCannon(pos rl.Vector2, cannons []CannonPost) (CannonPost, bool) {
	var best CannonPost
	found := false
	var bestDist float32
	for _, c := range cannons {
		d := rl.Vector2Distance(pos, c.HitCentre)
		if !found || d < bestDist {
			best, bestDist, found = c, d, true
		}
	}
	return best, found
}

// findFoe looks up a foe by ID; returns ok=false if it died or was never
// set (decideIn just reset targetID to "").
func findFoe(foes []Foe, id string) (Foe, bool) {
	if id == "" {
		return Foe{}, false
	}
	for _, f := range foes {
		if f.ID == id {
			return f, true
		}
	}
	return Foe{}, false
}

func frailestAlly(self Ally, allies []Ally) rl.Vector2 {
	frailest := self
	frac := healthFrac(self)
	for _, a := range allies {
		if a.IsDead {
			continue
		}
		if f := healthFrac(a); f < frac {
			frailest, frac = a, f
		}
	}
	return frailest.Pos
}
