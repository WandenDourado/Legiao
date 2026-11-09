package ui

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// AimIndicator is the discreet chevron drawn in front of the player, on the
// ray the next attack or spell will travel.
//
// It exists because the aim on Android is invisible until it is too late: the
// direction lives in AimJoystick.AimDir / SkillButton.SkillDir and is only
// ever drawn next to the BUTTON, at the bottom-right corner, while the player
// is looking at the character on the other side of the screen. A tap fires in
// a direction configured several seconds ago and nothing on the character says
// which one.
//
// It is deliberately quiet: at rest it is a barely-there mark (aimRestAlpha),
// and it only brightens and swells while a finger is on one of the action
// buttons. The goal is a hint the player reads without noticing they read it,
// not a second HUD competing with the fight — which is also why it wears the
// buttons' neutral (ui.ActionNeutral) and not a colour of its own.
type AimIndicator struct {
	// t is the eased presence, 0 = resting, 1 = a finger is aiming. Easing it
	// instead of switching is what keeps the mark from popping in and out on
	// every tap: the player sees it swell, not blink.
	t float32
	// shown is the direction actually drawn, chased towards the resolved one
	// instead of assigned. The three buttons hold three different saved
	// directions, so letting go of Q hands the mark back to the attack aim:
	// assigning it would make the chevron TELEPORT to the other side of the
	// character. Chased, it swings there, which reads as "back to your
	// attack aim" instead of a glitch. Zero until the first Draw.
	shown rl.Vector2
}

// AimHint is everything the renderer resolves about the aim for one frame.
// Dir is in world space.
//
// It carried an Accent too, so the chevron could wear the colour of whichever
// button was being aimed. That went out with the buttons' own colours (see
// ui.ActionNeutral): three hues in the corner read as three alerts, and the
// mark on the ground was repeating a cue nobody needed. The chevron now says
// "aiming" by brightening and swelling, which is the same information without
// a fourth colour on the screen.
type AimHint struct {
	Dir    rl.Vector2
	Aiming bool
}

const (
	// aimChevronDistance is how far from the spawn point the mark floats, in
	// world pixels. 120 clears the sprite in every direction: the figure
	// reaches ~99 px above Position (top of the head) and ~103 below it (the
	// soles), so a shorter radius would put a north or south aim inside the
	// character.
	aimChevronDistance = 120.0
	aimChevronLength   = 22.0
	// aimChevronSpread is the angle between the tip and each wing, in
	// radians (~41 degrees). Wider reads as a "V" lying flat, narrower as an
	// arrowhead standing up.
	aimChevronSpread = 0.72
	aimChevronThick  = 3.0

	// aimRestAlpha was 0.16 first and disappeared into the grass: on the
	// browser harness it took a 3x zoom to find the mark at all, which makes
	// it decoration rather than information. 0.24 is the lowest value where
	// the shape still reads at 1:1 over a bright tileset.
	aimRestAlpha   = 0.24
	aimActiveAlpha = 0.85
	// aimFadeRate is the easing speed, per second. At 8 the swell takes about
	// an eighth of a second — under the tap window, so even a quick tap shows
	// the direction it fired in.
	aimFadeRate   = 8.0
	aimActiveGrow = 1.25
	// aimTurnRate is how fast the drawn direction chases the resolved one,
	// per second. High enough (about 55 ms to converge) that a dragging thumb
	// never sees the mark lag behind it, low enough that the hand-off between
	// two buttons is a swing and not a jump.
	aimTurnRate = 18.0
)

// NewAimIndicator creates the indicator in its resting state.
func NewAimIndicator() *AimIndicator {
	return &AimIndicator{}
}

