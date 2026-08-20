package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Divine Avatar collection (Paladina ultimate) ---

// ActivateAvatar makes ownerID an avatar (or restarts the duration on recast).
func ActivateAvatar(m *Manager, ownerID string, pos rl.Vector2) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	if m.Avatars == nil {
		m.Avatars = make(map[string]*Avatar)
	}
	if av, ok := m.Avatars[ownerID]; ok {
		av.Remaining = AvatarDuration
		av.Position = pos
		return
	}
	m.Avatars[ownerID] = NewAvatar(ownerID, pos)
}

// HasAvatar reports whether ownerID is currently an invincible avatar.
func (m *Manager) HasAvatar(ownerID string) bool {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	av, ok := m.Avatars[ownerID]
	return ok && av.Active()
}

// SetAvatarAnchor keeps the emanation glued to its owner's position.
func (m *Manager) SetAvatarAnchor(ownerID string, pos rl.Vector2) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	if av, ok := m.Avatars[ownerID]; ok {
		av.Position = pos
	}
}

// UpdateAvatars advances every avatar and prunes the fully expired ones
// (kept briefly past expiry so the last embers finish). Host and client.
func (m *Manager) UpdateAvatars(dt float32) {
	m.ultMutex.Lock()
	defer m.ultMutex.Unlock()
	for id, av := range m.Avatars {
		av.Update(dt)
		if av.Remaining < -1.0 {
			delete(m.Avatars, id)
		}
	}
}

// DrawAvatars renders every avatar emanation (world space).
func (m *Manager) DrawAvatars() {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	for _, av := range m.Avatars {
		av.Draw()
	}
}
