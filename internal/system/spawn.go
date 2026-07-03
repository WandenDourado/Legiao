package system

import (
	"log"
	"math/rand"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/world"
)

const enemySpawnOffset float32 = 180

// SpawnSystem manages enemy spawning.
type SpawnSystem struct {
	spawnTimer    float32
	spawnInterval float32
	maxEnemies    int
	WorldBounds   world.Bounds
}

// NewSpawnSystem creates a new spawn system.
func NewSpawnSystem(bounds world.Bounds) *SpawnSystem {
	return &SpawnSystem{
		spawnTimer:    0,
		spawnInterval: 3.0, // Default spawn interval
		maxEnemies:    20,  // Default max enemies
		WorldBounds:   bounds,
	}
}

// Update checks if it's time to spawn new enemies.
// Should be called each frame with dt (delta time).
func (ss *SpawnSystem) Update(dt float32, em *entity.EntityManager) {
	ss.spawnTimer += dt

	if ss.spawnTimer >= ss.spawnInterval {
		ss.spawnTimer = 0
		ss.SpawnWave(em)
	}
}

// SpawnWave spawns a random number of enemies (1-3) at random positions.
func (ss *SpawnSystem) SpawnWave(em *entity.EntityManager) {
	if em.GetActiveEnemyCount() >= ss.maxEnemies {
		return // Max enemies reached
	}

	// Spawn 1-3 enemies per wave
	count := rand.Intn(3) + 1
	for i := 0; i < count; i++ {
		x, y := getRandomSpawnPosition(ss.WorldBounds)
		e := entity.NewEnemy(entity.EnemyTypeBasic, x, y)
		em.AddEnemy(e)
		log.Printf("[Spawn] Spawned enemy %s at (%.0f, %.0f)", e.ID, x, y)
	}
}

// getRandomSpawnPosition returns a random position at world edges.
func getRandomSpawnPosition(bounds world.Bounds) (float32, float32) {
	side := rand.Intn(4) // 0=top, 1=right, 2=bottom, 3=left
	var x, y float32

	switch side {
	case 0: // Top
		x = rand.Float32() * bounds.Width
		y = -enemySpawnOffset
	case 1: // Right
		x = bounds.Width + enemySpawnOffset
		y = rand.Float32() * bounds.Height
	case 2: // Bottom
		x = rand.Float32() * bounds.Width
		y = bounds.Height + enemySpawnOffset
	case 3: // Left
		x = -enemySpawnOffset
		y = rand.Float32() * bounds.Height
	}

	return x, y
}

// AutoSpawnWave is a convenience function to spawn enemies periodically.
// Used by the host in the main game loop.
func AutoSpawnWave(em *entity.EntityManager, players []entity.PlayerState) {
	// This is handled by SpawnSystem.Update() now
	// Kept for backward compatibility
}
