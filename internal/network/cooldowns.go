package network

// Shared, read-only view of the authoritative cooldowns. The host fills it
// from its own timers and clients fill it from MsgCooldown, so the HUD code is
// identical on both roles.

import "sync"

var (
	cooldowns      map[string]PlayerCooldowns
	cooldownsMutex sync.RWMutex
)

// SetCooldowns replaces the shared snapshot.
func SetCooldowns(snapshot map[string]PlayerCooldowns) {
	cooldownsMutex.Lock()
	cooldowns = snapshot
	cooldownsMutex.Unlock()
}

// ClearCooldowns drops every counter (used when a stage is reset).
func ClearCooldowns() {
	SetCooldowns(nil)
}

// SkillCooldown returns the seconds left on a player's skill, or 0 if ready.
func SkillCooldown(playerID, skillID string) float32 {
	cooldownsMutex.RLock()
	defer cooldownsMutex.RUnlock()
	return cooldowns[playerID].Skills[skillID]
}

// AttackCooldown returns the seconds left on a player's basic attack cadence.
func AttackCooldown(playerID string) float32 {
	cooldownsMutex.RLock()
	defer cooldownsMutex.RUnlock()
	return cooldowns[playerID].Attack
}

// LocalSkillCooldown is SkillCooldown for the player at this machine.
func LocalSkillCooldown(skillID string) float32 {
	return SkillCooldown(LocalPlayerID, skillID)
}
