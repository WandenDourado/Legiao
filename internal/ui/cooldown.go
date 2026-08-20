package ui

// Shared drawing for "this is recharging" feedback. Skills are round buttons
// on touch and round pips on desktop, so both share one overlay: a dark disc,
// a ring that empties as the timer runs out, and the seconds left in figures.

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawCooldownOverlay covers a circular button with its remaining cooldown.
// It draws nothing when remaining <= 0, so callers can hand it every slot
// without checking first. total is the skill's full cooldown and only drives
// the ring; pass 0 to get the number alone.
func DrawCooldownOverlay(center rl.Vector2, radius, remaining, total float32) {
	if remaining <= 0 {
		return
	}

	DrawCooldownRing(center, radius, remaining, total)

	text := FormatCooldown(remaining)
	fontSize := int32(radius * 0.95)
	if fontSize < 14 {
		fontSize = 14
	}
	textWidth := rl.MeasureText(text, fontSize)
	rl.DrawText(text,
		int32(center.X)-textWidth/2,
		int32(center.Y)-fontSize/2,
		fontSize, rl.White)
}

// DrawCooldownRing is the overlay without the figures: a dark disc and a ring
// that drains clockwise from the top, the direction the eye expects a timer to
// move. Used for the basic attack, whose cadence is fast enough that a number
// would be an unreadable blur.
func DrawCooldownRing(center rl.Vector2, radius, remaining, total float32) {
	if remaining <= 0 {
		return
	}
	rl.DrawCircleV(center, radius, rl.Fade(rl.Black, 0.55))
	if total <= 0 {
		return
	}
	sweep := 360 * (remaining / total)
	if sweep > 360 {
		sweep = 360
	}
	rl.DrawRing(center, radius*0.74, radius, -90, -90+sweep, 40,
		rl.Fade(rl.White, 0.30))
}

// FormatCooldown renders the seconds left: one decimal while the wait is short
// enough to matter frame by frame, whole seconds above that.
func FormatCooldown(remaining float32) string {
	if remaining <= 0 {
		return ""
	}
	if remaining < 10 {
		return fmt.Sprintf("%.1f", remaining)
	}
	return fmt.Sprintf("%.0f", remaining)
}
