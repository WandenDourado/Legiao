package network

// Host-side simulation for the four ultimate skills: meteor rain spawning and
// impacts, angelic resurrection + healing, celestial arrow piercing, and
// avatar anchoring/expiry. Invincibility itself is checked where enemy damage
// is applied (checkEnemyPlayerCollisions).

import (
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleUltimateTick advances every ultimate on the host for this frame.
func (h *Host) handleUltimateTick(dt float32) {
	h.tickMeteors(dt)
	h.tickAngelics(dt)
	h.tickCelestials(dt)
	h.tickAvatars(dt)
}

// tickMeteors spawns rain meteors across the map (broadcasting each so
// clients replicate the fall) and resolves impacts.
func (h *Host) tickMeteors(dt float32) {
	b := h.EntityManager.WorldBounds
	spawned := skill.StepMeteorRains(h.Skills, b.Width, b.Height, dt)
	for _, target := range spawned {
		h.broadcastUltimate("meteor_rain", "", target, rl.Vector2{})
	}
	dead := skill.StepMeteors(h.Skills, h.EntityManager.GetAllEnemies(), dt)
	for _, id := range dead {
		h.EntityManager.RemoveEnemy(id)
		h.broadcastCombatEvent("death", id, "enemy", 0, "")
	}
}

// tickAngelics fires the one-time resurrection when an altar appears,
// applies continuous healing and syncs it via heal events. The altar itself
// is FIXED where it was consecrated (no anchoring).
func (h *Host) tickAngelics(dt float32) {
	// One-time resurrection when an altar appears.
	if len(h.Skills.ConsumeAngelicResurrections()) > 0 {
		h.resurrectAllDead()
	}

	events := h.Skills.StepAngelics(dt, h.allyHealTargets())
	for _, ev := range events {
		h.playersMutex.Lock()
		if p, ok := h.players[ev.PlayerID]; ok {
			p.Health += ev.Amount
			if p.Health > p.MaxHealth {
				p.Health = p.MaxHealth
			}
			h.playersMutex.Unlock()
			h.broadcastCombatEvent("heal", ev.PlayerID, "player", p.Health, "")
			continue
		}
		h.playersMutex.Unlock()
	}
}

// resurrectAllDead revives every dead player IN PLACE with a fraction of max
// health and broadcasts the respawn events clients already understand.
func (h *Host) resurrectAllDead() {
	type revived struct {
		id     string
		health float32
	}
	h.playersMutex.Lock()
	list := make([]revived, 0)
	for id, p := range h.players {
		if p.IsDead {
			p.IsDead = false
			p.Health = skill.AngelicResurrectHealthPct * p.MaxHealth
			list = append(list, revived{id: id, health: p.Health})
		}
	}
	h.playersMutex.Unlock()
	for _, r := range list {
		h.broadcastCombatEvent("respawn", r.id, "player", r.health, "")
	}
}

// tickCelestials advances the piercing arrows and removes enemies they kill.
func (h *Host) tickCelestials(dt float32) {
	dead := skill.StepCelestials(h.Skills, h.EntityManager.GetAllEnemies(), dt)
	for _, id := range dead {
		h.EntityManager.RemoveEnemy(id)
		h.broadcastCombatEvent("death", id, "enemy", 0, "")
	}
}

// tickAvatars keeps each avatar glued to its owner and advances its life.
func (h *Host) tickAvatars(dt float32) {
	h.playersMutex.RLock()
	for id, p := range h.players {
		h.Skills.SetAvatarAnchor(id, rl.NewVector2(float32(p.X), float32(p.Y)))
	}
	h.playersMutex.RUnlock()
	h.Skills.UpdateAvatars(dt)
}

// broadcastUltimate tells clients to replicate an ultimate visual event.
func (h *Host) broadcastUltimate(skillID, ownerID string, origin, dir rl.Vector2) {
	payload := UltimateEventPayload{
		Skill:   skillID,
		OwnerID: ownerID,
		X:       int(origin.X),
		Y:       int(origin.Y),
		DirX:    dir.X,
		DirY:    dir.Y,
	}
	h.broadcast(Message{Type: MsgUltimate, Payload: MustMarshal(payload)})
}
