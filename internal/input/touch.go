package input

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const NoTouch = -1

// AndroidControlGeom is the single source of truth for on-screen control
// geometry on Android. Using one function for the attack hit-rect, the skill
// button layout, and the drawn visuals guarantees the touch hit-areas always
// match what the player sees and that the two buttons never overlap.
type AndroidControlGeom struct {
	AttackCenterX float32
	AttackCenterY float32
	AttackRadius  float32 // drawn radius of the fire/attack button
	SkillCenterX  float32
	SkillCenterY  float32
	SkillRadius   float32
	// Ultimate button: sits above the skill button, never overlapping it.
	UltCenterX float32
	UltCenterY float32
	UltRadius  float32
}

// ComputeAndroidControlGeom derives all control positions from the current
// screen size. The skill button sits above the attack button with a fixed gap
// so its circle never intersects the attack hit-rect, even on large screens.
func ComputeAndroidControlGeom(sw, sh float32) AndroidControlGeom {
	attackR := sh * 0.06
	attackCX := sw * 0.85
	attackCY := sh * 0.80
	skillR := attackR
	skillCX := attackCX
	// Place the skill button above the attack button by 3 radii + a gap so
	// the two circles never overlap (previously the gap was only 2r-25px,
	// which overlapped on tall screens and made the fire button "steal" the
	// skill touch).
	skillCY := attackCY - 3*attackR - 20
	// The ultimate button stacks above the skill button with the same gap
	// rule so the three circles form a clean, non-overlapping column.
	ultR := attackR
	ultCY := skillCY - 3*attackR - 20
	return AndroidControlGeom{
		AttackCenterX: attackCX,
		AttackCenterY: attackCY,
		AttackRadius:  attackR,
		SkillCenterX:  skillCX,
		SkillCenterY:  skillCY,
		SkillRadius:   skillR,
		UltCenterX:    attackCX,
		UltCenterY:    ultCY,
		UltRadius:     ultR,
	}
}

// AttackRect returns the fire-button hit-rect (a square matching the drawn
// circle's bounding box) for touch activation.
func (g AndroidControlGeom) AttackRect() rl.Rectangle {
	return rl.NewRectangle(
		g.AttackCenterX-g.AttackRadius,
		g.AttackCenterY-g.AttackRadius,
		g.AttackRadius*2,
		g.AttackRadius*2,
	)
}

// AimTapMaxDuration is the maximum touch duration (in seconds) for a quick tap
// to be recognized as a tap-to-fire rather than an aim-drag.
const AimTapMaxDuration = 0.150

// AimTapDragFrac and SkillTapDragFrac are the fraction of screen height that a
// touch must travel before it is reclassified from a tap into an aim-drag.
// Using a screen-relative threshold (instead of a fixed pixel count) keeps the
// distinction stable across devices: high-DPI screens report the same world
// movement for a smaller on-screen finger drift, so a fixed pixel limit made
// ordinary finger tremor count as aiming. The values are deliberately larger
// than the attack button radius (sh*0.06): a tap that only drifts inside the
// button (or just past its edge) must still count as a tap-to-fire, so only a
// real drag well beyond the button reconfigures the aim. Aim uses 0.05 and
// skill 0.04.
const AimTapDragFrac = 0.05
const SkillTapDragFrac = 0.04

// AimTapMaxDrag returns the drag distance (in pixels) below which a touch is
// still considered a tap-to-fire/cast, derived from the current screen height.
// A tap that moves less than this while held short is a "fire in saved
// direction"; beyond it the touch becomes an aim-drag that overwrites AimDir.
func AimTapMaxDrag(sh float32) float32 {
	return sh * AimTapDragFrac
}

// SkillTapMaxDrag is the drag threshold for the skill button (see
// AimTapMaxDrag). Kept marginally smaller than the attack threshold so the
// secondary aim gesture still aims at shorter drags.
func SkillTapMaxDrag(sh float32) float32 {
	return sh * SkillTapDragFrac
}

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
// sw/sh are the current screen dimensions, used to derive a screen-relative
// drag threshold so a quick tap is not mistaken for an aim-drag on high-DPI
// screens.
func (aj *AimJoystick) Update(attackRect rl.Rectangle, ts TouchState, skillCenter rl.Vector2, skillRadius float32, sw, sh float32) bool {
	fireNow := false
	maxDrag := AimTapMaxDrag(sh)

	// Scan all active touches to find one in the attack zone
	touchCount := rl.GetTouchPointCount()
	for i := int32(0); i < touchCount; i++ {
		touchID := rl.GetTouchPointId(i)
		touchPos := rl.GetTouchPosition(i)

		// New touch in attack zone — activate aim joystick.
		// The skill button sits above the fire button; if a touch lands in
		// both the attack rect and the skill circle, the skill button owns
		// it (checked first via the skillCenter exclusion below) so the fire
		// button never "steals" the skill touch.
		if !aj.IsActive && rl.CheckCollisionPointRec(touchPos, attackRect) {
			// Only claim this touch if the movement joystick doesn't own it
			// and it is not inside the skill button circle.
			if ts.JoystickTouchID != touchID &&
				!rl.CheckCollisionPointCircle(touchPos, skillCenter, skillRadius) {
				aj.TouchID = touchID
				aj.Origin = touchPos
				aj.Current = touchPos
				aj.IsActive = true
				aj.IsDragging = false
				aj.StartTime = time.Now()
				// Keep the previously configured AimDir so a tap (no drag)
				// fires in the last aimed direction. It is only overwritten
				// when the player drags during this touch.
			}
		}

		// Update position for our tracked touch
		if aj.IsActive && aj.TouchID == touchID {
			aj.Current = touchPos

			// Check if dragged past tap threshold
			dist := rl.Vector2Distance(aj.Current, aj.Origin)
			if dist > maxDrag {
				aj.IsDragging = true
				// Update aim direction
				delta := rl.Vector2Subtract(aj.Current, aj.Origin)
				length := rl.Vector2Length(delta)
				if length > 0 {
					aj.AimDir = rl.Vector2Scale(delta, 1.0/length)
					rl.TraceLog(rl.LogInfo, "FIREDBG aim updated dir=(%.2f,%.2f) dist=%.1f", aj.AimDir.X, aj.AimDir.Y, dist)
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
				rl.TraceLog(rl.LogInfo, "FIREDBG FIRE(tap) aimDir=(%.2f,%.2f) elapsed=%.3f", aj.AimDir.X, aj.AimDir.Y, elapsed)
			} else if aj.IsDragging {
				// Drag release — fire in aimed direction
				fireNow = true
				rl.TraceLog(rl.LogInfo, "FIREDBG FIRE(drag) aimDir=(%.2f,%.2f) elapsed=%.3f", aj.AimDir.X, aj.AimDir.Y, elapsed)
			} else {
				// Held but not dragged, past tap threshold — still fire in default dir
				fireNow = true
				rl.TraceLog(rl.LogInfo, "FIREDBG FIRE(held) aimDir=(%.2f,%.2f) elapsed=%.3f", aj.AimDir.X, aj.AimDir.Y, elapsed)
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
