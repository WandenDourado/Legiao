package network

// The host has no notion of a map zone on its own — zones live in game.World
// (doc/tilemap.md "Arena de mão única"), and game imports network, never the
// other way around (plan_bots_de_classe.md §14.1, the same shape as
// host_bot_portal.go's SetPartyPortals). This is the one channel through
// which the host learns the arena's one-way zone exists and where it is:
// game.World.UpdateArenaGate calls SetArenaLock every frame it already binds
// the zone for, so the host can apply the SAME rule to a bot's body that the
// zone already applies to the local human's — the piece that was missing,
// since a bot is a body the host moves and no client ever sends input for it.

import (
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	arenaLockMu    sync.RWMutex
	arenaLockZone  rl.Rectangle
	arenaLockArmed bool
)

// SetArenaLock records the arena's one-way zone and whether the rule is
// active on the current map. An empty zone / armed=false turns the rule off
// for every bot — the state a map with no arena, or a client machine that
// never sets CurrentHost, always leaves it in. Called once per frame by the
// host's own game.World.UpdateArenaGate.
func (h *Host) SetArenaLock(zone rl.Rectangle, armed bool) {
	arenaLockMu.Lock()
	arenaLockZone, arenaLockArmed = zone, armed
	arenaLockMu.Unlock()
}

// arenaLock returns the current arena zone and whether the rule is armed.
func arenaLock() (rl.Rectangle, bool) {
	arenaLockMu.RLock()
	defer arenaLockMu.RUnlock()
	return arenaLockZone, arenaLockArmed
}

// rectContains reports whether a world point falls inside a rectangle, the
// same half-open test tilemap.Zone.Contains uses — duplicated here instead of
// imported because the wire format for this rule is a bare rl.Rectangle, not
// a tilemap.Zone (see the package comment above).
func rectContains(r rl.Rectangle, p rl.Vector2) bool {
	return p.X >= r.X && p.X < r.X+r.Width && p.Y >= r.Y && p.Y < r.Y+r.Height
}

// ResetBotArenaLocks clears every bot's arena-lock flag. Called on stage
// restart and map load/travel (game.World.ApplyToHost,
// network.Host.ResetStage) — NOT on join/reconcile or absence handling,
// where other bots may still be legitimately sealed inside an arena a human
// hasn't left. A bot that stayed locked into the NEXT map would be the
// mirror image of the bug this file fixes.
func (h *Host) ResetBotArenaLocks() {
	h.playersMutex.Lock()
	for _, rt := range h.bots {
		rt.arenaLocked = false
	}
	h.playersMutex.Unlock()
}
