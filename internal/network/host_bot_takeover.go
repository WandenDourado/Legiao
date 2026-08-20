package network

// Taking over a bot's class: the human inherits the BODY (position, health,
// death state, revive countdown, cooldowns) instead of spawning fresh — a
// key swap, not a new character (plan §3, "Tomada de posse"). Recharged
// skills are deliberately NOT inherited: continuity of the body is the
// point, but a topped-up ultimate would turn "join this class" into a free
// cooldown reset.

import (
	"log"
	"strings"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// takeOverBot looks for a bot playing character class char and, if one
// exists, hands its body to newID. Caller must hold NEITHER playersMutex
// NOR cdMutex; this takes what it needs itself, in the documented order
// (playersMutex before cdMutex — host.go), and releases playersMutex before
// touching skills or broadcasting.
//
// Returns true if a bot was taken over — the caller must not also create a
// fresh player for newID.
func (h *Host) takeOverBot(newID, color, character string) bool {
	botKey := botIDFor(entity.CharacterType(character))

	h.playersMutex.Lock()
	botEntry, ok := h.players[botKey]
	if !ok {
		h.playersMutex.Unlock()
		return false
	}
	taken := *botEntry
	taken.PlayerID = newID
	taken.Color = color
	taken.Character = character
	delete(h.players, botKey)
	h.players[newID] = &taken
	if h.bots != nil {
		delete(h.bots, botKey)
	}
	h.playersMutex.Unlock()

	h.migrateAttackCooldown(botKey, newID)
	h.migrateSkillCooldowns(botKey, newID)
	h.migrateInvulnerability(botKey, newID)

	// The bot's own in-flight effects (a shield up, an avatar mid-flight)
	// do not belong to the human taking the body over — clear them and
	// tell every client to stop drawing what just lost its owner.
	for _, signal := range h.Skills.ClearOwner(botKey) {
		h.broadcastUltimate(signal, botKey, rl.Vector2{}, rl.Vector2{})
	}

	log.Printf("[Host] %s assume o corpo do bot de %s", newID, character)
	return true
}

func (h *Host) migrateAttackCooldown(oldID, newID string) {
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()
	if v, ok := h.attackCooldowns[oldID]; ok {
		h.attackCooldowns[newID] = v
		delete(h.attackCooldowns, oldID)
	}
}

// migrateSkillCooldowns rewrites every "oldID|skillID" entry in
// skillCooldowns/skillCharges to "newID|skillID". Both maps are keyed this
// way (cooldowns.go), unlike attackCooldowns which is keyed by playerID
// alone.
func (h *Host) migrateSkillCooldowns(oldID, newID string) {
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()
	prefix := oldID + "|"
	for key, v := range h.skillCooldowns {
		if suffix, ok := strings.CutPrefix(key, prefix); ok {
			h.skillCooldowns[newID+"|"+suffix] = v
			delete(h.skillCooldowns, key)
		}
	}
	for key, v := range h.skillCharges {
		if suffix, ok := strings.CutPrefix(key, prefix); ok {
			h.skillCharges[newID+"|"+suffix] = v
			delete(h.skillCharges, key)
		}
	}
}

func (h *Host) migrateInvulnerability(oldID, newID string) {
	invulnMu.Lock()
	defer invulnMu.Unlock()
	if v, ok := invulnUntil[oldID]; ok {
		invulnUntil[newID] = v
		delete(invulnUntil, oldID)
	}
}
