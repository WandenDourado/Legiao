package tilemap

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// GetCollisionRects reads the object group layer named "collision" from the
// map and returns all its rectangles. Returns nil if no collision layer
// is found.
func GetCollisionRects(tiledMap *TiledMap) []rl.Rectangle {
	for _, layer := range tiledMap.Layers {
		if layer.Type == "objectgroup" && layer.Name == "collision" {
			rects := make([]rl.Rectangle, 0, len(layer.Objects))
			for _, obj := range layer.Objects {
				// Skip degenerate rectangles (zero width or height).
				if obj.Width <= 0 || obj.Height <= 0 {
					continue
				}
				rects = append(rects, rl.NewRectangle(obj.X, obj.Y, obj.Width, obj.Height))
			}
			return rects
		}
	}
	return nil
}

// IsColliding performs an AABB collision check between a rectangle defined
// by (pos, width, height) and the given collision rectangles.
// Returns true if any overlap is found.
func IsColliding(pos rl.Vector2, width, height float32, rects []rl.Rectangle) bool {
	playerRect := rl.NewRectangle(pos.X-width/2, pos.Y-height/2, width, height)
	for _, r := range rects {
		if rl.CheckCollisionRecs(playerRect, r) {
			return true
		}
	}
	return false
}
