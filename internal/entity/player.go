package entity

import (
	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Game constants
const (
	// Screen dimensions
	ScreenWidth  = 1280
	ScreenHeight = 720

	// Target frames per second
	TargetFPS = 60

	// Player movement speed (units per second)
	PlayerSpeed = 200.0

	// Entity sizes (radius for circular collision)
	PlayerSize     = 20.0
	EnemySize      = 15.0
	ProjectileSize = 20.0

	// Respawn constants
	RespawnDelay         = 15.0 // Seconds before respawn
	RespawnHealthPercent = 0.15 // 15% health on respawn
)

// Sprint input thresholds
const (
	// SprintThreshold is the minimum joystick displacement (normalized 0-1)
	// that triggers sprint mode on Android.
	SprintThreshold = 0.70
)

// Player represents the player character.
type Player struct {
	// Color is a hex string like "#FF0000" representing the player's unique color.
	Color     string
	Position  rl.Vector2
	Velocity  rl.Vector2
	Health    float32
	MaxHealth float32
	Speed     float32
	Radius    float32
	IsDead    bool

	// Sprite animation fields
	WizardTexture rl.Texture2D
	AnimTimer     float32
	CurrentFrame  int
	CurrentRow    int
	LastRow       int
	IsSprinting   bool
	Initialized   bool
}

// NewPlayer creates a new player with default values.
func NewPlayer(spawn rl.Vector2) *Player {
	return &Player{
		Position:   spawn,
		Velocity:   rl.NewVector2(0, 0),
		Health:     100,
		MaxHealth:  100,
		Speed:      PlayerSpeed,
		Radius:     PlayerSize,
		IsDead:     false,
		CurrentRow: RowWalkDown,
		LastRow:    RowWalkDown,
	}
}

// Update updates the player's position based on input direction and delta time.
// dir is a normalized vector (dx, dy) in the range [-1, 1] for each axis.
func (p *Player) Update(dir rl.Vector2, dt float32, bounds world.Bounds) {
	p.Velocity.X = dir.X * p.Speed
	p.Velocity.Y = dir.Y * p.Speed

	p.Position.X += p.Velocity.X * dt
	p.Position.Y += p.Velocity.Y * dt

	if p.Position.X < 0 {
		p.Position.X = 0
	} else if p.Position.X > bounds.Width {
		p.Position.X = bounds.Width
	}
	if p.Position.Y < 0 {
		p.Position.Y = 0
	} else if p.Position.Y > bounds.Height {
		p.Position.Y = bounds.Height
	}

	p.updateAnimation(dir, dt)
}

// Respawn resets the player for respawn with the given health percentage.
func (p *Player) Respawn(healthPercent float32, x, y float32) {
	p.Health = p.MaxHealth * healthPercent
	p.Position.X = x
	p.Position.Y = y
	p.IsDead = false
}
