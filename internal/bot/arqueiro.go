package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// arqueiroBrain kites at range: keeps its distance, targets whatever
// threatens the frailest ally (tie-broken toward the most hurt foe, since
// finishing beats spreading damage), volleys when three or more enemies
// bunch up, and saves Flechas Celestiais for a boss or a sentry.
type arqueiroBrain struct {
	targetID   string
	decideIn   float32
	retreating bool
}

func (b *arqueiroBrain) Think(v View) Intent {
	// Suprema pronta e gargula viva: prioridade acima de tudo o resto (plan
	// doc/plan_avanco_bots_e_gargula.md §B4). Enquanto a suprema nao estiver
	// liberada pela campanha, v.UltimateReady ja vem falso e este bloco
	// simplesmente nao dispara — nos mapas anteriores a gargula continua
	// sendo problema do grupo, nao dele sozinho.
	if v.UltimateReady {
		if sentry, ok := nearestSentry(v.Self.Pos, v.Foes); ok {
			return b.huntSentry(v, sentry)
		}
	}

	if b.decideIn <= 0 {
		b.decideIn = decideEvery
		if foe, ok := mostThreateningFoe(v.Self, v.Allies, engageableFoes(v)); ok {
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
		case dist < arqueiroRetreatUnder:
			away := direction(target.Pos, v.Self.Pos)
			dest = rl.Vector2Add(v.Self.Pos, rl.Vector2Scale(away, arqueiroRetreatUnder))
		case dist > arqueiroKeepRange:
			dest = target.Pos
		default:
			dest = v.Self.Pos
		}
		usingTravel = false
	}
	if retreating {
		// Still fires while falling back (plan §A4) — only the destination
		// changes, not the attack below.
		nearest, hasNearest := nearestFoe(v.Self.Pos, v.Foes)
		dest = retreatDest(v, nearest.Pos, hasNearest)
		usingTravel = false
	}
	finishMove(&intent, v, dest, usingTravel)

	if hasTarget && rl.Vector2Distance(v.Self.Pos, target.Pos) <= arqueiroAttackRange {
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

// huntSentry approaches within celestialApproachMargin of the ultimate's
// real range before firing — loosed any further out, the arrow expires on
// the way (plan §B4, point 1) — and, if a second sentry lines up behind the
// first, aims through both: one activation for two towers instead of one
// each (point 2, spending only one of the two CelestialCharges).
func (b *arqueiroBrain) huntSentry(v View, sentry Foe) Intent {
	intent := Intent{}
	approachRange := v.UltimateRange - celestialApproachMargin
	if approachRange < 0 {
		approachRange = 0
	}
	if rl.Vector2Distance(v.Self.Pos, sentry.HitCentre) > approachRange {
		moveTo(&intent, v.Self.Pos, sentry.HitCentre, v.Allies)
		return intent
	}
	// Mira o HitCentre, nao Pos (plan §B4, point 3): a folga do raio ainda
	// acertaria mirando nos pés, mas seria folga, não desenho.
	aim := sentry.HitCentre
	if second, ok := secondSentryAligned(v.Self.Pos, sentry, v.Foes); ok {
		if through, ok2 := aimThrough(v.Self.Pos, sentry.HitCentre, second.HitCentre); ok2 {
			aim = through
		}
	}
	intent.Skill = &Cast{SkillID: "celestial_arrows", Aim: aim}
	return intent
}

// nearestSentry returns the closest living sentry to pos. The ONE function
// in this package allowed to look for an IsSentry foe on purpose — see
// Foe.IsSentry's doc (view.go) for why every ordinary target-selection
// helper refuses to.
func nearestSentry(pos rl.Vector2, foes []Foe) (Foe, bool) {
	var best Foe
	bestDist := float32(-1)
	for _, f := range foes {
		if !f.IsSentry {
			continue
		}
		d := rl.Vector2Distance(pos, f.HitCentre)
		if bestDist < 0 || d < bestDist {
			best, bestDist = f, d
		}
	}
	return best, bestDist >= 0
}

// secondSentryAligned finds another living sentry roughly along the ray
// from `from` through `first`, past it — mirrors foeBeyondAlly's math
// (steering.go) restricted to sentries, for the "one arrow, two towers"
// case (plan §B4, point 2).
func secondSentryAligned(from rl.Vector2, first Foe, foes []Foe) (Foe, bool) {
	rayDir := direction(from, first.HitCentre)
	if rayDir.X == 0 && rayDir.Y == 0 {
		return Foe{}, false
	}
	firstDist := rl.Vector2Distance(from, first.HitCentre)
	var best Foe
	bestDist := float32(-1)
	for _, f := range foes {
		if !f.IsSentry || f.ID == first.ID {
			continue
		}
		toFoe := rl.Vector2Subtract(f.HitCentre, from)
		proj := toFoe.X*rayDir.X + toFoe.Y*rayDir.Y
		if proj <= firstDist {
			continue // not beyond the first sentry
		}
		onRay := rl.Vector2Add(from, rl.Vector2Scale(rayDir, proj))
		if rl.Vector2Distance(f.HitCentre, onRay) > alignTolerance {
			continue
		}
		if bestDist < 0 || proj < bestDist {
			best, bestDist = f, proj
		}
	}
	return best, bestDist >= 0
}

func countFoesWithin(pos rl.Vector2, foes []Foe, radius float32) int {
	count := 0
	for _, f := range foes {
		if f.IsSentry {
			continue
		}
		if rl.Vector2Distance(pos, f.Pos) <= radius {
			count++
		}
	}
	return count
}