// Draw renders the chevron for this frame.
//
// origin must be the point the shot LEAVES FROM — the player's Position, which
// is where the host spawns the projectile. It was the soles (entity.GroundPoint)
// for a day, on the theory that a mark on the ground reads better than one
// floating at chest height. It does read better, and it is wrong: the fireball
// flies out of the chest and the mark sat 103 px below it, so the two were
// parallel rays a body-height apart and the player saw the spell leave along a
// line the indicator was not on. An indicator that is prettier than the thing
// it indicates is not an indicator.
//
// Must be called inside the camera block.
func (ai *AimIndicator) Draw(h AimHint, origin rl.Vector2, dt float32) {
	// Normalizing a zero vector yields NaN and would paint garbage across the
	// map, so the length is checked BEFORE the division, not after.
	if rl.Vector2Length(h.Dir) < 0.001 {
		return
	}
	ai.ease(h.Aiming, dt)
	dir := ai.turn(rl.Vector2Normalize(h.Dir), dt)

	alpha := aimRestAlpha + (aimActiveAlpha-aimRestAlpha)*ai.t
	scale := 1 + (aimActiveGrow-1)*ai.t
	col := ActionNeutral
	col.A = uint8(alpha * 255)

	// A plain circle around the spawn point: the tip lands at exactly Dir, and
	// the chevron points at exactly Dir.
	//
	// It used to land on an ellipse, the vertical axis squashed to 0.55 so the
	// mark would read as lying on the ground. That squash does not just move
	// the mark, it ROTATES it: a 45-degree aim came out drawn at 28.8, and
	// every direction that was not exactly horizontal or vertical was off by
	// up to 16 degrees. The one number this thing exists to communicate was
	// the one number it was allowed to distort.
	offset := rl.Vector2Scale(dir, aimChevronDistance)
	tip := rl.Vector2Add(origin, offset)

	back := rl.Vector2Scale(dir, -aimChevronLength*scale)
	left := rl.Vector2Add(tip, rotateVec2(back, aimChevronSpread))
	right := rl.Vector2Add(tip, rotateVec2(back, -aimChevronSpread))

	// A darker, slightly fatter pass underneath. A quarter-alpha white line on
	// bright grass has almost no contrast of its own; this dark rim is what
	// keeps the resting state readable without raising its alpha.
	shadow := rl.NewColor(0, 0, 0, uint8(alpha*140))
	rl.DrawLineEx(tip, left, aimChevronThick+2, shadow)
	rl.DrawLineEx(tip, right, aimChevronThick+2, shadow)

	rl.DrawLineEx(tip, left, aimChevronThick, col)
	rl.DrawLineEx(tip, right, aimChevronThick, col)
}

// ease moves t towards its target at aimFadeRate per second, clamped so a
// long frame cannot overshoot past the target and oscillate.
func (ai *AimIndicator) ease(aiming bool, dt float32) {
	target := float32(0)
	if aiming {
		target = 1
	}
	step := dt * aimFadeRate
	if step > 1 {
		step = 1
	}
	if step < 0 {
		step = 0
	}
	ai.t += (target - ai.t) * step
	if ai.t < 0.001 {
		ai.t = 0
	}
	if ai.t > 0.999 {
		ai.t = 1
	}
}

// turn chases the drawn direction towards the resolved one and returns it
// normalized. The first call snaps (there is nothing to chase from), so the
// mark does not sweep in from an arbitrary angle on the first frame of a run.
func (ai *AimIndicator) turn(target rl.Vector2, dt float32) rl.Vector2 {
	if rl.Vector2Length(ai.shown) < 0.001 {
		ai.shown = target
		return target
	}
	step := dt * aimTurnRate
	if step > 1 {
		step = 1
	}
	if step < 0 {
		step = 0
	}
	ai.shown = rl.Vector2Add(ai.shown, rl.Vector2Scale(rl.Vector2Subtract(target, ai.shown), step))
	// Chasing a unit vector through its INTERIOR shortens it, and an exact
	// 180-degree turn passes through zero. Both are handled by renormalizing
	// and falling back to the target when the length collapses.
	if rl.Vector2Length(ai.shown) < 0.05 {
		ai.shown = target
		return target
	}
	ai.shown = rl.Vector2Normalize(ai.shown)
	return ai.shown
}

// rotateVec2 rotates v by angle radians (screen space: +y is down, so a
// positive angle turns clockwise on screen).
func rotateVec2(v rl.Vector2, angle float32) rl.Vector2 {
	sin := float32(math.Sin(float64(angle)))
	cos := float32(math.Cos(float64(angle)))
	return rl.NewVector2(v.X*cos-v.Y*sin, v.X*sin+v.Y*cos)
}
