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

// NoTouch mirrors input.NoTouch as an int32. Not a cycle guard — this package
// does import input, right above — just the typed spelling the touch IDs
// need.
const NoTouch = int32(-1)

// SkillButton represents the on-screen "fireball" skill button (Android).
// It is positioned above the attack button and behaves like a second
// aim joystick: tap = cast in the last configured direction; press + drag =
// aim the projectile, release casts in the dragged direction and persists it.
type SkillButton struct {
	Position  rl.Vector2
	Radius    float32
	IsPressed bool
	// Label is the fallback drawn at the button centre when the icon file is
	// missing ("Q" for the primary skill, "R" for the ultimate).
	Label string
	// Icon is the glyph the button actually shows. There is no per-button
	// colour any more — see ui.ActionNeutral.
	Icon ActionIcon
	// Slot is which member of the cluster this is, for the shared aim lock.
	Slot input.ActionSlot
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
	TouchID    int32
	Origin     rl.Vector2
	Current    rl.Vector2
	IsActive   bool
	IsDragging bool
	StartTime  time.Time
	SkillDir   rl.Vector2 // last configured cast direction (persists across taps)
}

// NewSkillButton creates an unpositioned skill button. Call Layout(sw, sh)
// every frame (only on Android) before Update/Draw so the hit area always
// matches the drawn circle, even after rotation/resize.
func NewSkillButton() *SkillButton {
	return &SkillButton{
		Label:    "Q",
		Icon:     IconSkill,
		Slot:     input.SlotSkill,
		TouchID:  NoTouch,
		SkillDir: rl.NewVector2(0, 1), // default: cast down
	}
}

// NewUltimateButton creates the on-screen ULTIMATE button (Android). Same
// tap/drag-aim behavior as the skill button, but golden and positioned above
// it (see input.ComputeAndroidControlGeom).
func NewUltimateButton() *SkillButton {
	return &SkillButton{
		Label:      "R",
		Icon:       IconUltimate,
		Slot:       input.SlotUltimate,
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
//
// lock is the cluster's single-gesture rule: this button refuses a new touch
// while another one is being aimed, and owns the gesture itself while it is.
// Without it an aim drag that passed over this button armed it, and letting go
// cast a spell the player never asked for. See input.ActionLock.
func (sb *SkillButton) Update(lock *input.ActionLock) bool {
	castNow := false

	touchCount := rl.GetTouchPointCount()
	for i := int32(0); i < touchCount; i++ {
		touchID := rl.GetTouchPointId(i)
		touchPos := rl.GetTouchPosition(i)

		// New touch inside the skill button — activate
		if !sb.IsActive && rl.CheckCollisionPointCircle(touchPos, sb.Position, sb.Radius) &&
			lock.Claim(sb.Slot) {
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
			lock.Release(sb.Slot)
		}
	}

	return castNow
}

// Cancel drops an in-flight aim and frees the cluster, WITHOUT casting.
//
// For the frames the button stops being updated at all: a finished run, a body
// frozen inside a portal, an ultimate that stopped being available mid-aim.
// Release is normally detected by scanning touches, so a path that returns
// before that scan would leave the button armed and the cluster locked.
func (sb *SkillButton) Cancel(lock *input.ActionLock) {
	if sb.IsActive {
		lock.Release(sb.Slot)
	}
	sb.IsActive = false
	sb.IsDragging = false
	sb.IsPressed = false
	sb.TouchID = NoTouch
}

// Draw renders the skill button through the cluster's shared appearance.
// locked comes from the aim lock: true means another button owns the gesture
// and this one is not answering.
//
// The button no longer moves to the touch origin while aiming. It used to, and
// with the buttons stacked in a column that was harmless; in a cluster a
// button that jumps under the finger lands on top of its neighbours and the
// player loses track of what is where. The drag line and the ground chevron
// (ui.AimIndicator) carry the direction instead.
func (sb *SkillButton) Draw(locked bool) {
	DrawActionButton(ActionButtonState{
		Center:        sb.Position,
		Radius:        sb.Radius,
		Icon:          sb.Icon,
		Label:         sb.Label,
		Aiming:        sb.IsActive,
		Origin:        sb.Origin,
		Current:       sb.Current,
		Locked:        locked,
		Cooldown:      sb.Cooldown,
		CooldownTotal: sb.CooldownTotal,
	})
}

// SetCooldown feeds the button the authoritative recharge state for the frame.
func (sb *SkillButton) SetCooldown(remaining, total float32) {
	sb.Cooldown = remaining
	sb.CooldownTotal = total
}

// hudScale is the accessibility multiplier from Configuracoes > Interface >
// Escala do HUD e do Texto (0.8..settings.HUDScaleMax).
//
// Era lido SO aqui, e essa era a queixa: o jogador aumentava a escala e a
// barra de vida crescia sozinha enquanto o resto do texto continuava igual.
// Desde 20/09/2026 ele e o multiplicador de todo o texto do HUD, atraves de
// TextSize/FitTextSize (text.go).
var hudScale float32 = 1.0

// SetHUDScale applies the HUD scale preference. Expects an already-clamped
// value (settings.Settings.HUDScale, 0.8..settings.HUDScaleMax). E lido por
// todo o texto do HUD atraves de text.go, e nao so pela barra de vida.
func SetHUDScale(v float32) { hudScale = v }

// DrawHealthBar draws the player's health bar in the top-left corner.
// All dimensions are derived from the current screen size so the bar
// scales correctly on any resolution (fullscreen, windowed, etc.), times
// the player's chosen HUD scale.
func DrawHealthBar(health, maxHealth float32) {
	sw := float32(rl.GetScreenWidth())
	barWidth := sw * 0.20 * hudScale
	// Os numeros fixos sao pixels da tela nativa (Px, text.go): em 540p a
	// barra nao pode ficar com o dobro da altura relativa.
	barHeight := Px(20) * hudScale
	x := Px(10)
	y := Px(60)
	// 0.015 -> 0.019 e piso 14 -> MinTextSize (18), em 20/09/2026: numa janela
	// 1280x720 o "HP/HP" saia a 19 px, e era um dos textos que os testadores
	// citaram. A razao ja carrega hudScale, entao aqui e FitTextSize (so o
	// piso) e nao TextSize — ver text.go.
	fontSize := FitTextSize(sw * 0.019 * hudScale)

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
	rl.DrawRectangleLinesEx(rl.NewRectangle(float32(x), float32(y), barWidth, barHeight), max(Px(2), 1), rl.White)

	// Text
	healthText := fmt.Sprintf("%.0f/%.0f", health, maxHealth)
	rl.DrawText(healthText, int32(x+barWidth+Px(10)), int32(y), fontSize, rl.White)
}
