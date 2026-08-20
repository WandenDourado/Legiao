package skill

// Host-side damage for the burning ground the Bola de Fogo leaves behind.
//
// The zone used to be decoration only: FireGroundDamagePerSec was declared and
// never read, so the fire looked lethal and did nothing. Impact damage lives in
// StepFireballs; this is the part that keeps hurting after the explosion.
//
// Fire hurts MONSTERS only. Neither the caster nor allies burn — the Mago
// drops these zones on top of a melee fight by design.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
)

// FireGroundEvent reports the damage of one burning-ground tick so the caller
// can broadcast it. Dead enemies are NOT removed here.
type FireGroundEvent struct {
	EnemyID string
	Damage  float32
	Died    bool
}

// StepFireGrounds applies the burn to every monster standing in a fire zone.
// It only handles damage: ageing, particle emission and pruning stay in
// Manager.UpdateFire, which the host already calls each frame.
func StepFireGrounds(m *Manager, enemies []*entity.Enemy, dt float32) []FireGroundEvent {
	events := make([]FireGroundEvent, 0)

	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()

	for _, g := range m.FireGrounds {
		g.tickAccum += dt
		if g.tickAccum < fireGroundTickEvery {
			continue
		}
		g.tickAccum -= fireGroundTickEvery

		dmg := FireGroundDamagePerSec * fireGroundTickEvery
		for _, e := range enemies {
			if !e.IsActive || !g.Contains(e.Position) {
				continue
			}
			events = append(events, FireGroundEvent{
				EnemyID: e.ID,
				Damage:  dmg,
				Died:    e.TakeDamage(dmg),
			})
		}
	}
	return events
}
