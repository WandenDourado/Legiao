package game

import (
	"github.com/WandenDourado/Legiao/internal/collision"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// PlayerFootprint is the single definition of the player's collision box,
// shared by movement resolution, the portal trigger and the F3 debug overlay
// so they can never disagree about what is actually being tested.
//
// It returns the box CENTRE as well as its size, because the box is not
// centred on Position: it sits at the character's feet (see
// entity.PlayerGroundBox). Everything that tests the player against the map
// must use this centre, or it will be testing the character's chest against
// the ground.
func PlayerFootprint(p *entity.Player) (center rl.Vector2, width, height float32) {
	return entity.PlayerGroundBox(p)
}

// ResolveCollision keeps the player out of solid map space. The rule itself
// lives in the shared `collision` package, so players and monsters are
// resolved by the same code against the same map.
//
// Player.Update has already applied this frame's velocity, so the move is
// replayed from where the player stood before it: the shared resolver takes a
// start point and a delta, not an already-moved position.
//
// The resolver is handed the FOOT box, not the player's Position. The two
// differ by about 105 px, which is exactly the distance a character used to be
// able to plant their feet inside a tree trunk.
func ResolveCollision(p *entity.Player, grid *tilemap.CollisionGrid) {
	if grid == nil {
		return
	}
	wanted, width, height := PlayerFootprint(p)
	delta := rl.Vector2Scale(p.Velocity, rl.GetFrameTime())
	previousCenter := rl.Vector2Subtract(wanted, delta)
	resolved := collision.Resolve(previousCenter, delta, width, height, grid)
	entity.MoveByGroundCorrection(p, wanted, resolved)
}
