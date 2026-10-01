package game

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UpdatePortal decides whether the party leaves this map, and leaves on the
// World how the portals stand, so the drawing can show the group who it is
// still waiting for.
//
// It does NOT swap the world. The decision is the host's and the arrival is
// everyone's, so both roles travel through the same queue
// (network.ConsumeLocalTravel, applied by ApplyPendingTravel): a client that
// swapped its own map here would be back to the split party this replaced.
func UpdatePortal(current *World, p *entity.Player) {
	if current == nil || len(current.Portals) == 0 {
		return
	}
	// A portal only carries anyone once it has finished materialising, which on
	// a map with hordes means the whole run is cleared. Gating on the drawing
	// being fully open, and not merely on the wave state, keeps the rule the
	// player can see: what is not there yet does not work. The tally is dropped
	// with it, or a portal that closed again would keep a stale counter over it.
	if current.portalReveal < 1 {
		current.partyTally = nil
		if network.CurrentHost != nil {
			network.CurrentHost.SetPartyPortals(nil, false)
		}
		return
	}

	current.partyTally = countParty(current.Portals, portalParty(p))

	// Tell the host every open portal's rectangle — host_portal_presence.go
	// needs the actual box to decide who is standing inside one, the same
	// test countParty just ran, so waiting and the gate's tally can never
	// disagree. Active only once fully materialised AND nobody is still
	// mid-arrival (current.arrived): a party that just landed on top of a
	// portal must not vanish on the spot.
	if network.CurrentHost != nil {
		rects := make([]rl.Rectangle, len(current.Portals))
		targetIdx := 0
		for i, portal := range current.Portals {
			rects[i] = portal.Rect
			if current.partyTally[i].waiting() {
				targetIdx = i
			}
		}
		// rects[0] is the bot's default destination (host_bot_portal.go's
		// convention): whichever portal already has someone waiting, or the
		// first one otherwise.
		rects[0], rects[targetIdx] = rects[targetIdx], rects[0]
		network.CurrentHost.SetPartyPortals(rects, current.portalReveal >= 1 && !current.arrived)
	}

	// Landing on top of a portal must not immediately fire it again. It is
	// cleared only once NOBODY is standing in a portal, because the party
	// arrives together: clearing it as soon as one player stepped off would
	// re-fire the portal under the ones still on the arrival pad.
	if current.arrived {
		if !anyoneInside(current.partyTally) {
			current.arrived = false
		}
		return
	}
	// Only the machine that owns the simulation decides. A solo session has no
	// host object and is its own authority, which is why this asks who is NOT
	// deciding rather than who is.
	if network.Role == "client" {
		return
	}
	if portal, ok := readyPortal(current.Portals, current.partyTally); ok {
		log.Printf("[Portal] %s -> %s: o grupo inteiro entrou", current.Path, portal.TargetMap)
		network.StartTravel(portal.TargetMap, portal.TargetSpawn)
	}
}

// ApplyPendingTravel performs the map change the host announced, for whoever is
// running this loop. It returns the world it was given when there is nothing
// pending or the destination will not load, so the caller can always assign the
// result.
//
// It is called OUTSIDE the "player is alive" branch of the loop on purpose: a
// player who died on the way travels with the party and waits out the revive on
// the other side. Left inside, their machine would stay behind on a map the
// rest of the group has left.
func ApplyPendingTravel(current *World, p *entity.Player) *World {
	t, ok := network.ConsumeLocalTravel()
	if !ok {
		return current
	}
	// A reconnect catch-up (host_rejoin.go) is not a portal crossing: the
	// party never moved. If this machine is already on the right map there is
	// nothing to reload at all, and either way the local player has to land
	// back on the host-preserved position, not the map's spawn — see
	// reconnect_sync.go for why an ordinary frame never does this.
	if t.Reconnect {
		if current != nil && current.Path == t.TargetMap {
			resyncLocalPlayer(p)
			return current
		}
		next := travelTo(current, t.TargetMap, t.TargetSpawn, p)
		resyncLocalPlayer(p)
		return next
	}
	return travelTo(current, t.TargetMap, t.TargetSpawn, p)
}
