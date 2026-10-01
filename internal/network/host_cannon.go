package network

import (
	"encoding/json"
	"log"
	"sort"

	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A bola de fogo dos canhoes do corredor final (mapa 6).
//
// Ao contrario da esfera da Gargula Sentinela (host_sentry_orb.go), a bola do
// canhao NAO persegue: ela mira o jogador mais fraco ao alcance no instante do
// disparo e sai reta, como a Bola de Fogo do Mago. O acerto e resolvido contra
// QUALQUER jogador que a bola encoste no caminho, nao so contra quem foi
// mirado — um canhao nao lê "dono do golpe", ele varre a linha de tiro, e um
// aliado que se meta na frente para proteger quem estava marcado tambem leva o
// impacto. Isso e deliberado: e a mesma razao pela qual um escudo pode ser
// reforcado a tempo, mas nao pode ser jogado na frente de outra pessoa.
//
// DUAS PERGUNTAS DIFERENTES, e confundi-las foi o defeito de 28/08/2026:
//
//	ONDE ELA ESTOURA   cannonBallImpactPoint — o corpo da bola encostou em
//	                   alguem vivo, e o estouro nasce SOBRE ESSA PESSOA.
//	QUEM SANGRA        playersInCannonBlast — todo mundo dentro de
//	                   skill.CannonBlastRadius (630 px) desse ponto, para a
//	                   frente e para tras.
//
// Com as duas usando o mesmo teste, a explosao nascia 664 px antes do alvo e
// nao alcancava nem ele direito, muito menos quem vinha atras.

// handleCannonTick e chamado uma vez por quadro de UpdateSimulation.
func (h *Host) handleCannonTick(dt float32) {
	// 1. Cada canhao pronto e vivo dispara no jogador com MENOS VIDA ao
	//    alcance.
	//
	// Em duas partes, e a divisao importa: a primeira mexe na lista de
	// canhoes, a segunda le a lista de jogadores. Juntas elas travariam o
	// jogo — o golpe de espada entra com playersMutex na mao e pede
	// cannonsMutex (performSwordAttack -> damageCannonsInSweep), entao segurar
	// cannonsMutex aqui enquanto se pede playersMutex fecharia o ciclo. O
	// tempo entre as duas partes nao custa nada: no pior caso um canhao
	// derrubado neste instante ainda solta a ultima bola que ja estava armada.
	for _, c := range h.readyCannons(dt) {
		targetPos, ok := h.cannonAim(c.Position)
		if !ok {
			// Ninguem vivo em campo. Devolve o gatilho, senao o canhao ficaria
			// dois ciclos calado quando alguem voltar.
			h.rearmCannon(c, 0)
			continue
		}
		dir := rl.Vector2Subtract(targetPos, c.Position)
		id := skill.SpawnCannonBall(true, "", c.ID, c.Position, dir)
		h.broadcastCannonBall("cast", id, c.Position, dir)
	}

	// 2. Avanca as bolas e resolve o que elas encostam.
	for _, id := range skill.StepCannonBalls(dt) {
		// Expirou por tempo: some sem estouro, porque nao acertou nada.
		skill.RemoveCannonBall(true, id, false)
		h.broadcastCannonBall("expire", id, rl.Vector2{}, rl.Vector2{})
	}
	h.resolveCannonBallHits()
	skill.UpdateCannonBursts(true, dt)
}

// readyCannons conta o tempo de todo canhao de pe e devolve os que ficaram
// prontos neste quadro, ja com a recarga armada. Toda a mexida na lista
// acontece aqui dentro, sob cannonsMutex e sem pedir nenhum outro cadeado.
func (h *Host) readyCannons(dt float32) []*cannonPost {
	h.cannonsMutex.Lock()
	defer h.cannonsMutex.Unlock()
	var ready []*cannonPost
	for _, c := range h.liveCannons {
		if c.Destroyed {
			continue
		}
		c.attackTimer -= dt
		if c.attackTimer > 0 {
			continue
		}
		c.attackTimer = CannonCooldown
		ready = append(ready, c)
	}
	return ready
}

// rearmCannon reajusta o gatilho de um canhao (0 = pronto no proximo quadro).
func (h *Host) rearmCannon(c *cannonPost, timer float32) {
	h.cannonsMutex.Lock()
	defer h.cannonsMutex.Unlock()
	c.attackTimer = timer
}

// cannonAim decide para onde este posto atira, e a resposta e SEMPRE alguem:
// o canhao nunca fica calado enquanto houver gente viva no mapa.
//
// Duas miras, e a diferenca entre elas e o que o alcance significa (Gui,
// 28/08/2026 — "assim que a fase comecar, apos o dialogo, os canhoes ja devem
// comecar a lancar a bola de fogo"):
//
//   - DENTRO de CannonRange, a mira e a de sempre: o jogador com MENOS VIDA.
//     E o alcance em que o canhao esta caçando, e caçar e escolher a presa.
//   - FORA dele, o canhao dispara na direcao do jogador mais PROXIMO. Nao e
//     mira, e o corredor sendo varrido: a bola morre por tempo antes de
//     chegar (CannonBallTTL * CannonBallSpeed ~ 4140 px, contra 4000 de
//     alcance), entao ninguem sangra de longe — mas o grupo VE o corredor
//     pegando fogo desde o primeiro quadro da fase, em vez de andar meio
//     caminho no silencio e so entao descobrir do que se trata.
//
// O alcance continua sendo a linha em que se comeca a sangrar. O que ele
// deixou de ser e a linha em que o corredor comeca a existir.
func (h *Host) cannonAim(origin rl.Vector2) (rl.Vector2, bool) {
	if pos, ok := h.weakestPlayerWithinRange(origin, CannonRange); ok {
		return pos, true
	}
	return h.nearestPlayerAnywhere(origin)
}

// nearestPlayerAnywhere devolve a posicao do jogador vivo mais proximo, sem
// limite de distancia. So a DIRECAO dela e usada (ver cannonAim).
func (h *Host) nearestPlayerAnywhere(origin rl.Vector2) (rl.Vector2, bool) {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()

	var bestID string
	var bestPos rl.Vector2
	var bestDist float32
	found := false
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		d := rl.Vector2Distance(origin, pos)
		// Empate resolvido por ID, e nao pela ordem do mapa: dois jogadores a
		// mesma distancia nao podem fazer o canhao mudar de mira a cada quadro.
		if !found || d < bestDist || (d == bestDist && id < bestID) {
			bestID, bestPos, bestDist, found = id, pos, d, true
		}
	}
	return bestPos, found
}

