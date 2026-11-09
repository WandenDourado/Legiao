package nav

import rl "github.com/gen2brain/raylib-go/raylib"

// sqrt2 is the octile-distance diagonal step cost.
const sqrt2 = 1.4142135

// astarScratch is A*'s working memory, reused across searches instead of
// allocated fresh every time. Cells are marked "touched this search" with a
// generation stamp rather than cleared — zeroing tens of thousands of
// entries would cost more than the search itself on the bigger maps
// (world_05 is ~92k cells).
type astarScratch struct {
	gen      uint32
	seen     []uint32
	gScore   []float32
	cameFrom []int32
	open     []heapEntry
	seq      uint32
}

type heapEntry struct {
	cell int
	f    float32
	seq  uint32 // stable tie-break: without it, equal-cost paths can make the search order (and so the result) flap between runs
}

func (s *astarScratch) ensure(n int) {
	if len(s.seen) == n {
		return
	}
	s.seen = make([]uint32, n)
	s.gScore = make([]float32, n)
	s.cameFrom = make([]int32, n)
}

func less(a, b heapEntry) bool {
	if a.f != b.f {
		return a.f < b.f
	}
	return a.seq < b.seq
}

func (s *astarScratch) push(cell int, f float32) {
	s.seq++
	s.open = append(s.open, heapEntry{cell: cell, f: f, seq: s.seq})
	i := len(s.open) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if !less(s.open[i], s.open[parent]) {
			break
		}
		s.open[i], s.open[parent] = s.open[parent], s.open[i]
		i = parent
	}
}

func (s *astarScratch) pop() (heapEntry, bool) {
	if len(s.open) == 0 {
		return heapEntry{}, false
	}
	top := s.open[0]
	last := len(s.open) - 1
	s.open[0] = s.open[last]
	s.open = s.open[:last]
	i := 0
	for {
		l, r := 2*i+1, 2*i+2
		smallest := i
		if l < len(s.open) && less(s.open[l], s.open[smallest]) {
			smallest = l
		}
		if r < len(s.open) && less(s.open[r], s.open[smallest]) {
			smallest = r
		}
		if smallest == i {
			break
		}
		s.open[i], s.open[smallest] = s.open[smallest], s.open[i]
		i = smallest
	}
	return top, true
}

var neighborOffsets8 = [8][2]int{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
}

func octile(ax, ay, bx, by int) float32 {
	dx, dy := absInt(ax-bx), absInt(ay-by)
	if dx < dy {
		dx, dy = dy, dx
	}
	return float32(dx) + (sqrt2-1)*float32(dy)
}

// FindPath runs A* from `from` to `to` on the grid (octile distance,
// diagonal steps only when both orthogonal neighbours are free — no corner
// cutting: without that rule an agent clips a solid's corner, the step
// resolver refuses it, and an otherwise-correct route stalls on the very
// first cell), then string-pulls the result (smooth.go) into as few
// waypoints as line of sight allows.
//
// Returns ok=false when either endpoint is not walkable or no route
// connects them. out is reused when it has spare capacity.
func (g *Grid) FindPath(from, to rl.Vector2, out []rl.Vector2) ([]rl.Vector2, bool) {
	fromCell, ok1 := g.walkableCell(from)
	toCell, ok2 := g.walkableCell(to)
	if !ok1 || !ok2 {
		return out[:0], false
	}
	if fromCell == toCell {
		return append(out[:0], to), true
	}

	n := g.w * g.h
	g.scratch.ensure(n)
	g.scratch.gen++
	gen := g.scratch.gen
	g.scratch.open = g.scratch.open[:0]

	fx, fy := fromCell%g.w, fromCell/g.w
	tx, ty := toCell%g.w, toCell/g.w

	g.scratch.seen[fromCell] = gen
	g.scratch.gScore[fromCell] = 0
	g.scratch.cameFrom[fromCell] = -1
	g.scratch.push(fromCell, octile(fx, fy, tx, ty))

	found := false
	for {
		cur, ok := g.scratch.pop()
		if !ok {
			break
		}
		if g.scratch.seen[cur.cell] != gen {
			continue // stale entry from a lazily-deleted decrease-key
		}
		if cur.cell == toCell {
			found = true
			break
		}
		cx, cy := cur.cell%g.w, cur.cell/g.w
		for _, off := range neighborOffsets8 {
			nx, ny := cx+off[0], cy+off[1]
			if !g.inBounds(nx, ny) {
				continue
			}
			nIdx := g.index(nx, ny)
			if !g.free[nIdx] {
				continue
			}
			diag := off[0] != 0 && off[1] != 0
			stepCost := float32(1)
			if diag {
				// No corner cutting: both cells the diagonal would graze
				// must be free too, or the resolver refuses the step this
				// route is counting on.
				if !g.free[g.index(nx, cy)] || !g.free[g.index(cx, ny)] {
					continue
				}
				stepCost = sqrt2
			}
			tentative := g.scratch.gScore[cur.cell] + stepCost
			if g.scratch.seen[nIdx] == gen && tentative >= g.scratch.gScore[nIdx] {
				continue
			}
			g.scratch.seen[nIdx] = gen
			g.scratch.gScore[nIdx] = tentative
			g.scratch.cameFrom[nIdx] = int32(cur.cell)
			g.scratch.push(nIdx, tentative+octile(nx, ny, tx, ty))
		}
	}
	if !found {
		return out[:0], false
	}

	raw := make([]rl.Vector2, 0, 8)
	raw = append(raw, from)
	var cellsRev []int
	for cur := toCell; ; {
		cellsRev = append(cellsRev, cur)
		if cur == fromCell {
			break
		}
		cur = int(g.scratch.cameFrom[cur])
	}
	for i := len(cellsRev) - 1; i >= 0; i-- {
		c := cellsRev[i]
		raw = append(raw, g.cellCenter(c%g.w, c/g.w))
	}
	raw = append(raw, to)

	return g.smooth(raw, out), true
}
