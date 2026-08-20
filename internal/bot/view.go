// Package bot implements the class AIs that fill a vacant character slot
// when no human plays it. See doc/plan_bots_de_classe.md for the design.
//
// This package must never import internal/network: network imports bot to
// drive it, and the reverse would close a cycle. Everything the AI needs to
// know about progression, cooldowns and the portal arrives inside the View,
// filled in by the host.
package bot

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Ally is a party member as the bot's steering and targeting see it: itself
// or a human, alive or dead, host or remote.
type Ally struct {
	ID                string
	Char              entity.CharacterType
	Pos               rl.Vector2
	Health, MaxHealth float32
	IsDead            bool
	IsBot             bool
}

// Foe is an enemy as the bot's targeting sees it.
type Foe struct {
	ID                  string
	Pos, Vel            rl.Vector2
	Health, MaxHealth   float32
	AttackRange, Radius float32
	IsBoss              bool
	// IsSentry marks a castle sentry (global range, entity.EnemyTypeCastleSentry).
	// Every ordinary target-selection helper in this package (nearestFoe,
	// mostThreateningFoe, clusterCentre, countFoesWithin, foeBlocksLine,
	// foeBeyondAlly, anyFoeWithin) skips a sentry foe outright — no bot may
	// spend a basic attack or a common skill on one, since the host itself
	// refuses that damage (host.go's checkProjectileCollisions /
	// checkEnemyPlayerCollisions). The ONE exception is the Arqueiro's
	// ultimate priority rule (arqueiro.go, nearestSentry), which is the only
	// function allowed to look for IsSentry foes on purpose (plan
	// doc/plan_avanco_bots_e_gargula.md §B4, point 4).
	IsSentry bool
	// HitCentre is where a projectile actually needs to land — entity.Enemy's
	// HitCenter(), not Pos. The sentry's HitOffsetY (-67) means aiming at Pos
	// would send a celestial arrow over the creature's head.
	HitCentre rl.Vector2
}

// View is everything a Brain is allowed to know when deciding what to do
// this tick. It is built fresh by the host every time tickBots runs.
type View struct {
	Self   Ally
	Allies []Ally // never includes Self
	Foes   []Foe
	Bounds world.Bounds

	// PartyCentre is the average living position of EVERY party member,
	// bots included. Only combat/aim math should use it — an advancing bot
	// pulls its own weight into this average, which is exactly the feedback
	// loop HumanCentre exists to keep out of movement decisions (plan
	// doc/plan_avanco_bots_e_gargula.md §A2, cause 2).
	PartyCentre rl.Vector2
	// HumanCentre is the average living position of HUMAN party members
	// only. Every "follow the group" destination uses this, never
	// PartyCentre: it is the actual reference for "the party is advancing"
	// (plan §A3, R1). Meaningless when HasHumans is false.
	HumanCentre rl.Vector2
	// HasHumans is whether any human is alive right now. False means the
	// run is being decided by respawn timers, not movement — a bot should
	// hold its ground and keep fighting instead of picking a new leader
	// among the other bots.
	HasHumans bool
	// AdvanceDir is the living humans' average heading, smoothed over time
	// and held at its last value while the party is stationary — never the
	// zero vector once it has been established at least once (plan §A3,
	// R3). Formation posts are expressed relative to this, not to a fixed
	// world axis, so the escort re-orients itself as the party turns.
	AdvanceDir rl.Vector2

	Portal       rl.Vector2
	PortalActive bool
	// HumansAtPortal is whether at least one living human is already
	// standing inside an open portal, or within bot.PortalEscortRadius of
	// one (network.buildBotView tests this using the same box test
	// host_portal_presence.go already runs — a human with InPortal counts
	// directly, no distance measured). travelDest (steering.go) requires
	// this before treating the portal as a destination: without it a bot
	// would march to the door alone the instant a garrison map's own
	// "no waves" rule opens it (doc/plan_avanco_bots_e_gargula.md §A2,
	// cause 4) — the humans decide when the party travels.
	HumansAtPortal bool

	// EnemiesLeft is the horde still in the field plus what is yet to spawn.
	EnemiesLeft int

	// PrimaryReady is whether the class's Q skill is off cooldown.
	PrimaryReady bool
	// UltimateReady is whether both the cooldown AND the narrative gate
	// (progression) allow the ultimate to be cast right now.
	UltimateReady bool
	// RescueRecent is true for a short window after the last-stand rescue
	// just granted this bot its ultimate — the trigger that earned the
	// grant is still true, and the bot should not wait out its normal
	// reaction delay before using it.
	RescueRecent bool
	// UltimateRange is the current class's ultimate projectile range, when
	// that ultimate has one (skill.CelestialRange for the Arqueiro; zero
	// otherwise). internal/bot never imports internal/skill (plan §B4,
	// contract point 2 — "alcance da suprema chega pela View"), so this is
	// how the Arqueiro's sentry-hunting priority rule knows how close it
	// needs to walk before firing.
	UltimateRange float32

	Dt float32
}
