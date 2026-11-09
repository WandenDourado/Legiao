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

// Knockback empurra o inimigo para LONGE de `from`, e devolve se ele saiu do
// lugar. Hoje so o estouro do Escudo Sagrado usa (network/host_shield_burst.go).
//
// Duas regras herdadas de ResolveEnemyOverlap, e nao inventadas aqui:
//
//   - QUEM NAO ANDA NAO E EMPURRADO. Uma gargula sentinela e a Senhora das
//     Trevas tem `Speed 0` porque sao POSTOS: o mapa decidiu onde elas ficam.
//     Arrastar uma delas com uma explosao seria o mesmo defeito que a horda
//     empurrando a chefe para fora da ancora, que ja custou caro uma vez.
//   - O empurrao passa pelo resolvedor de mapa (`push`), entao ninguem
//     atravessa parede: um inimigo encurralado contra a pedra simplesmente
//     nao anda, em vez de ser arremessado para dentro dela.
//
// A velocidade fica intacta de proposito, tambem como em `push`: o corpo
// desliza, o sprite nao gira. O bicho e jogado para tras sem parar de encarar
// quem o jogou.
func (e *Enemy) Knockback(from rl.Vector2, distance float32, solid collision.Solid) bool {
	if e.Speed <= 0 || distance <= 0 {
		return false
	}
	delta := rl.Vector2Subtract(e.Position, from)
	if delta.X == 0 && delta.Y == 0 {
		// Em cima do centro exato: sem direcao para fugir. Empurra para baixo,
		// na direcao da camera, que e a unica escolha que nao esconde o bicho
		// atras do proprio jogador.
		delta = rl.NewVector2(0, 1)
	}
	before := e.Position
	e.push(rl.Vector2Scale(rl.Vector2Normalize(delta), distance), solid)
	return e.Position != before
}
