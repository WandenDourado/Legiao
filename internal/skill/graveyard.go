package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Graveyard constants tune the Necromante's Q skill (Cemitério): a rectangular
// strip of rotting, cursed ground raised in the aimed direction. Enemies
// crossing it take damage over time and are slowed.
const (
	// GraveyardLength is the world-space extent along the aim direction.
	GraveyardLength float32 = 520
	// GraveyardWidth is the world-space extent across the aim direction.
	GraveyardWidth float32 = 320
	// GraveyardOffset places the near edge this far in front of the caster.
	GraveyardOffset float32 = 50
	// GraveyardDuration is how long the cursed ground stays lethal.
	GraveyardDuration float32 = 6.0
	// GraveyardFadeTime is the graceful dissipation window at the end.
	GraveyardFadeTime float32 = 1.4
	// GraveyardDPS is the damage per second dealt to enemies inside.
	GraveyardDPS float32 = 14
	// graveyardTickEvery batches damage into steady increments.
	graveyardTickEvery float32 = 0.5
	// GraveyardSlowFactor multiplies enemy speed inside (0.45 = 55% slower).
	GraveyardSlowFactor float32 = 0.45
	// GraveyardSlowLinger keeps the slow briefly after leaving the area.
	GraveyardSlowLinger float32 = 0.6
	// GraveyardCooldown is the caster's cooldown in seconds.
	GraveyardCooldown float32 = 12.0
	// graveyardHandCount is how many skeleton hands claw out of the ground.
	graveyardHandCount = 24
)

// graveHand is one skeletal hand clawing out of the cursed ground. Local
// coordinates: u along the aim direction (0..Length), v across (-W/2..W/2).
type graveHand struct {
	u, v  float32
	scale float32 // 0.7..1.3 size variation
	delay float32 // seconds before it starts emerging
	phase float32 // individual sway phase
	flip  float32 // -1 or 1, mirrors the thumb side
}

// Graveyard is the cursed rectangular zone. All visuals are procedural.
type Graveyard struct {
	ID      string
	OwnerID string
	Origin  rl.Vector2 // near-edge center (world)
	Dir     rl.Vector2 // normalized aim direction
	Perp    rl.Vector2 // normalized perpendicular
	Age     float32
	// tickAccum batches the damage-over-time into graveyardTickEvery steps.
	tickAccum float32
	hands     []graveHand
	Souls     *ParticleEmitter
	soulAccum float32
}

// NewGraveyard raises cursed ground starting at origin, extending along dir.
func NewGraveyard(ownerID string, origin, dir rl.Vector2) *Graveyard {
	if dir.X == 0 && dir.Y == 0 {
		dir = rl.NewVector2(0, 1)
	}
	dir = rl.Vector2Normalize(dir)
	g := &Graveyard{
		ID:      generateID(),
		OwnerID: ownerID,
		Origin:  origin,
		Dir:     dir,
		Perp:    rl.NewVector2(-dir.Y, dir.X),
		Souls:   NewParticleEmitter(),
	}
	// Scatter hands over most of the area on a jittered grid so they cover it
	// without clumping. Positions are cosmetic, so host/client may differ.
	cols, rows := 6, graveyardHandCount/6
	for i := 0; i < graveyardHandCount; i++ {
		cu := (float32(i%cols) + 0.5) / float32(cols)
		cv := (float32(i/cols) + 0.5) / float32(rows)
		flip := float32(1)
		if rl.GetRandomValue(0, 1) == 0 {
			flip = -1
		}
		g.hands = append(g.hands, graveHand{
			u:     cu*GraveyardLength*0.9 + float32(rl.GetRandomValue(-28, 28)),
			v:     (cv-0.5)*GraveyardWidth*0.85 + float32(rl.GetRandomValue(-20, 20)),
			scale: 0.7 + float32(rl.GetRandomValue(0, 60))/100,
			delay: float32(rl.GetRandomValue(0, 90)) / 100,
			phase: float32(rl.GetRandomValue(0, 628)) / 100,
			flip:  flip,
		})
	}
	return g
}

// point maps local (u, v) coordinates to world space.
func (g *Graveyard) point(u, v float32) rl.Vector2 {
	return rl.NewVector2(
		g.Origin.X+g.Dir.X*u+g.Perp.X*v,
		g.Origin.Y+g.Dir.Y*u+g.Perp.Y*v,
	)
}

// Center returns the world-space center of the rectangle.
func (g *Graveyard) Center() rl.Vector2 { return g.point(GraveyardLength/2, 0) }

// Contains reports whether world point p lies inside the cursed rectangle.
func (g *Graveyard) Contains(p rl.Vector2) bool {
	rel := rl.Vector2Subtract(p, g.Origin)
	u := rel.X*g.Dir.X + rel.Y*g.Dir.Y
	v := rel.X*g.Perp.X + rel.Y*g.Perp.Y
	return u >= 0 && u <= GraveyardLength && float32(math.Abs(float64(v))) <= GraveyardWidth/2
}

// IsCursing reports whether the zone still damages/slows enemies.
func (g *Graveyard) IsCursing() bool { return g.Age <= GraveyardDuration }

// FadeAlpha is 1 while active, then ramps 1..0 during the fade window.
func (g *Graveyard) FadeAlpha() float32 {
	if g.Age <= GraveyardDuration {
		return 1
	}
	return clamp01(1 - (g.Age-GraveyardDuration)/GraveyardFadeTime)
}

// Expired reports whether the graveyard has fully dissipated.
func (g *Graveyard) Expired() bool {
	return g.Age > GraveyardDuration+GraveyardFadeTime
}
