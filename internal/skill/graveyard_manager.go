package skill

import (
	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Graveyard collection (Necromante Q) ---

// GraveEvent reports host-authoritative damage dealt by a graveyard tick.
type GraveEvent struct {
	EnemyID string
	Damage  float32
	Died    bool
}

// AddGraveyard registers a graveyard in the manager (thread-safe).
func (m *Manager) AddGraveyard(g *Graveyard) {
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	m.Graveyards[g.ID] = g
}

// GetAllGraveyards returns a snapshot of active graveyards.
func (m *Manager) GetAllGraveyards() []*Graveyard {
	m.necroMutex.RLock()
	defer m.necroMutex.RUnlock()
	list := make([]*Graveyard, 0, len(m.Graveyards))
	for _, g := range m.Graveyards {
		list = append(list, g)
	}
	return list
}

// SpawnGraveyard raises cursed ground for ownerID starting at origin along dir.
func SpawnGraveyard(m *Manager, ownerID string, origin, dir rl.Vector2) *Graveyard {
	g := NewGraveyard(ownerID, origin, dir)
	m.AddGraveyard(g)
	return g
}

// StepGraveyards advances every graveyard on the HOST: ages/animates them,
// slows enemies standing on cursed ground every frame, applies the damage-
// over-time in steady ticks, and prunes expired zones. Returns the damage
// events for the caller to broadcast (dead enemies are NOT removed here).
func StepGraveyards(m *Manager, enemies []*entity.Enemy, dt float32) []GraveEvent {
	events := make([]GraveEvent, 0)
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	for id, g := range m.Graveyards {
		g.advanceVisual(dt)
		if g.Expired() {
			delete(m.Graveyards, id)
			continue
		}
		if !g.IsCursing() {
			continue
		}
		g.tickAccum += dt
		tick := g.tickAccum >= graveyardTickEvery
		if tick {
			g.tickAccum -= graveyardTickEvery
		}
		for _, e := range enemies {
			if !e.IsActive || !g.Contains(e.Position) {
				continue
			}
			e.ApplySlow(GraveyardSlowFactor, GraveyardSlowLinger)
			if !tick {
				continue
			}
			dmg := GraveyardDPS * graveyardTickEvery
			died := e.TakeDamage(dmg)
			events = append(events, GraveEvent{EnemyID: e.ID, Damage: dmg, Died: died})
		}
	}
	return events
}

// AdvanceGraveyards animates graveyards on CLIENTS (visuals only, no
// gameplay) and prunes the fully dissipated ones.
func (m *Manager) AdvanceGraveyards(dt float32) {
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	for id, g := range m.Graveyards {
		g.advanceVisual(dt)
		if g.Expired() {
			delete(m.Graveyards, id)
		}
	}
}

// DrawGraveyards renders every graveyard in world space.
func (m *Manager) DrawGraveyards() {
	m.necroMutex.RLock()
	defer m.necroMutex.RUnlock()
	for _, g := range m.Graveyards {
		g.Draw()
	}
}
