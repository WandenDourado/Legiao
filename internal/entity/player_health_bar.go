package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// DrawAllyHealthBar renders a remote party member's health above their head,
// in world space. It exists apart from drawEnemyHealthBar (enemy.go) on
// purpose: a player is drawn CENTERED on Position while a directional
// monster is anchored at its feet, so the two need different geometry, and
// the ally reads "friend, not target" through softer color and a bordered
// bar shape that survives a screen full of monster bars in the same colors.
//
// health/maxHealth come straight off network.PlayerState: Health every tick,
// MaxHealth recomposed by the identity cache from the join-time announce
// (doc/network.md, "Identidade vai uma vez, estado vai sempre"). Callers must
// still skip the call while MaxHealth <= 0 (identity not applied yet) or the
// player is dead; this function only guards the arithmetic, not those two
// game-state decisions.
const (
	// allyBarAboveMargin is extra clearance above the character's frame top,
	// so the bar never touches hair/weapon geometry drawn near the top row.
	allyBarAboveMargin = 12
	// allyBarHalfWidthFactor derives the bar's half-width from PlayerSize,
	// the one shared hitbox radius every character uses.
	allyBarHalfWidthFactor = 1.4
	allyBarHeight          = 5.0
)

// allyBarColor maps a health fraction to the ally palette. Same thresholds as
// the monster bar (0.5, 0.25) so "how much health" reads the same way; the
// colors themselves are deliberately lighter/less saturated so an ally is
// never mistaken for a target at a glance.
func allyBarColor(percent float32) rl.Color {
	switch {
	case percent > 0.5:
		return rl.NewColor(150, 226, 170, 255) // soft mint green
	case percent > 0.25:
		return rl.NewColor(240, 208, 138, 255) // soft amber
	default:
		return rl.NewColor(235, 145, 148, 255) // soft rose red
	}
}

// allyBarLayout returns how far above Position the bar sits and its
// half-width, derived from the character's own frame size instead of the
// enemy's foot-anchored geometry. The character sprite is centered on
// Position (see GroundOffset), so the frame's top edge is already
// (FrameHeight/2)*RenderScale above Position; the bar clears that plus a
// fixed margin.
func allyBarLayout(def CharacterDef) (above, halfWidth float32) {
	scale := def.RenderScale
	if scale <= 0 {
		scale = 1
	}
	frameTop := float32(def.FrameHeight) / 2 * scale
	return frameTop + allyBarAboveMargin, PlayerSize * allyBarHalfWidthFactor
}

// allyBarFraction clamps health/maxHealth to [0,1] and reports whether the
// bar should be drawn at all. maxHealth <= 0 means identity has not been
// applied yet (or is a divide-by-zero guard) and the caller must skip drawing.
func allyBarFraction(health, maxHealth float32) (float32, bool) {
	if maxHealth <= 0 {
		return 0, false
	}
	frac := health / maxHealth
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	return frac, true
}

// DrawAllyHealthBar draws the bar itself. Callers decide whether to draw at
// all (MaxHealth <= 0, IsDead) — this function only handles geometry, color
// and the health fraction.
func DrawAllyHealthBar(def CharacterDef, x, y, health, maxHealth float32) {
	frac, ok := allyBarFraction(health, maxHealth)
	if !ok {
		return
	}

	above, halfWidth := allyBarLayout(def)
	barWidth := halfWidth * 2
	barX := x - halfWidth
	barY := y - above

	rl.DrawRectangle(int32(barX), int32(barY), int32(barWidth), int32(allyBarHeight), rl.Fade(rl.Black, 0.6))

	fillWidth := barWidth * frac
	rl.DrawRectangle(int32(barX), int32(barY), int32(fillWidth), int32(allyBarHeight), allyBarColor(frac))

	rl.DrawRectangleLines(int32(barX), int32(barY), int32(barWidth), int32(allyBarHeight), rl.Fade(rl.White, 0.35))
}
