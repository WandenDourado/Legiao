package entity

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Ground-fire constants tune the burning area left after an explosion.
const (
	FireGroundDuration float32 = 5.0
	FireGroundRadius   float32 = 550 // 10x base area (was 55)
)

// FireGround is a fixed world area that burns continuously for a duration.
type FireGround struct {
	ID      string
	Center  rl.Vector2
	Radius  float32
	TTL     float32
	Emitter *ParticleEmitter
	spawn   float32
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
