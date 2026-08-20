package entity

import (
	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Row order shared by all characters generated via the create-character-sprites skill:
// S, SW, W, N, NW. E, SE, and NE mirror W, SW, and NW.
const (
	RowWalkDown     = 0
	RowWalkDownLeft = 1
	RowWalkLeft     = 2
	RowWalkUp       = 3
	RowWalkUpLeft   = 4
)

// InitSprite loads the sprite texture for the player's character type.
func (p *Player) InitSprite() {
	def := GetCharacter(p.CharType)
	p.Texture = rl.LoadTexture(assets.Path(def.SpritePath))
	// Character frames are 128x192 drawn at 1.15x, so they are magnified too.
	// Bilinear does not remove the magnification, but it does stop it from
	// looking like hard pixel blocks.
	if p.Texture.ID != 0 {
		ApplySpriteFilter(p.Texture)
	}
	p.Initialized = true
}

// UnloadSprite unloads the player's character texture.
func (p *Player) UnloadSprite() {
	if p.Initialized {
		rl.UnloadTexture(p.Texture)
		p.Initialized = false
	}
}

// updateAnimation determines the correct sprite row and advances the frame timer.
func (p *Player) updateAnimation(dir rl.Vector2, dt float32) {
	def := GetCharacter(p.CharType)

	isMoving := dir.X != 0 || dir.Y != 0
	if !isMoving {
		p.CurrentFrame = 0
		p.AnimTimer = 0
		p.CurrentRow = p.LastRow
		return
	}

	p.CurrentRow = walkRowForDirection(dir)
	p.LastRow = p.CurrentRow

	frameTime := def.FrameTime
	if p.IsSprinting {
		frameTime = def.SprintTime
	}
	p.AnimTimer += dt
	if p.AnimTimer >= frameTime {
		p.AnimTimer -= frameTime
		p.CurrentFrame++
		if p.CurrentFrame >= def.Columns {
			p.CurrentFrame = 0
		}
	}
}

// Draw renders the player as an animated sprite from its character sprite sheet.
func (p *Player) Draw() {
	if !p.Initialized {
		col := hexToColor(p.Color)
		rl.DrawCircleV(p.Position, p.Radius, col)
		return
	}

	def := GetCharacter(p.CharType)
	// A corpse is the same sprite drawn through the grayscale shader, frozen
	// on the frame and facing the direction it died in.
	if p.IsDead {
		BeginDeathTint()
		defer EndDeathTint()
	}
	drawCharacterSprite(
		p.Texture,
		def,
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

// drawCharacterSprite renders a single frame from a character sprite sheet.
func drawCharacterSprite(texture rl.Texture2D, def CharacterDef, x, y float32, frame, row int, velX float32) {
	frame = validWalkFrame(frame, def.Columns)
	row = validWalkRow(row, def.Rows)

	fw := float32(def.FrameWidth)
	fh := float32(def.FrameHeight)
	scale := def.RenderScale
	if scale <= 0 {
		scale = 1
	}

	sourceRect := rl.NewRectangle(
		float32(frame)*fw,
		float32(row)*fh,
		fw,
		fh,
	)
	if shouldMirrorWalkRow(row, velX) {
		sourceRect.Width = -fw
	}

	destRect := rl.NewRectangle(
		x-fw*scale/2,
		y-fh*scale/2,
		fw*scale,
		fh*scale,
	)
	rl.DrawTexturePro(texture, sourceRect, destRect, rl.NewVector2(0, 0), 0, rl.White)
}
