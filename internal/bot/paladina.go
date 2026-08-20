package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// paladinaBrain holds the front line: stands between the nearest threat and
// the party's frailest member (monsters chase whoever is closest, so this
// is geometry, not a threat stat), shields under pressure, swings whenever
// something is in reach, and calls Avatar Divino when the group is falling.
type paladinaBrain struct {
	targetID   string
	decideIn   float32
	retreating bool
}

func (b *paladinaBrain) Think(v View) Intent {
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
	// Gated on Shield already being spent (v.PrimaryReady false): a front
	// line that retreats before trying to mitigate abandons the group
	// (plan §A4).
	retreating := paladinaRetreatHysteresis(&b.retreating, healthFrac(v.Self), v.PrimaryReady)

	intent := Intent{}

	// Position: between the target and the frailest ally, at sword range.
	dest := followDest(v)
	usingTravel := false
	if td, ok := travelDest(v); ok {
		dest, usingTravel = td, true
	}
	if hasTarget {
		frailest := frailestAlly(v.Self, v.Allies)
		mid := rl.Vector2Lerp(frailest, target.Pos, 0.6)
		dest = mid
		usingTravel = false
	}
	if retreating {
		nearest, hasNearest := nearestFoe(v.Self.Pos, v.Foes)
		dest = retreatDest(v, nearest.Pos, hasNearest)
		usingTravel = false
	}
	finishMove(&intent, v, dest, usingTravel)

	// Sword sweep whenever something is within reach — never while
	// retreating, or she would keep swinging with her back turned instead
	// of actually opening distance (plan §A4).
	if !retreating && hasTarget && rl.Vector2Distance(v.Self.Pos, target.Pos) <= frontRing {
		aim := target.Pos
		intent.Attack = &aim
	}

	// Shield: surrounded, or badly hurt.
	inMeleeRange := 0
	for _, f := range v.Foes {
		if rl.Vector2Distance(v.Self.Pos, f.Pos) <= frontRing {
			inMeleeRange++
		}
	}
	lowHealth := healthFrac(v.Self) < 0.5
	if v.PrimaryReady && (inMeleeRange >= shieldFoes || lowHealth) {
		intent.Skill = &Cast{SkillID: "shield", Aim: v.Self.Pos}
		return intent
	}

	// Avatar Divino: the group is falling apart.
	if v.UltimateReady && groupIsFalling(v.Self, v.Allies, v.EnemiesLeft) {
		intent.Skill = &Cast{SkillID: "divine_avatar", Aim: v.Self.Pos}
	}

	return intent
}

// findFoe looks up a foe by ID; returns ok=false if it died or was never
// set (decideIn just reset targetID to "").
func findFoe(foes []Foe, id string) (Foe, bool) {
	if id == "" {
		return Foe{}, false
	}
	for _, f := range foes {
		if f.ID == id {
			return f, true
		}
	}
	return Foe{}, false
}

func frailestAlly(self Ally, allies []Ally) rl.Vector2 {
	frailest := self
	frac := healthFrac(self)
	for _, a := range allies {
		if a.IsDead {
			continue
		}
		if f := healthFrac(a); f < frac {
			frailest, frac = a, f
		}
	}
	return frailest.Pos
}
