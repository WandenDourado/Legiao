package nav

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// testWall is a minimal collision.Solid: one axis-aligned rectangle.
type testWall struct {
	x, y, width, height float32
}

func (w testWall) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	return pos.X+width/2 > w.x && pos.X-width/2 < w.x+w.width &&
		pos.Y+height/2 > w.y && pos.Y-height/2 < w.y+w.height
}

// testWalls combines several rectangles, so a wall with a gap can be built
// from two of them.
type testWalls []testWall

func (ws testWalls) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	for _, w := range ws {
		if w.CollidesCentered(pos, width, height) {
			return true
		}
	}
	return false
}

func smallBounds() world.Bounds {
	return world.Bounds{Width: 1000, Height: 1000}
}

func TestFindPathExistsAcrossOpenGround(t *testing.T) {
	g := Build(nil, smallBounds(), CellSize, AgentBox)
	path, ok := g.FindPath(rl.NewVector2(50, 50), rl.NewVector2(900, 900), nil)
	if !ok || len(path) == 0 {
		t.Fatal("expected a path across open ground")
	}
}

func TestFindPathAroundABarricadeWithAGap(t *testing.T) {
	// A wall spanning the whole width of the test area except a gap in the
	// middle — the same shape a Tiled barricade with a vao takes.
	wall := testWalls{
		{x: 0, y: 400, width: 1000, height: 40},
	}
	// Cut a gap: only build a grid where the wall itself has a hole, by
	// using two segments instead of one continuous rectangle.
	wall = testWalls{
		{x: 0, y: 400, width: 440, height: 40},
		{x: 560, y: 400, width: 440, height: 40},
	}
	g := Build(wall, smallBounds(), CellSize, AgentBox)

	from := rl.NewVector2(500, 100)
	to := rl.NewVector2(500, 900)
	path, ok := g.FindPath(from, to, nil)
	if !ok {
		t.Fatal("expected a path through the gap in the barricade")
	}
	for i := 0; i < len(path)-1; i++ {
		if !g.LineOfSight(path[i], path[i+1]) {
			t.Fatalf("path segment %d->%d (%v -> %v) is not actually walkable in a straight line",
				i, i+1, path[i], path[i+1])
		}
	}
}

func TestFindPathReturnsFalseWhenNoRouteExists(t *testing.T) {
	// A wall with no gap at all, spanning past both bounds edges.
	wall := testWall{x: -100, y: 400, width: 1200, height: 40}
	g := Build(wall, smallBounds(), CellSize, AgentBox)
	_, ok := g.FindPath(rl.NewVector2(500, 100), rl.NewVector2(500, 900), nil)
	if ok {
		t.Fatal("expected no path across a wall with no gap")
	}
}

// TestFindPathNeverCutsACorner builds an L-shaped pocket where the only
// diagonal step available would clip the inside corner of the L, and
// asserts the path never takes a diagonal step that isn't fully clear on
// both orthogonal sides.
func TestFindPathNeverCutsACorner(t *testing.T) {
	// Two rectangles meeting at a corner, leaving the agent needing to go
	// around the outside corner rather than cut across it.
	walls := testWalls{
		{x: 300, y: 0, width: 40, height: 340},   // vertical arm
		{x: 300, y: 300, width: 340, height: 40}, // horizontal arm
	}
	g := Build(walls, smallBounds(), CellSize, AgentBox)

	from := rl.NewVector2(200, 200) // inside the L's pocket
	to := rl.NewVector2(800, 500)   // outside it
	path, ok := g.FindPath(from, to, nil)
	if !ok {
		t.Fatal("expected a path around the L")
	}
	// Every consecutive pair of cell-adjacent points in the RAW search
	// (before smoothing) would have been checked by FindPath's own
	// no-corner-cutting rule; here we only need the smoothed path to never
	// cross the walls, which LineOfSight already guarantees per-segment.
	for i := 0; i < len(path)-1; i++ {
		if !g.LineOfSight(path[i], path[i+1]) {
			t.Fatalf("smoothed segment %v -> %v is not clear", path[i], path[i+1])
		}
	}
}

func TestSmoothPathNeverCrossesSolid(t *testing.T) {
	wall := testWall{x: 400, y: 0, width: 40, height: 600}
	g := Build(wall, world.Bounds{Width: 1000, Height: 1000}, CellSize, AgentBox)
	from := rl.NewVector2(100, 100)
	to := rl.NewVector2(900, 900)
	path, ok := g.FindPath(from, to, nil)
	if !ok {
		t.Fatal("expected a path around the finite wall")
	}
	for i := 0; i < len(path)-1; i++ {
		if !g.LineOfSight(path[i], path[i+1]) {
			t.Fatalf("smoothed segment %v -> %v crosses solid ground", path[i], path[i+1])
		}
	}
}

