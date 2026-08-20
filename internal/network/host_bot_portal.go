package network

// The host has no notion of a portal on its own — portals live in game.World,
// and game imports network, never the other way around (plan §4/§14.1). This
// is the one channel through which the host learns portals exist and where
// they are: game.UpdatePortal calls SetPartyPortals every frame it already
// computes the party's portal tally for.

import (
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	partyPortalMu     sync.RWMutex
	partyPortalRects  []rl.Rectangle
	partyPortalActive bool
)

// SetPartyPortals records every open portal's rectangle (not just its
// centre — host_portal_presence.go needs the actual box to test who is
// standing inside one) and whether they are open for business (fully
// materialised and nobody mid-arrival — plan §14.1). rects[0], by the
// caller's convention, is whichever portal a bot with nothing better to
// aim at should walk toward. Called once per frame by the host's own
// game.UpdatePortal; a no-op on a client, which never runs bots.
func (h *Host) SetPartyPortals(rects []rl.Rectangle, active bool) {
	partyPortalMu.Lock()
	partyPortalRects, partyPortalActive = rects, active
	partyPortalMu.Unlock()
}

// partyPortal returns the centre a bot with no better destination should
// walk toward, and whether the door is open at all.
func partyPortal() (rl.Vector2, bool) {
	partyPortalMu.RLock()
	defer partyPortalMu.RUnlock()
	if !partyPortalActive || len(partyPortalRects) == 0 {
		return rl.Vector2{}, false
	}
	r := partyPortalRects[0]
	return rl.NewVector2(r.X+r.Width/2, r.Y+r.Height/2), true
}

// partyPortalRectsSnapshot returns every currently active portal rectangle,
// for host_portal_presence.go to test who is standing inside one.
func partyPortalRectsSnapshot() ([]rl.Rectangle, bool) {
	partyPortalMu.RLock()
	defer partyPortalMu.RUnlock()
	if !partyPortalActive {
		return nil, false
	}
	out := make([]rl.Rectangle, len(partyPortalRects))
	copy(out, partyPortalRects)
	return out, true
}
