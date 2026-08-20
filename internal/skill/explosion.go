package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Explosion is a one-shot procedural burst of fire at an impact point.
type Explosion struct {
	Position rl.Vector2
	Radius   float32
	TTL      float32
	MaxTTL   float32
	Emitter  *ParticleEmitter
}

// NewExplosion creates a one-shot explosion at pos with the given blast radius.
func NewExplosion(pos rl.Vector2, radius float32) *Explosion {
	e := &Explosion{
		Position: pos,
		Radius:   radius,
		TTL:      0.45,
		MaxTTL:   0.45,
		Emitter:  NewParticleEmitter(),
	}
	e.Emitter.Burst(pos, 40, radius*1.5, radius*3.5, 0.4, radius*0.4, rl.Orange)
	e.Emitter.Burst(pos, 20, radius*0.8, radius*2.0, 0.5, radius*0.25, rl.Yellow)
	return e
}

// Update advances particles and decays the timer. Returns false when finished.
func (e *Explosion) Update(dt float32) bool {
	e.TTL -= dt
	e.Emitter.Update(dt)
	return e.TTL > 0
}

// Draw renders the explosion shock ring plus particles with additive blending.
func (e *Explosion) Draw() {
	progress := 1 - clamp01(e.TTL/e.MaxTTL)
	ringR := e.Radius * (0.3 + progress*1.1)
	alpha := uint8(220 * (1 - progress))
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircle(int32(e.Position.X), int32(e.Position.Y), ringR, rl.NewColor(255, 140, 30, alpha))
	e.Emitter.Draw()
	rl.EndBlendMode()
}
