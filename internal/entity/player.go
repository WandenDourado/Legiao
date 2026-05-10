package entity

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Wizard sprite sheet constants
const (
	WizardFrameWidth  = 163
	WizardFrameHeight = 240
	WizardColumns     = 6
	WizardRows        = 4
	WizardFrameTime   = 0.12 // seconds per frame

	// Sprite rows
	RowWalkUp       = 0
	RowWalkDown     = 1
	RowWalkLeft     = 2
	RowWalkDownLeft = 3
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
	ProjectileSize = 5.0
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
	Initialized   bool
}

// NewPlayer creates a new player with default values.
func NewPlayer() *Player {
	return &Player{
		Position:   rl.NewVector2(float32(ScreenWidth/2), float32(ScreenHeight/2)),
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

// InitSprite loads the wizard texture for the player.
func (p *Player) InitSprite() {
	p.WizardTexture = rl.LoadTexture("assets/sprites/wizard/wizard.png")
	p.Initialized = true
}

// UnloadSprite unloads the wizard texture.
func (p *Player) UnloadSprite() {
	if p.Initialized {
		rl.UnloadTexture(p.WizardTexture)
		p.Initialized = false
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
	} else if p.Position.X > float32(ScreenWidth)-p.Radius {
		p.Position.X = float32(ScreenWidth) - p.Radius
	}
	if p.Position.Y < p.Radius {
		p.Position.Y = p.Radius
	} else if p.Position.Y > float32(ScreenHeight)-p.Radius {
		p.Position.Y = float32(ScreenHeight) - p.Radius
	}

	// Update sprite animation
	p.updateAnimation(dir, dt)
}

// updateAnimation determines the correct sprite row and advances the frame timer.
func (p *Player) updateAnimation(dir rl.Vector2, dt float32) {
	isMoving := dir.X != 0 || dir.Y != 0

	if isMoving {
		// Determine the correct row based on input direction
		// Prioritize the dominant axis for diagonal movement
		absX := dir.X
		if absX < 0 {
			absX = -absX
		}
		absY := dir.Y
		if absY < 0 {
			absY = -absY
		}

		if dir.Y < 0 && absY >= absX {
			// Moving up (or up-diagonal)
			p.CurrentRow = RowWalkUp
		} else if dir.Y > 0 && dir.X < 0 && absX > absY*0.5 {
			// Moving down-left diagonal
			p.CurrentRow = RowWalkDownLeft
		} else if dir.X < 0 {
			// Moving left
			p.CurrentRow = RowWalkLeft
		} else if dir.Y > 0 {
			// Moving down
			p.CurrentRow = RowWalkDown
		} else if dir.X > 0 {
			// Moving right (mirror of left)
			p.CurrentRow = RowWalkLeft
		}

		p.LastRow = p.CurrentRow

		// Advance animation timer
		p.AnimTimer += dt
		if p.AnimTimer >= WizardFrameTime {
			p.AnimTimer -= WizardFrameTime
			p.CurrentFrame++
			if p.CurrentFrame >= WizardColumns {
				p.CurrentFrame = 0
			}
		}
	} else {
		// Idle: freeze on frame 0 of the last active direction
		p.CurrentFrame = 0
		p.AnimTimer = 0
		p.CurrentRow = p.LastRow
	}
}

// Respawn resets the player for respawn with the given health percentage.
func (p *Player) Respawn(healthPercent float32, x, y float32) {
	p.Health = p.MaxHealth * healthPercent
	p.Position.X = x
	p.Position.Y = y
	p.IsDead = false
}

// Draw renders the player as an animated sprite from the wizard sprite sheet.
func (p *Player) Draw() {
	if !p.Initialized {
		// Fallback to circle if texture not loaded
		col := hexToColor(p.Color)
		rl.DrawCircleV(p.Position, p.Radius, col)
		return
	}

	// Calculate source rectangle from sprite sheet
	sourceRect := rl.NewRectangle(
		float32(p.CurrentFrame*WizardFrameWidth),
		float32(p.CurrentRow*WizardFrameHeight),
		WizardFrameWidth,
		WizardFrameHeight,
	)

	// Mirror for right-facing (row 2 with negative width)
	if p.Velocity.X > 0 && p.CurrentRow == RowWalkLeft {
		sourceRect.Width = -WizardFrameWidth
	}

	// Destination rectangle centered on player position
	destRect := rl.NewRectangle(
		p.Position.X-WizardFrameWidth/2,
		p.Position.Y-WizardFrameHeight/2,
		WizardFrameWidth,
		WizardFrameHeight,
	)

	rl.DrawTexturePro(p.WizardTexture, sourceRect, destRect, rl.NewVector2(0, 0), 0, rl.White)
}

// DrawPlayerAt renders a player at a specific position with a given color.
// For remote players, renders as a circle (they don't have local animation state).
func DrawPlayerAt(x, y float32, color string, radius float32) {
	col := hexToColor(color)
	rl.DrawCircleV(rl.NewVector2(x, y), radius, col)
}

// hexToColor converts a hex color string (e.g., "#FF5733") to rl.Color.
func hexToColor(hex string) rl.Color {
	if hex == "" {
		return rl.SkyBlue
	}
	// Remove the "#" prefix if present
	hex = strings.TrimPrefix(hex, "#")
	// Parse the hex string
	r, g, b := uint8(0), uint8(0), uint8(0)
	if len(hex) >= 2 {
		fmt.Sscanf(hex[0:2], "%02x", &r)
	}
	if len(hex) >= 4 {
		fmt.Sscanf(hex[2:4], "%02x", &g)
	}
	if len(hex) >= 6 {
		fmt.Sscanf(hex[4:6], "%02x", &b)
	}
	return rl.NewColor(r, g, b, 255)
}
