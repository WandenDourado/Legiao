package network

// What the host has to do to its own authoritative state when the party
// changes map. The world itself is loaded by the game package (ApplyToHost);
// what is left here is the players, and they are not optional.
//
// Every player is put on the arrival point at once. A living player would fix
// itself within a frame — its machine publishes the new position anyway — but a
// DEAD one publishes nothing, so without this the host would keep announcing a
// corpse at coordinates that belong to the map everyone just left, and revive
// it there. The map changed for the whole party; so does the party.

import "log"

// PlaceEveryoneAtSpawn moves every player to the host's current spawn point and
// announces the roster. Called right after the destination map has been applied
// to the host, so PlayerSpawn is already the arrival point.
func (h *Host) PlaceEveryoneAtSpawn() {
	h.playersMutex.Lock()
	count := len(h.players)
	for _, p := range h.players {
		p.X = int(h.PlayerSpawn.X)
		p.Y = int(h.PlayerSpawn.Y)
	}
	h.playersMutex.Unlock()

	// Whoever vanished waiting on the portal they just crossed has to
	// reappear on the new map — without this the party would arrive
	// invisible and, for a human, unable to move (input_handler.go freezes
	// on InPortal).
	h.clearAllPortalPresence()

	log.Printf("[Travel] %d jogadores colocados em %.0f,%.0f", count, h.PlayerSpawn.X, h.PlayerSpawn.Y)
	h.BroadcastRoster()
}
