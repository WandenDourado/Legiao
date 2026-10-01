package network

// Host-side handling for the Paladina's shield: anchor sync, animation and
// removal on death. Damage absorption itself happens where enemy melee damage
// is applied (checkEnemyPlayerCollisions), before health is reduced.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleShieldTick keeps every shield glued to its owner, advances the shield
// animations, and drops shields whose owners died or disconnected.
func (h *Host) handleShieldTick(dt float32) {
	h.playersMutex.RLock()
	anchors := make(map[string]rl.Vector2, len(h.players))
	deadOwners := make([]string, 0)
	for id, p := range h.players {
		if p.IsDead {
			deadOwners = append(deadOwners, id)
			continue
		}
		anchors[id] = rl.NewVector2(float32(p.X), float32(p.Y))
	}
	h.playersMutex.RUnlock()

	for id, pos := range anchors {
		h.Skills.SetShieldAnchor(id, pos)
	}
	for _, id := range deadOwners {
		if h.Skills.HasShield(id) {
			h.Skills.RemoveShield(id)
			h.broadcastShieldEvent(id, 0)
		}
	}
	h.Skills.UpdateShields(dt)
}

// broadcastShieldEvent syncs a shield's strength to clients as a combat event
// ("shield" on a player). hp > 0 activates/updates the aura; hp <= 0 breaks it.
func (h *Host) broadcastShieldEvent(ownerID string, hp float32) {
	h.broadcastCombatEvent("shield", ownerID, "player", hp, "")
}
