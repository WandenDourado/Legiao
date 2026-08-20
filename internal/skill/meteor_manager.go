package skill

import (
	"math/rand"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// MeteorRain is the host-side spawner state for the Mago's ultimate: for
// MeteorRainDuration it periodically picks random points across the WHOLE map
// and drops a meteor on each. Clients never hold rains — they receive each
// meteor spawn individually.
type MeteorRain struct {
	OwnerID   string
	Remaining float32
	accum     float32
}

// StartMeteorRain registers a rain owned by ownerID.
func StartMeteorRain(m *Manager, ownerID string, _ rl.Vector2) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	m.MeteorRains = append(m.MeteorRains, &MeteorRain{
		OwnerID:   ownerID,
		Remaining: MeteorRainDuration,
	})
}

// HasMeteorRain reports whether the owner has an active meteor rain.
func (m *Manager) HasMeteorRain(ownerID string) bool {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	for _, r := range m.MeteorRains {
		if r.OwnerID == ownerID {
			return true
		}
	}
	return false
}

// StepMeteorRains advances all rains and returns the ground targets of the
// meteors spawned this tick (already added to the manager). worldW/worldH is
// the map size; targets keep an inner margin so rings stay readable.
func StepMeteorRains(m *Manager, worldW, worldH float32, dt float32) []rl.Vector2 {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	spawned := make([]rl.Vector2, 0)
	kept := m.MeteorRains[:0]
	for _, r := range m.MeteorRains {
		r.Remaining -= dt
		r.accum += dt
		for r.accum >= MeteorRainInterval && r.Remaining > 0 {
			r.accum -= MeteorRainInterval
			margin := MeteorImpactRadius * 0.6
			target := rl.NewVector2(
				margin+rand.Float32()*(worldW-2*margin),
				margin+rand.Float32()*(worldH-2*margin),
			)
			mt := NewMeteor(target)
			m.addMeteorLocked(mt)
			spawned = append(spawned, target)
		}
		if r.Remaining > 0 {
			kept = append(kept, r)
		}
	}
	m.MeteorRains = kept
	return spawned
}

// AddMeteor registers a meteor (thread-safe). Used by clients replicating a
// meteor spawn received from the host.
func (m *Manager) AddMeteor(mt *Meteor) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	m.addMeteorLocked(mt)
}

func (m *Manager) addMeteorLocked(mt *Meteor) {
	if m.Meteors == nil {
		m.Meteors = make(map[string]*Meteor)
	}
	m.Meteors[mt.ID] = mt
}

// StepMeteors advances meteors on the HOST. On each impact frame it applies
// area damage to enemies and collects the ids of enemies that died. Finished
// meteors are pruned. Returns the dead enemy ids.
func StepMeteors(m *Manager, enemies []*entity.Enemy, dt float32) []string {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	dead := make([]string, 0)
	for id, mt := range m.Meteors {
		impactedNow, finished := mt.Advance(dt)
		if impactedNow {
			dead = append(dead,
				applyEnemyAreaDamage(enemies, mt.Target, MeteorImpactRadius, MeteorImpactDamage)...)
		}
		if finished {
			delete(m.Meteors, id)
		}
	}
	return dead
}

// AdvanceMeteors advances meteors on CLIENTS (visuals only, no damage) and
// prunes finished ones. The fall time is deterministic, so impact visuals
// stay in sync with the host without extra events.
func (m *Manager) AdvanceMeteors(dt float32) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	for id, mt := range m.Meteors {
		if _, finished := mt.Advance(dt); finished {
			delete(m.Meteors, id)
		}
	}
}

// DrawMeteors renders all meteors (world space).
func (m *Manager) DrawMeteors() {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	for _, mt := range m.Meteors {
		mt.Draw()
	}
}
