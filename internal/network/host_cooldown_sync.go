package network

// Cooldowns live on the host, but the counters are drawn on every machine.
// This file takes a snapshot of the authoritative timers each simulation tick
// and pushes it out, so a client's HUD never has to guess.

import "strings"

// broadcastCooldowns mirrors the current cooldowns to every client and to the
// host's own shared store (the HUD reads the same store on both roles).
func (h *Host) broadcastCooldowns() {
	payload := CooldownPayload{Players: h.cooldownSnapshot()}
	SetCooldowns(payload.Players)
	h.broadcast(Message{Type: MsgCooldown, Payload: MustMarshal(payload)})
}

// cooldownSnapshot converts the host's internal "playerID|skillID" keys into
// the per-player shape clients consume.
func (h *Host) cooldownSnapshot() map[string]PlayerCooldowns {
	h.cdMutex.Lock()
	defer h.cdMutex.Unlock()

	out := make(map[string]PlayerCooldowns, len(h.attackCooldowns))
	for key, remaining := range h.skillCooldowns {
		playerID, skillID, ok := strings.Cut(key, "|")
		if !ok {
			continue
		}
		entry := out[playerID]
		if entry.Skills == nil {
			entry.Skills = make(map[string]float32, 2)
		}
		entry.Skills[skillID] = remaining
		out[playerID] = entry
	}
	for playerID, remaining := range h.attackCooldowns {
		entry := out[playerID]
		entry.Attack = remaining
		out[playerID] = entry
	}
	return out
}
