package network

// O ultimo suspiro: o resgate roteirizado que salva o grupo da matilha.
//
// A cena tem tres momentos e cada um mora num lugar diferente, de proposito:
//
//	o gatilho   game.partyIsFalling -> o diretor de dialogo pede ArmLastStand
//	a fala      o dialogo congela o mundo, como qualquer cena
//	o resgate   ResolveLastStand, quando a ultima linha fecha
//
// Entre armar e resolver o Game Over fica SEGURADO. Sem isso a cena perderia
// a corrida: o grupo cai, o host anuncia o fim, e o resgate chega depois do
// jogo ter acabado — que e o mesmo que nao chegar.

import (
	"log"
	"sync"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Quem se ergue depende do MAPA (ver last_stand_heroes.go). O que segue vale
// para qualquer um deles.

// O resgate atende duas pessoas diferentes, com dois tamanhos.
//
// O HEROI DA FASE e quem age: ele volta INTEIRO e com dois segundos, porque
// precisa sair dos dentes, escolher a direcao e lancar. O resto do grupo so
// precisa nao morrer no proximo golpe — mas precisa disso de verdade, porque a
// cena dispara com todo mundo abaixo de um quarto da vida e devolver o jogo
// naquele estado e devolve-lo perdido.
const (
	// LastStandInvulnerability is how long the rescued hero cannot be hurt.
	//
	// Eram dois segundos, e em jogo isso nao dava. A cena termina com o heroi
	// no meio da horda que acabou de derrubar o grupo — na quinta onda do mapa
	// 5 sao 35 simultaneos — e dois segundos e o tempo de tomar um golpe, nao
	// de sair dos dentes, escolher a direcao e lancar. Cinco.
	LastStandInvulnerability float32 = 5.0
	// LastStandPartyHeal is the share of maximum health everyone else is
	// brought back to.
	LastStandPartyHeal float32 = 0.30
	// LastStandPartyInvulnerability is the breath the rest of the party gets
	// to reposition — half the caster's, because they are not the ones who
	// have to aim anything.
	LastStandPartyInvulnerability float32 = 1.0
)

type lastStandState struct {
	mu sync.RWMutex
	// armed is true from the moment the scene is requested until it resolves.
	armed bool
	// done marks the rescue as already spent for this run of the stage.
	done bool
	// npcActive is true while the summoned hero is on the field.
	npcActive bool
	npcPos    rl.Vector2
	// npcID e o dono do efeito do NPC em campo, e npcChar e quem desenhar.
	// Ficam no estado, e nao numa constante, porque o heroi e do mapa.
	npcID   string
	npcChar entity.CharacterType
}

var lastStand lastStandState

// LastStandPending reports whether the rescue is owed. The Game Over check
// consults it, which is the whole reason the flag exists.
func LastStandPending() bool {
	lastStand.mu.RLock()
	defer lastStand.mu.RUnlock()
	return lastStand.armed
}

// LastStandDone reports whether the rescue already happened in this run, so
// the scene cannot play twice on a stage the party keeps losing.
func LastStandDone() bool {
	lastStand.mu.RLock()
	defer lastStand.mu.RUnlock()
	return lastStand.done
}

// setLastStandNPC puts the summoned hero on or off the field. Host and client
// both call it — the host when it summons, the client when the ultimate's
// broadcast tells it one exists.
func setLastStandNPC(pos rl.Vector2, active bool) {
	setLastStandNPCAs(pos, active, "", "")
}

// setLastStandNPCAs faz o mesmo dizendo QUEM esta em campo.
func setLastStandNPCAs(pos rl.Vector2, active bool, npcID string,
	char entity.CharacterType) {
	lastStand.mu.Lock()
	defer lastStand.mu.Unlock()
	lastStand.npcActive = active
	lastStand.npcPos = pos
	lastStand.npcID = npcID
	lastStand.npcChar = char
}

// LastStandNPC returns the summoned hero's position, WHO it is, and whether it
// is on the field at all. The renderer draws it from this; nothing else needs
// it. O personagem vem junto porque ele muda de fase para fase, e desenhar o
// Necromante no mapa da Sacerdotisa seria um erro invisivel no codigo e
// gritante na tela.
func LastStandNPC() (rl.Vector2, entity.CharacterType, bool) {
	lastStand.mu.RLock()
	defer lastStand.mu.RUnlock()
	return lastStand.npcPos, lastStand.npcChar, lastStand.npcActive
}

// ArmLastStand holds the Game Over back because the scene is about to play.
// Called by the dialogue director the frame the scene starts.
func ArmLastStand() {
	lastStand.mu.Lock()
	defer lastStand.mu.Unlock()
	if lastStand.done {
		return
	}
	lastStand.armed = true
	log.Printf("[UltimoSuspiro] cena armada; Game Over segurado")
}

// ResetLastStand clears the scene for a fresh run of the stage.
//
// The fields are cleared one by one instead of assigning a zero struct: the
// state carries its own mutex, and overwriting the struct would copy a lock —
// which go vet rejects, and which here would also replace the very lock being
// held to do it.
func ResetLastStand() {
	lastStand.mu.Lock()
	defer lastStand.mu.Unlock()
	lastStand.armed = false
	lastStand.done = false
	lastStand.npcActive = false
	lastStand.npcPos = rl.Vector2{}
	lastStand.npcID = ""
	lastStand.npcChar = ""
}

// ResolveLastStand performs the rescue as the scene's last line closes.
//
// Two shapes, decided by who is in the party:
//
//   - Somebody plays the map's hero: they come back on their feet with full
//     health and a couple of seconds of immunity, and they cast the ultimate
//     THEMSELVES. The scene hands the moment to the player; it does not play
//     it for them.
//   - Nobody does: one answers anyway and casts on the spot. The rescue
//     cannot depend on the party's character picks.
//
// QUEM e o heroi vem de LastStandHeroFor(h.stageMap) — no mapa 2 o
// Necromante, no 3 a Sacerdotisa.
func (h *Host) ResolveLastStand() {
	lastStand.mu.Lock()
	if lastStand.done {
		lastStand.mu.Unlock()
		return
	}
	lastStand.done = true
	lastStand.armed = false
	lastStand.mu.Unlock()

	hero := LastStandHeroFor(h.stageMap)
	saved := h.reviveHero(hero.character)
	// Todo mundo que NAO e o heroi recebe o resgate menor. Sem isso a
	// cena devolvia o jogo com o grupo ainda abaixo de um quarto da vida, e o
	// primeiro lobo desfazia o que ela acabou de fazer.
	h.reviveParty(saved)
	if h.stageMap == "assets/maps/world_04.json" {
		origin := h.partyCentre()
		owner := saved
		if owner == "" {
			owner = hero.npcID
			setLastStandNPCAs(origin, true, hero.npcID, hero.character)
		}
		h.castCastleJudgment(owner, origin)
		return
	}
	if h.stageMap == "assets/maps/world_06.json" {
		origin := h.partyCentre()
		owner := saved
		if owner == "" {
			owner = hero.npcID
			setLastStandNPCAs(origin, true, hero.npcID, hero.character)
		}
		// Always cast the hero's ultimate and broadcast it, whether NPC or player-controlled
		hero.cast(h.Skills, owner, origin)
		h.BroadcastSkill(hero.skillID, owner, origin)
		h.castCannonJudgment(owner, origin)
		return
	}

	if saved != "" {
		log.Printf("[UltimoSuspiro] %s se ergueu; a ultimate e dele", saved)
		return
	}
	h.summonHeroNPC(hero)
}

// castCastleJudgment gives the map-4 rescue a precise destination contract:
// both island sentries are hit by Flechas Celestiais even though normal shots
// cannot touch their waterbound positions. It changes neither the Arqueiro's
// selectable ultimate nor its two-charge aiming rule outside this cutscene.
func (h *Host) castCastleJudgment(ownerID string, origin rl.Vector2) {
	for _, enemy := range h.EntityManager.GetAllEnemies() {
		if enemy.Type != entity.EnemyTypeCastleSentry || !enemy.IsActive {
			continue
		}
		dir := rl.Vector2Subtract(enemy.Position, origin)
		skill.SpawnCelestialArrow(h.Skills, ownerID, origin, dir)
		h.BroadcastSkillDir("celestial_arrows", ownerID, origin, dir)
	}
	log.Printf("[UltimoSuspiro] julgamento celestial do Arqueiro alcancou as sentinelas")
}

// castCannonJudgment e o resgate do mapa 6: com o Avatar dos Deuses erguido,
// a Paladina alcanca o fim do corredor e destroi os dois canhoes de uma vez.
//
// Como o julgamento do Arqueiro no mapa 4, isto e roteirizado a parte da
// magia selecionavel — Avatar dos Deuses e imunidade total, nao um ataque a
// distancia — porque o resgate precisa de um contrato PRECISO: os canhoes
// declarados pelo mapa, e nao um alvo que o jogador teria de mirar. `origin`
// e so de onde a explosao PARECE vir na tela (o centro do grupo); a
// Paladina nao anda fisicamente ate la — a narracao do dialogo e que conta
// a corrida, o mesmo recurso que a cena do mapa 4 usa para as flechas
// alcancarem ilhas que o Arqueiro nunca pisou.
func (h *Host) castCannonJudgment(ownerID string, origin rl.Vector2) {
	hit := h.DestroyCannons()
	for i, pos := range hit {
		h.broadcastCannonBall("impact", "judgment"+ownerID+string(rune('a'+i)), pos, rl.Vector2{})
	}
	if len(hit) == 0 {
		log.Printf("[UltimoSuspiro] julgamento da Paladina nao encontrou canhao de pe")
		return
	}
	log.Printf("[UltimoSuspiro] julgamento da Paladina destruiu %d canhao(oes); o caminho para a porta esta livre", len(hit))
}

// reviveParty brings everyone except `except` back to LastStandPartyHeal of
// their health, with a short window to get out of the way.
//
// A dead player is raised too: the scene is a rescue, and leaving a body on
// the ground for the thirty-second respawn while the ultimate clears the field
// would mean the player watches their own rescue.
func (h *Host) reviveParty(except string) {
	var revived []PlayerState
	h.playersMutex.Lock()
	for id, p := range h.players {
		if id == except {
			continue
		}
		p.IsDead = false
		p.RespawnIn = 0
		if healed := p.MaxHealth * LastStandPartyHeal; p.Health < healed {
			p.Health = healed
		}
		revived = append(revived, *p)
	}
	h.playersMutex.Unlock()

	for _, p := range revived {
		h.GrantInvulnerability(p.PlayerID, LastStandPartyInvulnerability)
		h.broadcastRespawn(p)
	}
	if len(revived) > 0 {
		log.Printf("[UltimoSuspiro] %d jogador(es) reerguidos a %.0f%% com %.0fs de folga",
			len(revived), LastStandPartyHeal*100, LastStandPartyInvulnerability)
	}
}

// reviveHero puts the map's hero back on their feet at full health, and
// returns their id — or "" when nobody in the party plays that character.
//
// It revives even a hero who is still standing: the scene fires with the
// party under a quarter of health, so "alive" here means barely, and sending
// them into the rescue on that sliver would waste the moment.
func (h *Host) reviveHero(character entity.CharacterType) string {
	var revived *PlayerState
	h.playersMutex.Lock()
	// Map iteration order is random, and a class CAN have more than one
	// player of it (plan §3, "classe duplicada"). Pick deterministically:
	// a present human first, then an absent one, then the bot last — a bot
	// invariably losing this pick if a human is in the running at all.
	best := -1
	for id, p := range h.players {
		if entity.CharacterType(p.Character) != character {
			continue
		}
		rank := 0
		switch {
		case isBotID(id):
			rank = 2
		case p.Absent:
			rank = 1
		}
		if best == -1 || rank < best {
			best, revived = rank, p
		}
	}
	if revived != nil {
		revived.IsDead = false
		revived.RespawnIn = 0
		revived.Health = revived.MaxHealth
		snapshot := *revived
		revived = &snapshot
	}
	h.playersMutex.Unlock()
	if revived == nil {
		return ""
	}
	h.GrantInvulnerability(revived.PlayerID, LastStandInvulnerability)
	// E a ultimate, que e o motivo da cena existir. Reerguer o heroi sem
	// devolver a magia dele entregava um resgate mudo: ele volta inteiro,
	// invulneravel, e a habilidade que o dialogo acabou de anunciar continua
	// em recarga. Ver ClearSkillCooldown.
	h.ClearSkillCooldown(revived.PlayerID, LastStandHeroFor(h.stageMap).skillID)
	// A campanha so entrega a suprema dele na fase SEGUINTE
	// (game.UltimatesGrantedOn), entao sem isto a cena anunciava a magia e o
	// host recusava o lancamento: o gate (skillUnlocked) e o mesmo de sempre,
	// e ele nao sabia que o resgate acabou de acontecer. A concessao vale so
	// PARA ESTA CORRIDA (ver progression.go) e precisa chegar ao cliente, ou o
	// botao do celular continua apagado e o R do desktop continua mudo.
	GrantUltimateForRun(character)
	h.BroadcastUltimateGrant(character)
	h.broadcastRespawn(*revived)
	return revived.PlayerID
}

// summonHeroNPC drops the map's hero into the fight and casts their ultimate
// from it.
//
// The NPC is deliberately NOT a player entry. Adding one would make it count
// in the player list, in the Game Over check and in the HUD, and it would have
// to die, respawn and be networked like everybody else. All it has to do is
// own an effect and be drawn, and every ultimate is keyed by a plain owner id.
func (h *Host) summonHeroNPC(hero lastStandHero) {
	pos := h.partyCentre()
	setLastStandNPCAs(pos, true, hero.npcID, hero.character)

	hero.cast(h.Skills, hero.npcID, pos)
	h.BroadcastSkill(hero.skillID, hero.npcID, pos)
	log.Printf("[UltimoSuspiro] ninguem e %s; um respondeu em (%.0f, %.0f)",
		hero.character, pos.X, pos.Y)
}

// partyCentre is where the summoned Necromante appears: the middle of whoever
// is still on the field.
//
// The legion only hunts within LegionLeashRadius of its owner, so dropping the
// NPC anywhere else would summon a rescue that cannot reach the fight.
func (h *Host) partyCentre() rl.Vector2 {
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()
	var sum rl.Vector2
	n := 0
	for _, p := range h.players {
		sum.X += float32(p.X)
		sum.Y += float32(p.Y)
		n++
	}
	if n == 0 {
		return h.PlayerSpawn
	}
	return rl.NewVector2(sum.X/float32(n), sum.Y/float32(n))
}

// tickLastStand keeps the summoned hero's effect anchored when it needs to be,
// and takes the NPC off the field once the effect is spent — it came for one
// thing.
func (h *Host) tickLastStand() {
	lastStand.mu.RLock()
	active, pos, npcID := lastStand.npcActive, lastStand.npcPos, lastStand.npcID
	lastStand.mu.RUnlock()
	if !active {
		return
	}
	hero := LastStandHeroFor(h.stageMap)
	if !hero.alive(h.Skills, npcID) {
		setLastStandNPC(rl.Vector2{}, false)
		// O cliente precisa saber para parar de desenha-lo. Cada heroi diz
		// qual e a mensagem: a legiao usa `legion_end`, que o cliente ja
		// escuta para dissolve-la.
		h.broadcastUltimate(hero.endSignal, npcID, rl.Vector2{}, rl.Vector2{})
		log.Printf("[UltimoSuspiro] a magia se gastou; o NPC se foi")
		return
	}
	// So a ultimate que SEGUE o dono precisa ser reancorada. O altar nao.
	if hero.anchor != nil {
		hero.anchor(h.Skills, npcID, pos)
	}
}