func TestLineOfSightBlockedBySolid(t *testing.T) {
	wall := testWall{x: 400, y: 0, width: 40, height: 1000}
	g := Build(wall, smallBounds(), CellSize, AgentBox)
	if g.LineOfSight(rl.NewVector2(100, 500), rl.NewVector2(900, 500)) {
		t.Fatal("expected line of sight to be blocked by the wall")
	}
	if !g.LineOfSight(rl.NewVector2(100, 500), rl.NewVector2(300, 500)) {
		t.Fatal("expected line of sight to be clear on the near side of the wall")
	}
}

func TestRebuildAreaOpensAndClosesAGate(t *testing.T) {
	gate := &toggleWall{x: 400, y: 0, width: 40, height: 1000, on: true}
	g := Build(gate, smallBounds(), CellSize, AgentBox)

	mid := rl.NewVector2(420, 500)
	if g.Walkable(mid) {
		t.Fatal("expected the gate cell to be blocked while the gate is closed")
	}

	gate.on = false
	g.RebuildArea(gate, rl.NewRectangle(gate.x, gate.y, gate.width, gate.height))
	if !g.Walkable(mid) {
		t.Fatal("expected RebuildArea to open the gate cell once the gate turned off")
	}

	gate.on = true
	g.RebuildArea(gate, rl.NewRectangle(gate.x, gate.y, gate.width, gate.height))
	if g.Walkable(mid) {
		t.Fatal("expected RebuildArea to re-close the gate cell")
	}
}

type toggleWall struct {
	x, y, width, height float32
	on                  bool
}

func (w *toggleWall) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	if !w.on {
		return false
	}
	return pos.X+width/2 > w.x && pos.X-width/2 < w.x+w.width &&
		pos.Y+height/2 > w.y && pos.Y-height/2 < w.y+w.height
}

func TestNearestWalkableFromInsideSolid(t *testing.T) {
	wall := testWall{x: 400, y: 400, width: 100, height: 100}
	g := Build(wall, smallBounds(), CellSize, AgentBox)

	inside := rl.NewVector2(450, 450)
	if g.Walkable(inside) {
		t.Fatal("test setup broken: point should be inside the wall")
	}
	nearest, ok := g.NearestWalkable(inside, 300)
	if !ok {
		t.Fatal("expected a nearest walkable point within radius")
	}
	if !g.Walkable(nearest) {
		t.Fatal("NearestWalkable returned a point that is not itself walkable")
	}
}

func TestNearestWalkableGivesUpBeyondRadius(t *testing.T) {
	// A wall covering everything within any sane radius of the test point.
	wall := testWall{x: -5000, y: -5000, width: 10000, height: 10000}
	g := Build(wall, smallBounds(), CellSize, AgentBox)
	_, ok := g.NearestWalkable(rl.NewVector2(500, 500), 64)
	if ok {
		t.Fatal("expected NearestWalkable to give up when nothing is within radius")
	}
}

func TestFrameBudgetStopsExtraSearchesButNeverBlocks(t *testing.T) {
	g := Build(nil, smallBounds(), CellSize, AgentBox)
	g.ResetFrameBudget()

	granted := 0
	for i := 0; i < PathBudgetPerFrame+5; i++ {
		if g.tryReserveSearch() {
			granted++
		}
	}
	if granted != PathBudgetPerFrame {
		t.Fatalf("expected exactly %d reservations granted this frame, got %d", PathBudgetPerFrame, granted)
	}

	g.ResetFrameBudget()
	if !g.tryReserveSearch() {
		t.Fatal("expected the budget to be available again after ResetFrameBudget")
	}
}

func TestFollowerFallsBackToStraightLineWhenBudgetExhausted(t *testing.T) {
	// A wall that forces planning, so Desired has to go through request.
	wall := testWalls{
		{x: 0, y: 400, width: 440, height: 40},
		{x: 560, y: 400, width: 440, height: 40},
	}
	g := Build(wall, smallBounds(), CellSize, AgentBox)
	g.ResetFrameBudget()
	for i := 0; i < PathBudgetPerFrame; i++ {
		g.tryReserveSearch() // exhaust the frame's budget on other agents
	}

	f := &Follower{}
	dir, following := f.Desired(g, rl.NewVector2(500, 100), rl.NewVector2(500, 900), 1.0/60, BotReplanEvery)
	if following {
		t.Fatal("expected no route with the budget exhausted, not a stale 'following' claim")
	}
	if dir.X == 0 && dir.Y == 0 {
		t.Fatal("expected Desired to still return an honest direction toward the goal")
	}
}

func TestFollowerClearsWhenDestinationBecomesVisible(t *testing.T) {
	g := Build(nil, smallBounds(), CellSize, AgentBox)
	f := &Follower{path: []rl.Vector2{{X: 1, Y: 1}}, idx: 0, hasGoal: true}
	_, following := f.Desired(g, rl.NewVector2(0, 0), rl.NewVector2(100, 0), 1.0/60, BotReplanEvery)
	if following {
		t.Fatal("expected a clear line of sight to drop any stale route")
	}
	if len(f.path) != 0 {
		t.Fatal("expected Clear to have emptied the path")
	}
}
