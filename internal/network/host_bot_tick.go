package network

// The bot AI tick: build each bot's View, ask its Brain what to do, and
// apply the Intent through the SAME gates a human's input goes through —
// HandleAttack and HandleSkillMessage — never a parallel path (plan §5).

import (
	"github.com/WandenDourado/Legiao/internal/ability"
	"github.com/WandenDourado/Legiao/internal/bot"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

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
	h.updateAdvanceDir(dt)

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

// updateAdvanceDir recomputes the whole party's smoothed heading from the
// living humans' current velocity, once per tick — every bot that frame
// reads the same value (host.advanceDir), so the escort re-forms around one
// consistent front instead of each bot picking its own idea of "forward"
// (doc/plan_avanco_bots_e_gargula.md §A3, R3).
//
// Bots are excluded from the average for the same reason HumanCentre
// excludes them (buildBotView): an advancing bot must not steer the very
// heading the rest of the escort re-forms around.
func (h *Host) updateAdvanceDir(dt float32) {
	h.playersMutex.RLock()
	var velSum rl.Vector2
	n := 0
	for pid, p := range h.players {
		if isBotID(pid) || p.IsDead {
			continue
		}
		velSum = rl.Vector2Add(velSum, rl.NewVector2(p.VelX, p.VelY))
		n++
	}
	h.playersMutex.RUnlock()

	if n == 0 || (velSum.X == 0 && velSum.Y == 0) {
		// Nobody alive to have a heading, or the party stopped moving:
		// keep the LAST known direction (plan's "parado o grupo, a
		// formacao usa a ultima direcao conhecida") instead of collapsing
		// to zero, which would pull every ranged bot on top of the humans.
		return
	}
	target := rl.Vector2Normalize(rl.Vector2Scale(velSum, 1/float32(n)))
	if h.advanceDir.X == 0 && h.advanceDir.Y == 0 {
		h.advanceDir = target
		return
	}
	t := bot.AdvanceDirSmoothing * dt
	if t > 1 {
		t = 1
	}
	h.advanceDir = rl.Vector2Normalize(rl.Vector2Lerp(h.advanceDir, target, t))
}

func (h *Host) tickOneBot(id string, dt float32) {
	view, rt, self, ok := h.buildBotView(id, dt)
	if !ok {
		return
	}
	// The Brain is ALWAYS called (InPortal is the one freeze — buildBotView
	// already filtered that out above). It used to be skipped whenever the
	// portal was active, on the assumption the field is always empty by
	// then — false for a garrison map: world_03 has no enemy_spawn_*
	// markers, so WaveState.Total is 0 and the portal is unlocked from the
	// first frame while all 83 monsters are still in the field
	// (doc/plan_avanco_bots_e_gargula.md §A2, cause 2). The portal is now
	// just a destination the Brain itself may pick, via travelDest
	// (steering.go) — never a shortcut around it.
	intent := rt.brain.Think(view)
	h.applyBotIntent(id, rt, self, intent, dt)
}

// buildBotView gathers everything a Brain is allowed to see. Reads
// h.players under RLock and releases it before returning — nothing here
// may be called while any lock this package takes is held (plan §9.4:
// playersMutex before cdMutex, never the reverse, and never held into
// HandleAttack/HandleSkillMessage). self is returned BY VALUE so the
// caller never touches PlayerState fields outside a lock.
func (h *Host) buildBotView(id string, dt float32) (bot.View, *botRuntime, PlayerState, bool) {
	// A separate lock from playersMutex (host_bot_portal.go) — safe to read
	// before taking playersMutex below, no ordering conflict.
	portalRects, portalRectsActive := partyPortalRectsSnapshot()

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
	var centreSum, humanCentreSum rl.Vector2
	living, humanLiving := 0, 0
	humansAtPortal := false
	for pid, p := range h.players {
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		if !p.IsDead {
			centreSum = rl.Vector2Add(centreSum, pos)
			living++
			// HumanCentre excludes every bot, this one included — a bot
			// that gets ahead of the group must never pull its own
			// "follow the party" reference forward with it (plan
			// doc/plan_avanco_bots_e_gargula.md §A2, cause 2).
			if !isBotID(pid) {
				humanCentreSum = rl.Vector2Add(humanCentreSum, pos)
				humanLiving++
				// HumansAtPortal (plan §A2, cause 4): a human already
				// InPortal counts directly — host_portal_presence.go ran
				// the same foot-box test this same tick, no need to redo
				// it — otherwise a live human within PortalEscortRadius of
				// any open portal rectangle also counts.
				if !humansAtPortal && portalRectsActive {
					if p.InPortal || nearAnyPortalRect(pos, portalRects) {
						humansAtPortal = true
					}
				}
			}
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
	var humanCentre rl.Vector2
	hasHumans := humanLiving > 0
	if hasHumans {
		humanCentre = rl.Vector2Scale(humanCentreSum, 1/float32(humanLiving))
	}

	char := rt.character
	enemies := h.EntityManager.GetAllEnemies()
	foes := make([]bot.Foe, 0, len(enemies))
	for _, e := range enemies {
		if !e.IsActive {
			continue
		}
		// A sentinela agora entra em Foes (antes era descartada aqui): sem
		// isso o Arqueiro nunca a enxerga para a supresma poder mata-la
		// (plan doc/plan_avanco_bots_e_gargula.md §B4). IsSentry e o que
		// mantém toda seleção de alvo comum longe dela — ver Foe.IsSentry.
		foes = append(foes, bot.Foe{
			ID:          e.ID,
			Pos:         e.Position,
			Vel:         e.Velocity,
			Health:      e.Health,
			MaxHealth:   e.MaxHealth,
			AttackRange: e.AttackRange,
			Radius:      e.Radius,
			IsBoss:      e.Type == entity.EnemyTypeDarkLady,
			IsSentry:    e.Type == entity.EnemyTypeCastleSentry,
			HitCentre:   e.HitCenter(),
		})
	}

	// Sentries are fixed posts, not the horde: counting them here would
	// inflate every EnemiesLeft-gated decision (Chuva de Meteoros, Legião
	// Espectral, groupIsFalling's "horde still in the field") every time a
	// map with sentries loads, which has nothing to do with the actual
	// wave.
	enemiesLeft := 0
	for _, f := range foes {
		if !f.IsSentry {
			enemiesLeft++
		}
	}
	if ws := GetWaveState(); ws.Remaining > enemiesLeft {
		enemiesLeft = ws.Remaining
	}

	portalCentre, portalActive := partyPortal()

	primaryID := ability.PrimaryAbilityOf(char)
	ultimateID := ability.UltimateAbilityOf(char)

	// UltimateRange only means anything for the Arqueiro today
	// (skill.CelestialRange, the one ultimate with a projectile reach a bot
	// needs to plan movement around) — zero for every other class, which
	// none of their brains read (plan §B4, contract point 2: "alcance da
	// suprema chega pela View").
	var ultimateRange float32
	if ultimateID == "celestial_arrows" {
		ultimateRange = skill.CelestialRange
	}

	v := bot.View{
		Self: bot.Ally{
			ID: id, Char: char, Pos: rl.NewVector2(float32(selfSnapshot.X), float32(selfSnapshot.Y)),
			Health: selfSnapshot.Health, MaxHealth: selfSnapshot.MaxHealth, IsDead: selfSnapshot.IsDead, IsBot: true,
		},
		Allies:         allies,
		Foes:           foes,
		Bounds:         h.WorldBounds,
		PartyCentre:    partyCentre,
		HumanCentre:    humanCentre,
		HasHumans:      hasHumans,
		AdvanceDir:     h.advanceDir,
		Portal:         portalCentre,
		PortalActive:   portalActive,
		HumansAtPortal: humansAtPortal,
		EnemiesLeft:    enemiesLeft,
		PrimaryReady:   primaryID == "" || h.skillOffCooldown(id, primaryID),
		UltimateReady:  ultimateID != "" && h.skillOffCooldown(id, ultimateID) && h.skillUnlocked(id, ultimateID, char),
		UltimateRange:  ultimateRange,
		Dt:             dt,
	}
	return v, rt, selfSnapshot, true
}

// nearAnyPortalRect reports whether pos is within bot.PortalEscortRadius of
// the CENTRE of any of rects — the same reference point partyPortal() (this
// package) already uses for "where a bot with nothing better to aim at
// should walk", so a human "near the portal" and a bot's own portal
// destination agree on what "the portal" means as a point.
func nearAnyPortalRect(pos rl.Vector2, rects []rl.Rectangle) bool {
	for _, r := range rects {
		centre := rl.NewVector2(r.X+r.Width/2, r.Y+r.Height/2)
		if rl.Vector2Distance(pos, centre) <= bot.PortalEscortRadius {
			return true
		}
	}
	return false
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
