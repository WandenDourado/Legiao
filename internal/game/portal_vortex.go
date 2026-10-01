package game

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// The body of the portal: the translucent surface, its luminous heart, the slow
// whirlpool inside it and the bright rim closing the shape. Everything here
// turns at a constant speed and breathes on a sine, never in steps — the portal
// is a permanent spell, so no movement is allowed to snap.

const (
	// portalSurfaceLayers stacks ellipses from the rim inward. Sixteen thin
	// layers read as a gradient; half a dozen thick ones read as onion rings.
	portalSurfaceLayers = 16
	// portalHeartLayers piles light in the middle, so the centre glows instead
	// of being the densest, darkest part of the stack.
	portalHeartLayers = 6
	// portalArms is how many coils of mist the whirlpool has.
	portalArms = 4
	// portalArmSteps is the segment count of one coil.
	portalArmSteps = 28
	// portalArmTurns is how far a coil wraps around the centre, in turns.
	portalArmTurns = 1.35
	// portalSpin is the rotation speed of the whole vortex, in radians per
	// second. Slow on purpose.
	portalSpin = 0.45
	// portalRipples is how many rings travel inward across the surface.
	portalRipples = 3
)

// drawPortalSurface fills the oval with luminous, translucent energy: deep
// azure at the rim fading to near-white in the middle, with no window onto the
// destination — the inside stays abstract.
func drawPortalSurface(s portalShape, t float32) {
	swell := 1 + 0.015*float32(math.Sin(float64(t)*1.3))
	for layer := 0; layer < portalSurfaceLayers; layer++ {
		depth := float32(layer) / (portalSurfaceLayers - 1)
		color := portalBlend(portalDeep, portalMist, depth/0.55)
		if depth >= 0.55 {
			color = portalBlend(portalMist, portalPale, (depth-0.55)/0.45)
		}
		scale := (1 - float32(layer)/portalSurfaceLayers) * swell
		rl.DrawEllipse(int32(s.Center.X), int32(s.Center.Y),
			s.RX*scale, s.RY*scale, s.fade(color, 0.13))
	}
}

// drawPortalHeart is the light gathered in the middle of the surface: the
// stacked ellipses only overlap near the centre, so the glow builds up there.
func drawPortalHeart(s portalShape, _ float32) {
	for layer := 0; layer < portalHeartLayers; layer++ {
		scale := 0.55 * (1 - float32(layer)/8)
		rl.DrawEllipse(int32(s.Center.X), int32(s.Center.Y),
			s.RX*scale, s.RY*scale, s.fade(portalPale, 0.05))
	}
}

// drawPortalVortex draws the whirlpool as spiral arms of mist. The arm coils
// outward while the whole pattern rotates, so the energy reads as FLOWING
// inward rather than as a picture being spun.
func drawPortalVortex(s portalShape, t float32) {
	spin := float64(t) * portalSpin
	for arm := 0; arm < portalArms; arm++ {
		offset := 2 * math.Pi * float64(arm) / portalArms
		previous := s.point(offset-spin, 0.05)

		for step := 1; step <= portalArmSteps; step++ {
			progress := float64(step) / portalArmSteps
			angle := offset - spin + progress*portalArmTurns*2*math.Pi

			// The wobble keeps the coil from looking like a drawn spiral; it is
			// small enough that an arm never crosses its neighbour.
			radius := float32(0.05 + 0.92*progress)
			radius += 0.03 * float32(math.Sin(float64(t)*1.7+progress*6))

			next := s.point(angle, radius)
			// Arms are thicker and brighter near the middle and thin out at the
			// rim, which is where the eye reads the direction of the flow.
			rl.DrawLineEx(previous, next,
				3.5-2.2*float32(progress),
				s.fade(portalPale, 0.16*float32(1-progress*0.55)))
			previous = next
		}
	}
}

// drawPortalRipples is the continuous rippling of the surface: rings that leave
// the rim and shrink toward the centre, fading in and out on a sine so neither
// end of the travel pops.
func drawPortalRipples(s portalShape, t float32) {
	for ripple := 0; ripple < portalRipples; ripple++ {
		phase := math.Mod(float64(t)*0.22+float64(ripple)/portalRipples, 1)
		scale := float32(1 - phase*0.85)
		alpha := 0.16 * float32(math.Sin(phase*math.Pi))
		rl.DrawEllipseLines(int32(s.Center.X), int32(s.Center.Y),
			s.RX*scale, s.RY*scale, s.fade(portalMist, alpha))
	}
}

// drawPortalRim closes the shape with the brightest line of the portal. There
// is no physical frame, so this edge is what tells the player where the light
// stops; it breathes slowly to keep the doorway alive.
func drawPortalRim(s portalShape, t float32) {
	breath := 0.50 + 0.16*float32(math.Abs(math.Sin(float64(t)*0.9)))

	// raylib only draws hairline ellipses, so a thick glowing stroke is faked
	// with passes falling off to each side of the true edge.
	falloff := [4]float32{1, 0.5, 0.25, 0.12}
	for pass := -3; pass <= 3; pass++ {
		distance := pass
		if distance < 0 {
			distance = -distance
		}
		offset := float32(pass) * 1.5
		rl.DrawEllipseLines(int32(s.Center.X), int32(s.Center.Y),
			s.RX+offset, s.RY+offset, s.fade(portalRim, breath*falloff[distance]))
	}

	// Two beads of light circling the edge, so the rim carries the same
	// rotation as the mist inside it.
	for bead := 0; bead < 2; bead++ {
		position := s.point(float64(t)*portalSpin+math.Pi*float64(bead), 1)
		rl.DrawCircleV(position, 3, s.fade(portalRim, 0.40*breath))
	}
}
