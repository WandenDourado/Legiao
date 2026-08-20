package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Sanctuary constants tune the Priestess (Sacerdotisa) Sanctuary skill.
const (
	// SanctuaryRadius is the world-space radius of the healing area.
	SanctuaryRadius float32 = 200
	// SanctuaryDuration is how long the sanctuary exists before fading out.
	SanctuaryDuration float32 = 7.0
	// SanctuaryFade is how long the graceful dissipation takes at the end.
	SanctuaryFade float32 = 1.5
	// SanctuaryHealPerSec is the continuous HP restored to allies inside the area.
	SanctuaryHealPerSec float32 = 12
	// SanctuaryCooldown is the time the caster must wait between casts.
	SanctuaryCooldown float32 = 14.0
	// SanctuaryOffset places the sanctuary this far in front of the caster.
	SanctuaryOffset float32 = 120
)

// Sanctuary is a procedurally-rendered healing zone cast by the Sacerdotisa.
// It has no sprite: the floor aura, pulsing light ring and floating sacred
// particles are all drawn with raylib primitives.
type Sanctuary struct {
	ID       string
	OwnerID  string
	Position rl.Vector2
	// Age is the elapsed time since the sanctuary was created.
	Age float32
	// Dead becomes true once the sanctuary has fully dissipated.
	Dead bool
	// HealTick accumulates dt so healing is applied in steady increments.
	HealAccum float32
}

// NewSanctuary creates a sanctuary centered at center (world-space).
func NewSanctuary(ownerID string, center rl.Vector2) *Sanctuary {
	return &Sanctuary{
		ID:       generateID(),
		OwnerID:  ownerID,
		Position: center,
		Age:      0,
		Dead:     false,
	}
}

// LifeRatio returns 0..1 progress through the sanctuary's active lifespan
// (excluding fade). Useful for ramps/pulses that should peak mid-life.
func (s *Sanctuary) LifeRatio() float32 {
	if SanctuaryDuration <= 0 {
		return 1
	}
	r := s.Age / SanctuaryDuration
	if r > 1 {
		return 1
	}
	return r
}

// FadeAlpha returns the current global opacity multiplier for the sanctuary.
// It is 1 while active, then linearly ramps 1..0 during the fade window.
func (s *Sanctuary) FadeAlpha() float32 {
	if s.Age <= SanctuaryDuration {
		return 1
	}
	over := s.Age - SanctuaryDuration
	a := 1 - over/SanctuaryFade
	if a < 0 {
		return 0
	}
	return a
}

// IsHealing reports whether the sanctuary is still in its healing window.
func (s *Sanctuary) IsHealing() bool {
	return s.Age <= SanctuaryDuration
}

// Contains reports whether world point p lies inside the sanctuary's radius.
func (s *Sanctuary) Contains(p rl.Vector2) bool {
	return rl.Vector2Distance(s.Position, p) <= SanctuaryRadius
}
