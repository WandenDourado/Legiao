package input

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const NoTouch = -1

// AimTapMaxDuration is the maximum touch duration (in seconds) for a quick tap
// to be recognized as a tap-to-fire rather than an aim-drag.
const AimTapMaxDuration = 0.150

// AimTapMaxDrag is the maximum drag distance (in pixels) during a quick tap
// for it to still be considered a tap-to-fire.
const AimTapMaxDrag = 10.0

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

// AimJoystick handles the dual-function action button on Android.
// Quick tap = fire in current facing direction.
// Press + drag = aim joystick; release = fire in aimed direction.
type AimJoystick struct {
	TouchID   int32
	Origin    rl.Vector2
	Current   rl.Vector2
	IsActive  bool
	IsDragging bool
	StartTime time.Time
	AimDir    rl.Vector2
}

// NewAimJoystick creates a new AimJoystick with default values.
func NewAimJoystick() AimJoystick {
	return AimJoystick{
		TouchID:  NoTouch,
		AimDir:   rl.NewVector2(0, 1), // default: aim down
	}
}

// Update processes touch input for the aim joystick.
// Returns true if the player should fire (on release).
// attackRect defines the zone where the aim joystick can be activated.
func (aj *AimJoystick) Update(attackRect rl.Rectangle, ts TouchState) bool {
	fireNow := false

	// Scan all active touches to find one in the attack zone
	touchCount := rl.GetTouchPointCount()
	for i := int32(0); i < touchCount; i++ {
		touchID := rl.GetTouchPointId(i)
		touchPos := rl.GetTouchPosition(i)

		// New touch in attack zone — activate aim joystick
		if !aj.IsActive && rl.CheckCollisionPointRec(touchPos, attackRect) {
			// Only claim this touch if the movement joystick doesn't own it
			if ts.JoystickTouchID != touchID {
				aj.TouchID = touchID
				aj.Origin = touchPos
				aj.Current = touchPos
				aj.IsActive = true
				aj.IsDragging = false
				aj.StartTime = time.Now()
				aj.AimDir = rl.NewVector2(0, 1) // default facing
			}
		}

		// Update position for our tracked touch
		if aj.IsActive && aj.TouchID == touchID {
			aj.Current = touchPos

			// Check if dragged past tap threshold
			dist := rl.Vector2Distance(aj.Current, aj.Origin)
			if dist > AimTapMaxDrag {
				aj.IsDragging = true
				// Update aim direction
				delta := rl.Vector2Subtract(aj.Current, aj.Origin)
				length := rl.Vector2Length(delta)
				if length > 0 {
					aj.AimDir = rl.Vector2Scale(delta, 1.0/length)
				}
			}
		}
	}

	// Check if our touch was released
	if aj.IsActive {
		touchFound := false
		for i := int32(0); i < touchCount; i++ {
			if rl.GetTouchPointId(i) == aj.TouchID {
				touchFound = true
				break
			}
		}

		if !touchFound {
			// Touch released — determine if it was a tap or a drag release
			elapsed := float32(time.Since(aj.StartTime).Seconds())
			if !aj.IsDragging && elapsed < AimTapMaxDuration {
				// Quick tap — fire in current/default direction
				fireNow = true
			} else if aj.IsDragging {
				// Drag release — fire in aimed direction
				fireNow = true
			} else {
				// Held but not dragged, past tap threshold — still fire in default dir
				fireNow = true
			}

			// Reset
			aj.IsActive = false
			aj.IsDragging = false
			aj.TouchID = NoTouch
		}
	}

	return fireNow
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

// SetMaxRadius sets the maximum joystick movement radius in pixels.
// Called each frame by the game loop to scale with screen dimensions.
func (ts *TouchState) SetMaxRadius(r float32) {
	ts.MaxRadius = r
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
