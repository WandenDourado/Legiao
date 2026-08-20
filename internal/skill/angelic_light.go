package skill

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// lightShaft is one column of divine light descending somewhere inside the
// altar — the gods blessing the consecrated ground. Offsets are relative to
// the altar center; the shaft fades in, holds, and fades out.
type lightShaft struct {
	offX     float32
	width    float32
	height   float32
	life     float32
	duration float32
}

// updateShafts spawns and ages the descending light columns (a few at a time
// so the blessing feels continuous but never noisy).
func (a *AngelicArea) updateShafts(dt float32) {
	if a.IsHealing() {
		a.shaftAcc += dt
		for a.shaftAcc >= 0.55 {
			a.shaftAcc -= 0.55
			a.shafts = append(a.shafts, &lightShaft{
				offX:     (rand.Float32()*2 - 1) * AngelicRadius * 0.8,
				width:    26 + rand.Float32()*30,
				height:   260 + rand.Float32()*90,
				duration: 1.6 + rand.Float32()*0.9,
			})
		}
	}
	kept := a.shafts[:0]
	for _, s := range a.shafts {
		s.life += dt
		if s.life < s.duration {
			kept = append(kept, s)
		}
	}
	a.shafts = kept
}

// shaftAlpha eases a shaft in and out over its life (0..1).
func (s *lightShaft) shaftAlpha() float32 {
	p := s.life / s.duration
	// smooth rise/fall: sin bump
	return float32(math.Sin(float64(p) * math.Pi))
}

// drawBlessingBeam renders the permanent central column of light plus the
// wandering descending shafts. Must be called inside additive blending.
func (a *AngelicArea) drawBlessingBeam(fade float32) {
	cx := a.Position.X
	cy := a.Position.Y

	// Great central beam: wide blue halo column + bright white core column.
	pulse := 1 + 0.06*float32(math.Sin(float64(a.Age)*2.6))
	beamH := int32(340)
	outerW := int32(120 * pulse)
	innerW := int32(46 * pulse)
	rl.DrawRectangleGradientV(int32(cx)-outerW/2, int32(cy)-beamH, outerW, beamH,
		rl.Fade(angelicBlue, 0), rl.Fade(angelicBlue, 0.30*fade))
	rl.DrawRectangleGradientV(int32(cx)-innerW/2, int32(cy)-beamH, innerW, beamH,
		rl.Fade(angelicWhite, 0), rl.Fade(angelicWhite, 0.42*fade))
	// Bloom where the beam touches the altar ground.
	rl.DrawCircleGradient(int32(cx), int32(cy), 110*pulse,
		rl.Fade(angelicWhite, 0.30*fade), rl.Blank)
	rl.DrawCircleGradient(int32(cx), int32(cy), 52*pulse,
		rl.Fade(angelicWhite, 0.45*fade), rl.Blank)

	// Wandering descending shafts across the consecrated ground.
	for _, s := range a.shafts {
		al := s.shaftAlpha() * fade
		x := cx + s.offX
		w := int32(s.width)
		h := int32(s.height)
		rl.DrawRectangleGradientV(int32(x)-w/2, int32(cy)-h, w, h,
			rl.Fade(angelicWhite, 0), rl.Fade(angelicWhite, 0.30*al))
		rl.DrawCircleGradient(int32(x), int32(cy), s.width*1.4,
			rl.Fade(angelicBlue, 0.25*al), rl.Blank)
	}
}
