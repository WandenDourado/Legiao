package ui

import (
	"fmt"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"

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

// SkillTapMaxDuration mirrors the AimJoystick tap duration threshold.
const SkillTapMaxDuration = 0.150

// NoTouch mirrors input.NoTouch since this package cannot import input
// (input imports the game package, which imports ui — would be a cycle).
const NoTouch = int32(-1)

// SkillButton represents the on-screen "fireball" skill button (Android).
// It is positioned above the attack button and behaves like a second
// aim joystick: tap = cast in the last configured direction; press + drag =
// aim the projectile, release casts in the dragged direction and persists it.
type SkillButton struct {
	Position  rl.Vector2
	Radius    float32
	IsPressed bool
	Color     rl.Color
	// Label is drawn at the button center ("Q" for the primary skill, "R"
	// for the ultimate). Accent tints the aim-mode feedback circle/line.
	Label      string
	Accent     rl.Color
	// IsUltimate selects the ultimate slot geometry in Layout().
	IsUltimate bool
	screenH    float32 // cached screen height for the drag threshold

	// Cooldown is the seconds left before this ability can be cast again and
	// CooldownTotal its full recharge. Both are fed every frame from the
	// host's authoritative snapshot and only drive the overlay drawn on top
	// of the button — the host is what actually refuses an early cast.
	Cooldown      float32
	CooldownTotal float32

	// Touch/aim state (Android only)
	TouchID   int32
	Origin    rl.Vector2
	Current   rl.Vector2
	IsActive  bool
	IsDragging bool
	StartTime time.Time
	SkillDir  rl.Vector2 // last configured cast direction (persists across taps)
}

// NewSkillButton creates an unpositioned skill button. Call Layout(sw, sh)
// every frame (only on Android) before Update/Draw so the hit area always
// matches the drawn circle, even after rotation/resize.
func NewSkillButton() *SkillButton {
	return &SkillButton{
		Color:    rl.Fade(rl.Orange, 0.7),
		Label:    "Q",
		Accent:   rl.Orange,
		TouchID:  NoTouch,
		SkillDir: rl.NewVector2(0, 1), // default: cast down
	}
}

// NewUltimateButton creates the on-screen ULTIMATE button (Android). Same
// tap/drag-aim behavior as the skill button, but golden and positioned above
// it (see input.ComputeAndroidControlGeom).
func NewUltimateButton() *SkillButton {
	return &SkillButton{
		Color:      rl.Fade(rl.Gold, 0.75),
		Label:      "R",
		Accent:     rl.Gold,
		IsUltimate: true,
		TouchID:    NoTouch,
		SkillDir:   rl.NewVector2(0, 1), // default: cast down
	}
}

// Layout recomputes the skill button geometry from the current screen size,
// using the shared Android control geometry so the drawn circle and hit area
// match the fire button's and never overlap it. Call once per frame on
// Android only.
func (sb *SkillButton) Layout(sw, sh float32) {
	geom := input.ComputeAndroidControlGeom(sw, sh)
	if sb.IsUltimate {
		sb.Position = rl.NewVector2(geom.UltCenterX, geom.UltCenterY)
		sb.Radius = geom.UltRadius
	} else {
		sb.Position = rl.NewVector2(geom.SkillCenterX, geom.SkillCenterY)
		sb.Radius = geom.SkillRadius
	}
	sb.screenH = sh
}

// Update processes touch input for the skill button (Android).
// Returns true when the player should cast (on tap or drag-release).
// Touch-based; mirrors AimJoystick: drag aims, tap casts the saved direction.
// The SkillDir is preserved across touches so a subsequent tap (no drag)
// casts in the last aimed direction rather than a fresh default.
func (sb *SkillButton) Update() bool {
	castNow := false

	touchCount := rl.GetTouchPointCount()
	for i := int32(0); i < touchCount; i++ {
		touchID := rl.GetTouchPointId(i)
		touchPos := rl.GetTouchPosition(i)

		// New touch inside the skill button — activate
		if !sb.IsActive && rl.CheckCollisionPointCircle(touchPos, sb.Position, sb.Radius) {
			sb.TouchID = touchID
			sb.Origin = touchPos
			sb.Current = touchPos
			sb.IsActive = true
			sb.IsDragging = false
			sb.StartTime = time.Now()
			sb.IsPressed = true
			// SkillDir is intentionally NOT reset: a tap casts the
			// previously configured direction.
		}

		// Track our touch and update aim on drag
		if sb.IsActive && sb.TouchID == touchID {
			sb.Current = touchPos
			dist := rl.Vector2Distance(sb.Current, sb.Origin)
			if dist > input.SkillTapMaxDrag(sb.screenH) {
				sb.IsDragging = true
				delta := rl.Vector2Subtract(sb.Current, sb.Origin)
				if length := rl.Vector2Length(delta); length > 0 {
					sb.SkillDir = rl.Vector2Scale(delta, 1.0/length)
				}
			}
		}
	}

	// Release detection
	if sb.IsActive {
		found := false
		for i := int32(0); i < touchCount; i++ {
			if rl.GetTouchPointId(i) == sb.TouchID {
				found = true
				break
			}
		}
		if !found {
			elapsed := float32(time.Since(sb.StartTime).Seconds())
			if !sb.IsDragging && elapsed < SkillTapMaxDuration {
				castNow = true // quick tap → cast saved direction
			} else if sb.IsDragging {
				castNow = true // drag-release → cast aimed direction
			} else {
				castNow = true // held, no drag, past tap window → cast saved dir
			}
			sb.IsActive = false
			sb.IsDragging = false
			sb.IsPressed = false
			sb.TouchID = NoTouch
		}
	}

	return castNow
}

// Draw renders the skill button. When the player is aiming (dragging), it
// shows a direction line and arrow so they can see where the projectile will
// go, mirroring the attack button's aim feedback.
func (sb *SkillButton) Draw() {
	color := sb.Color
	if sb.IsPressed {
		color = rl.Fade(sb.Accent, 0.9)
	}

	if sb.IsActive {
		// Aim mode: base at touch origin + direction line + arrow
		rl.DrawCircleV(sb.Origin, sb.Radius, rl.Fade(sb.Accent, 0.4))
		rl.DrawCircleLinesV(sb.Origin, sb.Radius, sb.Accent)
		rl.DrawLineV(sb.Origin, sb.Current, rl.Yellow)
		aimEnd := rl.Vector2Add(sb.Origin, rl.Vector2Scale(sb.SkillDir, sb.Radius*1.5))
		rl.DrawLineV(sb.Origin, aimEnd, sb.Accent)
	} else {
		rl.DrawCircleV(sb.Position, sb.Radius, color)
		rl.DrawCircleLinesV(sb.Position, sb.Radius, rl.White)
	}

	text := sb.Label
	textWidth := rl.MeasureText(text, 16)
	rl.DrawText(text,
		int32(sb.Position.X)-textWidth/2,
		int32(sb.Position.Y)-8,
		16, rl.White)

	// The recharge counter goes over the button itself: on a phone there is
	// no room for a separate bar, and the finger is already there.
	DrawCooldownOverlay(sb.Position, sb.Radius, sb.Cooldown, sb.CooldownTotal)
}

// SetCooldown feeds the button the authoritative recharge state for the frame.
func (sb *SkillButton) SetCooldown(remaining, total float32) {
	sb.Cooldown = remaining
	sb.CooldownTotal = total
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
