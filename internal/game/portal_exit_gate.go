package game

import (
	"github.com/WandenDourado/Legiao/internal/network"
)

// portalExitGate keeps a map's own collision pieces from blocking the portal
// that leads out of it once the phase is over. It exists because PortalsUnlocked
// (portal_gate.go) is already the single source of truth for "the phase is
// done" — the gate obeys that same sentence instead of a map declaring a
// second, hand-rolled rule of its own.
//
// world_06 is the motivating case: gates_gate_04, placed in the map long
// before the corridor cannons existed, sits inside portal_para_o_chefe and
// blocks 217 of its 512px width. Nothing in the game could ever open it — the
// only mechanism that disables a gate footprint, arenaGate, requires an
// arena_lock zone and a horde total the map does not declare — so the portal
// stayed partly obstructed from the first frame to the last, and the party had
// to squeeze through the two remaining slivers.
//
// This is agnostic of which map it runs on and costs nothing where no
// collision footprint sits inside a portal rectangle, which is every other
// map in the campaign.
type portalExitGate struct {
	mapPath    string
	generation int
	open       bool
}

// bind forgets state from a previous map or stage attempt, so a failed run
// begins with the exit closed again exactly like arenaGate's exitOpen does.
func (g *portalExitGate) bind(w *World) bool {
	gen := network.StageGeneration()
	path := ""
	if w != nil {
		path = w.Path
	}
	if path == g.mapPath && gen == g.generation {
		return false
	}
	changed := false
	if g.open && w != nil && w.Collision != nil {
		for _, portal := range w.Portals {
			if w.Collision.SetFootprintsEnabledOverlapping(portal.Rect, true) {
				changed = true
			}
		}
	}
	g.mapPath = path
	g.generation = gen
	g.open = false
	return changed
}

// UpdatePortalExitGate disables any collision footprint overlapping an open
// portal's rectangle the moment PortalsUnlocked turns true, and restores it on
// the next map or stage attempt via bind. It runs on every machine — host and
// client alike — because PortalsUnlocked reads only state the network layer
// already replicates (wave phase, cannon posts, climax), so every machine
// derives the same answer independently, the same way UpdateArenaGate does.
func (w *World) UpdatePortalExitGate() {
	if w == nil || len(w.Portals) == 0 {
		return
	}
	changed := w.portalExitGate.bind(w)
	g := &w.portalExitGate
	if !g.open && PortalsUnlocked() {
		g.open = true
		if w.Collision != nil {
			for _, portal := range w.Portals {
				if w.Collision.SetFootprintsEnabledOverlapping(portal.Rect, false) {
					changed = true
				}
			}
		}
	}
	if changed && network.CurrentHost != nil {
		// A malha de navegacao e derivada uma vez no carregamento e nao
		// observa a grade — mesma nota de arena_gate.go. Sem isto a horda (ou
		// um bot) continuaria planejando contra uma porta que ja saiu do
		// lugar.
		for _, portal := range w.Portals {
			network.CurrentHost.RebuildNavArea(portal.Rect)
		}
	}
}
