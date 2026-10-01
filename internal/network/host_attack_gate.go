package network

// Cadence gating for the basic attack. Skills have their own cooldowns; the
// basic attack is limited by the character's attack speed instead, so a fast
// character lands more hits per second than a slow one for the same input.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
)

// beginAttackCooldown returns false if playerID's basic attack is still on
// cadence cooldown; otherwise it arms the cooldown and returns true. The
// interval comes from the character definition (1 / AttackSpeed), so tuning a
// character's rhythm is a one-line change in entity.RegisterCharacter.
func (h *Host) beginAttackCooldown(playerID string, char entity.CharacterType) bool {
	// Test mode is checked before taking the lock: TestModeEnabled has its own
	// mutex and cdMutex is not reentrant.
	if h.TestModeEnabled(playerID) {
		return true
	}
	interval := entity.GetCharacter(char).AttackInterval()
	if interval <= 0 {
		return true // character with no declared cadence: unrestricted
	}
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()
	if h.attackCooldowns[playerID] > 0 {
		return false
	}
	h.attackCooldowns[playerID] = interval
	return true
}

// tickAttackCooldowns counts every armed attack cooldown down and drops the
// expired ones, so a ready attack simply has no entry in the map.
func (h *Host) tickAttackCooldowns(dt float32) {
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()
	for id, cd := range h.attackCooldowns {
		cd -= dt
		if cd <= 0 {
			delete(h.attackCooldowns, id)
			continue
		}
		h.attackCooldowns[id] = cd
	}
}
