package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// direction returns the normalized vector from a to b, or the zero vector
// when they coincide (so callers never have to special-case a NaN from
// normalizing a zero-length vector).
func direction(a, b rl.Vector2) rl.Vector2 {
	d := rl.Vector2Subtract(b, a)
	if d.X == 0 && d.Y == 0 {
		return d
	}
	return rl.Vector2Normalize(d)
}

// nearestFoe returns the closest living, non-sentry foe to pos. A sentry
// never comes back from this — see Foe.IsSentry's doc for why, and
// nearestSentry (arqueiro.go) for the one function allowed to find one.
func nearestFoe(pos rl.Vector2, foes []Foe) (Foe, bool) {
	var best Foe
	bestDist := float32(-1)
	for _, f := range foes {
		if f.IsSentry {
			continue
		}
		d := rl.Vector2Distance(pos, f.Pos)
		if bestDist < 0 || d < bestDist {
			best, bestDist = f, d
		}
	}
	return best, bestDist >= 0
}

// mostThreateningFoe picks whichever foe is closest to the most fragile
// ally (lowest health fraction), tie-broken by that ally being the most
// hurt. Used by classes whose job is to protect the group rather than
// simply hit whatever is nearest to themselves.
func mostThreateningFoe(self Ally, allies []Ally, foes []Foe) (Foe, bool) {
	frailest := self
	frailestFrac := healthFrac(self)
	for _, a := range allies {
		if a.IsDead {
			continue
		}
		if f := healthFrac(a); f < frailestFrac {
			frailest, frailestFrac = a, f
		}
	}
	return nearestFoe(frailest.Pos, foes)
}

func healthFrac(a Ally) float32 {
	if a.MaxHealth <= 0 {
		return 1
	}
	return a.Health / a.MaxHealth
}

// leadTarget predicts where a moving foe will be after `seconds`, so a
// projectile aimed here has a chance to actually land.
func leadTarget(f Foe, seconds float32) rl.Vector2 {
	return rl.Vector2Add(f.Pos, rl.Vector2Scale(f.Vel, seconds))
}

