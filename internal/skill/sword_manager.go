package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Sword sweep collection (Paladina basic attack) ---

// AddSword registers a sword sweep in the manager (thread-safe).
func (m *Manager) AddSword(s *SwordSweep) {
	m.swordMutex.Lock()
	defer m.swordMutex.Unlock()
	if m.Swords == nil {
		m.Swords = make(map[string]*SwordSweep)
	}
	m.Swords[s.ID] = s
}

// GetAllSwords returns a snapshot of active sword sweeps.
func (m *Manager) GetAllSwords() []*SwordSweep {
	m.swordMutex.RLock()
	defer m.swordMutex.RUnlock()
	list := make([]*SwordSweep, 0, len(m.Swords))
	for _, s := range m.Swords {
		list = append(list, s)
	}
	return list
}

// SetSwordAnchor glues every sweep owned by ownerID to the owner's live
// position, so the sword follows the character while she walks.
func (m *Manager) SetSwordAnchor(ownerID string, pos rl.Vector2) {
	m.swordMutex.Lock()
	defer m.swordMutex.Unlock()
	for _, s := range m.Swords {
		if s.OwnerID == ownerID {
			s.Position = pos
		}
	}
}

// AdvanceSwords steps every sweep and prunes finished ones.
func (m *Manager) AdvanceSwords(dt float32) {
	m.swordMutex.Lock()
	defer m.swordMutex.Unlock()
	for id, s := range m.Swords {
		if !s.Update(dt) {
			delete(m.Swords, id)
		}
	}
}

// DrawSwords renders all active sword sweeps.
func (m *Manager) DrawSwords() {
	for _, s := range m.GetAllSwords() {
		s.Draw()
	}
}
