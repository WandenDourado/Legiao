package skill

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// angelFeather is one small white plume drifting down inside the angelic
// area. Positions are RELATIVE to the area center so the feathers travel
// with the caster. Drawn procedurally: a slim two-triangle plume with a
// central quill line and a soft glow — no sprites.
type angelFeather struct {
	relX     float32 // horizontal offset from center at spawn
	startY   float32 // vertical offset where the fall starts (above the area)
	fallLen  float32 // total vertical distance of the fall
	life     float32 // elapsed
	duration float32 // total fall time
	sway     float32 // horizontal sway phase
	size     float32 // plume length
}

// updateFeathers spawns and ages the plumes (up to ~22 per second — a dense
// heavenly rain across the altar).
func (a *AngelicArea) updateFeathers(dt float32) {
	if a.IsHealing() {
		// Spawn continuously while the altar is active (dense, gentle rain
		// of plumes across the whole area).
		for i := 0; i < 2; i++ {
			if rand.Float32() < dt*11 {
				a.feathers = append(a.feathers, &angelFeather{
					relX:     (rand.Float32()*2 - 1) * AngelicRadius * 0.9,
					startY:   -240 - rand.Float32()*80,
					fallLen:  260 + rand.Float32()*120,
					duration: 2.6 + rand.Float32()*1.4,
					sway:     rand.Float32() * 2 * math.Pi,
					size:     10 + rand.Float32()*7,
				})
			}
		}
	}
	kept := a.feathers[:0]
	for _, f := range a.feathers {
		f.life += dt
		if f.life < f.duration {
			kept = append(kept, f)
		}
	}
	a.feathers = kept
}

// drawFeathers renders the plumes (already inside additive blending).
func (a *AngelicArea) drawFeathers(fade float32) {
	for _, f := range a.feathers {
		p := f.life / f.duration
		// Feathers materialize, drift down swaying, and dissolve near ground.
		alpha := fade
		if p < 0.15 {
			alpha *= p / 0.15
		} else if p > 0.8 {
			alpha *= (1 - p) / 0.2
		}
		swayX := float32(math.Sin(float64(f.life)*1.7+float64(f.sway))) * 26
		x := a.Position.X + f.relX + swayX
		y := a.Position.Y + f.startY + f.fallLen*p

		// Plume orientation rocks with the sway (like a leaf falling).
		tilt := float64(math.Sin(float64(f.life)*1.7+float64(f.sway))) * 0.7
		dir := rl.NewVector2(float32(math.Sin(tilt)), float32(math.Cos(tilt)))
		perp := rl.NewVector2(-dir.Y, dir.X)
		tipP := rl.NewVector2(x+dir.X*f.size*0.5, y+dir.Y*f.size*0.5)
		base := rl.NewVector2(x-dir.X*f.size*0.5, y-dir.Y*f.size*0.5)
		side1 := rl.NewVector2(x+perp.X*f.size*0.22, y+perp.Y*f.size*0.22)
		side2 := rl.NewVector2(x-perp.X*f.size*0.22, y-perp.Y*f.size*0.22)

		white := rl.Fade(angelicWhite, 0.8*alpha)
		// Two triangles (both windings) form the slim plume body.
		rl.DrawTriangle(base, side1, tipP, white)
		rl.DrawTriangle(base, tipP, side1, white)
		rl.DrawTriangle(base, side2, tipP, white)
		rl.DrawTriangle(base, tipP, side2, white)
		// Quill line + faint blue glow.
		rl.DrawLineEx(base, tipP, 1, rl.Fade(angelicBlue, 0.6*alpha))
		rl.DrawCircleGradient(int32(x), int32(y), f.size*0.8,
			rl.Fade(angelicBlue, 0.18*alpha), rl.Blank)
	}
}

// randomPointInRadius returns a uniformly distributed offset inside a circle.
func randomPointInRadius(radius float32) rl.Vector2 {
	ang := rand.Float64() * 2 * math.Pi
	dist := float32(math.Sqrt(rand.Float64())) * radius
	return rl.NewVector2(float32(math.Cos(ang))*dist, float32(math.Sin(ang))*dist)
}

// easeOutCubic maps 0..1 with a fast start and gentle settle.
func easeOutCubic(t float32) float32 {
	inv := 1 - t
	return 1 - inv*inv*inv
}
