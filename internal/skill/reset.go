package skill

// Reset empties every collection the manager owns.
//
// It exists for the stage restart: the field has to be clean before the horde
// runs again, and every visual anchored to a player (shields, avatars, sword
// sweeps, spectral legions) has to go with it or it would keep following a
// player who has already been put back on the spawn point.
//
// Each group is cleared under its own lock, which is why this is not simply a
// new Manager: casts arrive from client goroutines and may be writing while
// the reset runs.
func (m *Manager) Reset() {
	m.fireMutex.Lock()
	m.Fireballs = make(map[string]*Fireball)
	m.FireGrounds = nil
	m.Explosions = nil
	m.fireMutex.Unlock()

	m.sanctMutex.Lock()
	m.Sanctuaries = make(map[string]*Sanctuary)
	m.sanctuaryFX = nil
	m.sanctMutex.Unlock()

	m.arrowMutex.Lock()
	m.Arrows = make(map[string]*Arrow)
	m.arrowVolleys = nil
	m.arrowMutex.Unlock()

	m.shieldMutex.Lock()
	m.Shields = make(map[string]*Shield)
	m.shieldMutex.Unlock()

	m.swordMutex.Lock()
	m.Swords = make(map[string]*SwordSweep)
	m.swordMutex.Unlock()

	m.ultMutex.Lock()
	m.MeteorRains = nil
	m.Meteors = make(map[string]*Meteor)
	m.Angelics = make(map[string]*AngelicArea)
	m.Celestials = make(map[string]*CelestialArrow)
	m.Avatars = make(map[string]*Avatar)
	m.ultMutex.Unlock()

	m.necroMutex.Lock()
	m.Graveyards = make(map[string]*Graveyard)
	m.Legions = make(map[string]*Legion)
	m.necroMutex.Unlock()

	// As esferas da sentinela nao pertencem ao Manager (sao de um monstro, nao
	// de um personagem), mas o reset da fase e o mesmo: uma esfera que
	// sobrevive ao reinicio persegue um jogador que ja renasceu do outro lado
	// do mapa, e ninguem consegue explicar de onde ela veio.
	ResetSentryOrbs()
	// As bolas de canhao, pela mesma razao.
	ResetCannonBalls()
}
