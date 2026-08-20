package network

import (
	"encoding/json"
	"log"

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

// handleCannonTick e chamado uma vez por quadro de UpdateSimulation.
func (h *Host) handleCannonTick(dt float32) {
	// 1. Cada canhao pronto e vivo dispara no jogador com MENOS VIDA ao
	//    alcance.
	for _, c := range h.liveCannons {
		if c.Destroyed {
			continue
		}
		c.attackTimer -= dt
		if c.attackTimer > 0 {
			continue
		}
		targetPos, ok := h.weakestPlayerWithinRange(c.Position, CannonRange)
		if !ok {
			continue
		}
		c.attackTimer = CannonCooldown
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

// resolveCannonBallHits aplica o dano das bolas que encostaram em algum
// jogador vivo — qualquer um, nao so o mirado no lancamento (ver o comentario
// do arquivo).
func (h *Host) resolveCannonBallHits() {
	for _, b := range skill.GetCannonBalls(true) {
		playerID, hit := h.playerHitByCannonBall(b)
		if !hit {
			continue
		}
		skill.RemoveCannonBall(true, b.ID, true)
		h.broadcastCannonBall("impact", b.ID, b.Position, rl.Vector2{})
		h.applyCannonBallDamage(playerID)
	}
}

// playerHitByCannonBall devolve o primeiro jogador vivo que a bola encosta.
func (h *Host) playerHitByCannonBall(b *skill.CannonBall) (string, bool) {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		if rl.Vector2Distance(b.Position, pos) <= skill.CannonBallRadius+cannonBallPlayerRadius {
			return id, true
		}
	}
	return "", false
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
