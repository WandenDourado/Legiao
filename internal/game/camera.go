package game

import (
	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Camera2DState manages the rl.Camera2D that follows the player.
type Camera2DState struct {
	Camera rl.Camera2D
}

// NewCamera creates a Camera2DState with default values.
func NewCamera(sw, sh float32) Camera2DState {
	return Camera2DState{
		Camera: rl.Camera2D{
			Offset: rl.NewVector2(sw/2, sh/2),
			Target: rl.NewVector2(0, 0),
			Rotation: 0,
			Zoom:   1.0,
		},
	}
}

// Update moves the camera directly to the target position,
// then clamps the result to world bounds so the viewport never shows outside the map.
func (c *Camera2DState) Update(target rl.Vector2, sw, sh float32, bounds world.Bounds) {
	c.Camera.Offset = rl.NewVector2(sw/2, sh/2)
	c.Camera.Zoom = 1.0

	// Set camera target directly to player position
	c.Camera.Target.X = target.X
	c.Camera.Target.Y = target.Y

	// Clamp to world bounds, handling maps smaller than the screen
	halfW := sw / 2
	halfH := sh / 2
	if bounds.Width < sw {
		// Map narrower than screen – keep camera centered horizontally
		c.Camera.Target.X = bounds.Width / 2
	} else {
		c.Camera.Target.X = clamp(c.Camera.Target.X, halfW, bounds.Width-halfW)
	}
	if bounds.Height < sh {
		// Map shorter than screen – keep camera centered vertically
		c.Camera.Target.Y = bounds.Height / 2
	} else {
		c.Camera.Target.Y = clamp(c.Camera.Target.Y, halfH, bounds.Height-halfH)
	}
}

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
