package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// sacredParticle is a single floating motes inside the sanctuary. It drifts
// upward slowly with an individual fade in/out, giving the "sacred" feel.
type sacredParticle struct {
	offset rl.Vector2 // position relative to sanctuary center
	phase  float32    // per-particle phase for fade in/out
	speed  float32    // upward drift speed
	radius float32
	seed   float32
}

// newSacredParticle spawns one mote at a random offset inside the radius.
// The angle is turned into a full 360° unit vector (cos/sin) so motes are
// spread evenly across BOTH the left and right halves of the area.
func newSacredParticle() sacredParticle {
	ang := float32(rl.GetRandomValue(0, 359)) * math.Pi / 180
	dist := float32(rl.GetRandomValue(0, 100)) / 100 * SanctuaryRadius * 0.85
	dir := rl.NewVector2(float32(math.Cos(float64(ang))), float32(math.Sin(float64(ang))))
	return sacredParticle{
		offset: rl.Vector2Scale(dir, dist),
		phase:  float32(rl.GetRandomValue(0, 100)) / 100 * math.Pi * 2,
		speed:  20 + float32(rl.GetRandomValue(0, 30)),
		radius: 3 + float32(rl.GetRandomValue(0, 4)),
		seed:   float32(rl.GetRandomValue(0, 1000)) / 1000,
	}
}

// update advances the mote (upward drift + respawn when it leaves the area).
func (p *sacredParticle) update(dt float32) {
	p.phase += dt * 1.5
	p.offset.Y -= p.speed * dt
	if p.offset.Y < -SanctuaryRadius*0.8 {
		*p = newSacredParticle()
		p.offset.Y = SanctuaryRadius * 0.8
	}
}

// alpha returns the individual fade in/out factor (0..1..0 over its cycle).
func (p *sacredParticle) alpha() float32 {
	a := (float32(math.Sin(float64(p.phase))) + 1) / 2
	if a < 0 {
		return 0
	}
	if a > 1 {
		return 1
	}
	return a
}

// SanctuaryFX holds the procedural visuals for one sanctuary: the floating
// sacred particles and the pulsing light ring. It is pure presentation.
type SanctuaryFX struct {
	particles []sacredParticle
	ringPhase float32
}

// NewSanctuaryFX creates the particle/ring state for a sanctuary.
func NewSanctuaryFX() *SanctuaryFX {
	fx := &SanctuaryFX{ringPhase: 0}
	for i := 0; i < 24; i++ {
		fx.particles = append(fx.particles, newSacredParticle())
	}
	return fx
}

// update advances all particles and the ring pulse.
func (fx *SanctuaryFX) update(dt float32) {
	for i := range fx.particles {
		fx.particles[i].update(dt)
	}
	fx.ringPhase += dt * 2.0
}

// draw renders the full sanctuary procedurally at center with opacity a.
// Layers (back..front): floor aura gradient, pulsing light ring, particles.
func (fx *SanctuaryFX) draw(center rl.Vector2, a float32) {
	if a <= 0 {
		return
	}
	fx.drawFloorAura(center, a)
	fx.drawLightRing(center, a)
	fx.drawParticles(center, a)
}

// Sacred palette — RESTRICTED to gold (dourado), yellow (amarelo) and
// white (branco) only. No other hue is used. Opacity is kept low so a
// character drawn on top stays clearly visible above the ground effect.
var (
	colorGold         = rl.NewColor(255, 215, 0, 255)   // dourado
	colorWhite        = rl.NewColor(255, 255, 255, 255) // branco
	colorDivineYellow = rl.NewColor(255, 240, 130, 255) // amarelo
)

// lerpColor blends c0..c1 by t (0..1) on a per-channel basis.
// Avoids depending on an optional ColorLerp helper.
func lerpColor(c0, c1 rl.Color, t float32) rl.Color {
	return rl.NewColor(
		uint8(float32(c0.R)+float32(c1.R-c0.R)*t),
		uint8(float32(c0.G)+float32(c1.G-c0.G)*t),
		uint8(float32(c0.B)+float32(c1.B-c0.B)*t),
		255,
	)
}

