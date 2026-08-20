package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// paladinaBrain holds the front line: stands between the nearest threat and
// the party's frailest member (monsters chase whoever is closest, so this
// is geometry, not a threat stat), shields under pressure, swings whenever
// something is in reach, and calls Avatar Divino when the group is falling.
type paladinaBrain struct {
	targetID string
	decideIn float32
}

func (b *paladinaBrain) Think(v View) Intent {
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

	// Position: between the target and the frailest ally, at sword range.
	dest := v.PartyCentre
	if hasTarget {
		frailest := frailestAlly(v.Self, v.Allies)
		mid := rl.Vector2Lerp(frailest, target.Pos, 0.6)
		dest = mid
	} else if !withinFollowRadius(v.Self.Pos, v.PartyCentre) {
		dest = v.PartyCentre
	} else {
		dest = v.Self.Pos
	}
	moveTo(&intent, v.Self.Pos, dest, v.Allies)

	// Sword sweep whenever something is within reach.
	if hasTarget && rl.Vector2Distance(v.Self.Pos, target.Pos) <= frontRing {
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
