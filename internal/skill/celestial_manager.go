package skill

import (
	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Celestial Arrow collection (Arqueiro ultimate) ---

// SpawnCelestialArrow launches one celestial arrow from start toward dir.
func SpawnCelestialArrow(m *Manager, ownerID string, start, dir rl.Vector2) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	if m.Celestials == nil {
		m.Celestials = make(map[string]*CelestialArrow)
	}
	c := NewCelestialArrow(ownerID, start, dir)
	m.Celestials[c.ID] = c
}

// StepCelestials advances celestial arrows on the HOST. Arrows pierce: they
// damage each enemy once and keep flying (walls included — a heavenly shot
// crosses the whole map). Returns the ids of enemies that died.
func StepCelestials(m *Manager, enemies []*entity.Enemy, dt float32) []string {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	dead := make([]string, 0)
	for id, c := range m.Celestials {
		alive := c.Advance(dt)
		for _, e := range enemies {
			if c.HitIDs[e.ID] {
				continue
			}
			if rl.Vector2Distance(c.Position, e.HitCenter()) <= CelestialHitRadius+e.HitRadius() {
				c.HitIDs[e.ID] = true
				c.PierceFlash(e.Position)
				if e.TakeDamage(CelestialDamage) {
					dead = append(dead, e.ID)
				}
			}
		}
		if !alive {
			delete(m.Celestials, id)
		}
	}
	return dead
}

// AdvanceCelestials advances celestial arrows on CLIENTS (visuals only).
func (m *Manager) AdvanceCelestials(dt float32) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	for id, c := range m.Celestials {
		if !c.Advance(dt) {
			delete(m.Celestials, id)
		}
	}
}

// DrawCelestials renders all celestial arrows (world space).
func (m *Manager) DrawCelestials() {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	for _, c := range m.Celestials {
		c.Draw()
	}
}
