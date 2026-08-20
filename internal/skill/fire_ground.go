package skill

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Ground-fire constants tune the burning area left after an explosion.
const (
	FireGroundDuration float32 = 5.0
	FireGroundRadius   float32 = 550 // 10x base area (was 55)
	// fireGroundTickEvery batches the burn into steady increments instead of
	// nibbling health every frame, the same way the Cemiterio does: a monster
	// crossing the fire loses readable chunks and the host sends two events a
	// second per burning monster instead of sixty.
	fireGroundTickEvery float32 = 0.5
)

// FireGround is a fixed world area that burns continuously for a duration.
type FireGround struct {
	ID      string
	Center  rl.Vector2
	Radius  float32
	TTL     float32
	Emitter *ParticleEmitter
	spawn   float32
	// tickAccum batches the damage-over-time into fireGroundTickEvery steps.
	// Each zone carries its own, so two overlapping fires burn independently.
	tickAccum float32
}

// Contains reports whether a point is inside the burning circle. The monster's
// own radius is deliberately left out: with a 550-unit fire, adding it would
// only make the edge fuzzy, and "the monster is standing in the fire" is the
// rule a player can actually see.
func (g *FireGround) Contains(p rl.Vector2) bool {
	return rl.Vector2Distance(g.Center, p) <= g.Radius
}

// NewFireGround creates a burning zone at center that lasts FireGroundDuration seconds.
func NewFireGround(center rl.Vector2) *FireGround {
	return &FireGround{
		ID:      generateID(),
		Center:  center,
		Radius:  FireGroundRadius,
		TTL:     FireGroundDuration,
		Emitter: NewParticleEmitter(),
		spawn:   0,
	}
}

// Update counts down the timer and emits flame particles. Returns false at 0.
func (g *FireGround) Update(dt float32) bool {
	g.TTL -= dt
	g.spawn += dt
	rate := float32(0.04)
	for g.spawn >= rate {
		g.spawn -= rate
		ox := (rand.Float32()*2 - 1) * g.Radius
		oy := (rand.Float32()*2 - 1) * g.Radius
		if ox*ox+oy*oy > g.Radius*g.Radius {
			continue
		}
		pos := rl.NewVector2(g.Center.X+ox, g.Center.Y+oy)
		g.Emitter.Emit(pos, rl.NewVector2(ox*0.2, -30), 0.6, g.Radius*0.18, rl.Orange)
	}
	g.Emitter.Update(dt)
	return g.TTL > 0
}

// Draw renders the translucent ground patch and flames additively.
func (g *FireGround) Draw() {
	fade := clamp01(g.TTL / 1.0)
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircle(int32(g.Center.X), int32(g.Center.Y), g.Radius, rl.NewColor(255, 90, 20, uint8(70*fade)))
	g.Emitter.Draw()
	rl.EndBlendMode()
}
