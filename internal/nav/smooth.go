package nav

import rl "github.com/gen2brain/raylib-go/raylib"

// smooth string-pulls a raw cell-by-cell path into the few waypoints a
// creature would actually choose: from the current accepted point, jump as
// far down the list as line of sight still allows, back off one point at a
// time when it does not. This is what turns a staircase of cell centers
// into a straight walk with corners only where the map actually has them.
func (g *Grid) smooth(raw []rl.Vector2, out []rl.Vector2) []rl.Vector2 {
	out = out[:0]
	if len(raw) == 0 {
		return out
	}
	out = append(out, raw[0])
	i := 0
	for i < len(raw)-1 {
		j := len(raw) - 1
		for j > i+1 && !g.LineOfSight(raw[i], raw[j]) {
			j--
		}
		out = append(out, raw[j])
		i = j
	}
	return out
}
