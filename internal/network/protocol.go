package network

// Protocol defines message types and serialization for multiplayer.

import (
	"encoding/json"
	"time"
)

type MessageType string

const (
	MsgJoin        MessageType = "join"
	MsgInput       MessageType = "input"
	MsgStateUpdate MessageType = "state_update"
	MsgDisconnect  MessageType = "disconnect"

	// New message types for enemies, combat, and game state
	MsgEnemyUpdate      MessageType = "enemy_update"
	MsgProjectileUpdate MessageType = "projectile_update"
	MsgAttack           MessageType = "attack"
	MsgSkill            MessageType = "skill"
	MsgFireEvent        MessageType = "fire_event"
	MsgCombatEvent      MessageType = "combat_event"
	MsgGameOver         MessageType = "game_over"
	MsgRespawn          MessageType = "respawn"
	MsgSanctuary        MessageType = "sanctuary_event"
	MsgMelee            MessageType = "melee_event"
	MsgArrowVolley      MessageType = "arrow_volley_event"
	MsgUltimate         MessageType = "ultimate_event"
	// MsgCooldown mirrors the host's authoritative cooldowns so every client
	// can draw its own counters. The host owns the timers; clients only read.
	MsgCooldown MessageType = "cooldown"
	// MsgReset announces that the host restarted the stage after a Game Over.
	MsgReset MessageType = "reset_stage"
	// MsgTravel announces that the party moved to another map. The host is the
	// only one that decides it, and everybody obeys: without it a host walking
	// through a portal keeps simulating a map its clients have left.
	MsgTravel MessageType = "travel"
	// MsgTestMode toggles the no-cooldown test mode for one player. A client
	// sends it when the player presses F2; the host applies it to that
	// player's gates only.
	MsgTestMode MessageType = "test_mode"
	// MsgDialogue carries the line currently on screen. The host owns the
	// narrative: clients never send this, they only display what arrives.
	MsgDialogue MessageType = "dialogue"
	// MsgSentryOrb carries the shadow orb of the map-4 gargoyle sentry. It is
	// the first effect in the protocol that belongs to a MONSTER instead of a
	// player, which is why it does not travel on MsgSkill.
	MsgSentryOrb MessageType = "sentry_orb_event"
	// MsgCannonBall carries the fireball of the map-6 corridor cannons. Same
	// family as MsgSentryOrb (a monster's effect, not a player's), but the
	// projectile is straight-line instead of homing — see CannonBallPayload.
	MsgCannonBall MessageType = "cannon_ball_event"
	// MsgUltimateGrant announces that one character's ultimate is unlocked for
	// the REST OF THIS RUN ONLY, on top of whatever the campaign already
	// granted (internal/network/progression.go). Only the last-stand rescue
	// sends it, and only the host does — a client never grants itself
	// anything.
	MsgUltimateGrant MessageType = "ultimate_grant"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type JoinPayload struct {
	PlayerID  string `json:"player_id"`
	Color     string `json:"color"`
	Character string `json:"character"`
}

type InputPayload struct {
	PlayerID string `json:"player_id"`
	// Absolute position of the player
	X            int     `json:"x"`
	Y            int     `json:"y"`
	CurrentFrame int     `json:"current_frame"`
	CurrentRow   int     `json:"current_row"`
	IsSprinting  bool    `json:"is_sprinting"`
	VelX         float32 `json:"vel_x"`
	VelY         float32 `json:"vel_y"`
}

type StateUpdatePayload struct {
	Players []PlayerState `json:"players"`
}

// PlayerState represents a player's state in the game.
// Health, MaxHealth, and IsDead are omitted when empty for backward compatibility.
type PlayerState struct {
	PlayerID  string  `json:"player_id"`
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Color     string  `json:"color"`
	Character string  `json:"character"`
	Health    float32 `json:"health,omitempty"`
	MaxHealth float32 `json:"max_health,omitempty"`
	IsDead    bool    `json:"is_dead,omitempty"`
	// RespawnIn is the authoritative revive countdown in seconds, sent while
	// the player is dead. Clients display it; only the host counts it down.
	RespawnIn float32 `json:"respawn_in,omitempty"`
	// Absent marks a connection the host has lost but is still holding the
	// slot for (host_absence.go, ReconnectGrace). It is per-tick state like
	// IsDead, not identity, so it rides the ordinary snapshot with no
	// wire.go changes: a locked phone's status has to reach peers as fast as
	// any other health change.
	Absent bool `json:"absent,omitempty"`
	// absentSince is when the player went absent. Host-only bookkeeping for
	// ReconnectGrace; unexported so it never goes on the wire.
	absentSince time.Time
	// InPortal is true from the frame the host notices this (living) player
	// standing inside an open portal's rectangle until they leave it —
	// travel, or ESC/"SAIR" cancelling. It is per-tick state like IsDead and
	// Absent, not identity, so it rides the ordinary snapshot with no
	// wire.go changes (host_portal_presence.go).
	InPortal bool `json:"in_portal,omitempty"`
	// Animation state for sprite rendering
	CurrentFrame int     `json:"current_frame,omitempty"`
	CurrentRow   int     `json:"current_row,omitempty"`
	IsSprinting  bool    `json:"is_sprinting,omitempty"`
	VelX         float32 `json:"vel_x,omitempty"`
	VelY         float32 `json:"vel_y,omitempty"`
}

// EnemyState represents an enemy's state for network synchronization.
type EnemyState struct {
	EnemyID   string  `json:"enemy_id"`
	Type      string  `json:"type"` // "basic", "fast", "tank" - for future enemy types
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Health    float32 `json:"health"`
	MaxHealth float32 `json:"max_health"`
	Color     string  `json:"color"` // Blood red #8B0000
}

// WaveState mirrors the host's horde progression to clients for the HUD.
// Purely presentational: the host owns the real state machine.
type WaveState struct {
	Index        int     `json:"index"`        // 1-based wave number.
	Total        int     `json:"total"`        // How many waves the map has.
	Name         string  `json:"name"`         // Display name of the current wave.
	Phase        string  `json:"phase"`        // "fighting", "break" or "cleared".
	Remaining    int     `json:"remaining"`    // Alive plus not-yet-spawned.
	BreakTime    float32 `json:"break_time"`   // Seconds left in the break.
	Announcement string  `json:"announcement"` // Centred message during the break.
}

// EnemyUpdatePayload is broadcast by host to sync all enemies.
// The wave state rides along with it: both change every tick and are consumed
// together, so a separate message type would only add a second handler.
type EnemyUpdatePayload struct {
	Enemies []EnemyState `json:"enemies"`
	Wave    WaveState    `json:"wave"`
}

// ProjectileState represents a projectile's state.
type ProjectileState struct {
	ProjectileID string `json:"projectile_id"`
	OwnerID      string `json:"owner_id"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Active       bool   `json:"active"`
	Kind         string `json:"kind,omitempty"` // "fireball", "holy", "arrow", "basic"
	// DirX/DirY carry the normalized travel direction so clients can orient
	// directional projectiles (arrows, fireball tails).
	DirX float32 `json:"dir_x,omitempty"`
	DirY float32 `json:"dir_y,omitempty"`
}

// ProjectileUpdatePayload is broadcast by host to sync projectiles.
type ProjectileUpdatePayload struct {
	Projectiles []ProjectileState `json:"projectiles"`
}

// AttackPayload is sent by client when attack button is pressed.
type AttackPayload struct {
	PlayerID string `json:"player_id"`
	TargetX  int    `json:"target_x"` // Direction of attack
	TargetY  int    `json:"target_y"`
}

// SkillPayload is sent by client to activate a skill (e.g., fireball).
type SkillPayload struct {
	PlayerID string `json:"player_id"`
	Skill    string `json:"skill"` // "fireball"
	TargetX  int    `json:"target_x"`
	TargetY  int    `json:"target_y"`
}

// FireEventPayload carries fireball visual events to clients:
// "cast" (launch: origin + direction, spawns the traveling fireball) and
// "fire_explosion" (impact: explosion + ground fire).
type FireEventPayload struct {
	EventType string  `json:"event_type"` // "cast" or "fire_explosion"
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Radius    float32 `json:"radius"`
	OwnerID   string  `json:"owner_id,omitempty"`
	DirX      float32 `json:"dir_x,omitempty"`
	DirY      float32 `json:"dir_y,omitempty"`
}

// ArrowVolleyPayload carries an arrow-volley launch so clients can replicate
// the fan of arrows (origin + aim direction) for rendering.
type ArrowVolleyPayload struct {
	OwnerID string  `json:"owner_id"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	DirX    float32 `json:"dir_x"`
	DirY    float32 `json:"dir_y"`
}

// UltimateEventPayload carries an ultimate-skill activation so clients can
// replicate its visuals. Skill selects the effect; X/Y is the origin/center
// and DirX/DirY the aim direction (zero for untargeted ultimates).
type UltimateEventPayload struct {
	Skill   string  `json:"skill"`
	OwnerID string  `json:"owner_id"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	DirX    float32 `json:"dir_x"`
	DirY    float32 `json:"dir_y"`
}

// MeleePayload carries a Paladina sword-sweep launch so clients can replicate
// the swing (anchored to the owner, aimed toward Dir).
type MeleePayload struct {
	OwnerID string  `json:"owner_id"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	DirX    float32 `json:"dir_x"`
	DirY    float32 `json:"dir_y"`
	// Empowered marks a Divine Avatar swing (giant golden greatsword).
	Empowered bool `json:"empowered,omitempty"`
}

// SentryOrbPayload carries the gargoyle sentry's shadow orb to clients.
//
// "cast" traz a origem e o ID DO ALVO, e nao uma direcao: a trajetoria e
// perseguidora, entao o cliente refaz a curva sozinho contra a posicao
// replicada daquele jogador. "impact" traz o ponto do estouro e "expire" so o
// ID, para a esfera sumir sem estouro quando morreu de tempo.
type SentryOrbPayload struct {
	EventType string `json:"event_type"` // "cast", "impact" ou "expire"
	OrbID     string `json:"orb_id"`
	TargetID  string `json:"target_id,omitempty"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	// TTL so vai no "cast": com alcance global o tempo de voo varia por
	// disparo (host_sentry_orb.go, sentryOrbTTLFor), e sem ele o cliente
	// criaria a esfera com o padrao (9s) e a podaria antes de uma esfera
	// lancada de longe completar a viagem de verdade
	// (doc/plan_avanco_bots_e_gargula.md §B2).
	TTL float32 `json:"ttl,omitempty"`
}

// CannonBallPayload carries a corridor cannon's fireball to clients.
//
// "cast" traz a origem e a DIRECAO, e nao um ID de alvo: a bola nao persegue
// (ver host_cannon.go), entao o cliente refaz a trajetoria reta sozinho a
// partir de origem + direcao, como MsgFireEvent faz para a Bola de Fogo do
// Mago. "impact" traz o ponto do estouro e "expire" so o ID.
type CannonBallPayload struct {
	EventType string  `json:"event_type"` // "cast", "impact" ou "expire"
	BallID    string  `json:"ball_id"`
	X         int     `json:"x,omitempty"`
	Y         int     `json:"y,omitempty"`
	DirX      float32 `json:"dir_x,omitempty"`
	DirY      float32 `json:"dir_y,omitempty"`
}

// SanctuaryPayload carries a sanctuary spawn so clients can render it.
type SanctuaryPayload struct {
	SanctuaryID string `json:"sanctuary_id"`
	OwnerID     string `json:"owner_id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

// CombatEventPayload is broadcast by host for combat events.
type CombatEventPayload struct {
	EventType  string  `json:"event_type"`  // "damage", "death", "spawn"
	EntityID   string  `json:"entity_id"`   // player_id or enemy_id
	EntityType string  `json:"entity_type"` // "player" or "enemy"
	Value      float32 `json:"value"`       // damage amount or health
	KillerID   string  `json:"killer_id,omitempty"`
}

// GameOverPayload is broadcast when all players are dead.
type GameOverPayload struct {
	Message string `json:"message"`
}

// PlayerCooldowns is one player's remaining cooldowns. Skills maps skill ID ->
// seconds left; Attack is the basic-attack cadence gate derived from the
// character's attack speed. Only timers still running are sent, so a missing
// entry means "ready".
type PlayerCooldowns struct {
	Skills map[string]float32 `json:"skills,omitempty"`
	Attack float32            `json:"attack,omitempty"`
}

// CooldownPayload carries every player's cooldowns, keyed by player ID.
type CooldownPayload struct {
	Players map[string]PlayerCooldowns `json:"players"`
}

// ResetPayload tells clients the host restarted the stage: the horde starts
// over and everyone is revived at the spawn point carried here.
type ResetPayload struct {
	SpawnX int `json:"spawn_x"`
	SpawnY int `json:"spawn_y"`
}

// TravelPayload names the destination the party is moving to. The map file is
// the whole message: everything else about the destination (spawn, bounds,
// collision, portals) is read from it by whoever receives this.
//
// TargetSpawn is the name of an object in the destination's spawn layer; empty
// means the map's own player_spawn.
type TravelPayload struct {
	TargetMap   string `json:"target_map"`
	TargetSpawn string `json:"target_spawn,omitempty"`
	// Reconnect marks a catch-up sent to a single rejoining client
	// (sendCurrentMapTo, host_rejoin.go) rather than an actual party-wide
	// portal crossing. ApplyPendingTravel reads it to land the local player
	// back on the host-preserved position instead of the map's spawn point.
	Reconnect bool `json:"reconnect,omitempty"`
}

// TestModePayload toggles the no-cooldown test mode for a single player.
type TestModePayload struct {
	PlayerID string `json:"player_id"`
	Enabled  bool   `json:"enabled"`
}

// UltimateGrantPayload names the character whose ultimate the last-stand
// rescue just unlocked for the rest of this run.
type UltimateGrantPayload struct {
	Character string `json:"character"`
}

// RespawnPayload is sent when a player respawns.
type RespawnPayload struct {
	PlayerID string  `json:"player_id"`
	Health   float32 `json:"health"`
	X        int     `json:"x"`
	Y        int     `json:"y"`
}
