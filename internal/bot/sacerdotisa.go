package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// sacerdotisaBrain keeps the party alive first, damages second. Her basic
// bolt heals whichever LIVING ally it passes through — never herself
// (checkHolyProjectileHeals skips the caster) — without being consumed by
// them, so the mira picks a line THROUGH the most wounded ally whenever one
// exists in range, only falling back to the nearest threat once everyone is
// full. She holds further back than a straight damage dealer would
// (backLine, still inside the bolt's ~760px reach — boltRange) and breaks
// off immediately from anything inside panicLine. When nothing is near
// (calmRadius), she spends the peace topping the party off instead of
// standing idle.
type sacerdotisaBrain struct{}

func (b *sacerdotisaBrain) Think(v View) Intent {
	var intent Intent

	calm := !anyFoeWithin(v.Self.Pos, v.Foes, calmRadius)
	if calm {
		wounded, hasWounded := mostWoundedAlly(v.Allies)
		intent = b.recover(v, hasWounded, wounded)
	} else {
		nearest, hasNearest := nearestFoe(v.Self.Pos, v.Foes)
		wounded, hasWounded := mostWoundedAlly(v.Allies)
		inRange := hasWounded && rl.Vector2Distance(v.Self.Pos, wounded) <= boltRange
		var blocker Foe
		blocked := false
		if inRange {
			blocker, blocked = foeBlocksLine(v.Self.Pos, wounded, v.Foes)
		}

		switch {
		case inRange && !blocked:
			// The heal line: aim through the wounded ally, preferring
			// whatever also lands a foe beyond them.
			aim := wounded
			if beyond, ok := foeBeyondAlly(v.Self.Pos, wounded, v.Foes); ok {
				if through, ok2 := aimThrough(v.Self.Pos, wounded, beyond.Pos); ok2 {
					aim = through
				}
			}
			intent.Attack = &aim
			b.lineUpMove(&intent, v, wounded, hasNearest, nearest)

		case blocked:
			// A monster sits on the line between her and the ally who needs
			// the bolt: the heal would be spent on the blocker instead, so
			// she fights it — the same defensive decision either way.
			aim := leadTarget(blocker, 0.25)
			intent.Attack = &aim
			b.combatMove(&intent, v, nearest, hasNearest)

		default:
			// Nobody in reach needs healing (or nobody is wounded at all):
			// fall back to the party's usual threat targeting.
			if target, ok := mostThreateningFoe(v.Self, v.Allies, v.Foes); ok {
				aim := leadTarget(target, 0.25)
				intent.Attack = &aim
			}
			b.combatMove(&intent, v, nearest, hasNearest)
		}
	}

	if cast, ok := b.sanctuaryCast(v, calm); ok {
		intent.Skill = &cast
	} else if cast, ok := b.angelicCast(v); ok {
		intent.Skill = &cast
	}

	return intent
}

// lineUpMove positions her behind the wounded ally relative to whatever is
// threatening the party, so the heal line stays open instead of her
// wandering onto it. panicLine still wins over everything: a monster
// basically on top of her is not a geometry problem.
func (b *sacerdotisaBrain) lineUpMove(intent *Intent, v View, wounded rl.Vector2, hasNearest bool, nearest Foe) {
	if hasNearest && rl.Vector2Distance(v.Self.Pos, nearest.Pos) < panicLine {
		flee(intent, v.Self.Pos, nearest.Pos)
		return
	}
	threatRef := v.PartyCentre
	if hasNearest {
		threatRef = nearest.Pos
	}
	dest := wounded
	if behind := direction(threatRef, wounded); behind.X != 0 || behind.Y != 0 {
		dest = rl.Vector2Add(wounded, rl.Vector2Scale(behind, backLine*0.5))
	}
	moveTo(intent, v.Self.Pos, dest, v.Allies)
}

// combatMove is the plain backline/panic distance-keeping used whenever she
// is not actively lining up a heal: flee inside panicLine, retreat inside
// backLine, otherwise hold (or catch up with the party if nothing is near).
func (b *sacerdotisaBrain) combatMove(intent *Intent, v View, nearest Foe, hasNearest bool) {
	if !hasNearest {
		if !withinFollowRadius(v.Self.Pos, v.PartyCentre) {
			moveTo(intent, v.Self.Pos, v.PartyCentre, v.Allies)
		}
		return
	}
	dist := rl.Vector2Distance(v.Self.Pos, nearest.Pos)
	if dist < panicLine {
		flee(intent, v.Self.Pos, nearest.Pos)
		return
	}
	if dist < backLine {
		away := direction(nearest.Pos, v.Self.Pos)
		dest := rl.Vector2Add(v.Self.Pos, rl.Vector2Scale(away, backLine))
		moveTo(intent, v.Self.Pos, dest, v.Allies)
	}
}

