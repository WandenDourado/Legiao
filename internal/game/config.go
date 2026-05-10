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

	// Enemy constants
	EnemyHealth      = 100.0
	EnemyMaxHealth   = 100.0
	EnemySpeed       = 100.0 // Slower than player (PlayerSpeed=200)
	EnemyDamage      = 10.0
	EnemyColor       = "#8B0000" // Blood red
	EnemyAttackRange = 25.0
	EnemySpawnInterval = 3.0 // Seconds between spawn waves
	MaxEnemies       = 20

	// Projectile constants
	ProjectileSpeed   = 400.0
	ProjectileDamage  = 25.0
	ProjectileLifetime = 2.0 // Seconds before despawn

	// Combat constants
	EnemyAttackCooldown = 1.0 // Seconds between enemy attacks

	// Respawn constants
	RespawnDelay         = 15.0 // Seconds before respawn
	RespawnHealthPercent = 0.15  // 15% health on respawn

	// HUD constants
	AttackButtonSize = 35.0
)