// weakestPlayerWithinRange devolve a posicao do jogador VIVO com menos vida
// dentro do alcance de origin. Mesma logica de weakestPlayerInRange
// (host_sentry_orb.go), com o alcance passado explicitamente em vez de lido de
// um `*entity.Enemy`: o canhao nao e um.
func (h *Host) weakestPlayerWithinRange(origin rl.Vector2, rng float32) (rl.Vector2, bool) {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()

	var bestID string
	var bestPos rl.Vector2
	var bestHP float32
	found := false
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		if rl.Vector2Distance(origin, pos) > rng {
			continue
		}
		if !found || p.Health < bestHP || (p.Health == bestHP && id < bestID) {
			bestID, bestPos, bestHP, found = id, pos, p.Health, true
		}
	}
	return bestPos, found
}

// resolveCannonBallHits estoura as bolas que encostaram em alguem e cobra o
// dano de TODOS os jogadores dentro do estouro.
//
// A area, e nao a primeira pessoa (mudanca de 27/08/2026, a pedido do Gui):
// antes a bola parava no primeiro jogador que encostava e so ele sangrava, e
// dois aliados lado a lado no corredor viam a mesma explosao cobrir os dois
// enquanto so um perdia vida. Um canhao nao acerta uma pessoa, ele estoura.
//
// A explosao acontece ONDE a bola estava, e nao em cima de quem a disparou:
// quem se meteu na frente para proteger o alvo marcado leva o impacto junto,
// e agora leva junto o aliado que ele estava protegendo, se os dois couberem
// no raio. E deliberado — e a mesma razao pela qual um escudo pode ser
// reforcado a tempo mas nao pode ser jogado na frente de outra pessoa.
func (h *Host) resolveCannonBallHits() {
	for _, b := range skill.GetCannonBalls(true) {
		impact, ok := h.cannonBallImpactPoint(b.Position)
		if !ok {
			continue
		}
		// O estouro nasce EM CIMA DE QUEM FOI ATINGIDO, e nao onde a bola
		// estava. Por isso o `false` no Remove: ele deixaria a explosao no
		// lugar da bola, que e justamente o que fazia a onda nao chegar
		// atras.
		skill.RemoveCannonBall(true, b.ID, false)
		skill.AddCannonBurstAt(true, impact)
		h.broadcastCannonBall("impact", b.ID, impact, rl.Vector2{})
		for _, id := range h.playersInCannonBlast(impact) {
			h.applyCannonBallDamage(id)
		}
	}
}

