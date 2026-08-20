package skill

// ClearOwner removes every effect belonging to ownerID from every
// collection the Manager owns, and returns the "*_end" signals the caller
// must broadcast so clients stop drawing what just lost its owner.
//
// Two different key shapes exist (see Reset, which walks the same
// collections): Shields/Avatars/Angelics/Legions are keyed BY OWNER, so a
// single delete finds them; everything else is keyed by its own effect ID
// and carries an OwnerID field, so it needs a scan. Only avatar/angelic/
// legion have a client-side "_end" handler (client.go) — that is the only
// place an anchored effect's owner vanishing would otherwise leave a visual
// stuck on screen forever, since fireballs/arrows/sanctuaries/swords/
// meteors/celestials/graveyards all expire on their own clocks client-side.
func (m *Manager) ClearOwner(ownerID string) []string {
	var signals []string

	m.shieldMutex.Lock()
	delete(m.Shields, ownerID)
	m.shieldMutex.Unlock()

	m.ultMutex.Lock()
	if _, ok := m.Avatars[ownerID]; ok {
		delete(m.Avatars, ownerID)
		signals = append(signals, "avatar_end")
	}
	if _, ok := m.Angelics[ownerID]; ok {
		delete(m.Angelics, ownerID)
		signals = append(signals, "angelic_end")
	}
	for id, c := range m.Celestials {
		if c.OwnerID == ownerID {
			delete(m.Celestials, id)
		}
	}
	m.ultMutex.Unlock()

	m.necroMutex.Lock()
	if _, ok := m.Legions[ownerID]; ok {
		delete(m.Legions, ownerID)
		signals = append(signals, "legion_end")
	}
	for id, g := range m.Graveyards {
		if g.OwnerID == ownerID {
			delete(m.Graveyards, id)
		}
	}
	m.necroMutex.Unlock()

	m.fireMutex.Lock()
	for id, f := range m.Fireballs {
		if f.OwnerID == ownerID {
			delete(m.Fireballs, id)
		}
	}
	m.fireMutex.Unlock()

	m.arrowMutex.Lock()
	for id, a := range m.Arrows {
		if a.OwnerID == ownerID {
			delete(m.Arrows, id)
		}
	}
	m.arrowMutex.Unlock()

	m.sanctMutex.Lock()
	for id, s := range m.Sanctuaries {
		if s.OwnerID == ownerID {
			delete(m.Sanctuaries, id)
		}
	}
	m.sanctMutex.Unlock()

	m.swordMutex.Lock()
	for id, s := range m.Swords {
		if s.OwnerID == ownerID {
			delete(m.Swords, id)
		}
	}
	m.swordMutex.Unlock()

	return signals
}
