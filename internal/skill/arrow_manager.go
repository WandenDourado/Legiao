package skill

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Arrow volley collection (Arqueiro) ---

// AddArrow registers an arrow in the manager (thread-safe).
func (m *Manager) AddArrow(a *Arrow) {
	m.arrowMutex.Lock()
	defer m.arrowMutex.Unlock()
	if m.Arrows == nil {
		m.Arrows = make(map[string]*Arrow)
	}
	m.Arrows[a.ID] = a
}

// RemoveArrow deletes an arrow by ID.
func (m *Manager) RemoveArrow(id string) {
	m.arrowMutex.Lock()
	defer m.arrowMutex.Unlock()
	delete(m.Arrows, id)
}

// GetAllArrows returns a snapshot of active arrows.
func (m *Manager) GetAllArrows() []*Arrow {
	m.arrowMutex.RLock()
	defer m.arrowMutex.RUnlock()
	list := make([]*Arrow, 0, len(m.Arrows))
	for _, a := range m.Arrows {
		list = append(list, a)
	}
	return list
}

// volleyBurst schedules the follow-up waves of one cast: the same fan is
// fired ArrowVolleyWaves times, ArrowVolleyWaveDelay apart.
type volleyBurst struct {
	ownerID   string
	origin    rl.Vector2
	dir       rl.Vector2
	remaining int
	timer     float32
}

// SpawnArrowVolley fires the first fan of ArrowCount arrows immediately and
// schedules the remaining waves. Runs identically on host and clients (both
// step the schedule every frame), so a single spawn event keeps them in sync.
func SpawnArrowVolley(m *Manager, ownerID string, start, dir rl.Vector2) {
	m.arrowMutex.Lock()
	defer m.arrowMutex.Unlock()
	m.spawnFanLocked(ownerID, start, dir)
	if ArrowVolleyWaves > 1 {
		m.arrowVolleys = append(m.arrowVolleys, &volleyBurst{
			ownerID:   ownerID,
			origin:    start,
			dir:       dir,
			remaining: ArrowVolleyWaves - 1,
			timer:     ArrowVolleyWaveDelay,
		})
	}
}

// spawnFanLocked adds one full fan of arrows. Caller holds arrowMutex.
func (m *Manager) spawnFanLocked(ownerID string, start, dir rl.Vector2) {
	if m.Arrows == nil {
		m.Arrows = make(map[string]*Arrow)
	}
	for _, d := range VolleyDirections(dir) {
		a := NewArrow(ownerID, start, d)
		m.Arrows[a.ID] = a
	}
}

// stepArrowVolleysLocked fires due waves and drops finished bursts. Caller
// holds arrowMutex.
func (m *Manager) stepArrowVolleysLocked(dt float32) {
	kept := m.arrowVolleys[:0]
	for _, v := range m.arrowVolleys {
		v.timer -= dt
		for v.timer <= 0 && v.remaining > 0 {
			v.timer += ArrowVolleyWaveDelay
			v.remaining--
			m.spawnFanLocked(v.ownerID, v.origin, v.dir)
		}
		if v.remaining > 0 {
			kept = append(kept, v)
		}
	}
	m.arrowVolleys = kept
}

// StepArrows advances arrows on the host and resolves impacts against enemies
// and obstacles. Each arrow damages a single enemy then disappears. Returns
// the ids of enemies that died so the caller can remove/broadcast them.
func StepArrows(m *Manager, enemies []*entity.Enemy, collisionRects []rl.Rectangle, dt float32) []string {
	// Fire any wave that came due (host side of the 3x burst).
	m.arrowMutex.Lock()
	m.stepArrowVolleysLocked(dt)
	m.arrowMutex.Unlock()

	dead := make([]string, 0)
	for _, a := range m.GetAllArrows() {
		alive := a.Update(dt)
		hitObstacle := tilemap.IsColliding(a.Position, ArrowHitRadius*2, ArrowHitRadius*2, collisionRects)
		hitEnemy := firstEnemyHit(enemies, a.Position, ArrowHitRadius)
		if alive && !hitObstacle && hitEnemy == "" {
			continue
		}
		if hitEnemy != "" {
			if damageEnemyByID(enemies, hitEnemy, ArrowDamage) {
				dead = append(dead, hitEnemy)
			}
		}
		m.RemoveArrow(a.ID)
	}
	return dead
}

// damageEnemyByID applies dmg to the enemy with the given id and reports
// whether it died.
func damageEnemyByID(enemies []*entity.Enemy, id string, dmg float32) bool {
	for _, e := range enemies {
		if e.ID == id {
			return e.TakeDamage(dmg)
		}
	}
	return false
}

// AdvanceArrows animates arrows client-side (visual only) and prunes the ones
// that exceeded their range/lifetime. Host removal on hit is authoritative;
// clients simply let arrows fly out.
func (m *Manager) AdvanceArrows(dt float32) {
	m.arrowMutex.Lock()
	defer m.arrowMutex.Unlock()
	// Fire any wave that came due (client side of the 3x burst).
	m.stepArrowVolleysLocked(dt)
	for id, a := range m.Arrows {
		a.AdvanceVisual(dt)
		if a.Expired() {
			delete(m.Arrows, id)
		}
	}
}

// DrawArrows renders all arrows in world space.
func (m *Manager) DrawArrows() {
	m.arrowMutex.RLock()
	defer m.arrowMutex.RUnlock()
	for _, a := range m.Arrows {
		a.Draw()
	}
}
