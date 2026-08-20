package entity

import (
	"github.com/WandenDourado/Legiao/internal/collision"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// enemyFootprintScale converts an enemy's combat radius into its movement
// footprint. Radius covers the whole visible silhouette so that attacks
// connect where the art is; walking on that box would make a slime (90 px
// wide) barely fit through the 128 px gap a player (40 px box) strolls
// through. 0.45 puts every current enemy in the same 40-45 px band as the
// player, so monsters navigate the same openings the player does.
const enemyFootprintScale = 0.45

// EnemyFootprint is the single definition of an enemy's collision box, shared
// by movement resolution and the F3 debug overlay so the two can never
// disagree about what is being tested.
func EnemyFootprint(e *Enemy) (width, height float32) {
	size := e.Radius * 2 * enemyFootprintScale
	return size, size
}

// step displaces the enemy by delta against the map, sliding along obstacles
// and stepping around the ones it cannot slide past. The chosen way around is
// remembered on the enemy so it commits to it instead of jittering between
// left and right while the obstacle stays in front of it.
//
// Returns the displacement actually applied, which is not delta whenever the
// map got in the way.
func (e *Enemy) step(delta rl.Vector2, solid collision.Solid) rl.Vector2 {
	from := e.Position
	width, height := EnemyFootprint(e)
	next, dir := collision.ResolveDetour(from, delta, width, height, solid, e.detourDir)
	e.detourDir = dir
	e.Position = next
	return rl.Vector2Subtract(next, from)
}

// push applies a positional correction (crowd separation) through the same
// resolver, so enemies unstacking from a pile cannot shove each other into a
// tree. Velocity is left untouched: the correction must not turn the sprite.
func (e *Enemy) push(delta rl.Vector2, solid collision.Solid) {
	width, height := EnemyFootprint(e)
	e.Position = collision.Resolve(e.Position, delta, width, height, solid)
}
