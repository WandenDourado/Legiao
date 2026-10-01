package network

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/bot"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/nav"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// testWalls combines several one-rectangle Solids into one, so a wall with a
// gap can be built from two rectangles.
type testWalls []testWall

func (ws testWalls) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	for _, w := range ws {
		if w.CollidesCentered(pos, width, height) {
			return true
		}
	}
	return false
}

// TestDesiredBotMoveUsesTheMeshWhenTheGapIsFarAway proves the reason Fase 3
// exists: ResolveDetour alone only scans ~96 steps along the obstacle's face
// (collision/resolve.go), so a gap far from where the bot first meets the
// wall is invisible to it — the bot would grind along the face without ever
// finding the opening. The navigation mesh has no such limit: it plans the
// whole route.
func TestDesiredBotMoveUsesTheMeshWhenTheGapIsFarAway(t *testing.T) {
	wall := testWalls{
		{x: 1000, y: -2000, width: 40, height: 2740}, // covers y: -2000..740
		{x: 1000, y: 780, width: 40, height: 4000},   // covers y: 780..4780
	}
	bounds := world.Bounds{Width: 2000, Height: 2000}
	grid := nav.Build(wall, bounds, nav.CellSize, nav.AgentBox)

	rt := &botRuntime{character: entity.CharMago}
	pos := rl.NewVector2(0, 0)
	target := rl.NewVector2(2000, 0)

	crossed := false
	for frame := 0; frame < 3000; frame++ {
		grid.ResetFrameBudget()
		intent := bot.Intent{Dest: target, HasDest: true}
		move := desiredBotMove(rt, grid, pos, intent, botTestFrame)
		move = unstickMove(rt, pos, move, botTestFrame)
		if move.X != 0 || move.Y != 0 {
			delta := rl.Vector2Scale(move, botSpeed*botTestFrame)
			pos = resolveBotMove(rt, pos, delta, rt.character, wall, grid)
		}
		box, w, h := entity.GroundBoxAt(pos, rt.character)
		if wall.CollidesCentered(box, w, h) {
			t.Fatalf("frame %d: bot walked into the wall at %v", frame, pos)
		}
		if pos.X > 1040 {
			crossed = true
			break
		}
	}
	if !crossed {
		t.Fatal("bot never made it through the distant gap — the mesh route was not used")
	}
}

// TestDesiredBotMoveHasNoDestHoldsWithOnlyPush verifies the HasDest=false
// path: no destination means the mesh is not consulted at all, and only
// Intent.Push (already-computed separation) drives movement.
func TestDesiredBotMoveHasNoDestHoldsWithOnlyPush(t *testing.T) {
	rt := &botRuntime{character: entity.CharMago}
	intent := bot.Intent{HasDest: false, Push: rl.NewVector2(1, 0)}
	move := desiredBotMove(rt, nil, rl.NewVector2(0, 0), intent, botTestFrame)
	if move.X <= 0 || move.Y != 0 {
		t.Fatalf("expected movement driven purely by Push, got %+v", move)
	}

	intent = bot.Intent{HasDest: false}
	move = desiredBotMove(rt, nil, rl.NewVector2(0, 0), intent, botTestFrame)
	if move.X != 0 || move.Y != 0 {
		t.Fatalf("expected no movement with no destination and no push, got %+v", move)
	}
}
