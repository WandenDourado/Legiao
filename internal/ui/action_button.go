package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ActionButtonState is one action button's whole appearance for one frame.
//
// The three buttons are drawn by ONE function because they are one object to
// the player: a cluster where the difference between the middle and the two
// orbiting it must be size and glyph, never style. They used to be drawn in
// two different packages by two different functions (input.DrawAimJoystick
// for the fire button, SkillButton.Draw for the abilities) and had drifted
// apart — different fills, different rings, one with a knob at the finger and
// one without.
//
// It is a struct of values and not a method on the buttons because the fire
// button has no type of its own: it is an input.AimJoystick, which lives in a
// package that cannot import this one.
type ActionButtonState struct {
	Center rl.Vector2
	Radius float32
	Icon   ActionIcon
	// Label is drawn only when the icon file is missing. A button with
	// nothing on it is worse than a button with the wrong-language word.
	Label string
	// Aiming is true while a finger is on this button; Origin and Current are
	// the drag, read only then.
	Aiming  bool
	Origin  rl.Vector2
	Current rl.Vector2
	// Locked is true while ANOTHER button owns the aim gesture. See
	// input.ActionLock.
	Locked bool
	// Cooldown/CooldownTotal drive the recharge counter. RingOnly picks the
	// bare ring instead of the ring plus seconds: the basic attack recharges
	// one to three times a SECOND, and figures at that rate are an
	// unreadable blur.
	Cooldown      float32
	CooldownTotal float32
	RingOnly      bool
}

// ActionNeutral is the one colour all three buttons wear.
//
// They used to be red, orange and gold — one hue per slot, on the theory that
// colour is what tells the player which button is which. On a screen already
// carrying health bars, damage numbers, spell light and a wave counter, three
// saturated circles in the corner read as three ALERTS. And the cue was
// redundant: the glyphs already differ, and the middle one is half again as
// big as the other two.
//
// So the colour went out and the buttons wear the glyph's own near-white.
// What still separates them is size, position and drawing — none of which
// competes with the fight for attention. Slightly cool and not pure white so
// it sits with the UI rather than glaring out of it.
var ActionNeutral = rl.NewColor(238, 242, 248, 255)

const (
	// actionFillAlpha is the disc BEHIND the glyph. Low, because the PNG
	// carries its own translucent disc: a strong fill under it turns the
	// button into a solid blob and the icon stops reading.
	actionFillAlpha = 0.22
	// actionRingAlpha is the outline, and the only thing that draws the
	// button's edge against the map — the fill is too faint to. Raised from
	// 0.85 when the colour went out: a white ring has less contrast against
	// bright grass than a gold one did.
	actionRingAlpha = 0.95
	actionRingThick = 2.5
	// actionLockedDim multiplies every alpha while another button owns the
	// gesture. 0.38 is dark enough to read as "not now" at a glance and light
	// enough that the player can still see WHICH buttons went quiet.
	actionLockedDim = 0.38
	// actionAimGrow is how much the button swells under the finger.
	actionAimGrow = 1.06
)

// actionOpacity e o multiplicador global vindo de Configuracoes > Controles
// > Opacidade dos controles (0..1). Separado de actionLockedDim: os dois
// multiplicam, um e preferencia do jogador, o outro e estado de jogo (outro
// botao segurando o gesto).
var actionOpacity float32 = 1.0

// SetActionOpacity aplica a preferencia de opacidade dos controles. Espera
// vir clampada (settings.Settings.ControlsOpacity, 0..1).
func SetActionOpacity(v float32) { actionOpacity = v }

// DrawActionButton renders one button of the action cluster.
func DrawActionButton(s ActionButtonState) {
	dim := actionOpacity
	if s.Locked {
		dim *= actionLockedDim
	}
	radius := s.Radius
	if s.Aiming {
		radius *= actionAimGrow
	}

	// The drag line first, so the button sits on top of its own feedback.
	if s.Aiming {
		rl.DrawLineEx(s.Origin, s.Current, 2, rl.Fade(ActionNeutral, 0.55))
		rl.DrawCircleV(s.Current, s.Radius*0.28, rl.Fade(ActionNeutral, 0.7))
	}

	rl.DrawCircleV(s.Center, radius, rl.Fade(ActionNeutral, actionFillAlpha*dim))

	iconTint := rl.Fade(rl.White, dim)
	if s.Aiming {
		iconTint = rl.White
	}
	if !DrawActionIcon(s.Icon, s.Center, radius, iconTint) && s.Label != "" {
		// O raio do botao ja carrega a escala de controles do jogador, entao
		// so o piso se aplica. Ver text.go.
		size := FitTextSize(radius * 0.55)
		w := rl.MeasureText(s.Label, size)
		rl.DrawText(s.Label,
			int32(s.Center.X)-w/2,
			int32(s.Center.Y)-size/2,
			size, rl.Fade(rl.White, dim))
	}

	ring := actionRingAlpha * dim
	if s.Aiming {
		ring = 1
	}
	rl.DrawRing(s.Center, radius-Px(actionRingThick), radius, 0, 360, 48,
		rl.Fade(ActionNeutral, ring))

	if s.RingOnly {
		DrawCooldownRing(s.Center, s.Radius, s.Cooldown, s.CooldownTotal)
	} else {
		DrawCooldownOverlay(s.Center, s.Radius, s.Cooldown, s.CooldownTotal)
	}
}
