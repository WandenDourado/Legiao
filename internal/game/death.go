package game

// Death is authoritative on the host: it decides when a player falls, counts
// the revive down and brings them back. This file is the local mirror — it
// copies that verdict onto the player entity so the loop knows whether to move
// it and the renderer knows whether to grey it out.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
)

// SyncLocalPlayer applies the host's snapshot of the local player: health,
// death, and the revive countdown. Position is deliberately not copied while
// the player is alive — movement is predicted locally and the snapshot is only
// an echo of what this machine already sent. On a revive, though, the host
// chose the spawn point, so that one position wins.
func SyncLocalPlayer(p *entity.Player) {
	network.RemotePlayersMutex.Lock()
	state, ok := network.RemotePlayers[network.LocalPlayerID]
	network.RemotePlayersMutex.Unlock()
	if !ok {
		return
	}

	if state.MaxHealth > 0 {
		p.MaxHealth = state.MaxHealth
	}
	p.Health = state.Health
	p.RespawnIn = state.RespawnIn
	network.LocalRespawnIn = state.RespawnIn

	switch {
	case state.IsDead && !p.IsDead:
		p.Die()
	case !state.IsDead && p.IsDead:
		p.Respawn(1, float32(state.X), float32(state.Y))
		p.Health = state.Health
	}
	network.LocalPlayerDead = p.IsDead
	network.LocalPlayerInPortal = state.InPortal
}
