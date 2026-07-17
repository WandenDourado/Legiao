package entity

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Particle is a single procedural fire particle (no sprite, drawn with primitives).
type Particle struct {
	Pos     rl.Vector2
	Vel     rl.Vector2
	Life    float32
	MaxLife float32
	Radius  float32
	Color   rl.Color
}

// IsDead reports whether the particle has expired.
func (p *Particle) IsDead() bool { return p.Life <= 0 }

// step advances the particle and returns false when dead.
func (p *Particle) step(dt float32) bool {
	p.Life -= dt
	p.Pos.X += p.Vel.X * dt
	p.Pos.Y += p.Vel.Y * dt
	p.Vel.X *= 0.94
	p.Vel.Y *= 0.94
	if p.Radius > 0 {
		p.Radius *= 0.97
	}
	return p.Life > 0
}

// drawParticle renders a particle as an additive radial glow.
func drawParticle(p *Particle) {
	alpha := uint8(255 * clamp01(p.Life/p.MaxLife))
	c := rl.NewColor(p.Color.R, p.Color.G, p.Color.B, alpha)
	if p.Radius > 1 {
		rl.DrawCircleGradient(int32(p.Pos.X), int32(p.Pos.Y), p.Radius, c, rl.Blank)
	}
	rl.DrawCircle(int32(p.Pos.X), int32(p.Pos.Y), maxF(1, p.Radius*0.4), c)
}

// ParticleEmitter manages a pool of procedural particles.
type ParticleEmitter struct {
	particles []*Particle
}

// NewParticleEmitter creates an empty emitter.
func NewParticleEmitter() *ParticleEmitter { return &ParticleEmitter{} }

// Emit appends a particle with the given field values.
func (e *ParticleEmitter) Emit(pos, vel rl.Vector2, life, radius float32, c rl.Color) {
	e.particles = append(e.particles, &Particle{
		Pos: pos, Vel: vel, Life: life, MaxLife: life, Radius: radius, Color: c,
	})
}

// Burst emits n particles radiating from pos with speed in [minSpd,maxSpd].
func (e *ParticleEmitter) Burst(pos rl.Vector2, n int, minSpd, maxSpd, life, radius float32, c rl.Color) {
	for i := 0; i < n; i++ {
		a := rand.Float64() * 2 * math.Pi
		s := float64(minSpd) + rand.Float64()*float64(maxSpd-minSpd)
		sf := float32(s)
		vel := rl.NewVector2(float32(math.Cos(a))*sf, float32(math.Sin(a))*sf)
		e.Emit(pos, vel, life*float32(0.6+rand.Float64()*0.4), radius*float32(0.6+rand.Float64()*0.6), c)
	}
}

// Update advances all particles and drops dead ones.
func (e *ParticleEmitter) Update(dt float32) {
	alive := e.particles[:0]
	for _, p := range e.particles {
		if p.step(dt) {
			alive = append(alive, p)
		}
	}
	e.particles = alive
}

// Draw renders all particles under additive blending.
func (e *ParticleEmitter) Draw() {
	rl.BeginBlendMode(rl.BlendAdditive)
	for _, p := range e.particles {
		drawParticle(p)
	}
	rl.EndBlendMode()
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
