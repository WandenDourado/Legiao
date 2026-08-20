package game

// Resyncing the local player's OWN position after a reconnect.
//
// SyncLocalPlayer (death.go) deliberately never copies X/Y for a living
// player: position is predicted locally, and the snapshot is only an echo of
// what this machine already sent. That rule is correct for every ordinary
// frame, but a reconnect is not an ordinary frame — during the outage the
// host kept moving (or simply held) the authoritative X/Y for this player ID,
// while this machine's own p.Position froze at whatever it last was before
// the connection dropped. Nothing else in the loop ever pulls the two back
// together, so a reconnect needs this one, explicit correction.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// resyncLocalPlayer snaps the local player entity to whatever the host
// preserved for this playerID across the reconnect. Called once, from
// ApplyPendingTravel, only for a Reconnect-flagged arrival — never on an
// ordinary frame, or it would fight the local movement prediction every
// player relies on.
func resyncLocalPlayer(p *entity.Player) {
	network.RemotePlayersMutex.Lock()
	state, ok := network.RemotePlayers[network.LocalPlayerID]
	network.RemotePlayersMutex.Unlock()
	if !ok {
		return
	}
	p.Position.X = float32(state.X)
	p.Position.Y = float32(state.Y)
	p.Velocity = rl.Vector2{}
}
