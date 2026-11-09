package bot

import rl "github.com/gen2brain/raylib-go/raylib"

// necromanteBrain controls the field: drops the graveyard lane (a 520x320
// strip that slows to 45% for 6s — graveyard.go) across the PATH between an
// incoming cluster and the party, not on top of it, since the slow is worth
// however long a monster spends walking through it. The basic bolt (with
// its own lifesteal) targets the nearest foe to stay alive. Legiao
// Espectral comes out when surrounded or when the map hands over a mass of
// enemies.
type necromanteBrain struct {
	targetID   string
	decideIn   float32
	retreating bool
}

func (b *necromanteBrain) Think(v View) Intent {
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
	// Only the "too close, back off" case actually overrides the
	// destination — the same asymmetry that already existed before travel:
	// a target within attack range but past magoKeepRange does not pull her
	// toward it, so heading to the portal while still firing (below) is the
	// right outcome (plan §A2 "andar e atirar").
	if hasTarget && rl.Vector2Distance(v.Self.Pos, target.Pos) < magoKeepRange {
		away := direction(target.Pos, v.Self.Pos)
		dest = rl.Vector2Add(v.Self.Pos, rl.Vector2Scale(away, magoKeepRange))
		usingTravel = false
	}
	if retreating {
		nearest, hasNearest := nearestFoe(v.Self.Pos, v.Foes)
		dest = retreatDest(v, nearest.Pos, hasNearest)
		usingTravel = false
	}
	finishMove(&intent, v, dest, usingTravel)

	if hasTarget && rl.Vector2Distance(v.Self.Pos, target.Pos) <= necromanteAttackRange {
		aim := target.Pos
		intent.Attack = &aim
	}

	if v.PrimaryReady && v.EnemiesLeft >= graveyardMin {
		if cluster, count := clusterCentre(v.Foes, fireballBlastRadius); count >= 2 {
			// The lane is cast as a direction from Self; aim at a point on
			// the path between the party and the cluster so the strip lands
			// ahead of the incoming enemies rather than under the group.
			pathPoint := rl.Vector2Lerp(v.PartyCentre, cluster, 0.5)
			intent.Skill = &Cast{SkillID: "graveyard", Aim: pathPoint}
			return intent
		}
	}

	surrounded := countFoesWithin(v.Self.Pos, v.Foes, frontRing*1.5) >= 3
	if v.UltimateReady && (surrounded || v.EnemiesLeft >= 10) {
		intent.Skill = &Cast{SkillID: "spectral_legion", Aim: v.Self.Pos}
	}

	return intent
}
