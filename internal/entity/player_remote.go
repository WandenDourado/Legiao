package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// DrawWizardStateAt renders a player snapshot with the wizard sprite sheet.
func DrawWizardStateAt(texture rl.Texture2D, loaded bool, x, y float32, frame, row int, velX float32, color string, radius float32) {
	if !loaded {
		DrawPlayerAt(x, y, color, radius)
		return
	}

	drawWizardSprite(texture, x, y, frame, row, velX)
}
