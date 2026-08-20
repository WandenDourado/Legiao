package skill

import (
	"fmt"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Meteor Rain constants tune the Mago's ultimate (Chuva de Meteoros).
const (
	// MeteorRainDuration is how long the rain keeps spawning meteors.
	MeteorRainDuration float32 = 15.0
	// MeteorRainInterval is the delay between meteor spawns (~40/sec: the
	// rain must feel apocalyptic and sweep the whole map clean).
	MeteorRainInterval float32 = 0.025
	// MeteorRainCooldown is the caster's cooldown in seconds.
	MeteorRainCooldown float32 = 60.0
	// MeteorFallTime is how long a meteor streaks down before impact.
	MeteorFallTime float32 = 0.9
	// MeteorImpactRadius is the area-damage radius on ground impact.
	MeteorImpactRadius float32 = 160
	// MeteorImpactDamage kills any current enemy caught in the impact (100 HP
	// today): the ultimate is meant to clear the map of monsters.
	MeteorImpactDamage float32 = 100
	// meteorImpactTime is the impact flash/shockwave duration.
	meteorImpactTime float32 = 0.5
	// meteorFallDist is how far up-right the meteor spawns from its target.
	meteorFallDist float32 = 780
)

var meteorSeq uint64

func nextMeteorID() string {
	return fmt.Sprintf("mt%d", atomic.AddUint64(&meteorSeq, 1))
}

// Meteor is one falling rock of the meteor rain: a fiery streak dives toward
// a marked ground target, then explodes. Damage happens ONLY at impact (host).
type Meteor struct {
	ID     string
	Target rl.Vector2
	// Age runs 0..MeteorFallTime (falling), then continues through the
	// impact window. Impacted flips exactly once, at the impact frame.
	Age      float32
	Impacted bool
	Trail    *ParticleEmitter
}

// NewMeteor creates a meteor aimed at the given ground target.
func NewMeteor(target rl.Vector2) *Meteor {
	return &Meteor{ID: nextMeteorID(), Target: target, Trail: NewParticleEmitter()}
}

// fallDir is the fixed world-space dive direction (from up-right to target).
var meteorFallDir = rl.Vector2{X: -0.42, Y: 0.91} // normalized-ish

// HeadPos returns the meteor head's current world position while falling.
func (mt *Meteor) HeadPos() rl.Vector2 {
	p := clamp01(mt.Age / MeteorFallTime)
	remaining := (1 - p) * meteorFallDist
	return rl.NewVector2(
		mt.Target.X-meteorFallDir.X*remaining,
		mt.Target.Y-meteorFallDir.Y*remaining,
	)
}

// Advance ages the meteor and reports (impactedNow, finished). impactedNow is
// true only on the exact frame the meteor hits the ground, so the host can
// apply damage exactly once.
func (mt *Meteor) Advance(dt float32) (impactedNow, finished bool) {
	mt.Age += dt
	if !mt.Impacted && mt.Age >= MeteorFallTime {
		mt.Impacted = true
		impactedNow = true
		mt.Trail.Burst(mt.Target, 26, 220, 520, 0.45, 22, rl.Orange)
		mt.Trail.Burst(mt.Target, 14, 120, 300, 0.5, 14, rl.Yellow)
	}
	// While falling, feed the fiery tail (big rock = big burning wake).
	if !mt.Impacted {
		head := mt.HeadPos()
		mt.Trail.Emit(head, rl.Vector2Scale(meteorFallDir, -160), 0.34, 38, rl.Orange)
		mt.Trail.Emit(head, rl.Vector2Scale(meteorFallDir, -70), 0.28, 22, rl.Red)
		mt.Trail.Emit(head, rl.NewVector2(0, 0), 0.22, 15, rl.Yellow)
	}
	mt.Trail.Update(dt)
	finished = mt.Age >= MeteorFallTime+meteorImpactTime
	return impactedNow, finished
}

// Draw renders the meteor: ground target marker, falling streak with fiery
// head, and the impact flash + shockwave ring.
func (mt *Meteor) Draw() {
	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	if !mt.Impacted {
		p := clamp01(mt.Age / MeteorFallTime)
		// Ground marker: a ring that closes in as the meteor approaches.
		markR := MeteorImpactRadius * (1.15 - 0.55*p)
		rl.DrawRing(mt.Target, markR-3, markR+3, 0, 360, 40,
			rl.Fade(rl.NewColor(255, 90, 40, 255), 0.18+0.38*p))
		rl.DrawCircleGradient(int32(mt.Target.X), int32(mt.Target.Y),
			MeteorImpactRadius*0.5, rl.Fade(rl.NewColor(255, 70, 30, 255), 0.10+0.20*p), rl.Blank)

		// Falling streak: a HUGE burning rock with an elongated fire tail.
		head := mt.HeadPos()
		tail := rl.NewVector2(
			head.X-meteorFallDir.X*300,
			head.Y-meteorFallDir.Y*300,
		)
		rl.DrawLineEx(tail, head, 22, rl.Fade(rl.Red, 0.5))
		rl.DrawLineEx(tail, head, 12, rl.Fade(rl.Orange, 0.8))
		rl.DrawLineEx(tail, head, 5, rl.Fade(rl.NewColor(255, 250, 200, 255), 0.9))
		// Rock body: dark core silhouette wrapped in fire (reads as mass, not
		// just light).
		rl.DrawCircleGradient(int32(head.X), int32(head.Y), 88, rl.Fade(rl.Red, 0.7), rl.Blank)
		rl.DrawCircleGradient(int32(head.X), int32(head.Y), 54, rl.Fade(rl.Orange, 0.95), rl.Blank)
		rl.DrawCircle(int32(head.X), int32(head.Y), 24, rl.Fade(rl.NewColor(255, 252, 220, 255), 1.0))
		rl.EndBlendMode()
		// Solid molten-rock core drawn in normal blending over the glow.
		rl.DrawCircle(int32(head.X), int32(head.Y), 20, rl.NewColor(74, 40, 28, 255))
		rl.DrawCircleV(rl.NewVector2(head.X-6, head.Y-6), 9, rl.NewColor(255, 140, 40, 255))
		rl.DrawCircleV(rl.NewVector2(head.X+8, head.Y+5), 5.5, rl.NewColor(255, 200, 90, 255))
		rl.BeginBlendMode(rl.BlendAdditive)
	} else {
		// Impact: expanding shock ring + hot flash fading out.
		prog := clamp01((mt.Age - MeteorFallTime) / meteorImpactTime)
		r := MeteorImpactRadius * (0.35 + prog*1.05)
		a := 1 - prog
		rl.DrawRing(mt.Target, r-5, r+5, 0, 360, 48, rl.Fade(rl.NewColor(255, 150, 50, 255), 0.85*a))
		rl.DrawCircleGradient(int32(mt.Target.X), int32(mt.Target.Y),
			MeteorImpactRadius*(0.9-0.4*prog), rl.Fade(rl.Orange, 0.55*a), rl.Blank)
		rl.DrawCircleGradient(int32(mt.Target.X), int32(mt.Target.Y),
			MeteorImpactRadius*0.35*(1-prog), rl.Fade(rl.NewColor(255, 252, 220, 255), 0.9*a), rl.Blank)
	}

	for _, pt := range mt.Trail.particles {
		drawParticle(pt)
	}
}
