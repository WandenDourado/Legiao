package ui

import (
	"github.com/WandenDourado/legiao/internal/game"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// VirtualJoystick represents a simple on-screen joystick for touch input.
type VirtualJoystick struct {
	Center     rl.Vector2 // Center of the joystick base
	BaseRadius float32    // Radius of the base circle
	KnobRadius float32    // Radius of the knob circle
	KnobPos    rl.Vector2 // Current position of the knob
	IsDragging bool       // Whether the user is currently dragging the knob
	MaxOffset  float32    // Maximum distance the knob can move from the center
}

// NewVirtualJoystick creates a new joystick with default settings.
// The joystick is placed in the bottom-left corner of the screen.
func NewVirtualJoystick() *VirtualJoystick {
	return &VirtualJoystick{
		Center:     rl.NewVector2(150, float32(game.ScreenHeight)-150),
		BaseRadius: 80,
		KnobRadius: 40,
		KnobPos:    rl.NewVector2(150, float32(game.ScreenHeight)-150),
		IsDragging: false,
		MaxOffset:  50, // Knob can move up to 50 pixels from center
	}
}

// Update processes input and returns the normalized direction vector.
// If not dragging, returns (0,0).
func (vj *VirtualJoystick) Update() rl.Vector2 {
	mousePos := rl.GetMousePosition()
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		// Check if the touch/click is inside the base circle
		if rl.CheckCollisionPointCircle(mousePos, vj.Center, vj.BaseRadius) {
			vj.IsDragging = true
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		vj.IsDragging = false
		// Reset knob to center when released
		vj.KnobPos = vj.Center
		return rl.NewVector2(0, 0)
	}

	if vj.IsDragging {
		// Calculate vector from center to mouse position
		diff := rl.Vector2Subtract(mousePos, vj.Center)
		// Clamp the distance to MaxOffset
		diffLen := rl.Vector2Length(diff)
		if diffLen > vj.MaxOffset {
			diff = rl.Vector2Scale(rl.Vector2Normalize(diff), vj.MaxOffset)
		}
		// Set knob position
		vj.KnobPos = rl.Vector2Add(vj.Center, diff)
		// Return normalized direction (diff divided by MaxOffset)
		return rl.Vector2Scale(diff, 1.0/vj.MaxOffset)
	}
	// If not dragging, return zero
	return rl.NewVector2(0, 0)
}

// Draw renders the joystick (base and knob).
func (vj *VirtualJoystick) Draw() {
	// Draw base circle
	rl.DrawCircleV(vj.Center, vj.BaseRadius, rl.Fade(rl.Gray, 0.5))
	// Draw knob circle
	rl.DrawCircleV(vj.KnobPos, vj.KnobRadius, rl.Fade(rl.LightGray, 0.8))
}
