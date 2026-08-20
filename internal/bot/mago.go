package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// FireballBlastRadius mirrors skill.FireballExplosionRadius's ballpark for
// clustering purposes. It does not need to be exact — clusterCentre only
// uses it to decide which foe has the most neighbours, and a few pixels of
// slack does not change that ranking.
const fireballBlastRadius = 160.0

// magoBrain fights from mid-range, dropping Bola de Fogo (Q) on whatever
// cluster of foes is both dense and close to the party (fire never hurts
// allies, but it should not be wasted on a lone straggler), and calling
// Chuva de Meteoros once the field is a horde.
type magoBrain struct {
	targetID   string
	decideIn   float32
	retreating bool
}

func (b *magoBrain) Think(v View) Intent {
	if b.decideIn <= 0 {
		b.decideIn = decideEvery
		if foe, ok := nearestFoe(v.Self.Pos, engageableFoes(v)); ok {
			b.targetID = foe.ID
		} else {
			b.targetID = ""
		}
	} else {
		b.decideIn -= v.Dt
	}
	target, hasTarget := findFoe(v.Foes, b.targetID)
	retreating := retreatHysteresis(&b.retreating, healthFrac(v.Self), retreatUnder, rejoinAbove)

	intent := Intent{}

	dest := followDest(v)
	usingTravel := false
	if td, ok := travelDest(v); ok {
		dest, usingTravel = td, true
	}
	if hasTarget {
		dist := rl.Vector2Distance(v.Self.Pos, target.Pos)
		switch {
		case dist < magoKeepRange*0.5:
			away := direction(target.Pos, v.Self.Pos)
			dest = rl.Vector2Add(v.Self.Pos, rl.Vector2Scale(away, magoKeepRange*0.5))
		case dist > magoKeepRange:
			dest = target.Pos
		default:
			dest = v.Self.Pos
		}
		usingTravel = false
	}
	if retreating {
		nearest, hasNearest := nearestFoe(v.Self.Pos, v.Foes)
		dest = retreatDest(v, nearest.Pos, hasNearest)
		usingTravel = false
	}
	finishMove(&intent, v, dest, usingTravel)

	if hasTarget && rl.Vector2Distance(v.Self.Pos, target.Pos) <= magoAttackRange {
		aim := target.Pos
		intent.Attack = &aim
	}

	if v.PrimaryReady {
		if blast, count := clusterCentre(v.Foes, fireballBlastRadius); count >= magoClusterMin {
			intent.Skill = &Cast{SkillID: "fireball", Aim: blast}
			return intent
		}
	}

	if v.UltimateReady && v.EnemiesLeft >= 8 {
		aim := v.PartyCentre
		if hasTarget {
			aim = target.Pos
		}
		intent.Skill = &Cast{SkillID: "meteor_rain", Aim: aim}
	}

	return intent
}
