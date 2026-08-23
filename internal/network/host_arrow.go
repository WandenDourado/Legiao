package network

import (
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleArrowTick advances the Arqueiro's arrows on the host: moves them,
// resolves single-target enemy hits and obstacle impacts, and removes enemies
// that died. Enemy health itself is synced to clients via the regular enemy
// snapshot broadcast, so no per-hit event is required.
func (h *Host) handleArrowTick(dt float32) {
	dead := skill.StepArrows(
		h.Skills, h.EntityManager.GetAllEnemies(), h.solid, dt)
	for _, id := range dead {
		h.EntityManager.RemoveEnemy(id)
		h.broadcastCombatEvent("death", id, "enemy", 0, "")
	}
}

// broadcastArrowVolley tells clients to replicate an arrow volley (visuals
// only; damage stays host-authoritative).
func (h *Host) broadcastArrowVolley(ownerID string, origin, dir rl.Vector2) {
	payload := ArrowVolleyPayload{
		OwnerID: ownerID,
		X:       int(origin.X),
		Y:       int(origin.Y),
		DirX:    dir.X,
		DirY:    dir.Y,
	}
	h.broadcast(Message{Type: MsgArrowVolley, Payload: MustMarshal(payload)})
}
