package network

import (
	"encoding/json"
	"log"
)

// handleProjectileUpdate replaces the client projectile snapshot from the host.
func (c *Client) handleProjectileUpdate(payload []byte) {
	var update ProjectileUpdatePayload
	if err := json.Unmarshal(payload, &update); err != nil {
		log.Printf("client failed to unmarshal projectile update: %v", err)
		return
	}

	RemoteProjectilesMutex.Lock()
	RemoteProjectiles = make(map[string]ProjectileState)
	for _, p := range update.Projectiles {
		RemoteProjectiles[p.ProjectileID] = p
	}
	RemoteProjectilesMutex.Unlock()
	log.Printf("[Client] Updated RemoteProjectiles: %d projectiles", len(update.Projectiles))
}
