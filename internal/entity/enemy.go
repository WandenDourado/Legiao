package entity

import (
	"crypto/rand"
	"encoding/base64"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// EnemyType represents different types of enemies for future extensibility.
type EnemyType string

const (
	EnemyTypeBasic EnemyType = "basic"
	// Future enemy types:
	// EnemyTypeFast  EnemyType = "fast"
	// EnemyTypeTank  EnemyType = "tank"
)

// PlayerState is a minimal version for enemy AI - defined here to avoid import cycle.
type PlayerState struct {
	PlayerID string
	X        int
	Y        int
	Color    string
	Health   float32
	MaxHealth float32
	IsDead   bool
}

// Enemy represents an enemy entity in the game.
type Enemy struct {
	ID            string
	Type          EnemyType
	Position      rl.Vector2
	Velocity      rl.Vector2
	Health        float32
	MaxHealth     float32
	Speed         float32
	Radius        float32
	Color         string // Blood red #8B0000 for differentiation from players
	AttackDamage  float32
	AttackRange   float32
	AttackCooldown float32
	attackTimer   float32
	IsActive      bool
}

// NewEnemy creates a new enemy with the given type at position (x, y).
func NewEnemy(enemyType EnemyType, x, y float32) *Enemy {
	return &Enemy{
		ID:            generateID(),
		Type:          enemyType,
		Position:      rl.NewVector2(x, y),
		Health:        100.0,
		MaxHealth:     100.0,
		Speed:         100.0,
		Radius:        EnemySize,
		Color:         "#8B0000", // Blood red
		AttackDamage:  10.0,
		AttackRange:   25.0,
		AttackCooldown: 1.0,
		attackTimer:   0,
		IsActive:      true,
	}
}

// Update updates the enemy's AI and movement based on the nearest player.
// Returns true if the enemy is in range and can attack.
func (e *Enemy) Update(dt float32, players []PlayerState) bool {
	if !e.IsActive {
		return false
	}

	nearest := e.FindNearestPlayer(players)
	if nearest == nil {
		return false
	}

	targetPos := rl.NewVector2(float32(nearest.X), float32(nearest.Y))
	e.MoveTowardTarget(targetPos, dt)

	// Check if in attack range and update attack timer
	if e.IsInAttackRange(nearest) {
		e.attackTimer -= dt
		if e.attackTimer <= 0 {
			e.attackTimer = e.AttackCooldown
			return true // Enemy attacks this frame
		}
	}
	return false
}

// FindNearestPlayer returns the closest living player, or nil if none.
func (e *Enemy) FindNearestPlayer(players []PlayerState) *PlayerState {
	var nearest *PlayerState
	minDist := float32(math.MaxFloat32)

	for _, p := range players {
		if p.IsDead {
			continue
		}
		playerPos := rl.NewVector2(float32(p.X), float32(p.Y))
		dist := rl.Vector2Distance(e.Position, playerPos)
		if dist < minDist {
			minDist = dist
			nearest = &p
		}
	}
	return nearest
}

// MoveTowardTarget moves the enemy toward the target position.
func (e *Enemy) MoveTowardTarget(target rl.Vector2, dt float32) {
	direction := rl.Vector2Subtract(target, e.Position)
	length := rl.Vector2Length(direction)
	if length > 0 {
		direction = rl.Vector2Scale(direction, 1.0/length)
		e.Velocity = rl.Vector2Scale(direction, e.Speed)
		e.Position.X += e.Velocity.X * dt
		e.Position.Y += e.Velocity.Y * dt
	}
}

// IsInAttackRange returns true if the enemy is close enough to attack.
func (e *Enemy) IsInAttackRange(player *PlayerState) bool {
	playerPos := rl.NewVector2(float32(player.X), float32(player.Y))
	dist := rl.Vector2Distance(e.Position, playerPos)
	return dist <= e.AttackRange+e.Radius
}

// TakeDamage applies damage to the enemy. Returns true if the enemy died.
func (e *Enemy) TakeDamage(damage float32) bool {
	e.Health -= damage
	if e.Health <= 0 {
		e.Health = 0
		e.IsActive = false
		return true
	}
	return false
}

// Draw renders the enemy as a circle with blood red color.
func (e *Enemy) Draw() {
	if !e.IsActive {
		return
	}
	col := hexToColor(e.Color)
	rl.DrawCircleV(e.Position, e.Radius, col)
	rl.DrawCircleLinesV(e.Position, e.Radius, rl.Fade(rl.Black, 0.5))
}

// DrawHealthBar draws a small health bar above the enemy.
func (e *Enemy) DrawHealthBar() {
	if !e.IsActive {
		return
	}
	barWidth := e.Radius * 2
	barHeight := 3.0
	x := e.Position.X - e.Radius
	y := e.Position.Y - e.Radius - 8

	rl.DrawRectangle(int32(x), int32(y), int32(barWidth), int32(barHeight), rl.Fade(rl.Black, 0.5))

	healthPercent := e.Health / e.MaxHealth
	fillWidth := barWidth * healthPercent
	healthColor := rl.Red
	if healthPercent > 0.5 {
		healthColor = rl.Green
	} else if healthPercent > 0.25 {
		healthColor = rl.Orange
	}
	rl.DrawRectangle(int32(x), int32(y), int32(fillWidth), int32(barHeight), healthColor)
}

// generateID creates a unique ID for enemies and projectiles.
func generateID() string {
	randBytes := make([]byte, 8)
	rand.Read(randBytes)
	return base64.URLEncoding.EncodeToString(randBytes) + "-" + time.Now().Format("150405.000000")
}

// DrawEnemyAt renders an enemy at a specific position with a given color.
func DrawEnemyAt(x, y float32, color string, radius float32) {
	col := hexToColor(color)
	rl.DrawCircleV(rl.NewVector2(x, y), radius, col)
	rl.DrawCircleLinesV(rl.NewVector2(x, y), radius, rl.Fade(rl.Black, 0.5))
}

// DrawEnemyHealthBarAt draws a health bar for an enemy at the given position.
func DrawEnemyHealthBarAt(x, y, health, maxHealth, radius float32) {
	if maxHealth <= 0 {
		return
	}
	barWidth := radius * 2
	barHeight := 3.0
	barX := x - radius
	barY := y - radius - 8

	rl.DrawRectangle(int32(barX), int32(barY), int32(barWidth), int32(barHeight), rl.Fade(rl.Black, 0.5))

	healthPercent := health / maxHealth
	fillWidth := barWidth * healthPercent
	healthColor := rl.Red
	if healthPercent > 0.5 {
		healthColor = rl.Green
	} else if healthPercent > 0.25 {
		healthColor = rl.Orange
	}
	rl.DrawRectangle(int32(barX), int32(barY), int32(fillWidth), int32(barHeight), healthColor)
}
