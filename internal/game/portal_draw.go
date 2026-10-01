package game

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Portals are drawn from raylib primitives, not from an image asset: they are
// pure light, so an animated vortex reads better than a static sprite, costs no
// atlas space and stays coherent with the rule of `doc/art_style.md` that all
// glow belongs to the runtime and never to the painted art.
//
// The shape is an UPRIGHT oval standing on the floor, not a puddle: the portal
// is a doorway, so it is measured against the character (~186 px tall) exactly
// like a house door is, and it touches the ground without frame or pedestal.

// portalWidthRatio and portalHeightRatio measure the oval against the portal
// object of the map. A 128 px object gives a 115x186 px doorway, which is the
// character silhouette of the scale table in `doc/art_style.md`.
const (
	portalWidthRatio  = 0.45
	portalHeightRatio = 1.45
)

// portalFloorLayers is how many stacked ellipses build the light pool on the
// ground. Additive blending turns the stack into a gradient without a shader.
const portalFloorLayers = 6

// portalHaloLayers is the same trick for the light that bleeds onto whatever
// stands next to the portal.
const portalHaloLayers = 4

var (
	// Deep azure at the rim, sky mist in between, near-white in the middle and
	// on the edge: an old, stable spell, not a chaotic rift.
	portalDeep      = rl.NewColor(38, 104, 196, 255)
	portalMist      = rl.NewColor(120, 198, 240, 255)
	portalPale      = rl.NewColor(196, 238, 255, 255)
	portalTurquoise = rl.NewColor(96, 226, 214, 255)
	portalRim       = rl.NewColor(214, 246, 255, 255)
)

// DrawPortals paints every portal of this world, under the entities, and
// advances how far they have materialised. The progress lives on the World so
// it resets by itself when the map changes: a portal always opens in front of
// the players who earned it.
func (w *World) DrawPortals() {
	w.portalReveal = advancePortalReveal(w.portalReveal, PortalsUnlocked(), rl.GetFrameTime())
	if w.portalReveal <= 0 {
		return
	}

	t := float32(rl.GetTime())
	for i, portal := range w.Portals {
		// Each portal runs on its own clock offset so two doorways in the same
		// map never pulse in lockstep.
		shape := drawPortal(portal, t+float32(i)*0.7, w.portalReveal)
		if i < len(w.partyTally) {
			drawPortalCounter(shape, w.partyTally[i])
		}
	}
}

// drawPortal paints one doorway and returns the geometry it used, so the party
// counter is placed against the same oval instead of measuring it a second
// time.
func drawPortal(portal tilemap.Portal, t, reveal float32) portalShape {
	shape := portalShapeOf(portal, reveal)

	rl.BeginBlendMode(rl.BlendAdditive)
	drawPortalFloor(shape, t)
	drawPortalHalo(shape, t)
	rl.EndBlendMode()

	// The surface is the one part that is NOT additive. Added light over the
	// saturated grass of the map keeps the green underneath and the mist turns
	// teal; and a doorway you can see the ground through stops reading as a
	// doorway. So the surface is painted with alpha and actually covers what is
	// behind it, and only the glow around it is additive.
	drawPortalSurface(shape, t)

	rl.BeginBlendMode(rl.BlendAdditive)
	drawPortalHeart(shape, t)
	drawPortalVortex(shape, t)
	drawPortalRipples(shape, t)
	drawPortalRim(shape, t)
	drawPortalSparks(shape, t)
	rl.EndBlendMode()

	return shape
}

// drawPortalFloor is the pool of light on the ground plus the discreet ring
// that marks where the player has to stand. Flattened to about a third because
// the camera is top-down 3/4: a circle on the floor is seen as an ellipse.
func drawPortalFloor(s portalShape, t float32) {
	breath := 1 + 0.04*float32(math.Sin(float64(t)*0.9))
	for layer := 0; layer < portalFloorLayers; layer++ {
		spread := (1.5 - 0.2*float32(layer)) * breath
		rl.DrawEllipse(int32(s.Ground.X), int32(s.Ground.Y),
			s.RX*spread, s.RX*0.36*spread,
			s.fade(portalMist, 0.05))
	}

	ring := s.fade(portalRim, 0.20+0.08*float32(math.Abs(math.Sin(float64(t)*1.1))))
	rl.DrawEllipseLines(int32(s.Ground.X), int32(s.Ground.Y),
		s.RX*0.92*breath, s.RX*0.32*breath, ring)
}

// drawPortalHalo is the soft light the portal throws on its surroundings. It
// stays deliberately dim: the doorway lights the scene, it never blows it out.
func drawPortalHalo(s portalShape, t float32) {
	pulse := 1 + 0.03*float32(math.Sin(float64(t)*0.8))
	for layer := 0; layer < portalHaloLayers; layer++ {
		spread := (1.45 - 0.12*float32(layer)) * pulse
		rl.DrawEllipse(int32(s.Center.X), int32(s.Center.Y),
			s.RX*spread, s.RY*spread, s.fade(portalDeep, 0.05))
	}
}
