package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// DrawRemotePlayer renders a remote player snapshot using the correct character sprite sheet.
func DrawRemotePlayer(texture rl.Texture2D, loaded bool, def CharacterDef, x, y float32, frame, row int, velX float32, color string, radius float32) {
	if !loaded {
		DrawPlayerAt(x, y, color, radius)
		return
	}

	drawCharacterSprite(texture, def, x, y, frame, row, velX)
}

