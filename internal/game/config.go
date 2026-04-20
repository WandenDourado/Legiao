package game

// Config holds global game constants and settings.
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