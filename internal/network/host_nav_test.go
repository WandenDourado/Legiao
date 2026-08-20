package network

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/nav"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type toggleGate struct {
	x, y, width, height float32
	on                  bool
}

func (g *toggleGate) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	if !g.on {
		return false
	}
	return pos.X+width/2 > g.x && pos.X-width/2 < g.x+g.width &&
		pos.Y+height/2 > g.y && pos.Y-height/2 < g.y+g.height
}

// TestRebuildNavAreaFollowsTheArenaGate proves the arena gate's toggle
// (game/arena_gate.go) actually reaches the mesh: a cell inside the gate's
// footprint must flip walkability when RebuildNavArea is told about it,
// exactly like the gate's own collision does.
func TestRebuildNavAreaFollowsTheArenaGate(t *testing.T) {
	gate := &toggleGate{x: 400, y: 0, width: 40, height: 1000, on: true}
	bounds := world.Bounds{Width: 1000, Height: 1000}
	grid := nav.Build(gate, bounds, nav.CellSize, nav.AgentBox)
	h := &Host{EntityManager: &entity.EntityManager{Solid: gate, Nav: grid}}

	mid := rl.NewVector2(420, 500)
	if grid.Walkable(mid) {
		t.Fatal("expected the gate cell blocked while the gate is on")
	}

	gate.on = false
	h.RebuildNavArea(rl.NewRectangle(gate.x, gate.y, gate.width, gate.height))
	if !grid.Walkable(mid) {
		t.Fatal("expected RebuildNavArea to open the gate cell once the gate turned off")
	}

	gate.on = true
	h.RebuildNavArea(rl.NewRectangle(gate.x, gate.y, gate.width, gate.height))
	if grid.Walkable(mid) {
		t.Fatal("expected RebuildNavArea to re-close the gate cell")
	}
}

// TestRebuildNavAreaIsANoOpBeforeTheMeshExists guards the nil path: called
// before a map (and so a mesh) has ever loaded, this must not panic.
func TestRebuildNavAreaIsANoOpBeforeTheMeshExists(t *testing.T) {
	h := &Host{EntityManager: entity.NewEntityManager()}
	h.RebuildNavArea(rl.NewRectangle(0, 0, 10, 10))
}
