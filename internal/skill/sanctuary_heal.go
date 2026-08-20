package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// SanctuaryHealEvent describes one ally that should receive healing this tick.
// The caller (host) applies the HP change and broadcasts it so clients mirror.
type SanctuaryHealEvent struct {
	PlayerID string
	Amount   float32
}

// StepSanctuaryHealing applies continuous healing to every living ally inside
// the sanctuary for this dt. It accumulates fractional healing so low per-second
// rates still resolve smoothly, and only fires an event when a whole point is
// due. Players at full health are skipped (no overheal). Returns the events
// that must be applied + broadcast.
func StepSanctuaryHealing(s *Sanctuary, dt float32, allies map[string]PlayerHealTarget) []SanctuaryHealEvent {
	events := make([]SanctuaryHealEvent, 0)
	if !s.IsHealing() {
		return events
	}
	s.HealAccum += SanctuaryHealPerSec * dt
	whole := int(s.HealAccum)
	if whole > 0 {
		s.HealAccum -= float32(whole)
		for id, a := range allies {
			if a.IsDead || !s.Contains(rl.NewVector2(a.X, a.Y)) {
				continue
			}
			if a.Health >= a.MaxHealth {
				continue
			}
			amount := float32(whole)
			if a.Health+amount > a.MaxHealth {
				amount = a.MaxHealth - a.Health
			}
			events = append(events, SanctuaryHealEvent{PlayerID: id, Amount: amount})
		}
	}
	return events
}

// PlayerHealTarget is a minimal view of a player used for healing checks.
// It avoids depending on the network PlayerState type directly.
type PlayerHealTarget struct {
	X         float32
	Y         float32
	Health    float32
	MaxHealth float32
	IsDead    bool
}
