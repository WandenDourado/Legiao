package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// DrawRemotePlayer renders a remote player snapshot using the correct character
// sprite sheet. A dead one goes through the grayscale shader: same pose, no
// colour, which is the only cue separating a corpse from a teammate who has
// simply stopped moving.
//
// invuln is the seconds left of a host-granted immunity window
// (PlayerState.Invuln); while it runs the body blinks through the same shader
// (InvulnBlinkGray).
func DrawRemotePlayer(texture rl.Texture2D, loaded bool, def CharacterDef, x, y float32, frame, row int, velX float32, color string, radius float32, dead bool, invuln float32) {
	if !loaded {
		DrawPlayerAt(x, y, color, radius)
		return
	}

	if dead || InvulnBlinkGray(invuln) {
		BeginDeathTint()
		defer EndDeathTint()
	}
	drawCharacterSprite(texture, def, x, y, frame, row, velX)
}
