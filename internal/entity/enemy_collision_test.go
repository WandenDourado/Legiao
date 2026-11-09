package entity

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// testWall is a one-rectangle Solid: enough to prove enemies obey map
// obstacles without loading a Tiled map.
type testWall struct {
	x, y, width, height float32
}

func (w testWall) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	return pos.X+width/2 > w.x && pos.X-width/2 < w.x+w.width &&
		pos.Y+height/2 > w.y && pos.Y-height/2 < w.y+w.height
}

const testFrame = 1.0 / 60.0

func TestEnemyNeverEntersASolid(t *testing.T) {
	// A wall too tall to walk around: the enemy must press against it, never
	// through it, however long it chases.
	wall := testWall{x: 100, y: -4000, width: 100, height: 8000}
	e := NewEnemy(EnemyTypeBasic, 0, 0)
	target := rl.NewVector2(500, 0)
	width, height := EnemyFootprint(e)

	for frame := 0; frame < 600; frame++ {
		e.MoveTowardTarget(target, testFrame, MoveEnv{Solid: wall})
		if wall.CollidesCentered(e.Position, width, height) {
			t.Fatalf("frame %d: enemy walked into the wall at %v", frame, e.Position)
		}
	}
}

func TestEnemyWalksAroundABlockingObstacle(t *testing.T) {
	// A finite obstacle (a tree, a fence segment) between enemy and player.
	// Sliding alone would leave the enemy grinding against it; the detour has
	// to take it past the far edge.
	wall := testWall{x: 100, y: -100, width: 100, height: 200}
	e := NewEnemy(EnemyTypeBasic, 0, 0)
	target := rl.NewVector2(500, 0)

	for frame := 0; frame < 900; frame++ {
		e.MoveTowardTarget(target, testFrame, MoveEnv{Solid: wall})
	}
	if e.Position.X <= wall.x+wall.width {
		t.Fatalf("enemy stuck before the obstacle at %v", e.Position)
	}
}

func TestEnemyMovesFreelyWithoutAMap(t *testing.T) {
	// A nil Solid means "no map loaded", not "nothing moves".
	e := NewEnemy(EnemyTypeBasic, 0, 0)
	e.MoveTowardTarget(rl.NewVector2(500, 0), testFrame, MoveEnv{})
	if e.Position.X <= 0 {
		t.Fatalf("enemy did not move without a map: %v", e.Position)
	}
}

func TestEnemyFootprintIsComparableToThePlayers(t *testing.T) {
	// The movement box is derived from the combat radius, not equal to it:
	// walking on the full silhouette would keep monsters out of gaps the
	// player (a 40 px box) walks through.
	for _, enemyType := range []EnemyType{EnemyTypeBasic, EnemyTypeFast, EnemyTypeGarrison} {
		e := NewEnemy(enemyType, 0, 0)
		width, height := EnemyFootprint(e)
		if width != height {
			t.Fatalf("%s: footprint must be square, got %vx%v", enemyType, width, height)
		}
		if width < 30 || width > 60 {
			t.Fatalf("%s: footprint %v is outside the player's 40 px band", enemyType, width)
		}
	}
}
