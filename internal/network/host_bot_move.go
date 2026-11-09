package network

// Bot movement resolves against the map exactly like a human's local
// movement does (game/collision.go's ResolveCollision) — the same shared
// resolver, the same foot-box math — just without a *entity.Player to call
// it on, since a bot is a PlayerState.

import (
	"github.com/WandenDourado/Legiao/internal/bot"
	"github.com/WandenDourado/Legiao/internal/collision"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/nav"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// desiredBotMove turns a decided bot.Intent into an actual direction: the
// navigation mesh's Follower (rt.nav) gets the agent around whatever
// obstacle the straight line to Intent.Dest cannot solve — the ROUTE layer
// — and Intent.Push (separation from allies, decided by the brain) blends
// on top, same as the old seekAndSeparate used to. HasDest=false means
// "no destination this tick": only Push applies, and the Follower's route
// (if it had one) is dropped.
//
// This still is not the final word on where the bot's foot lands — that is
// resolveBotMove, one layer further down, which is what actually respects
// the map this frame (doc/plan_navegacao_bots_monstros.md §2).
func desiredBotMove(rt *botRuntime, grid *nav.Grid, pos rl.Vector2, intent bot.Intent, dt float32) rl.Vector2 {
	if !intent.HasDest {
		rt.nav.Clear()
		if intent.Push.X == 0 && intent.Push.Y == 0 {
			return rl.Vector2{}
		}
		return rl.Vector2Normalize(intent.Push)
	}
	dest := intent.Dest
	if projected, changed := clampDestToArenaLock(rt, dest); changed {
		// An old route pointed south of the threshold; it has to die here, or
		// the Follower keeps steering toward a point the bot can no longer
		// reach and just hovers at the seal instead of walking to the new one.
		rt.nav.Clear()
		dest = projected
	}
	seek, _ := rt.nav.Desired(grid, pos, dest, dt, nav.BotReplanEvery)
	move := rl.Vector2Add(seek, intent.Push)
	if move.X == 0 && move.Y == 0 {
		return move
	}
	return rl.Vector2Normalize(move)
}

// clampDestToArenaLock projects a destination south of the arena's one-way
// threshold back onto it, for a bot already locked in (doc/tilemap.md
// "Arena de mão única"). Without this, a bot whose Intent.Dest still points
// at the corridor (an ally to follow, a portal from the previous map) would
// have the mesh plan straight through a seal it physically cannot cross —
// applyArenaLock would then correct the resolved position every frame while
// the Follower kept re-aiming it south, the exact scrape-on-the-line the bug
// report described. The mesh itself stays global and shared with monsters
// (doc/plan_navegacao_bots_monstros.md §3); the per-agent rule belongs on
// the destination, not on the mesh.
//
// nav.CellSize insets the projected point from the boundary line itself so
// it lands solidly inside a walkable cell rather than exactly on the seam
// between the arena and the sealed strip.
func clampDestToArenaLock(rt *botRuntime, dest rl.Vector2) (rl.Vector2, bool) {
	zone, armed := arenaLock()
	if !armed || !rt.arenaLocked {
		return dest, false
	}
	limit := zone.Y + zone.Height - nav.CellSize
	if dest.Y <= limit {
		return dest, false
	}
	return rl.NewVector2(dest.X, limit), true
}

// applyArenaLock is the bot half of the arena's one-way rule
// (doc/tilemap.md "Arena de mão única"). game.World.UpdateArenaGate applies
// the equivalent clamp to the local human every frame through
// entity.MoveByGroundCorrection, but that call only ever reaches the
// client's own *entity.Player — never a bot's body, which the host itself
// moves and no client sends input for. network.SetArenaLock is the one
// channel that tells the host the zone exists at all (host_arena_lock.go);
// this is where the host applies it, per bot — armed by STANDING inside the
// zone, exactly like arenaGate.returnLocked, so a human who takes over a bot
// already inside the arena inherits a body that is correctly sealed in too.
func applyArenaLock(rt *botRuntime, char entity.CharacterType, pos rl.Vector2) rl.Vector2 {
	zone, armed := arenaLock()
	if !armed {
		return pos
	}
	center, _, height := entity.GroundBoxAt(pos, char)
	if !rt.arenaLocked && rectContains(zone, center) {
		rt.arenaLocked = true
	}
	if !rt.arenaLocked || center.X < zone.X || center.X >= zone.X+zone.Width {
		return pos
	}
	limit := zone.Y + zone.Height - height/2
	if center.Y > limit {
		// center.Y is a straight offset of pos.Y (GroundBoxAt never scales
		// it), so nudging pos.Y by the same amount the center overshot the
		// limit lands the foot box exactly on it — entity.MoveByGroundCorrection's
		// arithmetic without a *Player to apply it to.
		pos.Y += limit - center.Y
	}
	return pos
}

// stuckDistance/stuckWindow: a bot trying to move but netting less than
// stuckDistance of displacement over stuckWindow seconds is stuck against
// scenery (plan §9.1) — the same numbers bot/tuning.go documents for the
// AI side, duplicated here because this file must not import internal/bot
// (only network may import bot, never the reverse... but nothing stops
// this package from importing bot; it is entity/collision math that simply
// has no reason to).
const (
	stuckDistance = 40.0
	stuckWindow   = 1.5
)

// unstickMove detects a bot that intends to move but is not actually
// getting anywhere (wedged against a wall, corner, or another bot) and
// nudges it sideways to break free, using the same sliding resolver every
// other move already goes through. Returns the intent unchanged otherwise.
func unstickMove(rt *botRuntime, pos rl.Vector2, intent rl.Vector2, dt float32) rl.Vector2 {
	wantsToMove := intent.X != 0 || intent.Y != 0
	if !wantsToMove {
		rt.stuckFor = 0
		rt.lastPos = pos
		return intent
	}
	if rl.Vector2Distance(pos, rt.lastPos) >= stuckDistance {
		rt.stuckFor = 0
		rt.lastPos = pos
		return intent
	}
	rt.stuckFor += dt
	if rt.stuckFor < stuckWindow {
		return intent
	}
	// Still within stuckDistance after a full window: sidestep instead of
	// pushing straight into whatever is blocking.
	rt.stuckFor = 0
	rt.lastPos = pos
	return rl.Vector2Normalize(rl.NewVector2(-intent.Y, intent.X))
}

// botSpeed matches entity.PlayerSpeed: a bot walks like any character does,
// no faster and no slower.
const botSpeed = entity.PlayerSpeed

// resolveBotMove returns where pos ends up walking by delta, resolved
// against solid, using the character's foot box — entity.MoveByGroundCorrection's
// arithmetic without a *Player to apply it to (plan §4, entity section).
//
// Uses ResolveDetour, not the plain slide: a bot walking straight at a tree
// or a fence has nothing to slide along and used to just push against it
// until unstickMove's 1.5s timer sidestepped it for one frame before it went
// right back to pushing (doc/plan_navegacao_bots_monstros.md §1). Detouring
// commits to one way around the obstacle in rt.detourDir, the same
// commitment entity.(*Enemy).step already relies on, and clears it once the
// direct path opens again.
func resolveBotMove(rt *botRuntime, pos rl.Vector2, delta rl.Vector2, char entity.CharacterType, solid collision.Solid, grid *nav.Grid) rl.Vector2 {
	if delta.X == 0 && delta.Y == 0 {
		return pos
	}
	wanted, width, height := entity.GroundBoxAt(pos, char)
	// collision.Resolve gives an entity whose foot box already overlaps solid
	// space a free pass: it moves the FULL delta with no check at all, on the
	// assumption the entity spawned inside solid or got shoved there by a
	// crowd and just needs one frame to escape. A footprint that toggles
	// solid in game — this map's arena gate is exactly one, see
	// doc/tilemap.md "Arena de mão única" — can catch a bot standing on it at
	// the instant it turns back on, and that free pass is a bot walking
	// straight through whatever wall is next, not out of the footprint that
	// trapped it. Snapping to the nearest walkable cell FIRST keeps the
	// escape local: the free pass still fires (the snapped point starts
	// walkable), but there is nothing left to ghost through.
	if grid != nil && solid != nil && solid.CollidesCentered(wanted, width, height) {
		if free, ok := grid.NearestWalkable(wanted, nav.CellSize*4); ok {
			pos = rl.Vector2Add(pos, rl.Vector2Subtract(free, wanted))
			wanted = free
		}
	}
	resolved, committed := collision.ResolveDetour(wanted, delta, width, height, solid, rt.detourDir)
	rt.detourDir = committed
	diff := rl.Vector2Subtract(resolved, wanted)
	return rl.Vector2Add(pos, diff)
}

// stepBotAnimation advances a bot's headless frame/row bookkeeping so it
// rides the wire like any moving player.
func stepBotAnimation(rt *botRuntime, char entity.CharacterType, dir rl.Vector2, dt float32) (row, frame int) {
	def := entity.GetCharacter(char)
	moving := dir.X != 0 || dir.Y != 0
	rt.frame, rt.animTimer = entity.StepWalkAnimation(def, rt.frame, rt.animTimer, moving, false, dt)
	if moving {
		rt.lastRow = entity.WalkRowFor(dir)
	}
	return rt.lastRow, rt.frame
}
