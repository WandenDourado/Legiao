package network

import (
	"encoding/json"
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A habilidade da Gargula Sentinela do mapa 4.
//
// Ela nao passa pelo caminho normal de ataque de monstro (`AttackDamage`
// aplicado por proximidade em checkEnemyPlayerCollisions) por um motivo
// simples: aquele caminho tira vida no instante em que o inimigo "ataca", e a
// sentinela tem alcance 1900. O jogador levaria dano do outro lado do saguao
// sem nada atravessar a tela. A esfera existe para que o golpe tenha VIAGEM -
// tempo de ver, tempo de correr.
//
// Por isso a sentinela e filtrada de checkEnemyPlayerCollisions: o dano dela
// chega quando a esfera encosta, e so ai.

// handleSentryOrbTick e chamado uma vez por quadro de UpdateSimulation.
//
// attackedEnemies e o MESMO conjunto que o EntityManager.UpdateAll devolve e
// que checkEnemyPlayerCollisions consome. Reaproveita-lo em vez de manter um
// relogio proprio e o que garante que a sentinela dispare no ritmo do
// `AttackCooldown` da EnemyDef, e nao num numero paralelo que envelhece.
func (h *Host) handleSentryOrbTick(dt float32, attackedEnemies map[string]bool) {
	targets := h.livingPlayerPositions()

	// 1. Cada sentinela pronta lanca uma esfera no jogador com MENOS VIDA —
	// uma por vez (doc/plan_avanco_bots_e_gargula.md §B2): com alcance
	// global e uma esfera lenta (SentryOrbSpeed 300), a cadencia normal de
	// AttackCooldown sozinha empilharia uma dezena de esferas perseguindo o
	// mesmo jogador antes da primeira chegar.
	//
	// So a LANCADA passa pela porta de despertar (sentry_wake.go): antes de o
	// grupo alcancar o degrau que a fase declara, a gargula esta em campo, e
	// desenhada e pode morrer — ela so nao atira. O passo 2 corre sempre, ou
	// uma esfera ja no ar ficaria congelada no meio da tela.
	if h.sentriesMayFire() {
		h.launchSentryOrbs(attackedEnemies)
	}

	// 2. Avanca as esferas e resolve o que elas encostam.
	for _, id := range skill.StepSentryOrbs(dt, targets) {
		// Expirou por tempo: some sem estouro, porque nao acertou nada.
		skill.RemoveSentryOrb(true, id, false)
		h.broadcastSentryOrb("expire", id, "", rl.Vector2{}, 0)
	}
	h.resolveSentryOrbHits()
	skill.UpdateSentryBursts(true, dt)
}

// launchSentryOrbs da a cada gargula pronta — e sem esfera no ar — um alvo e
// uma esfera. Metade do passo 1 de handleSentryOrbTick, separada so para a
// porta de despertar caber numa linha la em cima.
func (h *Host) launchSentryOrbs(attackedEnemies map[string]bool) {
	for _, e := range h.EntityManager.GetAllEnemies() {
		if e.Type != entity.EnemyTypeCastleSentry || !e.IsActive {
			continue
		}
		if !attackedEnemies[e.ID] {
			continue
		}
		if skill.SentryHasLiveOrb(true, e.ID) {
			continue
		}
		targetID, targetPos, ok := h.weakestPlayerInRange(e)
		if !ok {
			continue
		}
		// Sai do peito da criatura, e nao dos pes: Position e a ancora no chao
		// (FootLine), entao lancar dali faria a esfera nascer no pedestal.
		origin := e.HitCenter()
		ttl := sentryOrbTTLFor(origin, targetPos)
		id := skill.SpawnSentryOrb(true, "", e.ID, targetID, origin, targetPos, ttl)
		h.broadcastSentryOrb("cast", id, targetID, origin, ttl)
	}
}

// livingPlayerPositions monta o mapa player_id -> posicao dos vivos, que e o
// que a perseguicao consome. Levantado UMA vez por quadro: com varias esferas
// em campo, refazer o mapa dentro do laco seria uma copia por projetil.
func (h *Host) livingPlayerPositions() map[string]rl.Vector2 {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	out := make(map[string]rl.Vector2, len(h.players))
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		out[id] = rl.NewVector2(float32(p.X), float32(p.Y))
	}
	return out
}

// weakestPlayerInRange devolve o jogador VIVO com menos vida dentro do alcance
// da sentinela.
//
// Menos vida, e nao mais perto: a sentinela e uma execucao, nao uma torre. Ela
// pune o time que deixou alguem cair e nao curou. O alvo e travado aqui, no
// lancamento, e a esfera nao re-mira depois - ver o comentario de leitura em
// skill/sentry_orb.go.
//
// Empate resolve pelo menor ID para ser deterministico: sem isso, a ordem de
// iteracao de mapa em Go faria a sentinela escolher alvos diferentes entre
// duas partidas com o mesmo estado.
func (h *Host) weakestPlayerInRange(e *entity.Enemy) (string, rl.Vector2, bool) {
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
		if rl.Vector2Distance(e.Position, pos) > e.AttackRange+e.Radius {
			continue
		}
		if !found || p.Health < bestHP || (p.Health == bestHP && id < bestID) {
			bestID, bestPos, bestHP, found = id, pos, p.Health, true
		}
	}
	return bestID, bestPos, found
}

