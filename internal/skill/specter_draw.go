package skill

// Procedural rendering of one specter, matching the ghoul reference: hunched
// gray corpse-flesh mass whose face is one huge gaping maw full of bone fangs,
// long arms with dark bandaged forearms ending in pale claws, hovering on a
// tattered wispy lower body. Frightening, dead, hungry.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Draw renders the ghoul in world space (front view, leaning toward Facing).
func (s *Specter) Draw() {
	a := s.alpha()
	if a <= 0 {
		return
	}
	t := float64(s.Age + s.Phase)
	bob := float32(math.Sin(t*4.2)) * 2.5
	lunge := clamp01(s.lungeT / specterLungeTime)
	pos := rl.NewVector2(
		s.Position.X+s.Facing.X*lunge*11,
		s.Position.Y+bob+s.Facing.Y*lunge*11,
	)
	r := SpecterRadius
	if s.Dying { // swells while dissolving into mist
		r *= 1 + s.DieAge/specterDissolve*0.5
	}
	lean := s.Facing.X * r * 0.3 // the maw leans toward the prey

	// Ground shadow (it hovers, but death has weight).
	rl.DrawEllipse(int32(pos.X), int32(pos.Y+r*1.6), r*1.1, r*0.32, rl.Fade(rl.Black, 0.30*a))

	// Long arms first (behind the torso): shoulder -> elbow -> bandaged
	// forearm -> three bone claws raking the ground.
	for _, side := range []float32{-1, 1} {
		sh := rl.NewVector2(pos.X+side*r*1.0, pos.Y-r*0.45)
		el := rl.NewVector2(pos.X+side*r*1.55, pos.Y+r*0.30)
		hd := rl.NewVector2(pos.X+side*r*1.35, pos.Y+r*1.25)
		rl.DrawLineEx(sh, el, r*0.5, rl.Fade(specterShade, a))
		rl.DrawLineEx(el, hd, r*0.44, rl.Fade(specterWrap, a)) // bandage wraps
		// Darker bands across the wraps.
		for i := 1; i <= 2; i++ {
			f := float32(i) / 3
			bx := el.X + (hd.X-el.X)*f
			by := el.Y + (hd.Y-el.Y)*f
			rl.DrawLineEx(rl.NewVector2(bx-r*0.24, by), rl.NewVector2(bx+r*0.24, by),
				2, rl.Fade(rl.NewColor(74, 44, 58, 255), a))
		}
		// Claws.
		for c := -1; c <= 1; c++ {
			tip := rl.NewVector2(hd.X+side*3+float32(c)*4.5, hd.Y+r*0.55)
			rl.DrawLineEx(hd, tip, 2.4, rl.Fade(specterBone, a))
		}
	}

	// Hovering lower body: tattered wisps instead of legs.
	for i, sx := range []float32{-0.5, 0.05, 0.5} {
		wob := float32(math.Sin(t*6+float64(i)*1.8)) * 2
		w := rl.NewVector2(pos.X+sx*r+wob, pos.Y+r*(1.05+0.22*float32(i%2)))
		rl.DrawCircleV(w, r*(0.30-0.06*float32(i)), rl.Fade(specterShade, 0.8*a))
	}

	// Torso, then the huge muscle hump looming over it.
	rl.DrawEllipse(int32(pos.X), int32(pos.Y+r*0.35), r*1.0, r*1.0, rl.Fade(specterShade, a))
	rl.DrawEllipse(int32(pos.X), int32(pos.Y+r*0.30), r*0.86, r*0.88, rl.Fade(specterFlesh, a))
	rl.DrawEllipse(int32(pos.X+lean*0.4), int32(pos.Y-r*0.55), r*1.32, r*0.82, rl.Fade(specterShade, a))
	rl.DrawEllipse(int32(pos.X+lean*0.4), int32(pos.Y-r*0.50), r*1.18, r*0.70, rl.Fade(specterFlesh, a))

	// THE MAW: a gaping dark cavity that IS the face, snapping hungrily —
	// it gapes wider as the ghoul bites (lunge).
	gape := 0.78 + 0.10*float32(math.Sin(t*3.1)) + 0.35*lunge
	mawC := rl.NewVector2(pos.X+lean, pos.Y-r*0.12)
	mawW := r * 0.80
	mawH := r * 1.0 * gape
	rl.DrawEllipse(int32(mawC.X), int32(mawC.Y), mawW+2.5, mawH+2.5, rl.Fade(specterShade, a))
	rl.DrawEllipse(int32(mawC.X), int32(mawC.Y), mawW, mawH, rl.Fade(specterMaw, a))

	// Bone fangs: four hanging from the upper jaw, three rising from the
	// lower jaw, slightly uneven like a rotten grin.
	fang := rl.Fade(specterBone, a)
	for i := 0; i < 4; i++ {
		fx := mawC.X - mawW*0.66 + float32(i)*mawW*0.44
		top := mawC.Y - mawH*0.92
		ln := r * (0.42 + 0.10*float32(i%2))
		drawFang(rl.NewVector2(fx, top+ln), rl.NewVector2(fx-2.6, top), rl.NewVector2(fx+2.6, top), fang)
	}
	for i := 0; i < 3; i++ {
		fx := mawC.X - mawW*0.48 + float32(i)*mawW*0.48
		bot := mawC.Y + mawH*0.92
		ln := r * (0.34 + 0.09*float32(i%2))
		drawFang(rl.NewVector2(fx, bot-ln), rl.NewVector2(fx-2.4, bot), rl.NewVector2(fx+2.4, bot), fang)
	}

	// Necro-purple aura; flares pale violet for an instant on each bite.
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(pos.X), int32(pos.Y), r*2.2, rl.Fade(specterGlow, 0.22*a), rl.Blank)
	if lunge > 0 {
		rl.DrawCircleGradient(int32(mawC.X), int32(mawC.Y), r*1.5, rl.Fade(specterLight, 0.35*lunge*a), rl.Blank)
	}
	rl.EndBlendMode()
}

// drawFang draws one tooth triangle in both winding orders so culling never
// hides it (same trick as the Arqueiro arrowhead).
func drawFang(tip, b1, b2 rl.Vector2, c rl.Color) {
	rl.DrawTriangle(tip, b1, b2, c)
	rl.DrawTriangle(tip, b2, b1, c)
}