// cannonBallImpactPoint devolve ONDE a bola estoura: a posicao do jogador vivo
// mais proximo que o corpo dela alcancou, ou falso se ela nao encostou em
// ninguem.
//
// O ponto de impacto e o CORPO ATINGIDO, e nao o centro da bola. Vale
// registrar por que nao foi feito do jeito que parecia obvio.
//
// O defeito relatado (Gui, 28/08/2026): "a onda do impacto nao chega nos
// jogadores de tras". A causa era o estouro nascer longe demais — o gatilho
// usava o raio do ESTOURO (630 + 34 = 664 px), entao a bola parava 664 px
// antes do alvo e a explosao, que se abre 630 px dali, mal alcancava quem foi
// atingido e nunca alcancava quem vinha atras. Era tambem o que fazia a bola
// parecer estourar em cadaver: detonando longe do alvo vivo, ela explodia em
// cima do que estivesse por ali, e um corpo caido no corredor e isso.
//
// A saida proposta era so detonar quando o CENTRO da bola encostasse no corpo.
// Conserta o alcance e quebra outras duas coisas: a bola tem 350 px de raio,
// entao ela passaria POR DENTRO do jogador por quase um segundo antes de
// explodir — o fogo atravessa a pessoa e nada acontece —, e a janela de acerto
// cairia para 34 px num tiro mirado ha varios segundos, o que faria quase toda
// bola errar e o corredor virar enfeite.
//
// Entao o gatilho continua sendo o corpo da bola encostando (ela estoura
// quando o fogo toca, que e o que se ve), e o que mudou foi o LUGAR do
// estouro: sobre o jogador. Dali os 630 px se abrem metade para a frente e
// metade para tras, e quem esta atras sangra — que era o pedido.
func (h *Host) cannonBallImpactPoint(ballPos rl.Vector2) (rl.Vector2, bool) {
	touch := skill.CannonBallRadius + cannonBallPlayerRadius

	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()

	var bestID string
	var best rl.Vector2
	var bestDist float32
	found := false
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		d := rl.Vector2Distance(ballPos, pos)
		if d > touch {
			continue
		}
		// O mais proximo do nucleo e quem a bola encostou primeiro. Empate
		// resolvido por ID para o estouro nao mudar de lugar por sorteio de
		// mapa.
		if !found || d < bestDist || (d == bestDist && id < bestID) {
			bestID, best, bestDist, found = id, pos, d, true
		}
	}
	return best, found
}

// playersInCannonBlast devolve todo jogador vivo dentro do estouro: um circulo
// de CannonBlastRadius centrado em origin, que e onde a bola parou.
//
// Circulo, e nao o primeiro corpo tocado (que era o comportamento ate
// 27/08/2026): dois aliados lado a lado viam a mesma explosao cobrir os dois e
// um saia ileso. E circulo simples, e nao a capsula esticada para a frente que
// tentei em 28/08: com o gatilho no centro da bola, a explosao ja nasce em
// cima do alvo e os 630 px alcancam quem esta atras por simetria — a capsula
// era remendo para um gatilho errado.
//
// Ordenado por ID para que duas execucoes com o mesmo estado cobrem o dano na
// mesma ordem — ninguem morre antes de outro por sorteio de mapa.
func (h *Host) playersInCannonBlast(origin rl.Vector2) []string {
	reach := skill.CannonBlastRadius + cannonBallPlayerRadius

	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	var caught []string
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		if rl.Vector2Distance(origin, pos) <= reach {
			caught = append(caught, id)
		}
	}
	sort.Strings(caught)
	return caught
}

// cannonBallPlayerRadius e a meia-largura do jogador para o teste de acerto,
// a mesma constante local que a esfera da sentinela usa (sentryOrbPlayerRadius)
// pela mesma razao: o jogo nao guarda um raio de jogador em lugar nenhum.
const cannonBallPlayerRadius float32 = 34

// applyCannonBallDamage tira a vida, respeitando avatar, invulnerabilidade e
// escudo — a MESMA ordem que checkEnemyPlayerCollisions e
// applySentryOrbDamage usam, de proposito: as tres tem que concordar.
func (h *Host) applyCannonBallDamage(playerID string) {
	if h.Skills.HasAvatar(playerID) || h.IsInvulnerable(playerID) {
		return
	}
	dmg := CannonDamage
	if leftover, hpAfter, had := h.Skills.AbsorbShieldDamage(playerID, dmg); had {
		h.broadcastShieldEvent(playerID, hpAfter)
		// hpAfter zerado com `had` verdadeiro so pode ser ESTE golpe:
		// AbsorbShieldDamage recusa escudo ja quebrado.
		if hpAfter <= 0 {
			h.burstBrokenShield(playerID)
		}
		if leftover <= 0 {
			return
		}
		dmg = leftover
	}

	h.playersMutex.Lock()
	p, ok := h.players[playerID]
	if !ok || p.IsDead {
		h.playersMutex.Unlock()
		return
	}
	p.Health -= dmg
	dead := p.Health <= 0
	if dead {
		h.markPlayerDead(p)
	}
	h.playersMutex.Unlock()

	if dead {
		h.broadcastCombatEvent("death", playerID, "player", 0, "")
		return
	}
	h.broadcastCombatEvent("damage", playerID, "player", dmg, "")
}

// broadcastCannonBall replica um evento de bola de canhao aos clientes. Mesmo
// padrao da esfera da sentinela, mas com direcao em vez de alvo travado: a
// bola nao persegue, entao o cliente so precisa saber para onde ela foi
// lancada para replicar a trajetoria reta sozinho.
func (h *Host) broadcastCannonBall(event, ballID string, pos, dir rl.Vector2) {
	payload := CannonBallPayload{
		EventType: event,
		BallID:    ballID,
		X:         int(pos.X),
		Y:         int(pos.Y),
		DirX:      dir.X,
		DirY:      dir.Y,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Host] falha ao serializar evento de bola de canhao: %v", err)
		return
	}
	h.broadcast(Message{Type: MsgCannonBall, Payload: data})
}
