package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// DrawWizardStateAt renders a player snapshot with the wizard sprite sheet.
func DrawWizardStateAt(texture rl.Texture2D, loaded bool, x, y float32, frame, row int, velX float32, color string, radius float32) {
	if !loaded {
		DrawPlayerAt(x, y, color, radius)
		return
	}

	if frame < 0 || frame >= WizardColumns {
		frame = 0
	}
	// Fallback to RowWalkUp if the texture doesn't have the 5th row (upper-diagonal)
	if texture.Height > 0 && row*WizardFrameHeight >= int(texture.Height) {
		if row == RowWalkUpLeft {
			row = RowWalkUp
		} else {
			row = RowWalkDown
		}
	}
	if row < 0 || row >= WizardRows {
		row = RowWalkDown
	}

	sourceRect := rl.NewRectangle(
		float32(frame*WizardFrameWidth),
		float32(row*WizardFrameHeight),
		WizardFrameWidth,
		WizardFrameHeight,
	)
	if velX > 0 && (row == RowWalkLeft || row == RowWalkDownLeft || row == RowWalkUpLeft) {
		sourceRect.Width = -WizardFrameWidth
	}

	destRect := rl.NewRectangle(
		x-WizardFrameWidth/2,
		y-WizardFrameHeight/2,
		WizardFrameWidth,
		WizardFrameHeight,
	)
	rl.DrawTexturePro(texture, sourceRect, destRect, rl.NewVector2(0, 0), 0, rl.White)
}
