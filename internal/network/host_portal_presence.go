package network

// Waiting inside a portal: a player who steps into the party's open portal
// rectangle vanishes and freezes, freeing the small rectangle for the rest
// of the group instead of the whole party orbiting it waiting for everyone
// to land inside at once. Human and bot both count — the door still opens
// only for the whole party, bots included (plan §3). This file only DECIDES
// the flag; who freezes because of it is host_bot_tick.go (bots) and
// game/input_handler.go (the local human's own input).

import (
	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// tickPortalPresence marks every living player InPortal if their foot box
// (entity.GroundBoxAt — the SAME box game/portal_party.go's countParty
// tests, so waiting and the gate's own tally can never disagree) sits
// inside one of the party's currently active portal rectangles. A dead
// player never enters portal-wait: it travels with the party like it
// always has, still fallen. Called once per simulation tick from
// UpdateSimulation, before tickBots.
func (h *Host) tickPortalPresence() {
	rects, active := partyPortalRectsSnapshot()

	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	for _, p := range h.players {
		if p.IsDead || !active {
			p.InPortal = false
			continue
		}
		pos := rl.NewVector2(float32(p.X), float32(p.Y))
		center, width, height := entity.GroundBoxAt(pos, entity.CharacterType(p.Character))
		p.InPortal = boxInsideAnyRect(center, width, height, rects)
	}
}

// boxInsideAnyRect mirrors tilemap.Portal.Contains's overlap test (a
// centered box against a rectangle) without importing tilemap, which this
// package must not depend on for portal geometry — the rectangles already
// arrived pre-resolved via SetPartyPortals.
func boxInsideAnyRect(center rl.Vector2, width, height float32, rects []rl.Rectangle) bool {
	left, right := center.X-width/2, center.X+width/2
	top, bottom := center.Y-height/2, center.Y+height/2
	for _, r := range rects {
		if left < r.X+r.Width && right > r.X && top < r.Y+r.Height && bottom > r.Y {
			return true
		}
	}
	return false
}

// clearAllPortalPresence drops InPortal for every player. Called on travel
// (host_travel.go) and stage reset (host_reset.go): without it the party
// would arrive on the next map — or a restarted stage — invisible and with
// no control, which is the worst failure this whole feature could have.
func (h *Host) clearAllPortalPresence() {
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	for _, p := range h.players {
		p.InPortal = false
	}
}
