package entity

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Projectile represents a projectile fired by a player.
type Projectile struct {
	ID        string
	OwnerID   string    // Player who fired it
	Position  rl.Vector2
	Velocity  rl.Vector2
	Damage    float32
	Radius    float32
	Color     string
	IsActive  bool
	Lifetime  float32 // Seconds until despawn
}

// NewProjectile creates a new projectile fired by ownerID from startPos toward target direction.
func NewProjectile(ownerID string, startPos rl.Vector2, direction rl.Vector2) *Projectile {
	speed := float32(400.0) // ProjectileSpeed
	return &Projectile{
		ID:       generateID(),
		OwnerID:  ownerID,
		Position:  startPos,
		Velocity:  rl.Vector2Scale(rl.Vector2Normalize(direction), speed),
		Damage:    25.0, // ProjectileDamage
		Radius:    ProjectileSize,
		Color:     "#FFFF00", // Yellow projectiles
		IsActive:  true,
		Lifetime:  2.0, // ProjectileLifetime
	}
}

// Update updates the projectile's position and lifetime.
// Returns true if the projectile is still active, false if it should be removed.
func (p *Projectile) Update(dt float32) bool {
	if !p.IsActive {
		return false
	}

	p.Position.X += p.Velocity.X * dt
	p.Position.Y += p.Velocity.Y * dt
	p.Lifetime -= dt

	// Check if lifetime expired
	if p.Lifetime <= 0 {
		p.IsActive = false
		return false
	}

	// Check if out of screen bounds
	if p.Position.X < -p.Radius || p.Position.X > float32(ScreenWidth)+p.Radius ||
		p.Position.Y < -p.Radius || p.Position.Y > float32(ScreenHeight)+p.Radius {
		p.IsActive = false
		return false
	}

	return true
}

// Draw renders the projectile as a small yellow circle.
func (p *Projectile) Draw() {
	if !p.IsActive {
		return
	}
	col := hexToColor(p.Color)
	rl.DrawCircleV(p.Position, p.Radius, col)
}

// DrawProjectileAt renders a projectile at a specific position.
func DrawProjectileAt(x, y float32) {
	rl.DrawCircleV(rl.NewVector2(x, y), ProjectileSize, rl.Yellow)
}
