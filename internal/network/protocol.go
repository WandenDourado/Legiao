package network

// Protocol defines message types and serialization for multiplayer.

import (
    "encoding/json"
)

type MessageType string

const (
    MsgJoin        MessageType = "join"
    MsgInput       MessageType = "input"
    MsgStateUpdate MessageType = "state_update"
    MsgDisconnect  MessageType = "disconnect"

    // New message types for enemies, combat, and game state
    MsgEnemyUpdate  MessageType = "enemy_update"
    MsgAttack       MessageType = "attack"
    MsgCombatEvent  MessageType = "combat_event"
    MsgGameOver     MessageType = "game_over"
    MsgRespawn      MessageType = "respawn"
)

type Message struct {
    Type    MessageType   `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

type JoinPayload struct {
    PlayerID string `json:"player_id"`
    Color    string `json:"color"`
}

type InputPayload struct {
    PlayerID string `json:"player_id"`
    // Absolute position of the player
    X int `json:"x"`
    Y int `json:"y"`
}

type StateUpdatePayload struct {
    Players []PlayerState `json:"players"`
}

// PlayerState represents a player's state in the game.
// Health, MaxHealth, and IsDead are omitted when empty for backward compatibility.
type PlayerState struct {
    PlayerID string  `json:"player_id"`
    X        int     `json:"x"`
    Y        int     `json:"y"`
    Color    string  `json:"color"`
    Health   float32 `json:"health,omitempty"`
    MaxHealth float32 `json:"max_health,omitempty"`
    IsDead   bool    `json:"is_dead,omitempty"`
}

// EnemyState represents an enemy's state for network synchronization.
type EnemyState struct {
    EnemyID   string  `json:"enemy_id"`
    Type      string  `json:"type"`       // "basic", "fast", "tank" - for future enemy types
    X         int     `json:"x"`
    Y         int     `json:"y"`
    Health    float32 `json:"health"`
    MaxHealth float32 `json:"max_health"`
    Color     string  `json:"color"`      // Blood red #8B0000
}

// EnemyUpdatePayload is broadcast by host to sync all enemies.
type EnemyUpdatePayload struct {
    Enemies []EnemyState `json:"enemies"`
}

// ProjectileState represents a projectile's state.
type ProjectileState struct {
    ProjectileID string  `json:"projectile_id"`
    OwnerID      string  `json:"owner_id"`
    X            int     `json:"x"`
    Y            int     `json:"y"`
    Active       bool    `json:"active"`
}

// ProjectileUpdatePayload is broadcast by host to sync projectiles.
type ProjectileUpdatePayload struct {
    Projectiles []ProjectileState `json:"projectiles"`
}

// AttackPayload is sent by client when attack button is pressed.
type AttackPayload struct {
    PlayerID string  `json:"player_id"`
    TargetX  int     `json:"target_x"`   // Direction of attack
    TargetY  int     `json:"target_y"`
}

// CombatEventPayload is broadcast by host for combat events.
type CombatEventPayload struct {
    EventType string  `json:"event_type"`   // "damage", "death", "spawn"
    EntityID  string  `json:"entity_id"`    // player_id or enemy_id
    EntityType string `json:"entity_type"`  // "player" or "enemy"
    Value     float32 `json:"value"`        // damage amount or health
    KillerID  string  `json:"killer_id,omitempty"`
}

// GameOverPayload is broadcast when all players are dead.
type GameOverPayload struct {
    Message string `json:"message"`
}

// RespawnPayload is sent when a player respawns.
type RespawnPayload struct {
    PlayerID string  `json:"player_id"`
    Health   float32 `json:"health"`
    X        int     `json:"x"`
    Y        int     `json:"y"`
}

