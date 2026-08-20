package game

import (
	"github.com/WandenDourado/Legiao/internal/network"
)

// The portal is the reward for finishing the map's horde run, so it stays shut
// — invisible and inert — until the last wave is cleared.

// portalRevealTime is how long the portal takes to materialise once the map is
// cleared, in seconds. Long enough to read as a doorway opening by itself.
const portalRevealTime float32 = 1.6

// PortalsUnlocked reports whether the portals of the current map may be seen and
// used.
//
// The gate reads the wave state the network layer publishes, which the host owns
// and mirrors to clients with every enemy update, so host and client open the
// portal on the same event without a message of their own.
//
// A map with no horde run at all (Total 0 — a terrain map, or the frames before
// the host publishes its first state) is NOT locked: locking it would leave a
// quiet map with no way out.
func PortalsUnlocked() bool {
	state := network.GetWaveState()
	if state.Total == 0 {
		return true
	}
	return network.WavePhase(state.Phase) == network.WavePhaseCleared
}

// advancePortalReveal moves the materialisation between 0 (shut) and 1 (open).
// It is a plain function of the previous value so the animation lives in the
// World that owns the portals and resets by itself when the map changes.
func advancePortalReveal(current float32, unlocked bool, dt float32) float32 {
	step := dt / portalRevealTime
	if !unlocked {
		step = -step
	}
	current += step
	if current < 0 {
		return 0
	}
	if current > 1 {
		return 1
	}
	return current
}
