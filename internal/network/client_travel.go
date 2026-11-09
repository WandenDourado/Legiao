package network

// Client-side handling of a map change announced by the host.
//
// The host has already dropped the world it left; this drops everything the
// client was mirroring of it. Without the wipe the previous map's monsters and
// projectiles keep being drawn over the new one until the next snapshot lands
// — at 20 Hz that is long enough to see, and they would be standing at
// coordinates that mean nothing here.

// applyTravel wipes the mirrored world and queues the local arrival.
func (c *Client) applyTravel(t TravelPayload) {
	RemoteEnemiesMutex.Lock()
	RemoteEnemies = make(map[string]EnemyState)
	RemoteEnemiesMutex.Unlock()

	RemoteProjectilesMutex.Lock()
	RemoteProjectiles = make(map[string]ProjectileState)
	RemoteProjectilesMutex.Unlock()

	if ClientSkills != nil {
		ClientSkills.Reset()
	}

	// Death is NOT cleared here. Arriving is not reviving: a player who died on
	// the way to the portal travels with the party and goes on waiting out the
	// host's revive countdown on the other side.
	RequestLocalTravel(t)
}
