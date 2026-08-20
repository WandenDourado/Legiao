package ui

// Desktop skill bar. On touch the cooldown is drawn straight on the ability
// buttons, but the keyboard has no buttons to draw on, so the same information
// gets a small bar of pips at the bottom of the screen — one per bound
// ability, labelled with the key that casts it.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// CooldownEntry is one slot of the desktop skill bar.
type CooldownEntry struct {
	Key       string  // key that casts it ("Q", "R")
	Label     string  // short skill name, drawn under the pip
	Remaining float32 // seconds left, 0 when ready
	Total     float32 // full cooldown, drives the drain ring
	Accent    rl.Color
}

// DrawSkillBar renders the desktop cooldown pips centred at the bottom of the
// screen. It is a no-op for an empty list.
func DrawSkillBar(sw, sh float32, entries []CooldownEntry) {
	if len(entries) == 0 {
		return
	}

	radius := sh * 0.035
	if radius < 22 {
		radius = 22
	}
	gap := radius * 0.7
	labelSize := labelFontSize(radius)

	// Slots are spaced by whatever is wider, the pips or their names. Spacing
	// by the circles alone printed "Bola de Fogo" over "Chuva de Meteoros",
	// because a skill name is far wider than the pip it belongs to.
	step := radius*2 + gap
	for _, e := range entries {
		if e.Label == "" {
			continue
		}
		if w := float32(rl.MeasureText(e.Label, labelSize)) + gap; w > step {
			step = w
		}
	}

	// Centres are spaced by step, so the row spans step*(n-1) between the
	// first and last centre; halving that centres the whole bar.
	x := sw/2 - step*float32(len(entries)-1)/2
	y := sh - radius - sh*0.06

	for _, e := range entries {
		center := rl.NewVector2(x, y)
		ready := e.Remaining <= 0

		fill := rl.Fade(e.Accent, 0.35)
		if ready {
			fill = rl.Fade(e.Accent, 0.75)
		}
		rl.DrawCircleV(center, radius, fill)
		rl.DrawCircleLinesV(center, radius, rl.White)

		keySize := int32(radius * 0.9)
		keyWidth := rl.MeasureText(e.Key, keySize)
		rl.DrawText(e.Key,
			int32(center.X)-keyWidth/2,
			int32(center.Y)-keySize/2,
			keySize, rl.White)

		// The overlay goes on top of the key so a recharging slot reads as a
		// number first and a key second.
		DrawCooldownOverlay(center, radius, e.Remaining, e.Total)

		if e.Label != "" {
			labelWidth := rl.MeasureText(e.Label, labelSize)
			rl.DrawText(e.Label,
				int32(center.X)-labelWidth/2,
				int32(center.Y+radius)+6,
				labelSize, rl.Fade(rl.White, 0.85))
		}

		x += step
	}
}

// labelFontSize is the skill-name size derived from the pip radius, floored so
// the names stay legible on a small window.
func labelFontSize(radius float32) int32 {
	size := int32(radius * 0.5)
	if size < 12 {
		size = 12
	}
	return size
}
