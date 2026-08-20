package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// AddSanctuary registers a sanctuary in the manager (thread-safe).
func (m *Manager) AddSanctuary(s *Sanctuary, fx *SanctuaryFX) {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	m.Sanctuaries[s.ID] = s
	if m.sanctuaryFX == nil {
		m.sanctuaryFX = make(map[string]*SanctuaryFX)
	}
	m.sanctuaryFX[s.ID] = fx
}

// RemoveSanctuary deletes a sanctuary by ID (used after full dissipation).
func (m *Manager) RemoveSanctuary(id string) {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	delete(m.Sanctuaries, id)
	if m.sanctuaryFX != nil {
		delete(m.sanctuaryFX, id)
	}
}

// GetAllSanctuaries returns a snapshot of active sanctuaries.
func (m *Manager) GetAllSanctuaries() []*Sanctuary {
	m.sanctMutex.RLock()
	defer m.sanctMutex.RUnlock()
	list := make([]*Sanctuary, 0, len(m.Sanctuaries))
	for _, s := range m.Sanctuaries {
		list = append(list, s)
	}
	return list
}

// SpawnSanctuary creates a sanctuary owned by ownerID at the given center.
func SpawnSanctuary(m *Manager, ownerID string, center rl.Vector2) {
	s := NewSanctuary(ownerID, center)
	m.AddSanctuary(s, NewSanctuaryFX())
}

// UpdateSanctuaries advances every sanctuary's lifetime + visuals. It returns
// heal events accumulated this tick for the caller to apply/broadcast. Each
// sanctuary is aged here; the host keeps authoritative ownership of removal.
func (m *Manager) UpdateSanctuaries(dt float32, allies map[string]PlayerHealTarget) []SanctuaryHealEvent {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	allEvents := make([]SanctuaryHealEvent, 0)
	for id, s := range m.Sanctuaries {
		s.Age += dt
		// advance visuals
		if m.sanctuaryFX != nil {
			if fx, ok := m.sanctuaryFX[id]; ok {
				fx.update(dt)
			}
		}
		// healing happens during the active window
		events := StepSanctuaryHealing(s, dt, allies)
		allEvents = append(allEvents, events...)
		// fully dissipated -> remove
		if s.Age > SanctuaryDuration+SanctuaryFade {
			s.Dead = true
			delete(m.Sanctuaries, id)
			if m.sanctuaryFX != nil {
				delete(m.sanctuaryFX, id)
			}
		}
	}
	return allEvents
}

// DrawSanctuaries renders all active sanctuaries (world space). Call inside a
// BeginMode2D block, after the floor and before/with the players as desired.
func (m *Manager) DrawSanctuaries() {
	m.sanctMutex.RLock()
	defer m.sanctMutex.RUnlock()
	for id, s := range m.Sanctuaries {
		a := s.FadeAlpha()
		if a <= 0 {
			continue
		}
		if m.sanctuaryFX != nil {
			if fx, ok := m.sanctuaryFX[id]; ok {
				fx.draw(s.Position, a)
			}
		}
	}
}

// AdvanceSanctuaries ages sanctuaries and advances their procedural FX without
// applying healing. Used on clients, where HP is already synced via combat
// events; this only keeps the visuals moving and prunes finished sanctuaries.
func (m *Manager) AdvanceSanctuaries(dt float32) {
	m.sanctMutex.Lock()
	defer m.sanctMutex.Unlock()
	for id, s := range m.Sanctuaries {
		s.Age += dt
		if m.sanctuaryFX != nil {
			if fx, ok := m.sanctuaryFX[id]; ok {
				fx.update(dt)
			}
		}
		if s.Age > SanctuaryDuration+SanctuaryFade {
			s.Dead = true
			delete(m.Sanctuaries, id)
			if m.sanctuaryFX != nil {
				delete(m.sanctuaryFX, id)
			}
		}
	}
}
