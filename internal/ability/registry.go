// Package ability implements character skills as data-driven, pluggable
// strategies (Strategy pattern) registered in a global registry (Registry
// pattern). Adding a new character with new skills requires only registering
// the skill and wiring it to the character — no per-character if/switch in
// input, host, or renderer code.
package ability

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// CastContext carries everything a skill needs to resolve its effect on the
// authoritative host. The host satisfies the HostLike interface below.
type CastContext struct {
	Host      HostLike
	PlayerID  string
	Character entity.CharacterType
	Position  rl.Vector2
	Aim       rl.Vector2 // world-space aim point (for targeted skills)
}

// HostLike is the minimal surface a skill needs from the host. It decouples
// skill implementations from the network package.
type HostLike interface {
	// SkillManager returns the authoritative skill manager (fireballs, sanctuaries).
	SkillManager() *skill.Manager
	// PlayerState returns the live state for playerID (nil if unknown/dead).
	PlayerState(playerID string) *PlayerStateView
	// BroadcastSkill spawns/visualizes the skill for every client.
	BroadcastSkill(skillID string, ownerID string, center rl.Vector2)
	// BroadcastSkillDir is BroadcastSkill for aimed skills that also need the
	// launch direction replicated on clients (e.g., arrow volley).
	BroadcastSkillDir(skillID string, ownerID string, origin, dir rl.Vector2)
}

// PlayerStateView is a read-only snapshot a skill may need about the caster.
type PlayerStateView struct {
	Character string
	X         float32
	Y         float32
	IsDead    bool
}

// Skill is a castable character ability.
type Skill interface {
	// ID is the wire/protocol identifier ("fireball", "sanctuary").
	ID() string
	// Cooldown is the per-caster cooldown in seconds.
	Cooldown() float32
	// Cast resolves the skill effect on the host. It must be safe to call
	// only when the caster is allowed to cast (host validates gating).
	Cast(ctx *CastContext)
	// Draw renders the skill's authoring/visuals for a given manager.
	Draw(m *skill.Manager)
}

// registry maps skill ID -> implementation.
var registry = map[string]Skill{}

// characterAbilities maps character type -> ordered list of skill IDs.
var characterAbilities = map[entity.CharacterType][]string{}

// RegisterSkill adds a skill implementation to the global registry.
func RegisterSkill(s Skill) {
	registry[s.ID()] = s
}

// BindAbility associates a skill ID with a character type. Call once per
// (character, skill) pair, typically from the character's registration site.
func BindAbility(char entity.CharacterType, skillID string) {
	characterAbilities[char] = append(characterAbilities[char], skillID)
}

// Get returns the registered skill for an ID, or nil if unknown.
func Get(skillID string) Skill {
	return registry[skillID]
}

// AbilitiesOf returns the ordered skill IDs bound to a character.
func AbilitiesOf(char entity.CharacterType) []string {
	return characterAbilities[char]
}

// PrimaryAbilityOf returns the first ability bound to a character, or "".
func PrimaryAbilityOf(char entity.CharacterType) string {
	return AbilityAt(char, 0)
}

// UltimateAbilityOf returns the second ability bound to a character (its
// ultimate/supreme skill), or "".
func UltimateAbilityOf(char entity.CharacterType) string {
	return AbilityAt(char, 1)
}

// AbilityAt returns the idx-th ability bound to a character, or "".
func AbilityAt(char entity.CharacterType, idx int) string {
	abs := characterAbilities[char]
	if idx < 0 || idx >= len(abs) {
		return ""
	}
	return abs[idx]
}

// Charged is an optional interface for skills with multiple casts per
// cooldown: the caster gets Charges() casts, and the cooldown is armed only
// after the last charge is spent (e.g., the Arqueiro's two celestial arrows).
type Charged interface {
	Charges() int
}

// CastByID resolves skillID on the host. Returns false if the skill is
// unknown (the caller is responsible for character/cooldown gating).
func CastByID(skillID string, ctx *CastContext) bool {
	s := registry[skillID]
	if s == nil {
		return false
	}
	s.Cast(ctx)
	return true
}

// DrawAll renders the visuals of every registered skill for the given manager.
// Used by the renderer so newly added skills are drawn without touching the
// draw code.
func DrawAll(m *skill.Manager) {
	for _, s := range registry {
		s.Draw(m)
	}
}
