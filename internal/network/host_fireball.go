package network

import (
	"encoding/json"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// HandleSkill activates the fireball skill for playerID aimed at world target.
func (h *Host) HandleSkill(playerID string, tx, ty int) {
	h.playersMutex.RLock()
	p, ok := h.players[playerID]
	h.playersMutex.RUnlock()
	if !ok || p.IsDead {
		return
	}
	start := rl.NewVector2(float32(p.X), float32(p.Y))
	dir := rl.NewVector2(float32(tx)-start.X, float32(ty)-start.Y)
	if rl.Vector2Length(dir) < 1 {
		return
	}
	entity.SpawnFireball(h.EntityManager, playerID, start, dir)
}

// SetCollisionRects provides obstacle rectangles used by skill projectiles.
func (h *Host) SetCollisionRects(rects []rl.Rectangle) {
	h.collisionRects = rects
}

// handleFireballTick advances fire projectiles and broadcasts the resulting
// fire visuals to clients. Fire damage only affects monsters (handled in
// entity.StepFireballs); players (self and allies) take NO fire damage.
func (h *Host) handleFireballTick(dt float32) {
	impacts := entity.StepFireballs(h.EntityManager, h.collisionRects, dt)
	for _, pos := range impacts {
		h.broadcastFireEvent("fire_explosion", pos)
	}
}

// broadcastFireEvent sends a fire visual event (explosion) to all clients.
func (h *Host) broadcastFireEvent(eventType string, pos rl.Vector2) {
	payload := FireEventPayload{
		EventType: eventType,
		X:         int(pos.X),
		Y:         int(pos.Y),
		Radius:    entity.FireballExplosionRadius,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.broadcast(Message{Type: MsgFireEvent, Payload: data})
}
