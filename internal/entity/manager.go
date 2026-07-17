package entity

import (
	"sync"

	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// EntityManager manages collections of enemies and projectiles in the game.
// It provides thread-safe operations for adding, removing, and updating entities.
type EntityManager struct {
	Enemies      map[string]*Enemy
	Projectiles  map[string]*Projectile
	Fireballs    map[string]*Fireball
	FireGrounds  []*FireGround
	Explosions   []*Explosion
	enemiesMutex sync.RWMutex
	projMutex    sync.RWMutex
	fireMutex    sync.RWMutex
	WorldBounds  world.Bounds
}

// NewEntityManager creates a new entity manager with initialized maps.
func NewEntityManager() *EntityManager {
	return &EntityManager{
		Enemies:     make(map[string]*Enemy),
		Projectiles: make(map[string]*Projectile),
		Fireballs:   make(map[string]*Fireball),
	}
}

// AddEnemy adds an enemy to the manager.
func (em *EntityManager) AddEnemy(e *Enemy) {
	em.enemiesMutex.Lock()
	defer em.enemiesMutex.Unlock()
	em.Enemies[e.ID] = e
}

// RemoveEnemy removes an enemy by ID.
func (em *EntityManager) RemoveEnemy(id string) {
	em.enemiesMutex.Lock()
	defer em.enemiesMutex.Unlock()
	delete(em.Enemies, id)
}

// GetEnemy returns an enemy by ID, or nil if not found.
func (em *EntityManager) GetEnemy(id string) *Enemy {
	em.enemiesMutex.RLock()
	defer em.enemiesMutex.RUnlock()
	return em.Enemies[id]
}

// GetAllEnemies returns a slice of all active enemies.
func (em *EntityManager) GetAllEnemies() []*Enemy {
	em.enemiesMutex.RLock()
	defer em.enemiesMutex.RUnlock()
	enemies := make([]*Enemy, 0, len(em.Enemies))
	for _, e := range em.Enemies {
		if e.IsActive {
			enemies = append(enemies, e)
		}
	}
	return enemies
}

// GetActiveEnemyCount returns the number of active enemies.
func (em *EntityManager) GetActiveEnemyCount() int {
	em.enemiesMutex.RLock()
	defer em.enemiesMutex.RUnlock()
	count := 0
	for _, e := range em.Enemies {
		if e.IsActive {
			count++
		}
	}
	return count
}

// AddProjectile adds a projectile to the manager.
func (em *EntityManager) AddProjectile(p *Projectile) {
	em.projMutex.Lock()
	defer em.projMutex.Unlock()
	em.Projectiles[p.ID] = p
}

// RemoveProjectile removes a projectile by ID.
func (em *EntityManager) RemoveProjectile(id string) {
	em.projMutex.Lock()
	defer em.projMutex.Unlock()
	delete(em.Projectiles, id)
}

// GetProjectile returns a projectile by ID, or nil if not found.
func (em *EntityManager) GetProjectile(id string) *Projectile {
	em.projMutex.RLock()
	defer em.projMutex.RUnlock()
	return em.Projectiles[id]
}

// GetAllProjectiles returns a slice of all active projectiles.
func (em *EntityManager) GetAllProjectiles() []*Projectile {
	em.projMutex.RLock()
	defer em.projMutex.RUnlock()
	projectiles := make([]*Projectile, 0, len(em.Projectiles))
	for _, p := range em.Projectiles {
		if p.IsActive {
			projectiles = append(projectiles, p)
		}
	}
	return projectiles
}

// UpdateAll updates all entities (enemies and projectiles).
// Returns a map of enemy IDs that attacked this frame.
// For host authoritative simulation.
func (em *EntityManager) UpdateAll(dt float32, players []PlayerState) map[string]bool {
	attackedEnemies := make(map[string]bool)

	// Update enemies
	em.enemiesMutex.Lock()
	for _, e := range em.Enemies {
		if e.IsActive {
			if e.Update(dt, players) {
				attackedEnemies[e.ID] = true
			}
		}
	}
	em.enemiesMutex.Unlock()

	// Update projectiles and remove inactive ones
	em.projMutex.Lock()
	for id, p := range em.Projectiles {
		if p.IsActive {
			if !p.Update(dt, em.WorldBounds) {
				delete(em.Projectiles, id)
			}
		} else {
			delete(em.Projectiles, id)
		}
	}
	em.projMutex.Unlock()

	return attackedEnemies
}

// DrawAll renders all active entities (enemies and projectiles).
func (em *EntityManager) DrawAll() {
	// Draw enemies
	em.enemiesMutex.RLock()
	for _, e := range em.Enemies {
		if e.IsActive {
			e.Draw()
			e.DrawHealthBar()
		}
	}
	em.enemiesMutex.RUnlock()

	// Draw projectiles
	em.projMutex.RLock()
	for _, p := range em.Projectiles {
		if p.IsActive {
			p.Draw()
		}
	}
	em.projMutex.RUnlock()
}

// Clear removes all entities (for game reset).
func (em *EntityManager) Clear() {
	em.enemiesMutex.Lock()
	em.Enemies = make(map[string]*Enemy)
	em.enemiesMutex.Unlock()

	em.projMutex.Lock()
	em.Projectiles = make(map[string]*Projectile)
	em.projMutex.Unlock()

	em.fireMutex.Lock()
	em.Fireballs = make(map[string]*Fireball)
	em.FireGrounds = nil
	em.Explosions = nil
	em.fireMutex.Unlock()
}

// DetectProjectileCollision checks if a projectile hit an enemy.
// Returns the enemy ID that was hit, or empty string if no hit.
func (em *EntityManager) DetectProjectileCollision(proj *Projectile) string {
	em.enemiesMutex.RLock()
	defer em.enemiesMutex.RUnlock()

	for _, e := range em.Enemies {
		if !e.IsActive {
			continue
		}
		dist := rl.Vector2Distance(proj.Position, e.Position)
		if dist <= proj.Radius+e.Radius {
			return e.ID
		}
	}
	return ""
}

// DetectEnemyPlayerCollision checks if any enemy is in attack range of any player.
// Returns a map of playerID -> damage to apply.
func (em *EntityManager) DetectEnemyPlayerCollision(players []PlayerState) map[string]float32 {
	damageMap := make(map[string]float32)

	em.enemiesMutex.RLock()
	defer em.enemiesMutex.RUnlock()

	for _, e := range em.Enemies {
		if !e.IsActive {
			continue
		}
		for _, p := range players {
			if p.IsDead {
				continue
			}
			playerPos := rl.NewVector2(float32(p.X), float32(p.Y))
			dist := rl.Vector2Distance(e.Position, playerPos)
			if dist <= e.AttackRange+e.Radius {
				damageMap[p.PlayerID] = e.AttackDamage
			}
		}
	}
	return damageMap
}
