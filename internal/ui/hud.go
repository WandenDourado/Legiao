package ui

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/entity"

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
		Center:     rl.NewVector2(150, float32(entity.ScreenHeight)-150),
		BaseRadius: 80,
		KnobRadius: 40,
		KnobPos:    rl.NewVector2(150, float32(entity.ScreenHeight)-150),
		IsDragging: false,
		MaxOffset:  50, // Knob can move up to 50 pixels from center
	}
}

// Update processes input and returns the normalized direction vector.
// If not dragging, returns (0,0).
// Only starts dragging if touch/click starts inside the joystick base.
func (vj *VirtualJoystick) Update() rl.Vector2 {
	mousePos := rl.GetMousePosition()

	// Only start dragging if click/touch STARTED inside the joystick base
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
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

// AttackButton represents an on-screen attack button.
type AttackButton struct {
	Position  rl.Vector2
	Radius    float32
	IsPressed bool
	Color     rl.Color
}

// NewAttackButton creates a new attack button in the bottom-right corner.
func NewAttackButton() *AttackButton {
	return &AttackButton{
		Position: rl.NewVector2(float32(entity.ScreenWidth)-100, float32(entity.ScreenHeight)-100),
		Radius:   35.0,
		Color:    rl.Fade(rl.Red, 0.7),
	}
}

// Update processes input and returns true if the button was just pressed.
func (ab *AttackButton) Update() bool {
	mousePos := rl.GetMousePosition()

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if rl.CheckCollisionPointCircle(mousePos, ab.Position, ab.Radius) {
			ab.IsPressed = true
			return true // Button was pressed this frame
		}
	}

	if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		ab.IsPressed = false
	}

	return false
}

// Draw renders the attack button.
func (ab *AttackButton) Draw() {
	// Draw button circle
	color := ab.Color
	if ab.IsPressed {
		color = rl.Fade(rl.Red, 0.9)
	}
	rl.DrawCircleV(ab.Position, ab.Radius, color)
	rl.DrawCircleLinesV(ab.Position, ab.Radius, rl.White)

	// Draw "FIRE" text
	text := "FIRE"
	textWidth := rl.MeasureText(text, 14)
	rl.DrawText(text,
		int32(ab.Position.X)-textWidth/2,
		int32(ab.Position.Y)-7,
		14, rl.White)
}

// DrawHealthBar draws the player's health bar in the top-left corner.
// All dimensions are derived from the current screen size so the bar
// scales correctly on any resolution (fullscreen, windowed, etc.).
func DrawHealthBar(health, maxHealth float32) {
	sw := float32(rl.GetScreenWidth())
	barWidth := sw * 0.20
	barHeight := float32(20)
	x := float32(10)
	y := float32(60)
	fontSize := int32(sw * 0.015)
	if fontSize < 14 {
		fontSize = 14
	}

	// Background
	rl.DrawRectangle(int32(x), int32(y), int32(barWidth), int32(barHeight), rl.Fade(rl.DarkGray, 0.8))

	// Health fill (green to red gradient based on health percentage)
	healthPercent := health / maxHealth
	fillWidth := int32(barWidth * float32(healthPercent))

	var healthColor rl.Color
	if healthPercent > 0.5 {
		healthColor = rl.Green
	} else if healthPercent > 0.25 {
		healthColor = rl.Orange
	} else {
		healthColor = rl.Red
	}

	rl.DrawRectangle(int32(x), int32(y), fillWidth, int32(barHeight), healthColor)

	// Border
	rl.DrawRectangleLinesEx(rl.NewRectangle(float32(x), float32(y), barWidth, barHeight), 2, rl.White)

	// Text
	healthText := fmt.Sprintf("%.0f/%.0f", health, maxHealth)
	rl.DrawText(healthText, int32(x)+int32(barWidth)+10, int32(y), fontSize, rl.White)
}
