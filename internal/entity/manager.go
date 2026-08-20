package entity

import (
	"sort"
	"sync"

	"github.com/WandenDourado/Legiao/internal/collision"
	"github.com/WandenDourado/Legiao/internal/nav"
	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// EntityManager manages collections of enemies and projectiles in the game.
// It provides thread-safe operations for adding, removing, and updating entities.
type EntityManager struct {
	Enemies      map[string]*Enemy
	Projectiles  map[string]*Projectile
	enemiesMutex sync.RWMutex
	projMutex    sync.RWMutex
	WorldBounds  world.Bounds
	// Solid is the loaded map's blocked space. Enemy movement is resolved
	// against it with the same rule the player uses, so monsters cannot walk
	// through trees, fences or houses. Nil until a map is applied, which means
	// "no obstacles" rather than "nothing moves".
	Solid collision.Solid
	// Nav is the walkability mesh derived from Solid (internal/nav), built
	// once per map load — a sibling of Solid rather than a replacement for
	// it: Solid still resolves each individual step, Nav decides which way
	// to route around an obstacle the step alone cannot solve. Nil until a
	// map is applied, same as Solid.
	Nav *nav.Grid
	// drawBuf e o slice que DrawAll reaproveita entre quadros para montar a
	// lista de inimigos visiveis. Ele existe para nao alocar 83 ponteiros por
	// quadro no world_03; nunca e lido fora do passe de desenho.
	drawBuf []*Enemy
}

// NewEntityManager creates a new entity manager with initialized maps.
// Skill collections (fireballs, sanctuaries, ...) live in the dedicated
// `skill` package instead, so this manager only owns enemies/projectiles.
func NewEntityManager() *EntityManager {
	return &EntityManager{
		Enemies:     make(map[string]*Enemy),
		Projectiles: make(map[string]*Projectile),
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

	// Update enemies. The active set is collected first so each enemy can
	// steer away from the others, then a positional pass removes whatever
	// overlap the steering could not prevent.
	em.enemiesMutex.Lock()
	active := make([]*Enemy, 0, len(em.Enemies))
	for _, e := range em.Enemies {
		if e.IsActive {
			active = append(active, e)
		}
	}
	// Map iteration order is random, and both the steering and the overlap
	// pass read positions that earlier entries have already changed. Sorting
	// keeps the host simulation deterministic frame to frame instead of
	// jittering with whatever order the map happened to yield.
	sort.Slice(active, func(i, j int) bool { return active[i].ID < active[j].ID })

	env := MoveEnv{Solid: em.Solid, Nav: em.Nav}
	for _, e := range active {
		if e.Update(dt, players, active, env) {
			attackedEnemies[e.ID] = true
		}
	}
	ResolveEnemyOverlap(active, em.Solid)
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
// DrawAll desenha o que a camera mostra. view e a janela visivel em unidades
// de mundo; uma janela de tamanho zero desliga o culling.
//
// Tres decisoes aqui, e todas vieram do world_03, onde a guarnicao poe 83
// monstros em campo desde o carregamento do mapa:
//
//  1. Quem esta fora da tela nao e desenhado. Antes os 83 eram, sempre.
//  2. Todos os SPRITES primeiro, todas as BARRAS depois. Intercalados, a barra
//     (que e retangulo) quebrava o batch de textura entre um monstro e o
//     proximo — dois flushes por inimigo em vez de um punhado por quadro.
//  3. A ordem e por TIPO, e nao a do map. A ordem de um map em Go e aleatoria
//     a cada iteracao, entao a textura trocava a cada inimigo E dois monstros
//     sobrepostos piscavam entre si a cada quadro, porque quem ficava na
//     frente mudava. Agrupar por tipo troca de textura uma vez por tipo e
//     torna a sobreposicao estavel. (Ordenar por Y daria profundidade de
//     verdade, mas e outra decisao: brigaria com o agrupamento por textura.)
func (em *EntityManager) DrawAll(view rl.Rectangle) {
	em.enemiesMutex.RLock()
	alive := 0
	for _, e := range em.Enemies {
		if e != nil && e.IsActive {
			alive++
		}
	}
	em.drawBuf = visibleEnemies(em.Enemies, view, em.drawBuf[:0])
	for _, e := range em.drawBuf {
		e.Draw()
	}
	for _, e := range em.drawBuf {
		e.DrawHealthBar()
	}
	drawn := len(em.drawBuf)
	em.enemiesMutex.RUnlock()

	// Draw projectiles
	projectiles := 0
	em.projMutex.RLock()
	for _, p := range em.Projectiles {
		if p.IsActive {
			p.Draw()
			projectiles++
		}
	}
	em.projMutex.RUnlock()

	RecordEnemyDraw(EnemyDrawCounts{Alive: alive, Drawn: drawn, Projectiles: projectiles})
}

// visibleEnemies filtra os ativos que aparecem na tela e os ordena por tipo,
// depois por ID. O ID desempata para a ordem nao depender da iteracao do map,
// que e aleatoria — mesma razao de UpdateAll ordenar antes de simular.
//
// Escreve no buffer recebido para nao alocar um slice por quadro.
func visibleEnemies(enemies map[string]*Enemy, view rl.Rectangle, buf []*Enemy) []*Enemy {
	for _, e := range enemies {
		if e == nil || !e.IsActive {
			continue
		}
		if !EnemyVisible(enemyDrawBoxOf(e), view) {
			continue
		}
		buf = append(buf, e)
	}
	sort.Slice(buf, func(i, j int) bool {
		if buf[i].Type != buf[j].Type {
			return buf[i].Type < buf[j].Type
		}
		return buf[i].ID < buf[j].ID
	})
	return buf
}

// Clear removes all entities (for game reset).
func (em *EntityManager) Clear() {
	em.enemiesMutex.Lock()
	em.Enemies = make(map[string]*Enemy)
	em.enemiesMutex.Unlock()

	em.projMutex.Lock()
	em.Projectiles = make(map[string]*Projectile)
	em.projMutex.Unlock()
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
