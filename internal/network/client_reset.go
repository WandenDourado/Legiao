package network

// Client-side handling of a stage reset announced by the host. The host has
// already cleared its own world; this drops everything the client was mirroring
// so the two do not disagree for the frames until the next full state update.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// applyStageReset wipes the client's mirrored world and queues the local
// player's return to the spawn point.
func (c *Client) applyStageReset(rp ResetPayload) {
	RemoteEnemiesMutex.Lock()
	RemoteEnemies = make(map[string]EnemyState)
	RemoteEnemiesMutex.Unlock()

	RemoteProjectilesMutex.Lock()
	RemoteProjectiles = make(map[string]ProjectileState)
	RemoteProjectilesMutex.Unlock()

	if ClientSkills != nil {
		ClientSkills.Reset()
	}
	ClearCooldowns()
	// Mesma limpeza que o host faz em ResetStage: a concessao da corrida (ver
	// progression.go) nao sobrevive a um reinicio de fase.
	ClearRunGrantedUltimates()

	// Espelha o RearmClimaxPending que o host acabou de fazer: o portal de um
	// mapa de emboscada e desenhado nas duas maquinas (game.PortalsUnlocked),
	// entao um cliente que nao re-armasse continuaria vendo a saida aberta
	// depois do F5. Ver climax_pending.go.
	RearmClimaxPending()

	GameOver = false
	LocalPlayerDead = false
	LocalRespawnIn = 0
	RequestLocalReset(rl.NewVector2(float32(rp.SpawnX), float32(rp.SpawnY)))
}
