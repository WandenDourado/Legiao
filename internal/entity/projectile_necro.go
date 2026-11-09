package entity

// Procedural rendering for the Necromante's basic attack: a bone-white skull
// engulfed in purple soulfire (reference: dark skull wreathed in violet
// flames). No sprites — raylib primitives only.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Necromancer identity palette: deep violet -> arcane purple -> pale lavender.
var (
	necroDeepViolet = rl.NewColor(74, 20, 120, 255)
	necroPurple     = rl.NewColor(150, 60, 220, 255)
	necroLavender   = rl.NewColor(226, 200, 255, 255)
	necroBone       = rl.NewColor(228, 220, 202, 255)
	necroSocket     = rl.NewColor(24, 8, 36, 255)
)

// drawNecroSkullProjectile renders the shadow skull traveling toward (dirX,
// dirY): a violet soulfire tail, a small bone skull with glowing eye sockets,
// and an additive purple halo.
func drawNecroSkullProjectile(pos rl.Vector2, dirX, dirY float32) {
	d := safeDir(dirX, dirY)
	perp := rl.NewVector2(-d.Y, d.X)
	t := rl.GetTime()
	flick := float32(math.Sin(t*26)) * 2

	// Soulfire tail: violet flame blobs dancing behind the skull.
	for i := 1; i <= 5; i++ {
		f := float32(i)
		wob := float32(math.Sin(t*20+float64(i)*1.9)) * (2 + f*1.3)
		tail := rl.NewVector2(
			pos.X-d.X*f*12+perp.X*wob,
			pos.Y-d.Y*f*12+perp.Y*wob,
		)
		alpha := uint8(200 - i*34)
		r := NecroAttackRadius * (0.95 - f*0.13)
		rl.DrawCircleV(tail, r, rl.NewColor(necroDeepViolet.R, necroDeepViolet.G, necroDeepViolet.B, alpha))
		rl.DrawCircleV(tail, r*0.5, rl.NewColor(necroPurple.R, necroPurple.G, necroPurple.B, alpha))
	}

	// Flame shroud hugging the skull.
	rl.DrawCircleV(pos, NecroAttackRadius+flick, rl.Fade(necroDeepViolet, 0.85))
	rl.DrawCircleV(pos, NecroAttackRadius-2+flick*0.6, rl.Fade(necroPurple, 0.75))

	// The skull itself: cranium, jaw, sockets, nose and teeth.
	r := NecroAttackRadius * 0.72
	cran := rl.NewVector2(pos.X, pos.Y-r*0.12)
	rl.DrawCircleV(cran, r, necroBone)
	jaw := rl.NewVector2(pos.X, pos.Y+r*0.62)
	rl.DrawEllipse(int32(jaw.X), int32(jaw.Y), r*0.62, r*0.5, necroBone)
	// Eye sockets with a violet ember glow inside.
	eyeOff := r * 0.42
	for _, s := range []float32{-1, 1} {
		eye := rl.NewVector2(cran.X+perp.X*eyeOff*s+d.X*r*0.18, cran.Y+perp.Y*eyeOff*s+d.Y*r*0.18)
		rl.DrawCircleV(eye, r*0.30, necroSocket)
		rl.DrawCircleV(eye, r*0.13, necroPurple)
	}
	// Nasal cavity + teeth gaps.
	nose := rl.NewVector2(cran.X+d.X*r*0.05, cran.Y+r*0.28)
	rl.DrawCircleV(nose, r*0.14, necroSocket)
	for i := -1; i <= 1; i++ {
		gx := jaw.X + perp.X*float32(i)*r*0.28
		gy := jaw.Y + perp.Y*float32(i)*r*0.28
		rl.DrawLineEx(rl.NewVector2(gx, gy-r*0.18), rl.NewVector2(gx, gy+r*0.22), 1.5, necroSocket)
	}

	// Additive halo + hot streak along the tail.
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(pos.X), int32(pos.Y), NecroAttackRadius*2.6,
		rl.Fade(necroPurple, 0.55), rl.Blank)
	rl.DrawCircleGradient(int32(pos.X), int32(pos.Y), NecroAttackRadius*1.1,
		rl.Fade(necroLavender, 0.45), rl.Blank)
	tailEnd := rl.NewVector2(pos.X-d.X*NecroAttackRadius*4, pos.Y-d.Y*NecroAttackRadius*4)
	rl.DrawLineEx(pos, tailEnd, NecroAttackRadius*0.7, rl.Fade(necroPurple, 0.28))
	rl.EndBlendMode()
}
