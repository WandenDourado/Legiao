package entity

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Fireball constants tune the projectile behavior.
const (
	FireballSpeed   float32 = 420
	FireballRadius  float32 = 140 // 10x base size (was 14)
	FireballTTL     float32 = 3.0
	FireballTrail   int = 6
	FireballRange   float32 = 900 // max travel distance before it auto-explodes

	// Skill tuning: explosion + ground fire.
	FireballExplosionRadius float32 = 700 // 10x base area (was 70)
	FireballExplosionDamage float32 = 25
	FireGroundDamagePerSec   float32 = 20
)

// Fireball is a procedurally-rendered fire projectile (no sprite).
type Fireball struct {
	ID        string
	OwnerID   string
	Position  rl.Vector2
	Velocity  rl.Vector2
	Origin    rl.Vector2
	Range     float32
	Traveled  float32
	Radius    float32
	TTL       float32
	Trail     *ParticleEmitter
}

// NewFireball creates a fireball launched from start toward dir (normalized).
func NewFireball(ownerID string, start, dir rl.Vector2) *Fireball {
	dn := dir
	if l := rl.Vector2Length(dn); l > 0 {
		dn = rl.Vector2Scale(dn, 1/l)
	}
	return &Fireball{
		ID:       generateID(),
		OwnerID:  ownerID,
		Position: start,
		Origin:   start,
		Velocity: rl.Vector2Scale(dn, FireballSpeed),
		Radius:   FireballRadius,
		Range:    FireballRange,
		TTL:      FireballTTL,
		Trail:    NewParticleEmitter(),
	}
}

// Update moves the fireball, spawns trail, decays TTL and tracks travel distance.
// Returns false when dead (TTL expired OR max range reached). This is the
// authoritative update used by StepFireballs, which owns fireball removal +
// explosion; callers must NOT also remove the fireball elsewhere.
func (f *Fireball) Update(dt float32) bool {
	f.TTL -= dt
	f.Position.X += f.Velocity.X * dt
	f.Position.Y += f.Velocity.Y * dt
	f.Traveled = rl.Vector2Distance(f.Origin, f.Position)
	f.emitTrail()
	f.Trail.Update(dt)
	return f.TTL > 0 && f.Traveled < f.Range
}

// AdvanceVisual moves the fireball and animates its trail WITHOUT deciding its
// life. Used by EntityManager.UpdateFire so the simulation (StepFireballs) keeps
// sole ownership of when a fireball is removed/explodes (avoid double-removal).
func (f *Fireball) AdvanceVisual(dt float32) {
	f.Position.X += f.Velocity.X * dt
	f.Position.Y += f.Velocity.Y * dt
	f.emitTrail()
	f.Trail.Update(dt)
}

// emitTrail spawns a few trail particles at the current position.
func (f *Fireball) emitTrail() {
	for i := 0; i < FireballTrail; i++ {
		f.Trail.Emit(f.Position, rl.Vector2Zero(), 0.35, f.Radius*0.9, rl.Orange)
	}
}

// Draw renders the fireball core and trail using additive blending.
func (f *Fireball) Draw() {
	f.Trail.Draw()
	rl.BeginBlendMode(rl.BlendAdditive)
	// Outer glow
	rl.DrawCircleGradient(int32(f.Position.X), int32(f.Position.Y), f.Radius*2.2, rl.Fade(rl.Orange, 0.5), rl.Blank)
	// Hot core
	rl.DrawCircleGradient(int32(f.Position.X), int32(f.Position.Y), f.Radius, rl.Yellow, rl.Red)
	rl.DrawCircle(int32(f.Position.X), int32(f.Position.Y), f.Radius*0.5, rl.Fade(rl.White, 0.9))
	rl.EndBlendMode()
}
