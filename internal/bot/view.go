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
}

// View is everything a Brain is allowed to know when deciding what to do
// this tick. It is built fresh by the host every time tickBots runs.
type View struct {
	Self   Ally
	Allies []Ally // never includes Self
	Foes   []Foe
	Bounds world.Bounds

	PartyCentre rl.Vector2

	Portal       rl.Vector2
	PortalActive bool

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

	Dt float32
}
