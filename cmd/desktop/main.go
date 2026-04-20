package main

import (
	"github.com/WandenDourado/legiao/internal/entity/player"
	"github.com/WandenDourado/legiao/internal/ui/hud"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// main initializes the window and runs the game loop.
// The game loop updates the player based on virtual joystick input and renders the scene.
func main() {
	// Initialize window
	rl.InitWindow(1280, 720, "Legião - Survival Shooter")
	defer rl.CloseWindow()

	// Set target FPS
	rl.SetTargetFPS(60)

	// Create game objects
	p := player.NewPlayer()
	vjoy := hud.NewVirtualJoystick()

	// Main game loop
	for !rl.WindowShouldClose() {
		// Calculate delta time
		dt := rl.GetFrameTime()

		// Update joystick to get input direction
		dir := vjoy.Update()

		// Update player with input and delta time
		p.Update(dir, dt)

		// Begin drawing
		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		// Draw player
		p.Draw()

		// Draw virtual joystick (for debugging/touch simulation)
		vjoy.Draw()

		rl.EndDrawing()
	}
}
