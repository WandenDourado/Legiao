package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Fireball constants tune the Mago Bola de Fogo skill.
const (
	FireballSpeed           float32 = 840
	FireballRadius          float32 = 140
	FireballTTL             float32 = 3.0
	FireballTrail           int     = 6
	FireballRange           float32 = 900
	FireballExplosionRadius float32 = 700
	FireballExplosionDamage float32 = 25
	FireGroundDamagePerSec  float32 = 20
	// FireballCooldown is the per-caster recharge in seconds. It sits in the
	// same tier as the Paladina's shield (6s): the Bola de Fogo is the Mago's
	// working tool, so it has to come back fast, but a 700-radius explosion
	// that also leaves burning ground cannot be spammed either.
	FireballCooldown float32 = 6.0
)

// Fireball is a procedurally rendered projectile. It has no sprite: the
// core, glow and trail are all drawn with raylib primitives.
type Fireball struct {
	ID       string
	OwnerID  string
	Position rl.Vector2
	Velocity rl.Vector2
	Origin   rl.Vector2
	Range    float32
	Traveled float32
	Radius   float32
	TTL      float32
	Trail    *ParticleEmitter
}

// NewFireball creates a fireball launched from start toward dir (normalized).
func NewFireball(ownerID string, start, dir rl.Vector2) *Fireball {
	d := rl.Vector2Normalize(dir)
	return &Fireball{
		ID:       generateID(),
		OwnerID:  ownerID,
		Position: start,
		Velocity: rl.Vector2Scale(d, FireballSpeed),
		Origin:   start,
		Range:    FireballRange,
		Traveled: 0,
		Radius:   FireballRadius,
		TTL:      FireballTTL,
		Trail:    NewParticleEmitter(),
	}
}

// Update advances the fireball and emits trail. Returns false when it should
// be removed (TTL expired or max range reached).
func (f *Fireball) Update(dt float32) bool {
	f.TTL -= dt
	f.Position = rl.Vector2Add(f.Position, rl.Vector2Scale(f.Velocity, dt))
	f.Traveled = rl.Vector2Distance(f.Origin, f.Position)
	f.emitTrail()
	f.Trail.Update(dt)
	return f.TTL > 0 && f.Traveled < f.Range
}

// AdvanceVisual moves the fireball + emits trail WITHOUT deciding removal
// (client-side visual update). It still tracks TTL/Traveled so the client can
// prune orphaned fireballs via Expired().
func (f *Fireball) AdvanceVisual(dt float32) {
	f.TTL -= dt
	f.Position = rl.Vector2Add(f.Position, rl.Vector2Scale(f.Velocity, dt))
	f.Traveled = rl.Vector2Distance(f.Origin, f.Position)
	f.emitTrail()
	f.Trail.Update(dt)
}

// Expired reports whether the fireball exceeded its range or lifetime.
func (f *Fireball) Expired() bool {
	return f.TTL <= 0 || f.Traveled >= f.Range
}

// emitTrail spawns a fire trail behind the fireball. The trail length scales
// with FireballRadius so it stays proportional to the projectile.
func (f *Fireball) emitTrail() {
	for i := 0; i < FireballTrail; i++ {
		// Spread particles back along the flight axis over a length of a few
		// radii, tapering the radius so the tail thins out behind the head.
		t := float32(i) / float32(FireballTrail-1) // 0 = head, 1 = far tail
		off := rl.Vector2Scale(f.Velocity, -0.07*float32(i+1))
		pos := rl.Vector2Add(f.Position, off)
		radius := f.Radius * (0.55 - 0.4*t)
		f.Trail.Emit(pos, rl.NewVector2(0, 0), 0.45, radius, rl.Orange)
	}
	f.Trail.Burst(f.Position, 2, 10, 40, 0.4, 14, rl.Yellow)
}

// Draw renders the fireball core + glow additively, then its trail.
// Outer flame is deep orange/red; only the small hot center is yellow.
func (f *Fireball) Draw() {
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircle(int32(f.Position.X), int32(f.Position.Y), f.Radius, rl.Fade(rl.Red, 0.55))
	rl.DrawCircle(int32(f.Position.X), int32(f.Position.Y), f.Radius*0.8, rl.Fade(rl.Orange, 0.85))
	rl.DrawCircle(int32(f.Position.X), int32(f.Position.Y), f.Radius*0.32, rl.Fade(rl.Yellow, 1.0))
	f.Trail.Draw()
	rl.EndBlendMode()
}
