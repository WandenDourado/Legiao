package entity

// The Fase 4 acceptance tests from doc/plan_navegacao_bots_monstros.md §8:
// a monster behind an L-shaped barricade reaches the target within a bounded
// simulated time, and a monster with a clear line of sight never bothers the
// mesh at all — pure simulation, no window, same shape as
// enemy_collision_test.go / orc_legion_test.go.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/nav"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestEnemyBehindAnLShapedBarricadeReachesTheTarget(t *testing.T) {
	// Two segments meeting at a corner: the enemy starts in the pocket they
	// form, the target is outside it. A local slide/detour alone tends to
	// hover at the inside corner; the mesh has to route it all the way
	// around the outside corner instead.
	walls := testWalls{
		{x: 300, y: 0, width: 40, height: 340},   // vertical arm
		{x: 300, y: 300, width: 340, height: 40}, // horizontal arm
	}
	bounds := world.Bounds{Width: 2000, Height: 2000}
	grid := nav.Build(walls, bounds, nav.CellSize, nav.AgentBox)

	e := NewEnemy(EnemyTypeBasic, 200, 200) // inside the L's pocket
	target := rl.NewVector2(800, 500)       // outside it
	env := MoveEnv{Solid: walls, Nav: grid}

	reached := false
	const dt = 1.0 / 60.0
	maxSeconds := 30.0
	for frame := 0; float64(frame)*dt < maxSeconds; frame++ {
		grid.ResetFrameBudget()
		e.MoveTowardTarget(target, dt, env)
		if rl.Vector2Distance(e.Position, target) < 40 {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("enemy never reached the target from behind the L-shaped barricade; ended at %v", e.Position)
	}
}

// testWalls combines several rectangles into one collision.Solid, so an
// L-shaped (or gapped) obstacle can be built from two of them. Mirrors
// internal/nav's own test helper of the same name.
type testWalls []testWall

func (ws testWalls) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	for _, w := range ws {
		if w.CollidesCentered(pos, width, height) {
			return true
		}
	}
	return false
}

// TestPackGetsThroughABarricadeGapWithinBudget is the "matilha nao
// engarrafa" acceptance check (plan §4/§7 fase 4): twenty monsters
// converging on the same gap in a barricade, sharing PathBudgetPerFrame,
// must all make it through in bounded simulated time — no monster left
// permanently stuck against the face, no crash, no infinite loop.
func TestPackGetsThroughABarricadeGapWithinBudget(t *testing.T) {
	// Bounds and every spawn/target point below stay within [0, bounds] on
	// purpose: the mesh only covers that rectangle (nav.Build), and a point
	// outside it is simply unwalkable — an easy trap to fall into with
	// negative test coordinates, and not a real one, since a monster's
	// position is always inside the map it was built from.
	walls := testWalls{
		{x: 1000, y: 0, width: 40, height: 700},   // covers y: 0..700
		{x: 1000, y: 740, width: 40, height: 760}, // covers y: 740..1500
	}
	bounds := world.Bounds{Width: 2000, Height: 1500}
	grid := nav.Build(walls, bounds, nav.CellSize, nav.AgentBox)
	env := MoveEnv{Solid: walls, Nav: grid}

	// Spread on a loose grid (80px, comfortably more than the slime's ~40px
	// movement footprint) — packed shoulder to shoulder, mutual separation
	// dominates the steering blend and drives the pack somewhere the mesh
	// never sent it, which is a test-realism problem, not a navigation one:
	// a real horde spawns with room to stand in.
	const packSize = 20
	pack := make([]*Enemy, packSize)
	for i := range pack {
		col, row := i%5, i/5
		pack[i] = NewEnemy(EnemyTypeBasic, float32(100+col*80), float32(100+row*80))
	}
	// The gap (y: 700..740) is 360-620px from every spawn — well beyond
	// collision.ResolveDetour's own ~96-step face scan, so getting through
	// requires the mesh, not just local sliding.
	target := rl.NewVector2(1900, 720)
	players := []PlayerState{{PlayerID: "target", X: int(target.X), Y: int(target.Y)}}

	const dt = 1.0 / 60.0
	maxSeconds := 60.0
	for frame := 0; float64(frame)*dt < maxSeconds; frame++ {
		grid.ResetFrameBudget()
		// Update, not raw MoveTowardTarget: this is what actually steers
		// with proper separation from the rest of the pack while moving,
		// not just a discrete post-pass. Twenty agents fighting for the same
		// gap need the real steering to avoid shoving each other off route.
		for _, e := range pack {
			e.Update(dt, players, pack, env)
		}
		ResolveEnemyOverlap(pack, walls)
	}

	for i, e := range pack {
		if e.Position.X <= walls[0].x+walls[0].width {
			t.Errorf("enemy %d never made it through the gap; ended at %v", i, e.Position)
		}
	}
}

func TestEnemyInOpenFieldNeverConsultsTheMesh(t *testing.T) {
	bounds := world.Bounds{Width: 2000, Height: 2000}
	grid := nav.Build(nil, bounds, nav.CellSize, nav.AgentBox)

	e := NewEnemy(EnemyTypeBasic, 0, 0)
	target := rl.NewVector2(1000, 0)
	env := MoveEnv{Nav: grid}

	const dt = 1.0 / 60.0
	for frame := 0; frame < 300; frame++ {
		grid.ResetFrameBudget()
		e.MoveTowardTarget(target, dt, env)
		if e.follower.Active() {
			t.Fatalf("frame %d: enemy consulted the navigation mesh with a clear line of sight to the target", frame)
		}
	}
}
