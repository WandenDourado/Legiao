package game

// Test mode (F2, desktop only): removes every recharge gate for the local
// player so a stage can be exercised without waiting. The host is what
// actually applies it, so a client pressing F2 asks for it over the wire and
// the answer holds for that client alone.

import (
	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UpdateTestMode toggles test mode when F2 is pressed. It is a keyboard-only
// affordance: there is no touch equivalent, and none is wanted — this is a
// development switch, not a game feature.
func UpdateTestMode(cfg Config) {
	if cfg.FullScreen {
		return
	}
	if !rl.IsKeyPressed(rl.KeyF2) {
		return
	}
	network.TestMode = !network.TestMode
	network.SetLocalTestMode(network.TestMode)
}
