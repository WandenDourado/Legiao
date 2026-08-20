package skill

// Procedural visuals for the Necromante's Cemitério: rotting darkened ground,
// skeletal hands clawing upward, drifting grave mist and rising soul wisps.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	graveRot    = rl.NewColor(22, 14, 26, 255)   // rotten earth
	graveRotRim = rl.NewColor(52, 32, 66, 255)   // bruised purple rim
	graveGreen  = rl.NewColor(110, 220, 140, 255) // sickly necro-green
	gravePurple = rl.NewColor(150, 60, 220, 255)
	graveBone   = rl.NewColor(215, 205, 182, 255)
)

// advanceVisual ages the zone and animates its cosmetic particles. Used by
// both the host step and the client advance.
func (g *Graveyard) advanceVisual(dt float32) {
	g.Age += dt
	if g.IsCursing() {
		// Rising soul wisps: alternating sickly green and violet motes.
		g.soulAccum += dt
		for g.soulAccum > 0.08 {
			g.soulAccum -= 0.08
			u := float32(rl.GetRandomValue(0, int32(GraveyardLength)))
			v := float32(rl.GetRandomValue(-int32(GraveyardWidth/2), int32(GraveyardWidth/2)))
			c := graveGreen
			if rl.GetRandomValue(0, 1) == 0 {
				c = gravePurple
			}
			vel := rl.NewVector2(float32(rl.GetRandomValue(-12, 12)), -30-float32(rl.GetRandomValue(0, 35)))
			g.Souls.Emit(g.point(u, v), vel, 1.1, 4, c)
		}
	}
	g.Souls.Update(dt)
}

// Draw renders the cursed ground in world space (back to front).
func (g *Graveyard) Draw() {
	f := g.FadeAlpha()
	if f <= 0 && len(g.Souls.particles) == 0 {
		return
	}
	angle := float32(math.Atan2(float64(g.Dir.Y), float64(g.Dir.X))) * 180 / math.Pi
	center := g.Center()
	emerge := clamp01(g.Age / 0.5) // ground rots in quickly on cast

	// Rotting ground: two rotated rects (dark core + bruised rim) plus
	// irregular rot blotches so the decay looks organic.
	rimRect := rl.NewRectangle(center.X, center.Y, GraveyardLength+26, GraveyardWidth+26)
	rl.DrawRectanglePro(rimRect, rl.NewVector2(rimRect.Width/2, rimRect.Height/2), angle,
		rl.Fade(graveRotRim, 0.42*f*emerge))
	rect := rl.NewRectangle(center.X, center.Y, GraveyardLength, GraveyardWidth)
	rl.DrawRectanglePro(rect, rl.NewVector2(rect.Width/2, rect.Height/2), angle,
		rl.Fade(graveRot, 0.72*f*emerge))
	for i, h := range g.hands { // reuse hand spots as blotch anchors
		r := (18 + float32(i%3)*9) * emerge
		rl.DrawCircleV(g.point(h.u, h.v*0.9), r, rl.Fade(graveRot, 0.5*f))
	}

	// Sickly green miasma pooling over the rot (additive, breathing).
	t := float32(rl.GetTime())
	breath := 0.5 + 0.5*float32(math.Sin(float64(t)*1.6))
	rl.BeginBlendMode(rl.BlendAdditive)
	for i := 0; i < 3; i++ {
		u := GraveyardLength * (0.2 + 0.3*float32(i))
		p := g.point(u, float32(math.Sin(float64(t)*0.7+float64(i)*2.1))*GraveyardWidth*0.2)
		rl.DrawCircleGradient(int32(p.X), int32(p.Y), 70+14*breath,
			rl.Fade(graveGreen, (0.10+0.05*breath)*f), rl.Blank)
	}
	rl.EndBlendMode()

	// Skeleton hands clawing out of the dirt.
	for i := range g.hands {
		g.drawHand(&g.hands[i], f)
	}

	// Souls drifting upward.
	rl.BeginBlendMode(rl.BlendAdditive)
	for _, pt := range g.Souls.particles {
		drawParticle(pt)
	}
	rl.EndBlendMode()
}

// drawHand renders one skeletal hand: palm, four clawing fingers with joints
// and a thumb, rising from a dark burst hole with a slow desperate sway.
func (g *Graveyard) drawHand(h *graveHand, fade float32) {
	rise := clamp01((g.Age - 0.35 - h.delay) / 0.7)
	if rise <= 0 {
		return
	}
	base := g.point(h.u, h.v)
	t := float32(rl.GetTime())
	sway := float32(math.Sin(float64(t)*1.9+float64(h.phase))) * 2.2 * h.scale
	s := h.scale * rise

	// Burst hole under the hand.
	rl.DrawEllipse(int32(base.X), int32(base.Y), 13*h.scale, 5.5*h.scale, rl.Fade(rl.Black, 0.55*fade))

	bone := rl.Fade(graveBone, fade)
	shade := rl.Fade(rl.NewColor(150, 140, 118, 255), fade)
	wrist := rl.NewVector2(base.X+sway*0.4, base.Y-7*s)
	rl.DrawLineEx(base, wrist, 5*s, shade) // forearm bone
	palm := rl.NewVector2(wrist.X+sway*0.6, wrist.Y-8*s)
	rl.DrawCircleV(palm, 5.2*s, bone) // palm

	// Four fingers fanning upward, two segments each, curling like claws.
	for i := 0; i < 4; i++ {
		fx := (float32(i) - 1.5) * 4.6 * s
		knee := rl.NewVector2(palm.X+fx+sway, palm.Y-9*s)
		tip := rl.NewVector2(knee.X+fx*0.35+sway*0.6, knee.Y-7*s+float32(i%2)*2*s)
		rl.DrawLineEx(palm, knee, 2.4*s, bone)
		rl.DrawLineEx(knee, tip, 1.8*s, bone)
		rl.DrawCircleV(knee, 1.6*s, shade) // knuckle
	}
	// Thumb to the side.
	thumb := rl.NewVector2(palm.X+8*s*h.flip, palm.Y-3*s)
	rl.DrawLineEx(palm, thumb, 2.2*s, bone)

	// Faint green grave-light licking the bones.
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(palm.X), int32(palm.Y-6*s), 16*s,
		rl.Fade(graveGreen, 0.16*fade), rl.Blank)
	rl.EndBlendMode()
}
