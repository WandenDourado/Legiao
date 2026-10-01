package nav

// ResetFrameBudget must be called exactly once per simulation frame, before
// any agent asks for a route. It is what keeps a whole pack colliding with
// the same barricade in the same frame from spending an unbounded number of
// A* searches in that one tick (plan §3.4).
func (g *Grid) ResetFrameBudget() {
	g.searchesThisFrame = 0
}

// tryReserveSearch reports whether the caller may run a search this frame,
// consuming one of PathBudgetPerFrame slots if so. A caller refused a slot
// keeps whatever path (or straight line) it already had — it never waits.
func (g *Grid) tryReserveSearch() bool {
	if g.searchesThisFrame >= PathBudgetPerFrame {
		return false
	}
	g.searchesThisFrame++
	return true
}
