package skill

import (
	"sync"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Manager owns the host/server-authoritative collections for character skills
// (fireballs, ground fire, explosions, sanctuaries). It lives in the `skill`
// package so the entity package stays focused on players/enemies/projectiles.
type Manager struct {
	Fireballs  map[string]*Fireball
	FireGrounds []*FireGround
	Explosions []*Explosion
	Sanctuaries  map[string]*Sanctuary
	sanctuaryFX map[string]*SanctuaryFX
	Arrows      map[string]*Arrow
	// arrowVolleys holds the pending follow-up waves of arrow-volley casts.
	arrowVolleys []*volleyBurst
	Shields     map[string]*Shield
	// Swords holds active Paladina basic-attack sweeps.
	Swords map[string]*SwordSweep
	// Ultimate-skill collections. They share ultMutex: ultimates are rare
	// events, so one coarse lock keeps the manager struct manageable.
	MeteorRains []*MeteorRain
	Meteors     map[string]*Meteor
	Angelics    map[string]*AngelicArea
	Celestials  map[string]*CelestialArrow
	Avatars     map[string]*Avatar
	// Necromante collections (graveyards + spectral legions) share necroMutex.
	// Efeitos do chefe do mapa 7. Mutex proprio: eles pulsam duas vezes por
	// segundo (nevoa) e nao devem disputar o `ultMutex`, que serve a eventos
	// raros.
	Thorns []*Thorn
	Fog    *DarkFog

	Graveyards map[string]*Graveyard
	Legions    map[string]*Legion
	necroMutex sync.RWMutex
	fireMutex   sync.RWMutex
	sanctMutex  sync.RWMutex
	arrowMutex  sync.RWMutex
	shieldMutex sync.RWMutex
	swordMutex  sync.RWMutex
	ultMutex    sync.RWMutex
	bossMutex   sync.RWMutex
}

// NewManager creates an empty skill manager.
func NewManager() *Manager {
	return &Manager{
		Fireballs:  make(map[string]*Fireball),
		Sanctuaries: make(map[string]*Sanctuary),
		Arrows:     make(map[string]*Arrow),
		Shields:    make(map[string]*Shield),
		Swords:     make(map[string]*SwordSweep),
		Meteors:    make(map[string]*Meteor),
		Angelics:   make(map[string]*AngelicArea),
		Celestials: make(map[string]*CelestialArrow),
		Avatars:    make(map[string]*Avatar),
		Graveyards: make(map[string]*Graveyard),
		Legions:    make(map[string]*Legion),
	}
}

// --- Fireball ---

// AddFireball registers a fireball in the manager (thread-safe).
func (m *Manager) AddFireball(f *Fireball) {
	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()
	m.Fireballs[f.ID] = f
}

// RemoveFireball deletes a fireball by ID.
func (m *Manager) RemoveFireball(id string) {
	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()
	delete(m.Fireballs, id)
}

// GetAllFireballs returns a snapshot of active fireballs.
func (m *Manager) GetAllFireballs() []*Fireball {
	m.fireMutex.RLock()
	defer m.fireMutex.RUnlock()
	list := make([]*Fireball, 0, len(m.Fireballs))
	for _, f := range m.Fireballs {
		list = append(list, f)
	}
	return list
}

// AddFireGround appends a burning ground zone.
func (m *Manager) AddFireGround(g *FireGround) {
	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()
	m.FireGrounds = append(m.FireGrounds, g)
}

// GetAllFireGrounds returns a snapshot of active ground-fire zones.
func (m *Manager) GetAllFireGrounds() []*FireGround {
	m.fireMutex.RLock()
	defer m.fireMutex.RUnlock()
	list := make([]*FireGround, len(m.FireGrounds))
	copy(list, m.FireGrounds)
	return list
}

// AddExplosion appends a one-shot explosion.
func (m *Manager) AddExplosion(e *Explosion) {
	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()
	m.Explosions = append(m.Explosions, e)
}

// SpawnFireball launches a new fireball from start toward dir (world-space).
func SpawnFireball(m *Manager, ownerID string, start, dir rl.Vector2) {
	m.AddFireball(NewFireball(ownerID, start, dir))
}

// AdvanceFireballs animates client-side fireball visuals and prunes expired
// ones (range/TTL) so orphaned fireballs never fly forever. The authoritative
// removal-on-impact still comes from the host's explosion event.
func (m *Manager) AdvanceFireballs(dt float32) {
	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()
	for id, f := range m.Fireballs {
		f.AdvanceVisual(dt)
		if f.Expired() {
			delete(m.Fireballs, id)
		}
	}
}

// RemoveFireballsNear deletes every fireball within radius of pos. Clients use
// it when the host reports an impact, so the visual fireball vanishes exactly
// where the explosion happens.
func (m *Manager) RemoveFireballsNear(pos rl.Vector2, radius float32) {
	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()
	for id, f := range m.Fireballs {
		if rl.Vector2Distance(f.Position, pos) <= radius {
			delete(m.Fireballs, id)
		}
	}
}

// StepFireballs advances fire projectiles and resolves impacts (enemy/obstacle
// collisions). On impact it spawns an explosion + ground fire and applies area
// damage to enemies. Returns the world positions of impacts and the ids of
// enemies that died, so the caller can broadcast visuals / remove the dead.
// Player health is NOT mutated here (network layer does that, outside `skill`).
func StepFireballs(m *Manager, enemies []*entity.Enemy, collisionRects []rl.Rectangle, dt float32) ([]rl.Vector2, []string) {
	impacts := make([]rl.Vector2, 0)
	dead := make([]string, 0)
	for _, fb := range m.GetAllFireballs() {
		alive := fb.Update(dt)
		hitObstacle := tilemap.IsColliding(fb.Position, fb.Radius*2, fb.Radius*2, collisionRects)
		hitEnemy := firstEnemyHit(enemies, fb.Position, fb.Radius)
		if alive && !hitObstacle && hitEnemy == "" {
			continue
		}
		m.RemoveFireball(fb.ID)
		m.AddExplosion(NewExplosion(fb.Position, FireballExplosionRadius))
		m.AddFireGround(NewFireGround(fb.Position))
		impacts = append(impacts, fb.Position)
		dead = append(dead, applyEnemyAreaDamage(enemies, fb.Position, FireballExplosionRadius, FireballExplosionDamage)...)
	}
	return impacts, dead
}

// firstEnemyHit returns the id of the first enemy overlapping pos, or "".
func firstEnemyHit(enemies []*entity.Enemy, pos rl.Vector2, radius float32) string {
	for _, e := range enemies {
		// HitCenter em todo teste de ACERTO: a posicao do monstro e a ancora
		// (o pe, em quem tem FootLine) e a caixa de acerto e o corpo.
		if rl.Vector2Distance(pos, e.HitCenter()) <= radius+e.HitRadius() {
			return e.ID
		}
	}
	return ""
}

// applyEnemyAreaDamage deals dmg to every enemy within radius of center and
// returns the ids of enemies that died.
func applyEnemyAreaDamage(enemies []*entity.Enemy, center rl.Vector2, radius, dmg float32) []string {
	dead := make([]string, 0)
	for _, e := range enemies {
		if rl.Vector2Distance(center, e.HitCenter()) > radius+e.HitRadius() {
			continue
		}
		if e.TakeDamage(dmg) {
			dead = append(dead, e.ID)
		}
	}
	return dead
}

// UpdateFire animates ground fire and explosions and drops finished ones.
// Fireball MOVEMENT is intentionally NOT handled here: on the host it is owned
// exclusively by StepFireballs (advancing here too made fireballs fly at
// double speed), and on clients it is handled by AdvanceFireballs.
func (m *Manager) UpdateFire(dt float32) {
	m.fireMutex.Lock()
	defer m.fireMutex.Unlock()

	keptG := m.FireGrounds[:0]
	for _, g := range m.FireGrounds {
		if g.Update(dt) {
			keptG = append(keptG, g)
		}
	}
	m.FireGrounds = keptG
	keptE := m.Explosions[:0]
	for _, e := range m.Explosions {
		if e.Update(dt) {
			keptE = append(keptE, e)
		}
	}
	m.Explosions = keptE
}

// DrawFire renders ground fire, fireballs, and explosions in world space.
func (m *Manager) DrawFire() {
	m.fireMutex.RLock()
	defer m.fireMutex.RUnlock()
	for _, g := range m.FireGrounds {
		g.Draw()
	}
	for _, f := range m.Fireballs {
		f.Draw()
	}
	for _, e := range m.Explosions {
		e.Draw()
	}
}
