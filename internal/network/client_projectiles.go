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
	// O instantaneo ANTERIOR sai inteiro daqui antes de ser trocado: e contra
	// ele que se descobre quem acabou de atacar (client_attack_sound.go).
	prev := RemoteProjectiles
	RemoteProjectiles = make(map[string]ProjectileState)
	for _, p := range update.Projectiles {
		RemoteProjectiles[p.ProjectileID] = p
	}
	RemoteProjectilesMutex.Unlock()

	// Fora do mutex de proposito: anunciar precisa do mutex dos JOGADORES, e
	// segurar os dois ao mesmo tempo e como se cria uma ordem de travas.
	announceNewProjectiles(prev, update.Projectiles)
	log.Printf("[Client] Updated RemoteProjectiles: %d projectiles", len(update.Projectiles))
}
