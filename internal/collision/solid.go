// Package collision owns the one movement-resolution rule shared by every
// entity that walks on the map: players, monsters, and anything else that must
// not pass through a tree, a fence or a house.
//
// It deliberately knows nothing about entities or tilemaps. It talks to the
// map through the Solid interface, which is what allows `entity` to respect
// map obstacles without importing `tilemap` (and without an import cycle).
package collision

import rl "github.com/gen2brain/raylib-go/raylib"

// Solid answers whether an axis-aligned box centered on a world point overlaps
// blocked space. *tilemap.CollisionGrid already satisfies it, and a test can
// substitute a hand-written grid without loading a map.
type Solid interface {
	CollidesCentered(pos rl.Vector2, width, height float32) bool
}

// blocked is the nil-safe form of the interface call. A nil Solid means "no
// map loaded yet": everything is walkable, which keeps menus, tests and the
// frames before the first map finishes loading from freezing every entity in
// place.
func blocked(s Solid, pos rl.Vector2, width, height float32) bool {
	if s == nil {
		return false
	}
	return s.CollidesCentered(pos, width, height)
}
