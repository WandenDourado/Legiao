package network

import (
	"log"
	"math/rand"

	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	enemySpawnOffset      float32 = 180
	enemyMinPlayerDist    float32 = 520
	enemySpawnMaxAttempts         = 16
)

// spawnEnemies spawns enemies away from alive players.
func (h *Host) spawnEnemies() {
	if h.EntityManager.GetActiveEnemyCount() >= 20 {
		return
	}

	count := rand.Intn(3) + 1
	for i := 0; i < count; i++ {
		x, y := h.getRandomSpawnPosition()
		e := entity.NewEnemy(entity.EnemyTypeBasic, x, y)
		h.EntityManager.AddEnemy(e)
		log.Printf("[Host] Spawned enemy %s at (%.0f, %.0f)", e.ID, x, y)
	}
}

// getRandomSpawnPosition returns a world-edge spawn far from alive players.
func (h *Host) getRandomSpawnPosition() (float32, float32) {
	players := h.getAlivePlayerPositions()
	best := rl.Vector2{}
	bestDistSq := float32(-1)
	minDistSq := enemyMinPlayerDist * enemyMinPlayerDist

	for i := 0; i < enemySpawnMaxAttempts; i++ {
		candidate := h.randomEdgeSpawnCandidate()
		distSq := nearestPlayerDistanceSq(candidate, players)
		if distSq >= minDistSq {
			return candidate.X, candidate.Y
		}
		if distSq > bestDistSq {
			best = candidate
			bestDistSq = distSq
		}
	}

	return best.X, best.Y
}

func (h *Host) randomEdgeSpawnCandidate() rl.Vector2 {
	w := h.WorldBounds.Width
	hh := h.WorldBounds.Height

	switch rand.Intn(4) {
	case 0:
		return rl.NewVector2(rand.Float32()*w, -enemySpawnOffset)
	case 1:
		return rl.NewVector2(w+enemySpawnOffset, rand.Float32()*hh)
	case 2:
		return rl.NewVector2(rand.Float32()*w, hh+enemySpawnOffset)
	default:
		return rl.NewVector2(-enemySpawnOffset, rand.Float32()*hh)
	}
}

func (h *Host) getAlivePlayerPositions() []rl.Vector2 {
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()

	players := make([]rl.Vector2, 0, len(h.players))
	for _, p := range h.players {
		if !p.IsDead {
			players = append(players, rl.NewVector2(float32(p.X), float32(p.Y)))
		}
	}
	return players
}

func nearestPlayerDistanceSq(candidate rl.Vector2, players []rl.Vector2) float32 {
	if len(players) == 0 {
		return enemyMinPlayerDist * enemyMinPlayerDist
	}

	best := float32(-1)
	for _, player := range players {
		distSq := rl.Vector2DistanceSqr(candidate, player)
		if best < 0 || distSq < best {
			best = distSq
		}
	}
	return best
}
