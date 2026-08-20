package bot

import "github.com/WandenDourado/Legiao/internal/entity"

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

	// A4: retreat with hysteresis. retreatUnder/rejoinAbove are health
	// fractions; entering low and only rejoining well above it stops a bot
	// from flickering combat/retreat every tick a stray hit lands (plan
	// doc/plan_avanco_bots_e_gargula.md §A4).
	retreatUnder = 0.35
	rejoinAbove  = 0.60
	// paladinaRetreatUnder is lower than the shared retreatUnder: she is the
	// front line, and is gated further by having already cast Shield
	// (paladinaRetreatHysteresis, steering.go) before she is allowed to
	// fall back at all — a front line that retreats before trying to
	// mitigate abandons the group.
	paladinaRetreatUnder = 0.25
	// retreatExtraBack is how much further behind the formation post a
	// retreating bot falls back to, away from whatever enemy is nearest.
	retreatExtraBack = 300.0

	// A5: useful attack range for each class's basic-attack projectile —
	// AttackSpeed * Lifetime from entity/projectile.go. Firing past this
	// spends the attack's cadence on a bolt that expires before it can
	// possibly land. Sacerdotisa already has boltRange for this purpose.
	arqueiroAttackRange   = 1120.0 // ArrowAttackSpeed(700) * Lifetime(1.6)
	magoAttackRange       = 840.0  // FireballAttackSpeed(420) * Lifetime(2.0)
	necromanteAttackRange = 800.0  // NecroAttackSpeed(400) * Lifetime(2.0)

	// celestialApproachMargin is how far short of the Arqueiro ultimate's
	// real range (View.UltimateRange) he stops and fires instead of closing
	// the last stretch — launching exactly at max range risks the arrow
	// expiring a step short of a moving target (plan doc/plan_avanco_bots_e_gargula.md
	// §B4, point 1).
	celestialApproachMargin = 600.0

	// engageRadius is how far from the bot itself OR from the living
	// humans' centre an enemy still counts as "in the way" for target
	// selection (doc/plan_avanco_bots_e_gargula.md §A3, R2). A foe outside
	// this simply does not exist for the decision — it is the difference
	// between "there is a monster on the map" and "there is a monster on my
	// path". A little past the Arqueiro's own keep range (600) so he never
	// sees his current target vanish mid-kite, and about a screen wide.
	engageRadius = 900.0

	// AdvanceDirSmoothing is the exponential-smoothing rate (per second)
	// the party's advance direction chases the living humans' average
	// velocity. Low enough that one player strafing does not spin the whole
	// formation, high enough to follow an actual push within a second or
	// two (plan §A3, R3). Exported: network.Host.updateAdvanceDir computes
	// the smoothing itself (it owns the per-tick state, host_bot_tick.go),
	// but the rate is still a tuning number and belongs in this file.
	AdvanceDirSmoothing = 2.0

	// PortalEscortRadius is how close a living human must be to an open
	// portal — or already standing inside one — before a bot treats travel
	// as the decision for the tick (View.HumansAtPortal, travelDest). The
	// humans decide when the party moves on; the bots only escort
	// (doc/plan_avanco_bots_e_gargula.md §A2, cause 4). Exported:
	// network.buildBotView needs it — only the host can see the portal
	// rectangles and player positions this test requires.
	PortalEscortRadius = 1200.0
)

// formationOffset is a class's post relative to the human front, expressed
// in the advance direction's own frame: forward is positive AHEAD of the
// humans and negative behind them; lateral is the perpendicular offset, sign
// picks a side. Table from doc/plan_avanco_bots_e_gargula.md §A3, R3.
//
// Arqueiro and Mago take opposite lateral sides at the same depth so neither
// stands in the other's line of fire; Necromante mirrors the Arqueiro's side
// but sits further out so the three ranged posts do not stack.
type formationOffset struct {
	forward, lateral float32
}

var classFormation = map[entity.CharacterType]formationOffset{
	entity.CharPaladina:    {forward: 160},
	entity.CharArqueiro:    {forward: -250, lateral: -220},
	entity.CharMago:        {forward: -250, lateral: 220},
	entity.CharNecromante:  {forward: -250, lateral: -340},
	entity.CharSacerdotisa: {forward: -350},
}
