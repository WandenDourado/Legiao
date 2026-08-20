package bot

// Every clock and threshold the AIs use, in one place, so a tuning pass
// (plan §6, "afinacao") never has to go hunting through five class files.
// Values are the plan's informed guesses (doc/plan_bots_de_classe.md §13.5),
// meant to be adjusted here and nowhere else.
const (
	// decideEvery is how often a bot re-evaluates its target. Below this a
	// bot flickers between two equidistant targets instead of committing.
	decideEvery = 0.25

	// reactDelay is how long a bot waits before responding to a fresh fact
	// (an enemy entering range, an ally crossing a health threshold). Zero
	// would read as reflexes no human has; this is a human-plausible
	// window.
	reactDelay = 0.25

	// allySeparation is the minimum distance a bot tries to keep from
	// other party members, so a single area hit cannot wipe a stacked
	// group.
	allySeparation = 90.0

	// followRadius is how far a bot lets itself drift from the living
	// party's centre before beelining back, chosen to stay inside a
	// typical screen so it never looks abandoned.
	followRadius = 700.0

	// stuckDistance/stuckWindow define "stuck": less than stuckDistance of
	// net movement over stuckWindow seconds while a destination is active.
	stuckDistance = 40.0
	stuckWindow   = 1.5

	// Paladina.
	frontRing  = 140.0 // sword reach; how close is "in front"
	shieldFoes = 3     // enemies in melee range that call for a shield

	// Sacerdotisa. backLine keeps her outside melee and still inside the
	// bolt's ~760px useful range (HolyAttackSpeed * Lifetime,
	// entity/projectile.go); panicLine is "a monster is basically on top of
	// me, flee first, decide later"; calmRadius is the "field is clear
	// nearby, go heal" threshold — local peace, not the horde counter.
	backLine           = 420.0
	panicLine          = 260.0
	calmRadius         = 900.0
	sanctuaryAllies    = 2 // allies below sanctuaryHealthCut worth a cast
	sanctuaryHealthCut = 0.70
	// boltRange mirrors entity.HolyAttackSpeed(380) * Lifetime(2.0) —
	// the bolt travels ~760px before it expires on its own.
	boltRange = 760.0
	// sanctuaryApproachRange is how close she needs to be to a wounded ally
	// before casting Sanctuary is worth it; a little past skill.SanctuaryRadius
	// (200) since the area drops slightly ahead of her, not exactly on her feet.
	sanctuaryApproachRange = 220.0
	// lineBlockTolerance is the extra slack (beyond a foe's own radius)
	// used when deciding a monster sits on the heal line.
	lineBlockTolerance = 30.0
	// alignTolerance is how far off the heal ray a foe may sit and still
	// count as "beyond the wounded ally", worth aiming a hit at too.
	alignTolerance = 80.0

	// Arqueiro.
	arqueiroKeepRange    = 600.0
	arqueiroRetreatUnder = 320.0

	// Mago.
	magoKeepRange  = 450.0
	magoClusterMin = 3

	// Necromante.
	graveyardMin = 4 // incoming enemies that justify the 12s cooldown
)
