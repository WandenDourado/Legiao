package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// arqueiroBrain kites at range: keeps its distance, targets whatever
// threatens the frailest ally (tie-broken toward the most hurt foe, since
// finishing beats spreading damage), volleys when three or more enemies
// bunch up, and saves Flechas Celestiais for a boss or a sentry.
type arqueiroBrain struct {
	targetID string
	decideIn float32
}

func (b *arqueiroBrain) Think(v View) Intent {
	if b.decideIn <= 0 {
		b.decideIn = decideEvery
		if foe, ok := mostThreateningFoe(v.Self, v.Allies, v.Foes); ok {
			b.targetID = foe.ID
		} else {
			b.targetID = ""
		}
	} else {
		b.decideIn -= v.Dt
	}
	target, hasTarget := findFoe(v.Foes, b.targetID)

	intent := Intent{}

	dest := v.PartyCentre
	if hasTarget {
		dist := rl.Vector2Distance(v.Self.Pos, target.Pos)
		switch {
		case dist < arqueiroRetreatUnder:
			away := direction(target.Pos, v.Self.Pos)
			dest = rl.Vector2Add(v.Self.Pos, rl.Vector2Scale(away, arqueiroRetreatUnder))
		case dist > arqueiroKeepRange:
			dest = target.Pos
		default:
			dest = v.Self.Pos
		}
	} else if !withinFollowRadius(v.Self.Pos, v.PartyCentre) {
		dest = v.PartyCentre
	} else {
		dest = v.Self.Pos
	}
	moveTo(&intent, v.Self.Pos, dest, v.Allies)

	if hasTarget {
		aim := leadTarget(target, 0.25)
		intent.Attack = &aim
	}

	if v.PrimaryReady && countFoesWithin(v.Self.Pos, v.Foes, arqueiroKeepRange) >= 3 {
		aim := v.Self.Pos
		if hasTarget {
			aim = target.Pos
		}
		intent.Skill = &Cast{SkillID: "arrow_volley", Aim: aim}
		return intent
	}

	if v.UltimateReady && hasTarget && (target.IsBoss || target.AttackRange > 1000) {
		intent.Skill = &Cast{SkillID: "celestial_arrows", Aim: target.Pos}
	}

	return intent
}

func countFoesWithin(pos rl.Vector2, foes []Foe, radius float32) int {
	count := 0
	for _, f := range foes {
		if rl.Vector2Distance(pos, f.Pos) <= radius {
			count++
		}
	}
	return count
}
