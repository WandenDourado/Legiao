package game

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ResolveCollision checks the player against the map collision rectangles and
// reverts to axis-aligned submovement on overlap (simple slide).
func ResolveCollision(p *entity.Player, rects []rl.Rectangle) {
	if len(rects) == 0 {
		return
	}
	playerW := p.Radius * 2
	playerH := p.Radius * 2
	if tilemap.IsColliding(p.Position, playerW, playerH, rects) {
		// Try reverting X only
		testPos := rl.NewVector2(p.Position.X-p.Velocity.X*rl.GetFrameTime(), p.Position.Y)
		if !tilemap.IsColliding(testPos, playerW, playerH, rects) {
			p.Position.X = testPos.X
			return
		}
		// Try reverting Y only
		testPos = rl.NewVector2(p.Position.X, p.Position.Y-p.Velocity.Y*rl.GetFrameTime())
		if !tilemap.IsColliding(testPos, playerW, playerH, rects) {
			p.Position.Y = testPos.Y
			return
		}
		// Revert both axes
		p.Position.X -= p.Velocity.X * rl.GetFrameTime()
		p.Position.Y -= p.Velocity.Y * rl.GetFrameTime()
	}
}