// clusterCentre returns the position of whichever foe has the most other
// foes within radius of it, and how many that is (itself included). Used by
// area-damage classes (Mago) to find the best place to drop a blast.
func clusterCentre(foes []Foe, radius float32) (rl.Vector2, int) {
	var best rl.Vector2
	bestCount := 0
	for _, f := range foes {
		if f.IsSentry {
			continue
		}
		count := 0
		for _, g := range foes {
			if g.IsSentry {
				continue
			}
			if rl.Vector2Distance(f.Pos, g.Pos) <= radius {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			best = f.Pos
		}
	}
	return best, bestCount
}

// separation returns a push vector away from any ally closer than minDist,
// so a group of bots does not stand on top of each other in front of an
// area attack.
func separation(pos rl.Vector2, allies []Ally, minDist float32) rl.Vector2 {
	var push rl.Vector2
	for _, a := range allies {
		if a.IsDead {
			continue
		}
		d := rl.Vector2Distance(pos, a.Pos)
		if d > 0 && d < minDist {
			away := rl.Vector2Scale(direction(a.Pos, pos), (minDist-d)/minDist)
			push = rl.Vector2Add(push, away)
		}
	}
	return push
}

// moveTo sets Dest/HasDest/Push on intent so the host's navigation layer
// (internal/nav, via host_bot_tick.go) can get the agent to dest — around
// obstacles when the straight line is blocked — blended with the ordinary
// separation from allies. This is what every brain calls instead of
// deciding its own direction: the brain says WHERE, the navigation layer
// says BY WHAT ROUTE (plan §2's split between the decision and route
// layers).
func moveTo(intent *Intent, selfPos, dest rl.Vector2, allies []Ally) {
	intent.Dest = dest
	intent.HasDest = true
	intent.Push = separation(selfPos, allies, allySeparation)
}

// flee sets intent to move straight away from threat, no separation and no
// navigation — panicLine-class reactions are an immediate reflex, not a
// planned route.
func flee(intent *Intent, selfPos, threat rl.Vector2) {
	dir := fleeFrom(selfPos, threat)
	intent.Dest = rl.Vector2Add(selfPos, rl.Vector2Scale(dir, 200))
	intent.HasDest = true
}

// groupIsFalling is the shared "call the ultimate now" trigger: two party
// members (self included) under 30% health, or one dead with a horde still
// in the field.
func groupIsFalling(self Ally, allies []Ally, enemiesLeft int) bool {
	low := 0
	if healthFrac(self) < 0.30 {
		low++
	}
	deadCount := 0
	for _, a := range allies {
		if a.IsDead {
			deadCount++
			continue
		}
		if healthFrac(a) < 0.30 {
			low++
		}
	}
	return low >= 2 || (deadCount >= 1 && enemiesLeft > 0)
}

// anyFoeWithin reports whether any foe sits within radius of pos.
func anyFoeWithin(pos rl.Vector2, foes []Foe, radius float32) bool {
	for _, f := range foes {
		if f.IsSentry {
			continue
		}
		if rl.Vector2Distance(pos, f.Pos) <= radius {
			return true
		}
	}
	return false
}

// fleeFrom is the direction straight away from a threat.
func fleeFrom(pos, threat rl.Vector2) rl.Vector2 {
	return direction(threat, pos)
}

// pointToSegmentDistance is the shortest distance from p to the segment ab.
func pointToSegmentDistance(p, a, b rl.Vector2) float32 {
	ab := rl.Vector2Subtract(b, a)
	lenSq := ab.X*ab.X + ab.Y*ab.Y
	if lenSq == 0 {
		return rl.Vector2Distance(p, a)
	}
	t := ((p.X-a.X)*ab.X + (p.Y-a.Y)*ab.Y) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	proj := rl.Vector2Add(a, rl.Vector2Scale(ab, t))
	return rl.Vector2Distance(p, proj)
}

// foeBlocksLine returns the first foe sitting close enough to the segment
// from `from` to `through` that a piercing bolt aimed at `through` would be
// consumed by it first, instead of reaching (and healing) whoever is at
// `through`.
func foeBlocksLine(from, through rl.Vector2, foes []Foe) (Foe, bool) {
	for _, f := range foes {
		if f.IsSentry {
			continue
		}
		if pointToSegmentDistance(f.Pos, from, through) <= f.Radius+lineBlockTolerance {
			return f, true
		}
	}
	return Foe{}, false
}

// foeBeyondAlly returns the closest foe that lies roughly along the ray
// from `from` through `ally`, PAST the ally — the case where aiming a
// piercing shot at the ally also lines up a hit on a monster behind them.
func foeBeyondAlly(from, ally rl.Vector2, foes []Foe) (Foe, bool) {
	rayDir := direction(from, ally)
	if rayDir.X == 0 && rayDir.Y == 0 {
		return Foe{}, false
	}
	allyDist := rl.Vector2Distance(from, ally)
	var best Foe
	bestDist := float32(-1)
	for _, f := range foes {
		if f.IsSentry {
			continue
		}
		toFoe := rl.Vector2Subtract(f.Pos, from)
		proj := toFoe.X*rayDir.X + toFoe.Y*rayDir.Y
		if proj <= allyDist {
			continue // not beyond the ally
		}
		onRay := rl.Vector2Add(from, rl.Vector2Scale(rayDir, proj))
		if rl.Vector2Distance(f.Pos, onRay) > alignTolerance {
			continue
		}
		if bestDist < 0 || proj < bestDist {
			best, bestDist = f, proj
		}
	}
	return best, bestDist >= 0
}

// partyCentreOrSelf falls back to the bot's own position when it has no
// living allies to average with, so followRadius never fights an empty
// party.
func withinFollowRadius(pos, centre rl.Vector2) bool {
	return rl.Vector2Distance(pos, centre) <= followRadius
}

// followDest is the "no target, catch up with the group" destination every
// brain falls back to: the class's formation post relative to the living
// humans' front when at least one human is alive and the bot is out of
// followRadius from it, or the bot's own position otherwise — holding the
// line and fighting rather than beelining toward wherever the other bots
// happen to be (doc/plan_avanco_bots_e_gargula.md §A3, R1 and R3). Humans
// decide where the party is going; without one alive, there is no "group"
// to follow, only a run waiting on respawn.
func followDest(v View) rl.Vector2 {
	if !v.HasHumans {
		return v.Self.Pos
	}
	post := formationPost(v)
	if withinFollowRadius(v.Self.Pos, post) {
		return v.Self.Pos
	}
	return post
}

// travelDest returns the portal as a destination when — and only when —
// traveling is the right call right now: the door is open, nothing nearby
// needs fighting, and a human is already at (or near) the door
// (HumansAtPortal). Without the third condition a bot would march to the
// portal alone the instant a garrison map's own "no waves" rule opens it —
// world_03 has no enemy_spawn_* markers at all, so WaveState.Total is 0 and
// game.PortalsUnlocked() is already true from the first frame while the
// garrison is still in the field (doc/plan_avanco_bots_e_gargula.md §A2,
// causes 2 and 4). Humans decide when the party travels; the bots only
// escort.
func travelDest(v View) (rl.Vector2, bool) {
	if !v.PortalActive || !v.HumansAtPortal {
		return rl.Vector2{}, false
	}
	if len(engageableFoes(v)) > 0 {
		return rl.Vector2{}, false
	}
	return v.Portal, true
}

// finishMove is the LAST step of a brain's destination chain (alvo engajado
// > recuo > travelDest > formationPost): it applies dest to intent, with
// Push left at ZERO when usingTravel is true instead of the ordinary moveTo
// blend. Separation exists so a crowd does not eat one area hit together —
// exactly the thing that would shove everyone back OUT of the portal's small
// rectangle (doc/tilemap.md "Quem entra no portal some e espera"), so travel
// must not fight it. usingTravel is decided by the CALLER: only it knows
// whether combat targeting or a retreat already overrode travelDest's result
// this tick.
func finishMove(intent *Intent, v View, dest rl.Vector2, usingTravel bool) {
	if usingTravel {
		intent.Dest, intent.HasDest = dest, true
		return
	}
	moveTo(intent, v.Self.Pos, dest, v.Allies)
}

// formationPost is where a bot's class wants to stand relative to the
// humans' front line and heading, per classFormation (tuning.go). This is
// what "seguir o grupo" actually means once the party is advancing rather
// than idling (plan §A3, R3): the escort re-forms around the humans instead
// of collapsing onto their exact position. A class with no table entry
// (never one of the five playable characters) falls back to the human
// centre itself, so a stray call cannot divide by a zero direction.
func formationPost(v View) rl.Vector2 {
	off, ok := classFormation[v.Self.Char]
	if !ok {
		return v.HumanCentre
	}
	fwd := v.AdvanceDir
	if fwd.X == 0 && fwd.Y == 0 {
		// No heading has ever been established (the run just started and
		// nobody has moved yet) — pick a stable arbitrary forward instead
		// of leaving every ranged bot stacked exactly on the humans.
		fwd = rl.NewVector2(0, -1)
	}
	lat := rl.NewVector2(-fwd.Y, fwd.X)
	post := rl.Vector2Add(v.HumanCentre, rl.Vector2Scale(fwd, off.forward))
	return rl.Vector2Add(post, rl.Vector2Scale(lat, off.lateral))
}

// retreatHysteresis decides whether a bot should be retreating this tick,
// given whether it was retreating last tick. Entering below `under` and only
// rejoining above `above` is what stops a bot from flickering in and out of
// combat every time a single hit crosses one exact threshold (plan
// doc/plan_avanco_bots_e_gargula.md §A4). Mutates *retreating and returns
// its new value.
func retreatHysteresis(retreating *bool, frac, under, above float32) bool {
	switch {
	case *retreating && frac >= above:
		*retreating = false
	case !*retreating && frac < under:
		*retreating = true
	}
	return *retreating
}

// paladinaRetreatHysteresis is retreatHysteresis with an extra gate on
// entering retreat: she may not fall back until Shield is no longer
// available to her (shieldReady false — on cooldown, or just cast), since a
// front line that retreats before trying to mitigate abandons the group
// (plan §A4).
func paladinaRetreatHysteresis(retreating *bool, frac float32, shieldReady bool) bool {
	switch {
	case *retreating && frac >= rejoinAbove:
		*retreating = false
	case !*retreating && frac < paladinaRetreatUnder && !shieldReady:
		*retreating = true
	}
	return *retreating
}

// retreatDest is where a retreating bot falls back to: its usual formation
// post, pushed retreatExtraBack further behind, away from whatever enemy is
// nearest — so retreating actually opens distance from the threat instead of
// just holding the same line (plan §A4).
func retreatDest(v View, nearestFoePos rl.Vector2, hasNearest bool) rl.Vector2 {
	post := formationPost(v)
	if !hasNearest {
		return post
	}
	away := direction(nearestFoePos, v.Self.Pos)
	if away.X == 0 && away.Y == 0 {
		return post
	}
	return rl.Vector2Add(post, rl.Vector2Scale(away, retreatExtraBack))
}

// engageableFoes filters foes down to the ones close enough to matter for a
// decision right now: within engageRadius of the bot itself, or of the
// living humans' centre (plan §A3, R2). Everything else is invisible to
// target selection — a bot chasing something on the far side of the map is
// exactly the bug this exists to stop.
func engageableFoes(v View) []Foe {
	out := make([]Foe, 0, len(v.Foes))
	for _, f := range v.Foes {
		if rl.Vector2Distance(v.Self.Pos, f.Pos) <= engageRadius {
			out = append(out, f)
			continue
		}
		if v.HasHumans && rl.Vector2Distance(v.HumanCentre, f.Pos) <= engageRadius {
			out = append(out, f)
		}
	}
	return out
}
