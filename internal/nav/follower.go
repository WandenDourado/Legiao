package nav

import rl "github.com/gen2brain/raylib-go/raylib"

// Follower is the route state for ONE agent: the current smoothed path (if
// any), which waypoint it is walking toward, and the goal that path was
// computed for. One Follower belongs to one bot or one monster for its
// whole lifetime — it is what lets an agent commit to a route across
// frames instead of re-deciding (and re-searching) every one of them.
type Follower struct {
	path     []rl.Vector2
	idx      int
	goal     rl.Vector2
	hasGoal  bool
	replanIn float32
}

// Active reports whether this Follower currently has a goal it is
// mid-route toward. Callers that only want the mesh consulted once they
// are actually committed to a detour (and not before, and not the instant
// it clears) use this for that hysteresis — see
// entity.(*Enemy).moveTowardTarget's non-progress gate.
func (f *Follower) Active() bool {
	return f.hasGoal
}

// Clear drops the current route. Called whenever a goal is abandoned —
// destination reached, or the direct line opened back up.
func (f *Follower) Clear() {
	f.path = f.path[:0]
	f.idx = 0
	f.hasGoal = false
	f.replanIn = 0
}

// Desired returns the direction to move this frame toward goal, and
// whether it is currently following a planned route (false means "walking
// the straight line," which is also what it returns while no route is
// available yet — never nothing).
//
// replanEvery is the caller's own clock (BotReplanEvery or FoeReplanEvery):
// Desired never asks for a new search more often than that, on top of the
// shared per-frame budget every Grid enforces regardless of caller.
func (f *Follower) Desired(g *Grid, pos, goal rl.Vector2, dt, replanEvery float32) (rl.Vector2, bool) {
	if g == nil || g.LineOfSight(pos, goal) {
		f.Clear()
		return direction(pos, goal), false
	}

	f.replanIn -= dt
	goalMoved := !f.hasGoal || rl.Vector2Distance(f.goal, goal) > WaypointReached
	needsPath := len(f.path) == 0 || f.idx >= len(f.path)

	if !needsPath && !goalMoved && !g.LineOfSight(pos, f.path[f.idx]) {
		// A shove (ResolveEnemyOverlap, another agent's own separation) can
		// knock an agent off the straight line to its current waypoint
		// without moving the goal at all (plan §6, risk 5). Reanchor to
		// whichever remaining waypoint is farthest but still visible before
		// paying for a full replan — usually all a shove actually needs.
		if !f.reanchor(g, pos) {
			needsPath = true
		}
	}

	if (goalMoved || needsPath) && f.replanIn <= 0 {
		f.replanIn = replanEvery
		f.request(g, pos, goal)
	}

	if len(f.path) == 0 {
		return direction(pos, goal), false
	}
	for f.idx < len(f.path) && rl.Vector2Distance(pos, f.path[f.idx]) <= WaypointReached {
		f.idx++
	}
	if f.idx >= len(f.path) {
		return direction(pos, goal), false
	}
	return direction(pos, f.path[f.idx]), true
}

// reanchor advances idx to the farthest remaining waypoint still visible
// from pos, so a shove that only knocked the agent slightly off its current
// leg does not force a full replan. Returns false when not even the
// waypoint it was already walking toward is visible any more, which means
// the shove was bad enough that only a fresh search will do.
func (f *Follower) reanchor(g *Grid, pos rl.Vector2) bool {
	for j := len(f.path) - 1; j >= f.idx; j-- {
		if g.LineOfSight(pos, f.path[j]) {
			f.idx = j
			return true
		}
	}
	return false
}

// request spends one search (if the frame budget has room) trying to plan
// from pos to goal. A refused reservation leaves the Follower exactly as it
// was — replanIn already ticked over in Desired, so it retries at the next
// allowed tick rather than hammering the budget every frame.
func (f *Follower) request(g *Grid, pos, goal rl.Vector2) {
	if !g.tryReserveSearch() {
		return
	}
	f.goal = goal
	f.hasGoal = true

	from, ok1 := g.NearestWalkable(pos, g.cell*4)
	to, ok2 := g.NearestWalkable(goal, g.cell*4)
	if !ok1 || !ok2 {
		f.path = f.path[:0]
		return
	}
	if path, ok := g.FindPath(from, to, f.path[:0]); ok {
		f.path = path
		f.idx = 0
		return
	}
	f.path = f.path[:0]
}

// direction is the normalized vector from a to b, or the zero vector when
// they coincide.
func direction(a, b rl.Vector2) rl.Vector2 {
	d := rl.Vector2Subtract(b, a)
	if d.X == 0 && d.Y == 0 {
		return d
	}
	return rl.Vector2Normalize(d)
}