// recover is the calmRadius behaviour: no monster worth worrying about
// nearby, so close on whoever is hurt and keep firing until they are not,
// self-cast Sanctuary if she is the one who needs it, and otherwise just
// stay with the group — never firing into an empty field.
func (b *sacerdotisaBrain) recover(v View, hasWounded bool, wounded rl.Vector2) Intent {
	var intent Intent
	if !hasWounded {
		if !withinFollowRadius(v.Self.Pos, v.PartyCentre) {
			moveTo(&intent, v.Self.Pos, v.PartyCentre, v.Allies)
		}
		return intent
	}
	if rl.Vector2Distance(v.Self.Pos, wounded) > boltRange {
		moveTo(&intent, v.Self.Pos, wounded, v.Allies)
	}
	if rl.Vector2Distance(v.Self.Pos, wounded) <= boltRange {
		aim := wounded
		intent.Attack = &aim
	}
	return intent
}

// sanctuaryCast covers both triggers: the general one (two party members,
// herself included, below sanctuaryHealthCut — and she must already be near
// them, since the area drops at her feet) and the calm-only one (nothing to
// fight, and she herself could use it — her only self-heal).
func (b *sacerdotisaBrain) sanctuaryCast(v View, calm bool) (Cast, bool) {
	if !v.PrimaryReady {
		return Cast{}, false
	}
	count := woundedClusterAt(v.Self.Pos, v.Allies)
	if healthFrac(v.Self) < sanctuaryHealthCut {
		count++
	}
	if count >= sanctuaryAllies && nearWoundedCluster(v) {
		return Cast{SkillID: "sanctuary", Aim: v.Self.Pos}, true
	}
	if calm && healthFrac(v.Self) < 1 {
		return Cast{SkillID: "sanctuary", Aim: v.Self.Pos}, true
	}
	return Cast{}, false
}

func (b *sacerdotisaBrain) angelicCast(v View) (Cast, bool) {
	if !v.UltimateReady {
		return Cast{}, false
	}
	deadAllies, criticalCount := 0, 0
	for _, a := range v.Allies {
		if a.IsDead {
			deadAllies++
			continue
		}
		if healthFrac(a) < 0.35 {
			criticalCount++
		}
	}
	if deadAllies >= 1 || criticalCount >= 3 {
		return Cast{SkillID: "angelic_area", Aim: v.Self.Pos}, true
	}
	return Cast{}, false
}

func mostWoundedAlly(allies []Ally) (rl.Vector2, bool) {
	var pos rl.Vector2
	found := false
	lowest := float32(1)
	for _, a := range allies {
		if a.IsDead {
			continue
		}
		if f := healthFrac(a); f < 1 && (!found || f < lowest) {
			pos, lowest, found = a.Pos, f, true
		}
	}
	return pos, found
}

// nearWoundedCluster reports whether she is close enough to a wounded ally
// (or is herself the wounded one) that casting Sanctuary right now would
// actually land on somebody, instead of spending the 14s recharge on empty
// ground.
func nearWoundedCluster(v View) bool {
	if healthFrac(v.Self) < sanctuaryHealthCut {
		return true
	}
	for _, a := range v.Allies {
		if a.IsDead {
			continue
		}
		if healthFrac(a) < sanctuaryHealthCut && rl.Vector2Distance(v.Self.Pos, a.Pos) <= sanctuaryApproachRange {
			return true
		}
	}
	return false
}

// aimThrough returns a point beyond `through` on the ray from `from`
// through it, so a piercing projectile aimed there crosses `through` on the
// way to roughly where `beyond` is. Returns ok=false when through and from
// coincide.
func aimThrough(from, through, beyond rl.Vector2) (rl.Vector2, bool) {
	dir := direction(from, through)
	if dir.X == 0 && dir.Y == 0 {
		return beyond, false
	}
	dist := rl.Vector2Distance(from, beyond)
	if dist < rl.Vector2Distance(from, through) {
		dist = rl.Vector2Distance(from, through) + 200
	}
	return rl.Vector2Add(from, rl.Vector2Scale(dir, dist)), true
}

// woundedClusterAt counts allies below sanctuaryHealthCut, which is the
// signal that a Sanctuary cast here is worth its cooldown.
func woundedClusterAt(pos rl.Vector2, allies []Ally) int {
	count := 0
	for _, a := range allies {
		if a.IsDead {
			continue
		}
		if healthFrac(a) < sanctuaryHealthCut {
			count++
		}
	}
	return count
}
