package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Angelic Area constants tune the Sacerdotisa's ultimate (Área Angelical).
const (
	// AngelicRadius is the world-space radius of the angelic altar zone.
	AngelicRadius float32 = 720
	// AngelicDuration is how long the area lasts before fading.
	AngelicDuration float32 = 12.0
	// AngelicFade is the graceful dissipation window at the end.
	AngelicFade float32 = 1.5
	// AngelicHealPerSec is the continuous HP restored to allies inside.
	AngelicHealPerSec float32 = 20
	// AngelicResurrectHealthPct is the health fraction the dead return with.
	AngelicResurrectHealthPct float32 = 0.5
	// AngelicCooldown is the caster's cooldown in seconds.
	AngelicCooldown float32 = 60.0
)

// Angelic palette: white and soft heavenly blues (per the spell design).
var (
	angelicWhite = rl.NewColor(255, 255, 255, 255)
	angelicBlue  = rl.NewColor(150, 200, 255, 255)
	angelicDeep  = rl.NewColor(90, 150, 255, 255)
)

// AngelicArea is the Sacerdotisa's ultimate: she consecrates an ALTAR — a
// great heavenly zone fixed on the ground where it was cast. It resurrects
// the fallen once (host consumes ResurrectPending) and constantly heals
// allies inside. Visuals: white/blue floor glow bathed in divine light, a
// central blessing beam, descending light shafts, boundary rings and falling
// angel feathers.
type AngelicArea struct {
	OwnerID  string
	Position rl.Vector2 // fixed altar center (does NOT follow the caster)
	Age      float32
	// ResurrectPending is consumed exactly once by the host tick to revive
	// all dead players when the area appears.
	ResurrectPending bool
	// HealAccum accumulates fractional healing (same pattern as Sanctuary).
	HealAccum float32
	feathers  []*angelFeather
	shafts    []*lightShaft
	spawnAcc  float32
	shaftAcc  float32
	sparkles  *ParticleEmitter
}

// NewAngelicArea creates the area centered on the caster.
func NewAngelicArea(ownerID string, center rl.Vector2) *AngelicArea {
	return &AngelicArea{
		OwnerID:          ownerID,
		Position:         center,
		ResurrectPending: true,
		sparkles:         NewParticleEmitter(),
	}
}

// Finished reports whether the area fully dissipated.
func (a *AngelicArea) Finished() bool {
	return a.Age > AngelicDuration+AngelicFade
}

// FadeAlpha is 1 while active, then ramps 1..0 during the fade window.
func (a *AngelicArea) FadeAlpha() float32 {
	if a.Age <= AngelicDuration {
		return 1
	}
	return clamp01(1 - (a.Age-AngelicDuration)/AngelicFade)
}

// IsHealing reports whether the area still heals.
func (a *AngelicArea) IsHealing() bool { return a.Age <= AngelicDuration }

// Contains reports whether world point p lies inside the area.
func (a *AngelicArea) Contains(p rl.Vector2) bool {
	return rl.Vector2Distance(a.Position, p) <= AngelicRadius
}

// Update advances animation: feathers, light shafts, sparkles, timers.
func (a *AngelicArea) Update(dt float32) {
	a.Age += dt
	a.updateFeathers(dt)
	a.updateShafts(dt)
	// Gentle rising sparkles across the area.
	a.spawnAcc += dt
	for a.spawnAcc >= 0.08 {
		a.spawnAcc -= 0.08
		off := randomPointInRadius(AngelicRadius * 0.92)
		pos := rl.Vector2Add(a.Position, off)
		a.sparkles.Emit(pos, rl.NewVector2(0, -34), 1.1, 5, angelicBlue)
	}
	a.sparkles.Update(dt)
}

// Draw renders the heavenly zone (world space).
func (a *AngelicArea) Draw() {
	fade := a.FadeAlpha()
	if fade <= 0 {
		return
	}
	// Entrance bloom: the area blossoms out over the first 0.6s.
	grow := easeOutCubic(clamp01(a.Age / 0.6))
	r := AngelicRadius * grow

	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	// Divine ground wash: the whole terrain inside the altar bathes in a
	// clear light (the gods' blessing), white heart over a blue body.
	rl.DrawCircleGradient(int32(a.Position.X), int32(a.Position.Y), r,
		rl.Fade(angelicBlue, 0.22*fade), rl.Blank)
	rl.DrawCircleGradient(int32(a.Position.X), int32(a.Position.Y), r*0.7,
		rl.Fade(angelicWhite, 0.22*fade), rl.Blank)
	breath := (float32(math.Sin(float64(a.Age)*1.3)) + 1) / 2 // slow holy breathing
	rl.DrawCircleGradient(int32(a.Position.X), int32(a.Position.Y), r*0.9,
		rl.Fade(angelicWhite, 0.08*breath*fade), rl.Blank)

	// Central blessing beam: a great column of light consecrating the altar.
	a.drawBlessingBeam(fade)

	// Rotating broad light rays (very low alpha — divine light shafts).
	spin := a.Age * 9
	for i := 0; i < 6; i++ {
		ang := float64(spin)*math.Pi/180 + float64(i)*math.Pi/3
		tip := rl.NewVector2(
			a.Position.X+float32(math.Cos(ang))*r,
			a.Position.Y+float32(math.Sin(ang))*r,
		)
		perp := rl.NewVector2(-float32(math.Sin(ang))*26, float32(math.Cos(ang))*26)
		p1 := rl.Vector2Add(a.Position, perp)
		p2 := rl.Vector2Subtract(a.Position, perp)
		c := rl.Fade(angelicWhite, 0.07*fade)
		rl.DrawTriangle(p1, p2, tip, c)
		rl.DrawTriangle(p1, tip, p2, c)
	}

	// Boundary: solid outer ring + slow pulsing inner ring.
	rl.DrawRing(a.Position, r-3, r+3, 0, 360, 64, rl.Fade(angelicBlue, 0.55*fade))
	pulse := (float32(math.Sin(float64(a.Age)*2.1)) + 1) / 2 // 0..1
	pr := r * (0.75 + 0.1*pulse)
	rl.DrawRing(a.Position, pr-1.5, pr+1.5, 0, 360, 56, rl.Fade(angelicDeep, 0.30*(1-pulse)*fade))

	// Rotating heavenly arcs on the boundary (counter-rotation = alive).
	spin2 := -a.Age * 30
	for i := 0; i < 4; i++ {
		start := spin2 + float32(i)*90
		rl.DrawRing(a.Position, r+6, r+10, start, start+40, 24, rl.Fade(angelicWhite, 0.5*fade))
	}

	a.drawFeathers(fade)
	for _, p := range a.sparkles.particles {
		drawParticle(p)
	}
}
