package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Divine Avatar constants tune the Paladina's ultimate (Avatar dos Deuses).
const (
	// AvatarDuration is how long the paladin stays an invincible avatar.
	AvatarDuration float32 = 15.0
	// AvatarCooldown is the caster's cooldown in seconds.
	AvatarCooldown float32 = 60.0
	// AvatarRadius is the visual radius of the golden emanation.
	AvatarRadius float32 = 110
	// avatarWaveEvery is the interval between outward golden shockwaves.
	avatarWaveEvery float32 = 1.5
	// avatarWaveLife is how long each shockwave expands before vanishing.
	avatarWaveLife float32 = 0.9
)

var (
	avatarGold  = rl.NewColor(255, 210, 90, 255)
	avatarLight = rl.NewColor(255, 245, 200, 255)
)

// Avatar is the Paladina transfigured: while it lasts the owner is IMMUNE to
// all damage (host checks HasAvatar before applying any). Golden energy
// emanates continuously: ground glyph, light column, halo, rising embers and
// periodic shockwaves. All primitives, no sprites.
type Avatar struct {
	OwnerID   string
	Position  rl.Vector2 // anchor, synced to the owner every frame
	Remaining float32
	Time      float32
	waveTimer float32
	waves     []float32 // ages of live shockwaves
	Embers    *ParticleEmitter
}

// NewAvatar creates a full-duration avatar anchored at pos.
func NewAvatar(ownerID string, pos rl.Vector2) *Avatar {
	return &Avatar{
		OwnerID:   ownerID,
		Position:  pos,
		Remaining: AvatarDuration,
		Embers:    NewParticleEmitter(),
	}
}

// Active reports whether the avatar still protects its owner.
func (av *Avatar) Active() bool { return av.Remaining > 0 }

// fade ramps the whole effect out over the final 1.2s.
func (av *Avatar) fade() float32 {
	return clamp01(av.Remaining / 1.2)
}

// Update advances timers, embers and shockwaves.
func (av *Avatar) Update(dt float32) {
	av.Time += dt
	av.Remaining -= dt

	// Rising golden embers around the body.
	if av.Remaining > 0 {
		for i := 0; i < 2; i++ {
			off := randomPointInRadius(AvatarRadius * 0.55)
			pos := rl.Vector2Add(av.Position, off)
			vel := rl.NewVector2(off.X*0.25, -80-float32(rl.GetRandomValue(0, 70)))
			av.Embers.Emit(pos, vel, 0.7, 7, avatarGold)
		}
		// Periodic outward shockwave: raw divine power pulsing off the avatar.
		av.waveTimer += dt
		if av.waveTimer >= avatarWaveEvery {
			av.waveTimer -= avatarWaveEvery
			av.waves = append(av.waves, 0)
		}
	}
	kept := av.waves[:0]
	for _, w := range av.waves {
		w += dt
		if w < avatarWaveLife {
			kept = append(kept, w)
		}
	}
	av.waves = kept
	av.Embers.Update(dt)
}

// Draw renders the golden transfiguration (world space).
func (av *Avatar) Draw() {
	f := av.fade()
	if f <= 0 && len(av.Embers.particles) == 0 {
		return
	}
	pulse := 1 + 0.05*float32(math.Sin(float64(av.Time)*3.4))
	r := AvatarRadius * pulse

	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	// Ground light: a warm golden pool under the avatar's feet (no sigil —
	// just consecrated ground).
	gy := av.Position.Y + 62
	rl.DrawCircleGradient(int32(av.Position.X), int32(gy), r*0.95,
		rl.Fade(avatarGold, 0.18*f), rl.Blank)

	// Column of light rising FROM THE GROUND. The sprite is ~220px tall with
	// its center at Position, so the feet sit ~110px below center — the
	// pillar base must reach that line, not the waist/knees.
	pillarBase := av.Position.Y + 112
	colW := int32(64)
	colH := int32(350)
	rl.DrawRectangleGradientV(
		int32(av.Position.X)-colW/2, int32(pillarBase)-colH, colW, colH,
		rl.Fade(avatarLight, 0), rl.Fade(avatarGold, 0.30*f))

	// Pulsing body glow (the breathing golden emanation).
	rl.DrawCircleGradient(int32(av.Position.X), int32(av.Position.Y), r*0.85,
		rl.Fade(avatarGold, 0.26*f), rl.Blank)

	// Halo floating above the head.
	haloY := av.Position.Y - 118 + 4*float32(math.Sin(float64(av.Time)*2.2))
	halo := rl.NewVector2(av.Position.X, haloY)
	rl.DrawRing(halo, 16, 20, 0, 360, 32, rl.Fade(avatarLight, 0.85*f))
	rl.DrawCircleGradient(int32(halo.X), int32(halo.Y), 30, rl.Fade(avatarGold, 0.35*f), rl.Blank)

	// Expanding shockwaves.
	for _, w := range av.waves {
		p := w / avatarWaveLife
		wr := r * (1 + p*1.6)
		rl.DrawRing(av.Position, wr-3, wr+3, 0, 360, 56,
			rl.Fade(avatarGold, 0.5*(1-p)*f))
	}

	for _, pt := range av.Embers.particles {
		drawParticle(pt)
	}
}
