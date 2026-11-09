package game

// Stage reset. A Game Over is final until the host restarts the stage, either
// with F5 on the keyboard or with the button on the Game Over overlay. The
// same overlay tells everyone else that they are waiting on the host.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// uiGameOver owns the overlay and the restart button hit-area. It is drawn by
// the renderer and read here, exactly like the on-screen ability buttons.
var uiGameOver = ui.NewGameOverPanel()

// UpdateStageReset handles the host's restart command and applies a reset that
// has already been decided (by this host or by the one we are connected to) to
// the local player. Called once per frame from input processing.
func UpdateStageReset(p *entity.Player, sw, sh float32) {
	if network.GameOver && CanResetStage() {
		if rl.IsKeyPressed(rl.KeyF5) || uiGameOver.Update(sw, sh) {
			network.CurrentHost.ResetStage()
		}
	}

	if spawn, ok := network.ConsumeLocalReset(); ok {
		// Full health: a restarted stage is a fresh run, not a revive.
		p.Respawn(1, spawn.X, spawn.Y)
	}
}

// CanResetStage reports whether this machine is the one that may restart the
// stage. Only the host simulates the world, so only the host can rewind it.
func CanResetStage() bool {
	return network.Role == "host" && network.CurrentHost != nil
}
