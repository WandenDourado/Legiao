package system

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// CheckProjectileCollisions checks if any projectile hit an enemy.
// If a hit is detected, applies damage to the enemy and removes the projectile.
// Returns a map of enemy ID -> damage applied (for broadcasting).
func CheckProjectileCollisions(em *entity.EntityManager) map[string]float32 {
	damageMap := make(map[string]float32)

	projectiles := em.GetAllProjectiles()
	enemies := em.GetAllEnemies()

	for _, proj := range projectiles {
		for _, e := range enemies {
			dist := rl.Vector2Distance(proj.Position, e.Position)
			if dist <= proj.Radius+e.Radius {
				// Hit! Apply damage
				if e.TakeDamage(proj.Damage) {
					// Enemy died
					log.Printf("[Combat] Enemy %s died from projectile", e.ID)
					damageMap[e.ID] = -1 // -1 indicates death
				} else {
					damageMap[e.ID] = proj.Damage
				}
				// Remove projectile on hit
				em.RemoveProjectile(proj.ID)
				break // Projectile can only hit one enemy
			}
		}
	}

	return damageMap
}

// CheckEnemyPlayerCollision checks if enemies are in attack range of players.
// Returns a map of playerID -> damage to apply.
func CheckEnemyPlayerCollision(em *entity.EntityManager, players map[string]network.PlayerState) map[string]float32 {
	damageMap := make(map[string]float32)

	enemies := em.GetAllEnemies()

	for _, e := range enemies {
		if !e.IsActive {
			continue
		}
		for playerID, p := range players {
			if p.IsDead {
				continue
			}
			playerPos := rl.NewVector2(float32(p.X), float32(p.Y))
			dist := rl.Vector2Distance(e.Position, playerPos)
			if dist <= e.AttackRange+e.Radius {
				damageMap[playerID] = e.AttackDamage
			}
		}
	}

	return damageMap
}

// ApplyPlayerDamage applies damage to players and handles death/respawn.
// Returns true if any player died.
func ApplyPlayerDamage(players map[string]*network.PlayerState, damageMap map[string]float32) bool {
	anyDied := false

	for playerID, damage := range damageMap {
		if p, ok := players[playerID]; ok {
			p.Health -= damage
			if p.Health <= 0 {
				p.Health = 0
				p.IsDead = true
				anyDied = true
				log.Printf("[Combat] Player %s died", playerID)
			}
		}
	}

	return anyDied
}

// CheckGameOver returns true if all players are dead.
func CheckGameOver(players map[string]*network.PlayerState) bool {
	if len(players) == 0 {
		return true
	}

	for _, p := range players {
		if !p.IsDead {
			return false // At least one player alive
		}
	}
	return true // All players dead
}

// HandleEnemyDeath removes dead enemies and broadcasts the event.
func HandleEnemyDeath(em *entity.EntityManager, enemyID string) {
	em.RemoveEnemy(enemyID)
	log.Printf("[Combat] Enemy %s removed", enemyID)
}
