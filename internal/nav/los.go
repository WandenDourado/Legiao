package nav

import rl "github.com/gen2brain/raylib-go/raylib"

// LineOfSight reports whether a straight walk from a to b never crosses a
// blocked cell. Sampled at half-cell steps rather than a full grid
// raycast: cheap, and never coarse enough to step clean over a one-cell
// obstacle since CellSize is already sized to the smallest gap that matters.
func (g *Grid) LineOfSight(a, b rl.Vector2) bool {
	dist := rl.Vector2Distance(a, b)
	if dist == 0 {
		return g.Walkable(a)
	}
	step := g.cell * 0.5
	steps := int(dist/step) + 1
	dir := rl.Vector2Scale(rl.Vector2Subtract(b, a), 1/dist)
	for i := 0; i <= steps; i++ {
		t := float32(i) * step
		if t > dist {
			t = dist
		}
		p := rl.Vector2Add(a, rl.Vector2Scale(dir, t))
		if !g.Walkable(p) {
			return false
		}
	}
	return true
}
