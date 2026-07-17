package entity

import (
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// AddFireball registers a fireball in the manager (thread-safe).
func (em *EntityManager) AddFireball(f *Fireball) {
	em.fireMutex.Lock()
	defer em.fireMutex.Unlock()
	em.Fireballs[f.ID] = f
}

// RemoveFireball deletes a fireball by ID.
func (em *EntityManager) RemoveFireball(id string) {
	em.fireMutex.Lock()
	defer em.fireMutex.Unlock()
	delete(em.Fireballs, id)
}

// GetAllFireballs returns a snapshot of active fireballs.
func (em *EntityManager) GetAllFireballs() []*Fireball {
	em.fireMutex.RLock()
	defer em.fireMutex.RUnlock()
	list := make([]*Fireball, 0, len(em.Fireballs))
	for _, f := range em.Fireballs {
		list = append(list, f)
	}
	return list
}

// AddFireGround appends a burning ground zone.
func (em *EntityManager) AddFireGround(g *FireGround) {
	em.fireMutex.Lock()
	defer em.fireMutex.Unlock()
	em.FireGrounds = append(em.FireGrounds, g)
}

// GetAllFireGrounds returns a snapshot of active ground-fire zones.
func (em *EntityManager) GetAllFireGrounds() []*FireGround {
	em.fireMutex.RLock()
	defer em.fireMutex.RUnlock()
	list := make([]*FireGround, len(em.FireGrounds))
	copy(list, em.FireGrounds)
	return list
}

// SpawnFireball launches a new fireball from start toward dir (world-space).
func SpawnFireball(em *EntityManager, ownerID string, start, dir rl.Vector2) {
	em.AddFireball(NewFireball(ownerID, start, dir))
}

// StepFireballs advances fire projectiles and resolves impacts (enemy/obstacle
// collisions). On impact it spawns an explosion + ground fire and applies area
// damage to enemies directly. Returns the world positions of impacts so the
// caller can broadcast visuals and apply player damage. Player health is NOT
// mutated here (network layer does that to stay outside the entity package).
func StepFireballs(em *EntityManager, collisionRects []rl.Rectangle, dt float32) []rl.Vector2 {
	impacts := make([]rl.Vector2, 0)
	for _, fb := range em.GetAllFireballs() {
		alive := fb.Update(dt)
		hitObstacle := tilemap.IsColliding(fb.Position, fb.Radius*2, fb.Radius*2, collisionRects)
		hitEnemy := firstEnemyHit(em, fb.Position, fb.Radius)
		if alive && !hitObstacle && hitEnemy == "" {
			continue
		}
		em.RemoveFireball(fb.ID)
		em.AddExplosion(NewExplosion(fb.Position, FireballExplosionRadius))
		em.AddFireGround(NewFireGround(fb.Position))
		impacts = append(impacts, fb.Position)
		applyEnemyAreaDamage(em, fb.Position, FireballExplosionRadius, FireballExplosionDamage)
	}
	return impacts
}

// firstEnemyHit returns the id of the first enemy overlapping pos, or "".
func firstEnemyHit(em *EntityManager, pos rl.Vector2, radius float32) string {
	for _, e := range em.GetAllEnemies() {
		if rl.Vector2Distance(pos, e.Position) <= radius+e.Radius {
			return e.ID
		}
	}
	return ""
}

// applyEnemyAreaDamage deals dmg to every enemy within radius of center.
func applyEnemyAreaDamage(em *EntityManager, center rl.Vector2, radius, dmg float32) {
	for _, e := range em.GetAllEnemies() {
		if rl.Vector2Distance(center, e.Position) > radius+e.Radius {
			continue
		}
		if e.TakeDamage(dmg) {
			em.RemoveEnemy(e.ID)
		}
	}
}

// AddExplosion appends a one-shot explosion.
func (em *EntityManager) AddExplosion(e *Explosion) {
	em.fireMutex.Lock()
	defer em.fireMutex.Unlock()
	em.Explosions = append(em.Explosions, e)
}

// UpdateFire animates fireballs (visual only), ground fire, and explosions;
// drops finished ground fires and explosions. Fireball REMOVAL + explosion is
// owned exclusively by StepFireballs (host sim), so fireballs are only advanced
// here, never deleted, to avoid a double-removal that would skip the blast.
func (em *EntityManager) UpdateFire(dt float32) {
	em.fireMutex.Lock()
	defer em.fireMutex.Unlock()

	for _, f := range em.Fireballs {
		f.AdvanceVisual(dt)
	}
	keptG := em.FireGrounds[:0]
	for _, g := range em.FireGrounds {
		if g.Update(dt) {
			keptG = append(keptG, g)
		}
	}
	em.FireGrounds = keptG
	keptE := em.Explosions[:0]
	for _, e := range em.Explosions {
		if e.Update(dt) {
			keptE = append(keptE, e)
		}
	}
	em.Explosions = keptE
}

// DrawFire renders ground fire, fireballs, and explosions in world space.
func (em *EntityManager) DrawFire() {
	em.fireMutex.RLock()
	defer em.fireMutex.RUnlock()
	for _, g := range em.FireGrounds {
		g.Draw()
	}
	for _, f := range em.Fireballs {
		f.Draw()
	}
	for _, e := range em.Explosions {
		e.Draw()
	}
}
