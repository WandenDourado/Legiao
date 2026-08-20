package game

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Geometry and colour helpers shared by every layer of the portal drawing.

// portalShape is the measured geometry of one portal: an upright oval whose
// base rests on the floor at the map object's position.
type portalShape struct {
	// Center is the middle of the upright oval, above the ground.
	Center rl.Vector2
	// Ground is where the oval meets the floor, the interaction point.
	Ground rl.Vector2
	RX, RY float32
	// Reveal is how far the portal has materialised, 0 to 1. Every layer of the
	// drawing is faded by it, so the doorway opens instead of popping in.
	Reveal float32
}

// portalShapeOf derives the oval from the portal rectangle. The rectangle stays
// the trigger area; only the drawing stands up out of it.
//
// While it opens the oval also grows, from a bit over half size to full, and
// the base stays put: the growth is applied before the centre is offset, so the
// portal never floats off the floor mid-animation.
func portalShapeOf(portal tilemap.Portal, reveal float32) portalShape {
	ground := portal.Center()
	grow := 0.55 + 0.45*reveal
	rx := portal.Rect.Width * portalWidthRatio * grow
	ry := portal.Rect.Height * portalHeightRatio / 2 * grow
	return portalShape{
		Center: rl.NewVector2(ground.X, ground.Y-ry),
		Ground: ground,
		RX:     rx,
		RY:     ry,
		Reveal: reveal,
	}
}

// fade applies a layer's own alpha and the materialisation together, so the
// whole portal dims as one while it opens.
func (s portalShape) fade(color rl.Color, alpha float32) rl.Color {
	return rl.Fade(color, alpha*s.Reveal)
}

// point maps an angle and a normalized radius (1 = the rim) onto the oval.
func (s portalShape) point(angle float64, radius float32) rl.Vector2 {
	return rl.NewVector2(
		s.Center.X+s.RX*radius*float32(math.Cos(angle)),
		s.Center.Y+s.RY*radius*float32(math.Sin(angle)),
	)
}

// portalBlend mixes two portal colours, so the surface can fade from one to the
// next across its layers instead of showing rings of flat colour.
func portalBlend(from, to rl.Color, f float32) rl.Color {
	if f < 0 {
		f = 0
	} else if f > 1 {
		f = 1
	}
	mix := func(a, b uint8) uint8 { return uint8(float32(a) + (float32(b)-float32(a))*f) }
	return rl.NewColor(mix(from.R, to.R), mix(from.G, to.G), mix(from.B, to.B), 255)
}
