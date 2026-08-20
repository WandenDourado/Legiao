package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Shield collection (Paladina) ---

// ActivateShield creates a shield for ownerID or, if one is already active,
// regenerates it back to full strength (values never stack).
func ActivateShield(m *Manager, ownerID string, pos rl.Vector2) {
	m.shieldMutex.Lock()
	defer m.shieldMutex.Unlock()
	if m.Shields == nil {
		m.Shields = make(map[string]*Shield)
	}
	if s, ok := m.Shields[ownerID]; ok && !s.Finished() {
		s.Position = pos
		s.Regenerate()
		return
	}
	m.Shields[ownerID] = NewShield(ownerID, pos)
}

// SetShieldHP force-sets a shield's strength (client sync from host events).
// A zero-or-less value triggers the break/shatter animation.
func (m *Manager) SetShieldHP(ownerID string, hp float32) {
	m.shieldMutex.Lock()
	defer m.shieldMutex.Unlock()
	s, ok := m.Shields[ownerID]
	if !ok {
		return
	}
	if hp <= 0 {
		if s.HP > 0 {
			s.Absorb(s.HP + 1) // consume remaining strength -> shatter FX
		}
		return
	}
	s.HP = hp
	s.HitFlash = 0.25
}

// AbsorbShieldDamage routes dmg through ownerID's shield (host authoritative).
// It returns the damage that leaks to the player, the shield HP after the hit,
// and whether a shield was present to absorb anything.
func (m *Manager) AbsorbShieldDamage(ownerID string, dmg float32) (leftover, hpAfter float32, had bool) {
	m.shieldMutex.Lock()
	defer m.shieldMutex.Unlock()
	s, ok := m.Shields[ownerID]
	if !ok || s.Broken() {
		return dmg, 0, false
	}
	leftover, _ = s.Absorb(dmg)
	return leftover, s.HP, true
}

// SetShieldAnchor keeps a shield glued to its owner's current position.
func (m *Manager) SetShieldAnchor(ownerID string, pos rl.Vector2) {
	m.shieldMutex.Lock()
	defer m.shieldMutex.Unlock()
	if s, ok := m.Shields[ownerID]; ok {
		s.Position = pos
	}
}

// RemoveShield drops ownerID's shield immediately (e.g., owner died).
func (m *Manager) RemoveShield(ownerID string) {
	m.shieldMutex.Lock()
	defer m.shieldMutex.Unlock()
	delete(m.Shields, ownerID)
}

// HasShield reports whether ownerID currently has an unbroken shield.
func (m *Manager) HasShield(ownerID string) bool {
	m.shieldMutex.RLock()
	defer m.shieldMutex.RUnlock()
	s, ok := m.Shields[ownerID]
	return ok && !s.Broken()
}

// UpdateShields advances shield animations and prunes finished (shattered)
// shields. Safe for both host and client managers.
func (m *Manager) UpdateShields(dt float32) {
	m.shieldMutex.Lock()
	defer m.shieldMutex.Unlock()
	for id, s := range m.Shields {
		s.Update(dt)
		if s.Finished() {
			delete(m.Shields, id)
		}
	}
}

// DrawShields renders every active shield aura in world space.
func (m *Manager) DrawShields() {
	m.shieldMutex.RLock()
	defer m.shieldMutex.RUnlock()
	for _, s := range m.Shields {
		s.Draw()
	}
}
