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

// nearestFoe returns the closest living foe to pos.
func nearestFoe(pos rl.Vector2, foes []Foe) (Foe, bool) {
	var best Foe
	bestDist := float32(-1)
	for _, f := range foes {
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
		count := 0
		for _, g := range foes {
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