// drawFloorAura draws concentric translucent circles to fake a radial
// ground aura using only the sacred palette (gold .. divine yellow).
// Opacity is kept low so a player standing inside stays visible on top.
func (fx *SanctuaryFX) drawFloorAura(center rl.Vector2, a float32) {
	// outer tint: soft divine yellow base (kept opaque so the ground
	// tint underneath cannot shift the yellow toward pink).
	rl.DrawCircleV(center, SanctuaryRadius, rl.Fade(colorDivineYellow, 0.22*a))
	const steps = 8
	for i := steps; i >= 1; i-- {
		t := float32(i) / steps
		r := SanctuaryRadius * t
		// brighter (gold) toward the center, fading to divine yellow outward
		base := lerpColor(colorDivineYellow, colorGold, 1-t)
		alpha := uint8(50 * (1 - t*0.5) * a)
		rl.DrawCircleV(center, r, rl.Fade(base, float32(alpha)/255))
	}
	// solid-ish yellow inner core — kept opaque enough that the ground
	// tint underneath cannot shift it toward pink.
	rl.DrawCircleV(center, SanctuaryRadius*0.32, rl.Fade(colorGold, 0.15*a))
	rl.DrawCircleV(center, SanctuaryRadius*0.20, rl.Fade(colorDivineYellow, 0.15*a))
}

// drawLightRing draws a pulsing ring of light around the area border plus
// subtle radial rays emanating outward, using gold + divine yellow.
func (fx *SanctuaryFX) drawLightRing(center rl.Vector2, a float32) {
	pulse := 0.5 + 0.5*float32(math.Sin(float64(fx.ringPhase)))
	// base ring (gold)
	rl.DrawCircleLinesV(center, SanctuaryRadius, rl.Fade(colorGold, 0.85*a))
	// pulsing outer ring (divine yellow)
	ringR := SanctuaryRadius + 8 + pulse*14
	rl.DrawCircleLinesV(center, ringR, rl.Fade(colorDivineYellow, 0.5*(1-pulse)*a))

	// radial rays around the border (subtle, slow rotation)
	rays := 12
	for i := 0; i < rays; i++ {
		ang := float32(i)/float32(rays)*math.Pi*2 + fx.ringPhase*0.2
		dir := rl.NewVector2(float32(math.Cos(float64(ang))), float32(math.Sin(float64(ang))))
		inner := rl.Vector2Add(center, rl.Vector2Scale(dir, SanctuaryRadius))
		outer := rl.Vector2Add(center, rl.Vector2Scale(dir, SanctuaryRadius+10+pulse*16))
		rl.DrawLineV(inner, outer, rl.Fade(colorDivineYellow, 0.7*a))
	}
}

// drawParticles renders the floating motes. Mostly gentle white sparkle
// stars (with a soft glow) that cover much of the area; a few use gold/yellow
// to read as divine embers. All colors stay within the gold/yellow/white palette.
func (fx *SanctuaryFX) drawParticles(center rl.Vector2, a float32) {
	for _, p := range fx.particles {
		pos := rl.Vector2Add(center, p.offset)
		pa := p.alpha() * a
		if pa <= 0 {
			continue
		}
		// white sparkle star (core + glow) — dominant, covers the magic
		rl.DrawCircleV(pos, p.radius*1.9, rl.Fade(colorWhite, pa*0.35))
		rl.DrawCircleV(pos, p.radius, rl.Fade(colorWhite, pa))
		drawSparkle(pos, p.radius*3.2, pa)
		// a few golden/divine-yellow embers layered underneath
		if int(p.seed*10)%3 == 0 {
			rl.DrawCircleV(pos, p.radius*1.4, rl.Fade(colorGold, pa*0.4))
		} else if int(p.seed*10)%3 == 1 {
			rl.DrawCircleV(pos, p.radius*1.4, rl.Fade(colorDivineYellow, pa*0.4))
		}
	}
}

// drawSparkle draws a small 4-point star (plus) centered at pos, used for the
// white sacred sparkles. l is the half-length of each arm.
func drawSparkle(pos rl.Vector2, l float32, a float32) {
	if a <= 0 {
		return
	}
	rl.DrawLineV(
		rl.NewVector2(pos.X-l, pos.Y),
		rl.NewVector2(pos.X+l, pos.Y),
		rl.Fade(colorWhite, a*0.9),
	)
	rl.DrawLineV(
		rl.NewVector2(pos.X, pos.Y-l),
		rl.NewVector2(pos.X, pos.Y+l),
		rl.Fade(colorWhite, a*0.9),
	)
}
