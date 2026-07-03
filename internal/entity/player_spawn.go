package entity

import (
	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// InitialPlayerSpawn returns the default player spawn at the center of the world.
func InitialPlayerSpawn(bounds world.Bounds) rl.Vector2 {
	return rl.NewVector2(bounds.Width*0.5, bounds.Height*0.5)
}
