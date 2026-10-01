// Package nav is the route layer between deciding WHERE to go (internal/bot
// and the monster AI) and what the map lets a single step DO
// (internal/collision). See doc/plan_navegacao_bots_monstros.md.
//
// Pure: it depends only on internal/collision (for the Solid it is built
// from), internal/world (for map bounds) and raylib's rl.Vector2. It must
// never import internal/network, internal/game or internal/entity — if a
// change here needs any of those, the design has left the rails.
package nav

import rl "github.com/gen2brain/raylib-go/raylib"

// Grid is a walkability mesh derived once from the map's collision: a cell
// is free when an AgentBox-sized box centered on it does not collide.
type Grid struct {
	cell    float32
	agent   float32
	originX float32
	originY float32
	w, h    int
	free    []bool

	// Reused A* scratch space (astar.go) and the per-frame search budget
	// (budget.go). Neither is safe for concurrent use — the simulation
	// that drives bots and monsters is single-goroutine, and Grid assumes it.
	scratch           astarScratch
	searchesThisFrame int
}

func (g *Grid) index(cx, cy int) int { return cy*g.w + cx }

func (g *Grid) cellOf(p rl.Vector2) (int, int) {
	cx := int((p.X - g.originX) / g.cell)
	cy := int((p.Y - g.originY) / g.cell)
	return cx, cy
}

func (g *Grid) inBounds(cx, cy int) bool {
	return cx >= 0 && cy >= 0 && cx < g.w && cy < g.h
}

func (g *Grid) cellCenter(cx, cy int) rl.Vector2 {
	return rl.NewVector2(
		g.originX+(float32(cx)+0.5)*g.cell,
		g.originY+(float32(cy)+0.5)*g.cell,
	)
}

// walkableCell returns the cell index p falls in, and whether that cell is
// both in bounds and free.
func (g *Grid) walkableCell(p rl.Vector2) (int, bool) {
	cx, cy := g.cellOf(p)
	if !g.inBounds(cx, cy) {
		return 0, false
	}
	idx := g.index(cx, cy)
	return idx, g.free[idx]
}

// Walkable reports whether p falls inside a free cell.
func (g *Grid) Walkable(p rl.Vector2) bool {
	_, ok := g.walkableCell(p)
	return ok
}

// NearestWalkable returns the free cell center closest to p within
// maxRadius, searching outward ring by ring. An agent can find itself
// standing inside what the mesh calls solid — spawned there, or shoved in
// by a crowd — and refusing to plan from or to that point would leave it
// with no route at all instead of the nearest way out.
func (g *Grid) NearestWalkable(p rl.Vector2, maxRadius float32) (rl.Vector2, bool) {
	if g.Walkable(p) {
		return p, true
	}
	cx, cy := g.cellOf(p)
	maxRing := int(maxRadius/g.cell) + 1

	var best rl.Vector2
	bestDist := float32(-1)
	for r := 1; r <= maxRing; r++ {
		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				if absInt(dx) != r && absInt(dy) != r {
					continue // ring only: interior cells were already checked at a smaller r
				}
				nx, ny := cx+dx, cy+dy
				if !g.inBounds(nx, ny) || !g.free[g.index(nx, ny)] {
					continue
				}
				c := g.cellCenter(nx, ny)
				d := rl.Vector2Distance(p, c)
				if d > maxRadius {
					continue
				}
				if bestDist < 0 || d < bestDist {
					best, bestDist = c, d
				}
			}
		}
		if bestDist >= 0 {
			return best, true
		}
	}
	return rl.Vector2{}, false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