// resolveSentryOrbHits aplica o dano das esferas que encostaram num jogador.
//
// Passa pelas MESMAS portas que o golpe corpo-a-corpo de qualquer monstro -
// Avatar Divino, janela de invulnerabilidade e Escudo Sagrado - de proposito.
// Uma magia nova que ignorasse o escudo da Paladina nao seria dificuldade, e
// sim um buraco na regra.
func (h *Host) resolveSentryOrbHits() {
	for _, o := range skill.GetSentryOrbs(true) {
		playerID, hit := h.playerHitBySentryOrb(o)
		if !hit {
			continue
		}
		skill.RemoveSentryOrb(true, o.ID, true)
		h.broadcastSentryOrb("impact", o.ID, "", o.Position, 0)
		h.applySentryOrbDamage(playerID)
	}
}

// playerHitBySentryOrb devolve o jogador que a esfera encostou.
func (h *Host) playerHitBySentryOrb(o *skill.SentryOrb) (string, bool) {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		if rl.Vector2Distance(o.Position, pos) <= skill.SentryOrbRadius+sentryOrbPlayerRadius {
			return id, true
		}
	}
	return "", false
}

// sentryOrbPlayerRadius e a meia-largura do jogador para o teste de acerto da
// esfera. O jogo nao guarda um raio de jogador em lugar nenhum - o movimento
// usa a pegada retangular -, entao este numero e local e explicito em vez de
// escondido dentro da conta.
const sentryOrbPlayerRadius float32 = 34

// applySentryOrbDamage tira a vida, respeitando avatar, invulnerabilidade e
// escudo, e anuncia o resultado. E uma copia deliberada da ordem que
// checkEnemyPlayerCollisions usa: as duas tem que concordar, e concordar por
// leitura e mais seguro que por abstracao apressada.
func (h *Host) applySentryOrbDamage(playerID string) {
	if h.Skills.HasAvatar(playerID) || h.IsInvulnerable(playerID) {
		return
	}
	dmg := entity.GetEnemyDef(entity.EnemyTypeCastleSentry).AttackDamage
	if leftover, hpAfter, had := h.Skills.AbsorbShieldDamage(playerID, dmg); had {
		h.broadcastShieldEvent(playerID, hpAfter)
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

// sentryOrbTTLFor is how long an orb launched from origin toward target
// should live: enough to reach almost anywhere across a map this size, with
// slack for the target moving and for the turn-rate-limited chase
// (skill.SentryOrbTurnRate) catching up. The 1.5x and +2s are the plan's
// own numbers (doc/plan_avanco_bots_e_gargula.md §B2); SentryOrbMaxTTL caps
// a wildly out-of-range shot from lingering forever. The slowness is
// DELIBERATE (skill/sentry_orb.go's own comment already defends it): from
// far away the orb becomes a slow, visible inevitability crossing the
// screen — that reads as dread, not as a bug.
func sentryOrbTTLFor(origin, target rl.Vector2) float32 {
	dist := rl.Vector2Distance(origin, target)
	ttl := dist/skill.SentryOrbSpeed*1.5 + 2
	if ttl > skill.SentryOrbMaxTTL {
		ttl = skill.SentryOrbMaxTTL
	}
	return ttl
}

// broadcastSentryOrb replica um evento de esfera aos clientes.
//
// Padrao de sincronia mais barato que funciona (ver a skill): UM evento de
// nascimento carregando origem + id do alvo, e o cliente refaz a perseguicao
// sozinho contra a posicao replicada daquele jogador. Nao da para usar so
// origem + direcao como a bola de fogo, porque a trajetoria depende de para
// onde o alvo correu; e nao vale um array novo no snapshot, porque o visual
// nao precisa ser exato - vida e morte continuam vindo dos eventos do host.
//
// ttl so importa no evento "cast": sem ele o cliente criaria a esfera com o
// SentryOrbTTL padrao (9s) e a podaria antes de uma esfera de alcance global
// completar a viagem real, sumindo da tela sem o evento de impacto nunca ter
// chegado.
func (h *Host) broadcastSentryOrb(event, orbID, targetID string, pos rl.Vector2, ttl float32) {
	payload := SentryOrbPayload{
		EventType: event,
		OrbID:     orbID,
		TargetID:  targetID,
		X:         int(pos.X),
		Y:         int(pos.Y),
		TTL:       ttl,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Host] falha ao serializar evento de esfera da sentinela: %v", err)
		return
	}
	h.broadcast(Message{Type: MsgSentryOrb, Payload: data})
}
