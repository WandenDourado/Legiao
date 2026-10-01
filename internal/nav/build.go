package nav

import (
	"github.com/WandenDourado/Legiao/internal/collision"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Build derives a walkability mesh from s, covering bounds at cell
// resolution, testing an agent x agent box centered on each cell. Zero
// cell/agent fall back to CellSize/AgentBox — the values every real caller
// should pass unless a test needs something coarser to stay fast.
//
// This is the only place cost matters: it runs once per map load, not once
// per frame (see RebuildArea for the in-game case).
func Build(s collision.Solid, bounds world.Bounds, cell, agent float32) *Grid {
	if cell <= 0 {
		cell = CellSize
	}
	if agent <= 0 {
		agent = AgentBox
	}
	w := int(bounds.Width/cell) + 1
	h := int(bounds.Height/cell) + 1
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	g := &Grid{cell: cell, agent: agent, w: w, h: h, free: make([]bool, w*h)}
	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			g.free[g.index(cx, cy)] = !collidesAgent(s, g.cellCenter(cx, cy), agent)
		}
	}
	return g
}

// RebuildArea recomputes walkability only for the cells overlapping area.
// The arena gate (internal/game/arena_gate.go) changes the map's collision
// IN GAME by toggling a footprint region; without this the mesh would keep
// insisting the gate is however it was at map load, and the whole pack
// would either walk a route that no longer exists or believe none does.
func (g *Grid) RebuildArea(s collision.Solid, area rl.Rectangle) {
	minCX, minCY := g.cellOf(rl.NewVector2(area.X, area.Y))
	maxCX, maxCY := g.cellOf(rl.NewVector2(area.X+area.Width, area.Y+area.Height))
	for cy := minCY; cy <= maxCY; cy++ {
		for cx := minCX; cx <= maxCX; cx++ {
			if !g.inBounds(cx, cy) {
				continue
			}
			g.free[g.index(cx, cy)] = !collidesAgent(s, g.cellCenter(cx, cy), g.agent)
		}
	}
}

func collidesAgent(s collision.Solid, pos rl.Vector2, agent float32) bool {
	if s == nil {
		return false
	}
	return s.CollidesCentered(pos, agent, agent)
}
