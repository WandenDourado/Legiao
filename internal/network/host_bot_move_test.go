package network

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// testWall is a one-rectangle collision.Solid — the same minimal stub
// internal/entity/enemy_collision_test.go uses, so a bot can be proven to
// obey and get around an obstacle without loading a Tiled map.
type testWall struct {
	x, y, width, height float32
}

func (w testWall) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	return pos.X+width/2 > w.x && pos.X-width/2 < w.x+w.width &&
		pos.Y+height/2 > w.y && pos.Y-height/2 < w.y+w.height
}

const botTestFrame = 1.0 / 60.0

// TestResolveBotMoveWalksAroundAnObstacle proves the fix for the bot pushing
// a tree on the way to the portal: a bot walking straight at a finite
// obstacle (the same shape internal/entity/enemy_collision_test.go proves
// enemies already get around) must reach the far side instead of grinding
// against the face forever (plan §1, "remendo imediato").
func TestResolveBotMoveWalksAroundAnObstacle(t *testing.T) {
	wall := testWall{x: 100, y: -100, width: 100, height: 200}
	rt := &botRuntime{character: entity.CharMago}
	pos := rl.NewVector2(0, 0)
	target := rl.NewVector2(500, 0)

	for frame := 0; frame < 900; frame++ {
		dir := rl.Vector2Normalize(rl.Vector2Subtract(target, pos))
		delta := rl.Vector2Scale(dir, botSpeed*botTestFrame)
		pos = resolveBotMove(rt, pos, delta, rt.character, wall, nil)

		box, w, h := entity.GroundBoxAt(pos, rt.character)
		if wall.CollidesCentered(box, w, h) {
			t.Fatalf("frame %d: bot walked into the wall at %v", frame, pos)
		}
	}
	if pos.X <= wall.x+wall.width {
		t.Fatalf("bot stuck before the obstacle at %v — grinding against the face instead of detouring", pos)
	}
}

// TestResolveBotMoveClearsTheCommitmentOnceThePathIsOpen checks the other
// half of ResolveDetour's contract: once nothing blocks the direct path,
// rt.detourDir must go back to zero, or the bot would keep sidestepping
// long after the obstacle is behind it.
func TestResolveBotMoveClearsTheCommitmentOnceThePathIsOpen(t *testing.T) {
	rt := &botRuntime{character: entity.CharMago}
	pos := rl.NewVector2(0, 0)
	delta := rl.NewVector2(botSpeed*botTestFrame, 0)

	resolveBotMove(rt, pos, delta, rt.character, nil, nil)

	if rt.detourDir.X != 0 || rt.detourDir.Y != 0 {
		t.Fatalf("expected no committed detour with nothing in the way, got %+v", rt.detourDir)
	}
}
