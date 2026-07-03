package entity

import (
	"fmt"
	"strings"

	"github.com/WandenDourado/Legiao/internal/assets"
	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Wizard sprite sheet constants
const (
	WizardFrameWidth              = 165
	WizardFrameHeight             = 246
	WizardColumns                 = 6
	WizardRows                    = 5
	WizardFrameTime       float32 = 0.12 // seconds per frame (walk)
	WizardSprintFrameTime float32 = 0.08 // seconds per frame (sprint — faster playback)

	// Sprite rows
	RowWalkUp       = 0
	RowWalkDown     = 1
	RowWalkLeft     = 2
	RowWalkDownLeft = 3
	RowWalkUpLeft   = 4
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

// InitSprite loads the wizard texture for the player.
func (p *Player) InitSprite() {
	p.WizardTexture = rl.LoadTexture(assets.Path("assets/sprites/wizard/wizard.png"))
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
func (p *Player) Update(dir rl.Vector2, dt float32, bounds world.Bounds) {
	// Calculate velocity from input direction and speed
	p.Velocity.X = dir.X * p.Speed
	p.Velocity.Y = dir.Y * p.Speed

	// Update position
	p.Position.X += p.Velocity.X * dt
	p.Position.Y += p.Velocity.Y * dt

	// Keep player within world bounds
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

		if dir.Y < 0 && dir.X < 0 && absX > absY*0.5 {
			// Moving up-left diagonal
			p.CurrentRow = RowWalkUpLeft
		} else if dir.Y < 0 && dir.X > 0 && absX > absY*0.5 {
			// Moving up-right diagonal (mirrored)
			p.CurrentRow = RowWalkUpLeft
		} else if dir.Y < 0 && absY >= absX {
			// Moving up (or up-diagonal but mostly vertical)
			p.CurrentRow = RowWalkUp
		} else if dir.Y > 0 && dir.X < 0 && absX > absY*0.5 {
			// Moving down-left diagonal
			p.CurrentRow = RowWalkDownLeft
		} else if dir.Y > 0 && dir.X > 0 && absX > absY*0.5 {
			// Moving down-right diagonal (mirrored)
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

		// Advance animation timer — faster during sprint
		frameTime := WizardFrameTime
		if p.IsSprinting {
			frameTime = WizardSprintFrameTime
		}
		p.AnimTimer += dt
		if p.AnimTimer >= frameTime {
			p.AnimTimer -= frameTime
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

	currentRow := p.CurrentRow
	// Fallback to RowWalkUp if the texture doesn't have the 5th row (upper-diagonal)
	if p.Initialized && p.WizardTexture.Height > 0 && currentRow*WizardFrameHeight >= int(p.WizardTexture.Height) {
		if currentRow == RowWalkUpLeft {
			currentRow = RowWalkUp
		} else {
			currentRow = RowWalkDown
		}
	}

	// Calculate source rectangle from sprite sheet
	sourceRect := rl.NewRectangle(
		float32(p.CurrentFrame*WizardFrameWidth),
		float32(currentRow*WizardFrameHeight),
		WizardFrameWidth,
		WizardFrameHeight,
	)

	// Mirror for right-facing (left-facing rows with negative width)
	if p.Velocity.X > 0 && (currentRow == RowWalkLeft || currentRow == RowWalkDownLeft || currentRow == RowWalkUpLeft) {
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
