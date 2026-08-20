package network

import (
	"log"
	"math/rand"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	enemySpawnOffset      float32 = 180
	enemyMinPlayerDist    float32 = 520
	enemySpawnMaxAttempts         = 16
)

// StartWaveRun installs the map's enemy_spawn_* markers and the run that map
// declares in waveRuns. Called once after the map loads; until it is, nothing
// spawns.
//
// It takes the map path and not just the markers because the run belongs to
// the MAP: with the definitions in a package variable, every map loaded played
// the world_01 waves. E, por ser o unico ponto por onde o mapa chega ao host,
// e aqui que `stageMap` e guardado — o ultimo suspiro tambem depende dele.
func (h *Host) StartWaveRun(mapPath string, points []tilemap.SpawnPoint) {
	h.stageMap = mapPath
	defs := WaveDefsFor(mapPath)
	if defs == nil && len(points) > 0 {
		logMissingRun(mapPath, len(points))
	}
	h.Waves = NewWaveRunner(points, defs)
	// Publish the opening break immediately. Without this, a freshly loaded
	// horde map spends one frame looking like Total == 0, which is the shared
	// signal for a quiet map with an already-open portal.
	SetWaveState(h.Waves.State(0))
}

// updateWaves ticks the wave state machine. Enemy spawning happens only here:
// the old fixed 3-second timer that fed enemies forever is gone, because a map
// now has a finite run of waves and then goes quiet.
func (h *Host) updateWaves(dt float32) {
	if h.Waves == nil {
		return
	}
	alive := h.EntityManager.GetActiveEnemyCount()
	h.Waves.Update(dt, alive, h.getAlivePlayerPositions(), h.spawnEnemyAt)
	// A gargula NAO passa por `spawnEnemyAt`: ela nao vem do anel de spawn, ela
	// e ancorada num posto que o mapa declarou. Por isso o pedido e separado —
	// e por isso ele e uma vez por horda, e nao uma reconciliacao por quadro.
	if want := h.Waves.TakeSentryOrder(); want > 0 {
		h.armSentries(h.stageMap, h.stageSentries, want)
	}
	SetWaveState(h.Waves.State(alive))
	h.updateBossState()
	h.updateBoss(dt)
}

// spawnEnemyAt places one enemy of the given type at a world position. A zero
// position means the wave runner had no usable spawn point, in which case we
// fall back to the world-edge behaviour rather than dumping the enemy on the
// map origin.
func (h *Host) spawnEnemyAt(enemyType entity.EnemyType, pos rl.Vector2) {
	if pos.X == 0 && pos.Y == 0 {
		log.Printf("[Host] sem ponto de spawn utilizavel; usando borda do mundo")
		x, y := h.getRandomSpawnPosition()
		pos = rl.NewVector2(x, y)
	}
	pos = h.clampToWorld(pos)

	e := entity.NewEnemy(enemyType, pos.X, pos.Y)
	h.EntityManager.AddEnemy(e)
	log.Printf("[Host] Spawned %s %s at (%.0f, %.0f)", e.Type, e.ID, pos.X, pos.Y)
}

// clampToWorld keeps a position inside the map. It is a no-op when the bounds
// are unset: clamping against a zero-sized world would push everything to the
// origin, which is exactly the bug this guard exists to prevent.
func (h *Host) clampToWorld(pos rl.Vector2) rl.Vector2 {
	if h.WorldBounds.Width <= 0 || h.WorldBounds.Height <= 0 {
		log.Printf("[Host] WorldBounds nao definido; posicao de spawn nao sera limitada")
		return pos
	}
	return rl.NewVector2(
		clampFloat(pos.X, 0, h.WorldBounds.Width),
		clampFloat(pos.Y, 0, h.WorldBounds.Height),
	)
}

func clampFloat(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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
