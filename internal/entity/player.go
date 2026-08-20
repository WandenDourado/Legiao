package entity

import (
	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
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
	PlayerSize = 20.0
	// EnemySize is the legacy placeholder radius from when enemies were plain
	// circles. Enemy hitboxes now come from EnemyDef.Radius so that art and
	// hitbox are sized together; kept for the debug fallback only.
	EnemySize = 15.0
	// EnemySlimeRadius matches the slime's visible silhouette: the frame is
	// 128 px scaled by 1.15, and the blob spans ~68% of it, so ~100 px across.
	EnemySlimeRadius = 45.0
	// EnemyWolfRadius covers the wolf's torso. The animal is ~180 px long but
	// only ~55 px wide, so no single circle fits it: this one is generous
	// laterally and leaves the muzzle and tail outside the hitbox.
	EnemyWolfRadius = 50.0
	// EnemyOrcRadius covers the orc's torso, not its reach. The body is ~70 px
	// wide in the source frame and is drawn at 1.6x, so the trunk spans ~112 px
	// on screen; the espadao adds another ~90 px that deliberately stays
	// OUTSIDE the hitbox. A circle that swallowed the blade would let the
	// player connect with empty air beside the monster.
	//
	// Unlike the slime and the wolf, this circle sits at the orc's FEET rather
	// than at the middle of the drawing, because EnemyDef.FootLine is set. That
	// is what makes a standing figure collide with what it is standing on.
	EnemyOrcRadius = 45.0
	ProjectileSize = 20.0

	// Respawn constants. A dead player stays on the ground, greyed out and
	// unable to act, until the timer runs out. The wait is long enough to
	// hurt the group, and the health it gives back is small enough that
	// coming back is not a free reset.
	RespawnDelay         = 30.0 // Seconds before respawn
	RespawnHealthPercent = 0.30 // 30% health on respawn
)

// Sprint input thresholds
const (
	// SprintThreshold is the minimum joystick displacement (normalized 0-1)
	// that triggers sprint mode on Android.
	SprintThreshold = 0.70
)

// Player represents the player character.
type Player struct {
	// CharType identifies which character this player is using.
	CharType CharacterType
	// Color is a hex string like "#FF0000" representing the player's unique color.
	Color     string
	Position  rl.Vector2
	Velocity  rl.Vector2
	Health    float32
	MaxHealth float32
	Speed     float32
	Radius    float32
	IsDead    bool
	// RespawnIn is how many seconds are left until the host revives this
	// player. It mirrors the authoritative countdown and exists so the HUD
	// can show it; nothing in the entity package acts on it.
	RespawnIn float32

	// Sprite animation fields
	Texture     rl.Texture2D
	AnimTimer   float32
	CurrentFrame int
	CurrentRow   int
	LastRow      int
	IsSprinting  bool
	Initialized  bool
}

// NewPlayer creates a new player with default values for the given character type.
func NewPlayer(spawn rl.Vector2, charType CharacterType) *Player {
	return &Player{
		CharType:   charType,
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

// Update updates the player's position based on input direction and delta time.
// dir is a normalized vector (dx, dy) in the range [-1, 1] for each axis.
func (p *Player) Update(dir rl.Vector2, dt float32, bounds world.Bounds) {
	p.Velocity.X = dir.X * p.Speed
	p.Velocity.Y = dir.Y * p.Speed

	p.Position.X += p.Velocity.X * dt
	p.Position.Y += p.Velocity.Y * dt

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

	p.updateAnimation(dir, dt)
}

// Die freezes the player where it stands. Movement is skipped by the caller
// while IsDead is set; resetting the animation here is what makes the corpse
// stop mid-stride instead of keeping the last walk frame it happened to be on.
func (p *Player) Die() {
	p.IsDead = true
	p.CurrentFrame = 0
	p.AnimTimer = 0
	// Velocity is left alone on purpose: the sprite is mirrored from it, so
	// zeroing it would flip a body that died running to the right.
}

// Respawn resets the player for respawn with the given health percentage.
func (p *Player) Respawn(healthPercent float32, x, y float32) {
	p.Health = p.MaxHealth * healthPercent
	p.Position.X = x
	p.Position.Y = y
	p.IsDead = false
	p.RespawnIn = 0
}
