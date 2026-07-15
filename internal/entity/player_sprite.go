package entity

import (
	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Wizard sprite sheet constants for skills/create-character-sprites output.
const (
	WizardFrameWidth              = 128
	WizardFrameHeight             = 192
	WizardColumns                 = 8
	WizardRows                    = 5
	WizardFrameTime       float32 = 0.12
	WizardSprintFrameTime float32 = 0.08

	// Row order: S, SW, W, N, NW. E, SE, and NE mirror W, SW, and NW.
	RowWalkDown     = 0
	RowWalkDownLeft = 1
	RowWalkLeft     = 2
	RowWalkUp       = 3
	RowWalkUpLeft   = 4
)

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

// updateAnimation determines the correct sprite row and advances the frame timer.
func (p *Player) updateAnimation(dir rl.Vector2, dt float32) {
	isMoving := dir.X != 0 || dir.Y != 0
	if !isMoving {
		p.CurrentFrame = 0
		p.AnimTimer = 0
		p.CurrentRow = p.LastRow
		return
	}

	p.CurrentRow = walkRowForDirection(dir)
	p.LastRow = p.CurrentRow

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
}

// Draw renders the player as an animated sprite from the wizard sprite sheet.
func (p *Player) Draw() {
	if !p.Initialized {
		col := hexToColor(p.Color)
		rl.DrawCircleV(p.Position, p.Radius, col)
		return
	}

	drawWizardSprite(
		p.WizardTexture,
		p.Position.X,
		p.Position.Y,
		p.CurrentFrame,
		p.CurrentRow,
		p.Velocity.X,
	)
}

// DrawPlayerAt renders a player at a specific position with a given color.
func DrawPlayerAt(x, y float32, color string, radius float32) {
	col := hexToColor(color)
	rl.DrawCircleV(rl.NewVector2(x, y), radius, col)
}

func drawWizardSprite(texture rl.Texture2D, x, y float32, frame, row int, velX float32) {
	frame = validWalkFrame(frame)
	row = validWalkRow(row)

	sourceRect := rl.NewRectangle(
		float32(frame*WizardFrameWidth),
		float32(row*WizardFrameHeight),
		WizardFrameWidth,
		WizardFrameHeight,
	)
	if shouldMirrorWalkRow(row, velX) {
		sourceRect.Width = -WizardFrameWidth
	}

	destRect := rl.NewRectangle(
		x-WizardFrameWidth/2,
		y-WizardFrameHeight/2,
		WizardFrameWidth,
		WizardFrameHeight,
	)
	rl.DrawTexturePro(texture, sourceRect, destRect, rl.NewVector2(0, 0), 0, rl.White)
}
