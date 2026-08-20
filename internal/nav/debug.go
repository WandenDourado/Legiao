package nav

import rl "github.com/gen2brain/raylib-go/raylib"

// ForEachCell calls fn for every cell's center and free/blocked state.
// The only caller is a debug overlay (game's F4 toggle) — nothing on any
// gameplay path uses this, and iterating every cell every frame is not
// something UpdateSimulation should ever pay for.
func (g *Grid) ForEachCell(fn func(center rl.Vector2, half float32, free bool)) {
	half := g.cell / 2
	for cy := 0; cy < g.h; cy++ {
		for cx := 0; cx < g.w; cx++ {
			fn(g.cellCenter(cx, cy), half, g.free[g.index(cx, cy)])
		}
	}
}

// Path returns the Follower's current smoothed waypoints, for the debug
// overlay to draw an agent's route. The caller must not mutate the result.
func (f *Follower) Path() []rl.Vector2 {
	return f.path
}
