package main

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/ui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	// No Android, InitWindow(0, 0, ...) abre em tela cheia na resolução nativa
	rl.InitWindow(0, 0, "Legião")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)

	// Inicializa os objetos do jogo
	p := entity.NewPlayer()
	vjoy := ui.NewVirtualJoystick()

	for !rl.WindowShouldClose() {
		dt := rl.GetFrameTime()

		// O VirtualJoystick já trata entradas de toque/mouse
		dir := vjoy.Update()
		p.Update(dir, dt)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Black)

		p.Draw()
		vjoy.Draw()

		rl.EndDrawing()
	}
}
