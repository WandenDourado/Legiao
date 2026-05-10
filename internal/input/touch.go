package input

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

const NoTouch = -1

// TouchState holds the state of multi-touch inputs for joystick and attack button.
// Uses stable touch IDs (not array indices) for reliable tracking.
type TouchState struct {
	JoystickTouchID int32
	AttackTouchID   int32
	JoystickOrigin   rl.Vector2
	JoystickCurrent  rl.Vector2
	IsAttacking     bool
	MaxRadius       float32
}

// NewTouchState creates a new TouchState with default values.
// MaxRadius is the maximum joystick movement in pixels.
func NewTouchState() TouchState {
	return TouchState{
		JoystickTouchID: NoTouch,
		AttackTouchID:   NoTouch,
		IsAttacking:     false,
		MaxRadius:       60.0,
	}
}

// Update processes all active touches and updates the state.
// Should be called once per frame, before reading state.
// Zones: Left half = joystick, Right half = attack button.
func (ts *TouchState) Update(joystickRect rl.Rectangle, attackRect rl.Rectangle) {
	activeIDs := make(map[int32]bool)

	// Scan all active touches
	touchCount := rl.GetTouchPointCount()
	for i := int32(0); i < touchCount; i++ {
		touchID := rl.GetTouchPointId(i) // Unique stable ID
		touchPos := rl.GetTouchPosition(i)

		activeIDs[touchID] = true

		// Check if this is a new touch for joystick
		if ts.JoystickTouchID == NoTouch && rl.CheckCollisionPointRec(touchPos, joystickRect) {
			ts.JoystickTouchID = touchID
			ts.JoystickOrigin = touchPos
			ts.JoystickCurrent = touchPos
		}

		// Check if this is a new touch for attack button
		if ts.AttackTouchID == NoTouch && rl.CheckCollisionPointRec(touchPos, attackRect) {
			ts.AttackTouchID = touchID
			ts.IsAttacking = true
		}

		// Update positions for already tracked touches
		if ts.JoystickTouchID == touchID {
			ts.JoystickCurrent = touchPos
		}
	}

	// Release joystick if touch is gone
	if ts.JoystickTouchID != NoTouch && !activeIDs[ts.JoystickTouchID] {
		ts.JoystickTouchID = NoTouch
	}

	// Release attack button if touch is gone
	if ts.AttackTouchID != NoTouch && !activeIDs[ts.AttackTouchID] {
		ts.AttackTouchID = NoTouch
		ts.IsAttacking = false
	}

	// Fallback: handle mouse input for desktop testing
	if ts.JoystickTouchID == NoTouch && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		if rl.CheckCollisionPointRec(mousePos, joystickRect) {
			ts.JoystickTouchID = -2 // Special ID for mouse
			ts.JoystickOrigin = mousePos
			ts.JoystickCurrent = mousePos
		}
	}
	if ts.JoystickTouchID == -2 {
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			ts.JoystickCurrent = rl.GetMousePosition()
		}
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			ts.JoystickTouchID = NoTouch
		}
	}

	// Mouse attack button
	if ts.AttackTouchID == NoTouch && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		mousePos := rl.GetMousePosition()
		if rl.CheckCollisionPointRec(mousePos, attackRect) {
			ts.AttackTouchID = -2
			ts.IsAttacking = true
		}
	}
	if ts.AttackTouchID == -2 && rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		ts.AttackTouchID = NoTouch
		ts.IsAttacking = false
	}
}

// GetJoystickDelta returns the normalized joystick direction vector.
// Returns rl.Vector2{} (zero) when joystick is not active.
// Values are clamped to MaxRadius and normalized to [-1, 1].
func (ts *TouchState) GetJoystickDelta() rl.Vector2 {
	if ts.JoystickTouchID == NoTouch {
		return rl.NewVector2(0, 0)
	}

	delta := rl.Vector2Subtract(ts.JoystickCurrent, ts.JoystickOrigin)
	length := rl.Vector2Length(delta)

	if length > ts.MaxRadius {
		delta = rl.Vector2Scale(rl.Vector2Normalize(delta), ts.MaxRadius)
	}

	if ts.MaxRadius > 0 {
		delta.X /= ts.MaxRadius
		delta.Y /= ts.MaxRadius
	}

	return delta
}
