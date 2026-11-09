package skill

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/sound"

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
	sound.PlayAtFor(sound.SFXGraveyard, origin, GraveyardDuration)
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

// DrawGraveyardGrounds desenha o CHAO de cada cemiterio. Chamado antes dos
// corpos (game/renderer.go), junto com o selo do Mago; ver o cabecalho de
// graveyard_draw.go para por que o efeito vai em dois passes.
func (m *Manager) DrawGraveyardGrounds() {
	m.necroMutex.RLock()
	defer m.necroMutex.RUnlock()
	// Ordem estavel, nao a do mapa: ver draw_order.go.
	for _, id := range orderedIDs(m.Graveyards) {
		m.Graveyards[id].DrawGround()
	}
}

// DrawGraveyardOverlays desenha as maos e as almas de cada cemiterio, depois
// dos corpos. E este o passe que GraveyardSkill.Draw entrega ao ability.DrawAll.
func (m *Manager) DrawGraveyardOverlays() {
	m.necroMutex.RLock()
	defer m.necroMutex.RUnlock()
	for _, id := range orderedIDs(m.Graveyards) {
		m.Graveyards[id].DrawOverlay()
	}
}
