package network

// The bot AI tick: build each bot's View, ask its Brain what to do, and
// apply the Intent through the SAME gates a human's input goes through —
// HandleAttack and HandleSkillMessage — never a parallel path (plan §5).

import (
	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/bot"
	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// tickBots drives every bot for one simulation frame. Called from
// UpdateSimulation, after updateWaves and before EntityManager.UpdateAll:
// a bot's movement this tick already counts for which player an enemy
// targets in the SAME tick, exactly like a human's input, which also
// arrived before the tick began (plan §13.4).
func (h *Host) tickBots(dt float32) {
	// One search budget shared by every bot this frame — reset here, not
	// per-bot, or the budget would effectively multiply by however many
	// bots are in the party (doc/plan_navegacao_bots_monstros.md §3.4).
	if h.EntityManager != nil && h.EntityManager.Nav != nil {
		h.EntityManager.Nav.ResetFrameBudget()
	}

	h.playersMutex.RLock()
	ids := make([]string, 0, len(h.bots))
	for id := range h.bots {
		ids = append(ids, id)
	}
	h.playersMutex.RUnlock()

	for _, id := range ids {
		h.tickOneBot(id, dt)
	}
}

func (h *Host) tickOneBot(id string, dt float32) {
	view, rt, self, ok := h.buildBotView(id, dt)
	if !ok {
		return
	}
	var intent bot.Intent
	if view.PortalActive {
		// The door is open and the party is heading through it: walking to
		// it beats anything combat-related, and the field is empty by the
		// time a portal opens anyway (portalReveal only completes once the
		// run is cleared — game/portal.go). No Push: separation is exactly
		// what stops everyone from fitting in the same small rectangle.
		intent.Dest, intent.HasDest = view.Portal, true
	} else {
		intent = rt.brain.Think(view)
	}
	h.applyBotIntent(id, rt, self, intent, dt)
}

// buildBotView gathers everything a Brain is allowed to see. Reads
// h.players under RLock and releases it before returning — nothing here
// may be called while any lock this package takes is held (plan §9.4:
// playersMutex before cdMutex, never the reverse, and never held into
// HandleAttack/HandleSkillMessage). self is returned BY VALUE so the
// caller never touches PlayerState fields outside a lock.
func (h *Host) buildBotView(id string, dt float32) (bot.View, *botRuntime, PlayerState, bool) {
	h.playersMutex.RLock()
	rt, hasRuntime := h.bots[id]
	self, hasSelf := h.players[id]
	if !hasRuntime || !hasSelf || self.IsDead || self.InPortal {
		// InPortal: this bot just vanished into the portal it is standing
		// in and freezes there — no Intent, no movement, no attack — until
		// the whole party is inside too (or the flag is cleared by travel/
		// reset). This is what actually frees the small rectangle for the
		// rest of the group instead of everyone orbiting it forever.
		h.playersMutex.RUnlock()
		return bot.View{}, nil, PlayerState{}, false
	}
	selfSnapshot := *self
	allies := make([]bot.Ally, 0, len(h.players)-1)
	var centreSum rl.Vector2
	living := 0
	for pid, p := range h.players {
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		if !p.IsDead {
			centreSum = rl.Vector2Add(centreSum, pos)
			living++
		}
		if pid == id {
			continue
		}
		allies = append(allies, bot.Ally{
			ID:        pid,
			Char:      entity.CharacterType(p.Character),
			Pos:       pos,
			Health:    p.Health,
			MaxHealth: p.MaxHealth,
			IsDead:    p.IsDead,
			IsBot:     isBotID(pid),
		})
	}
	h.playersMutex.RUnlock()

	partyCentre := rl.NewVector2(float32(selfSnapshot.X), float32(selfSnapshot.Y))
	if living > 0 {
		partyCentre = rl.Vector2Scale(centreSum, 1/float32(living))
	}

	char := rt.character
	enemies := h.EntityManager.GetAllEnemies()
	foes := make([]bot.Foe, 0, len(enemies))
	for _, e := range enemies {
		if !e.IsActive || e.Type == entity.EnemyTypeCastleSentry {
			continue
		}
		foes = append(foes, bot.Foe{
			ID:          e.ID,
			Pos:         e.Position,
			Vel:         e.Velocity,
			Health:      e.Health,
			MaxHealth:   e.MaxHealth,
			AttackRange: e.AttackRange,
			Radius:      e.Radius,
			IsBoss:      e.Type == entity.EnemyTypeDarkLady,
		})
	}

	enemiesLeft := len(foes)
	if ws := GetWaveState(); ws.Remaining > enemiesLeft {
		enemiesLeft = ws.Remaining
	}

	portalCentre, portalActive := partyPortal()

	primaryID := ability.PrimaryAbilityOf(char)
	ultimateID := ability.UltimateAbilityOf(char)

	v := bot.View{
		Self: bot.Ally{
			ID: id, Char: char, Pos: rl.NewVector2(float32(selfSnapshot.X), float32(selfSnapshot.Y)),
			Health: selfSnapshot.Health, MaxHealth: selfSnapshot.MaxHealth, IsDead: selfSnapshot.IsDead, IsBot: true,
		},
		Allies:        allies,
		Foes:          foes,
		Bounds:        h.WorldBounds,
		PartyCentre:   partyCentre,
		Portal:        portalCentre,
		PortalActive:  portalActive,
		EnemiesLeft:   enemiesLeft,
		PrimaryReady:  primaryID == "" || h.skillOffCooldown(id, primaryID),
		UltimateReady: ultimateID != "" && h.skillOffCooldown(id, ultimateID) && h.skillUnlocked(id, ultimateID, char),
		Dt:            dt,
	}
	return v, rt, selfSnapshot, true
}

// skillOffCooldown is a read-only check, unlike beginSkillCooldown: the bot
// needs to know whether a cast is worth attempting before it commits to
// one, without consuming a charge just by looking.
func (h *Host) skillOffCooldown(playerID, skillID string) bool {
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()
	return h.skillCooldowns[playerID+"|"+skillID] <= 0
}

// applyBotIntent turns a decided Intent into the same calls a human's
// client would have produced: movement resolved against the map, then
// HandleAttack / HandleSkillMessage — never a bot-only code path.
func (h *Host) applyBotIntent(id string, rt *botRuntime, self PlayerState, intent bot.Intent, dt float32) {
	pos := rl.NewVector2(float32(self.X), float32(self.Y))
	move := desiredBotMove(rt, h.EntityManager.Nav, pos, intent, dt)
	move = unstickMove(rt, pos, move, dt)
	if move.X != 0 || move.Y != 0 {
		delta := rl.Vector2Scale(move, botSpeed*dt)
		pos = resolveBotMove(rt, pos, delta, rt.character, h.EntityManager.Solid, h.EntityManager.Nav)
	}
	// After the step resolves, not before: arenaLocked arms on actually
	// BEING inside the zone, the same rule arenaGate.returnLocked uses for
	// the local human (doc/tilemap.md "Arena de mão única").
	pos = applyArenaLock(rt, rt.character, pos)
	row, frame := stepBotAnimation(rt, rt.character, move, dt)

	var published PlayerState
	h.playersMutex.Lock()
	if p, ok := h.players[id]; ok && !p.IsDead {
		p.X, p.Y = int(pos.X), int(pos.Y)
		p.CurrentRow, p.CurrentFrame = row, frame
		p.VelX, p.VelY = move.X*botSpeed, move.Y*botSpeed
		p.IsSprinting = false
		published = *p
	}
	h.playersMutex.Unlock()

	// The host's own screen draws remote players (bots included) from
	// RemotePlayers, which BroadcastStateUpdate otherwise only refreshes at
	// SnapshotHz (20Hz) — plenty for the network, but a bot standing next to
	// a human every frame would visibly stutter at that rate. This is a
	// LOCAL write only, same as UpdatePlayerState always was; tickBots never
	// broadcasts on its own (host.go's snapshot timer still owns that).
	if published.PlayerID != "" {
		UpdatePlayerState(published)
	}

	if intent.Attack != nil {
		h.HandleAttack(id, intent.Attack.X, intent.Attack.Y)
	}
	if intent.Skill != nil {
		h.HandleSkillMessage(id, intent.Skill.SkillID, intent.Skill.Aim.X, intent.Skill.Aim.Y)
	}
}
