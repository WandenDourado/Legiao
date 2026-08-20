package network

import (
	"encoding/json"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

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
	if entity.CharacterType(p.Character) != entity.CharMago {
		return // only the Mago may cast Bola de Fogo (fireball)
	}
	start := rl.NewVector2(float32(p.X), float32(p.Y))
	dir := rl.NewVector2(float32(tx)-start.X, float32(ty)-start.Y)
	if rl.Vector2Length(dir) < 1 {
		return
	}
	skill.SpawnFireball(h.Skills, playerID, start, dir)
}

// SetCollisionRects provides obstacle rectangles used by skill projectiles.
func (h *Host) SetCollisionRects(rects []rl.Rectangle) {
	h.collisionRects = rects
}

// handleFireballTick advances fire projectiles and broadcasts the resulting
// fire visuals to clients. Fire damage only affects monsters (handled in
// skill.StepFireballs); players (self and allies) take NO fire damage.
func (h *Host) handleFireballTick(dt float32) {
	impacts, dead := skill.StepFireballs(
		h.Skills, h.EntityManager.GetAllEnemies(), h.collisionRects, dt)
	for _, id := range dead {
		h.EntityManager.RemoveEnemy(id)
	}
	for _, pos := range impacts {
		h.broadcastFireEvent("fire_explosion", pos)
	}

	// The burning ground the explosion leaves behind keeps damaging monsters
	// standing in it. The enemy list is read again on purpose: the impact
	// above may have just removed some of them.
	for _, ev := range skill.StepFireGrounds(h.Skills, h.EntityManager.GetAllEnemies(), dt) {
		if ev.Died {
			h.EntityManager.RemoveEnemy(ev.EnemyID)
			h.broadcastCombatEvent("death", ev.EnemyID, "enemy", 0, "")
			continue
		}
		h.broadcastCombatEvent("damage", ev.EnemyID, "enemy", ev.Damage, "")
	}
}

// broadcastFireCast replicates a fireball LAUNCH on clients: origin +
// direction so they can spawn and animate the traveling fireball.
func (h *Host) broadcastFireCast(ownerID string, origin, dir rl.Vector2) {
	payload := FireEventPayload{
		EventType: "cast",
		X:         int(origin.X),
		Y:         int(origin.Y),
		OwnerID:   ownerID,
		DirX:      dir.X,
		DirY:      dir.Y,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.broadcast(Message{Type: MsgFireEvent, Payload: data})
}

// broadcastFireEvent sends a fire visual event (explosion) to all clients.
func (h *Host) broadcastFireEvent(eventType string, pos rl.Vector2) {
	payload := FireEventPayload{
		EventType: eventType,
		X:         int(pos.X),
		Y:         int(pos.Y),
		Radius:    skill.FireballExplosionRadius,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.broadcast(Message{Type: MsgFireEvent, Payload: data})
}
