package network

// Test mode: a per-player switch that removes every cooldown gate so a stage
// can be exercised without waiting for recharges. It is authoritative like
// everything else — a client asks for it with MsgTestMode and the host decides.

import "log"

// SetTestMode enables or disables test mode for one player. It affects only
// that player's gates, so a host can test while the rest of the group plays
// normally.
func (h *Host) SetTestMode(playerID string, enabled bool) {
	h.testMutex.Lock()
	if enabled {
		if h.testPlayers == nil {
			h.testPlayers = make(map[string]bool)
		}
		h.testPlayers[playerID] = true
	} else {
		delete(h.testPlayers, playerID)
	}
	h.testMutex.Unlock()
	log.Printf("[Host] modo teste %v para %s", enabled, playerID)
}

// TestModeEnabled reports whether playerID is currently ignoring cooldowns.
func (h *Host) TestModeEnabled(playerID string) bool {
	h.testMutex.RLock()
	defer h.testMutex.RUnlock()
	return h.testPlayers[playerID]
}
