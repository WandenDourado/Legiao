package nav

// Every constant the navigation mesh and its followers use, in one place —
// see doc/plan_navegacao_bots_monstros.md §10 for the reasoning behind each
// number. Adjust here, nowhere else.
const (
	// CellSize is the mesh resolution: half the width of the widest agent
	// footprint that walks in this game, which is what guarantees a real
	// ~60px gap contains at least one free cell center. A coarser cell can
	// make a legitimate opening invisible to the mesh.
	CellSize = 32.0

	// AgentBox is the square footprint tested at each cell center when the
	// mesh is built — the widest walking character's footprint (the wolf,
	// 45) with slack. Anything with a bigger footprint needs its own mesh;
	// see the package doc.
	AgentBox = 48.0

	// PathBudgetPerFrame caps how many A* searches may run in a single
	// simulation frame, so a whole pack blocked on the same obstacle cannot
	// spend an unbounded amount of time in one tick.
	PathBudgetPerFrame = 8

	// BotReplanEvery / FoeReplanEvery are how often ONE agent of that kind
	// may ask for a new route. A bot cannot look like it is thinking; a
	// monster can, and there are many more of them.
	BotReplanEvery = 0.4
	FoeReplanEvery = 0.7

	// FoeStuckBefore is how long a monster's direct approach must fail to
	// progress before it asks for a route — the "hit it, look, go around"
	// beat described in the plan, not instant tactical awareness.
	FoeStuckBefore = 0.4

	// WaypointReached is how close an agent must get to a waypoint before
	// it advances to the next one. Smaller than a cell, so the agent lets
	// go of a waypoint instead of orbiting it.
	WaypointReached = 24.0

	// NarrowGapSeparation is how much a following agent's separation push
	// is scaled down while both its sides are blocked — full separation in
	// a narrow gap pushes the whole group back out of the passage, the same
	// failure the portal wait had to fix.
	NarrowGapSeparation = 0.25
)
