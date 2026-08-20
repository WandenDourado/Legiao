package network

// The navigation mesh is built once per map load (game/world_state.go), but
// the arena gate (game/arena_gate.go) changes the map's collision IN GAME by
// toggling a footprint region open or closed. Without telling the mesh, it
// would keep insisting the gate is however it was at load time, and the
// whole pack would either walk a route that no longer exists or believe none
// does — doc/plan_navegacao_bots_monstros.md §3.2.

import rl "github.com/gen2brain/raylib-go/raylib"

// BotRoute returns the mesh-planned waypoints a bot is currently following,
// for the F4 debug overlay only — nothing on any gameplay path reads this.
func (h *Host) BotRoute(botID string) []rl.Vector2 {
	h.playersMutex.RLock()
	defer h.playersMutex.RUnlock()
	rt, ok := h.bots[botID]
	if !ok {
		return nil
	}
	return rt.nav.Path()
}

// RebuildNavArea recomputes mesh walkability for the cells overlapping
// area, against the CURRENT collision (the same Solid the mesh itself was
// built from — game callers never need to pass it again). A no-op before
// the mesh exists, same as every other Nav access.
func (h *Host) RebuildNavArea(area rl.Rectangle) {
	if h.EntityManager == nil || h.EntityManager.Nav == nil {
		return
	}
	h.EntityManager.Nav.RebuildArea(h.EntityManager.Solid, area)
}
