package entity

import (
	"github.com/WandenDourado/legiao/internal/game"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Player represents the player character.
type Player struct {
	Position rl.Vector2
	Velocity rl.Vector2
	Health   float32
	Speed    float32
	Radius   float32
}

// NewPlayer creates a new player with default values.
func NewPlayer() *Player {
	return &Player{
		Position: rl.NewVector2(float32(game.ScreenWidth/2), float32(game.ScreenHeight/2)),
		Velocity: rl.NewVector2(0, 0),
		Health:   100,
		Speed:    game.PlayerSpeed,
		Radius:   game.PlayerSize,
	}
}

// Update updates the player's position based on input direction and delta time.
// dir is a normalized vector (dx, dy) in the range [-1, 1] for each axis.
func (p *Player) Update(dir rl.Vector2, dt float32) {
	// Calculate velocity from input direction and speed
	p.Velocity.X = dir.X * p.Speed
	p.Velocity.Y = dir.Y * p.Speed

	// Update position
	p.Position.X += p.Velocity.X * dt
	p.Position.Y += p.Velocity.Y * dt

	// Keep player within screen bounds (optional, but good for testing)
	if p.Position.X < p.Radius {
		p.Position.X = p.Radius
	} else if p.Position.X > float32(game.ScreenWidth)-p.Radius {
		p.Position.X = float32(game.ScreenWidth) - p.Radius
	}
	if p.Position.Y < p.Radius {
		p.Position.Y = p.Radius
	} else if p.Position.Y > float32(game.ScreenHeight)-p.Radius {
		p.Position.Y = float32(game.ScreenHeight) - p.Radius
	}
}

// Draw renders the player as a circle.
func (p *Player) Draw() {
	rl.DrawCircleV(p.Position, p.Radius, rl.SkyBlue)
}
