package entity

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Projectile kinds customize the basic attack per character.
const (
	// KindBasic is the legacy yellow ball (fallback for unknown characters).
	KindBasic = "basic"
	// KindFireball is the Mago's basic attack: a blazing fire ball.
	KindFireball = "fireball"
	// KindHoly is the Sacerdotisa's basic attack: a sacred light bolt that
	// heals allies it passes through and damages enemies.
	KindHoly = "holy"
	// KindArrow is the Arqueiro's basic attack: a wooden arrow.
	KindArrow = "arrow"
	// KindNecroSkull is the Necromante's basic attack: a shadowy skull wreathed
	// in purple soulfire that steals life from enemies it hits.
	KindNecroSkull = "necroskull"
)

// Per-kind tuning for basic attacks.
const (
	FireballAttackDamage float32 = 25
	FireballAttackSpeed  float32 = 420
	FireballAttackRadius float32 = 20

	HolyAttackDamage float32 = 18
	HolyAttackHeal   float32 = 12
	HolyAttackSpeed  float32 = 380
	HolyAttackRadius float32 = 13

	ArrowAttackDamage float32 = 22
	ArrowAttackSpeed  float32 = 700
	ArrowAttackRadius float32 = 10

	// NecroAttackLifesteal is the flat HP restored to the Necromante per
	// enemy hit by a shadow skull.
	NecroAttackDamage    float32 = 22
	NecroAttackLifesteal float32 = 8
	NecroAttackSpeed     float32 = 400
	NecroAttackRadius    float32 = 16
)

// AttackKindFor returns the basic-attack projectile kind for a character.
// The Paladina has no projectile (melee sword sweep) and returns "".
func AttackKindFor(ct CharacterType) string {
	switch ct {
	case CharMago:
		return KindFireball
	case CharSacerdotisa:
		return KindHoly
	case CharArqueiro:
		return KindArrow
	case CharNecromante:
		return KindNecroSkull
	case CharPaladina:
		return "" // melee: handled by the sword sweep, no projectile
	}
	return KindBasic
}

// Projectile represents a projectile fired by a player.
type Projectile struct {
	ID       string
	OwnerID  string // Player who fired it
	Kind     string // Visual/behavior kind (KindFireball, KindHoly, ...)
	Position rl.Vector2
	Velocity rl.Vector2
	Damage   float32
	Heal     float32 // Healing applied to allies (KindHoly only)
	Lifesteal float32 // HP restored to the OWNER on enemy hit (KindNecroSkull)
	Radius   float32
	Color    string
	IsActive bool
	Lifetime float32 // Seconds until despawn

	// HealedAllies tracks which players this projectile already healed so a
	// holy bolt heals each ally at most once while passing through.
	HealedAllies map[string]bool
}

// NewProjectile creates a new projectile fired by ownerID from startPos toward target direction.
func NewProjectile(ownerID string, startPos rl.Vector2, direction rl.Vector2) *Projectile {
	speed := float32(400.0) // ProjectileSpeed
	return &Projectile{
		ID:       generateID(),
		OwnerID:  ownerID,
		Kind:     KindBasic,
		Position: startPos,
		Velocity: rl.Vector2Scale(rl.Vector2Normalize(direction), speed),
		Damage:   25.0, // ProjectileDamage
		Radius:   ProjectileSize,
		Color:    "#FFFF00", // Yellow projectiles
		IsActive: true,
		Lifetime: 2.0, // ProjectileLifetime
	}
}

// NewAttackProjectile creates the character-specific basic-attack projectile.
// Returns nil for melee characters (Paladina).
func NewAttackProjectile(ownerID string, ct CharacterType, startPos, direction rl.Vector2) *Projectile {
	kind := AttackKindFor(ct)
	if kind == "" {
		return nil
	}
	p := NewProjectile(ownerID, startPos, direction)
	p.Kind = kind
	switch kind {
	case KindFireball:
		p.Damage = FireballAttackDamage
		p.Radius = FireballAttackRadius
		p.Velocity = rl.Vector2Scale(rl.Vector2Normalize(direction), FireballAttackSpeed)
		p.Color = "#FF6A00"
	case KindHoly:
		p.Damage = HolyAttackDamage
		p.Heal = HolyAttackHeal
		p.Radius = HolyAttackRadius
		p.Velocity = rl.Vector2Scale(rl.Vector2Normalize(direction), HolyAttackSpeed)
		p.Color = "#FFF2C0"
		p.HealedAllies = make(map[string]bool)
	case KindArrow:
		p.Damage = ArrowAttackDamage
		p.Radius = ArrowAttackRadius
		p.Velocity = rl.Vector2Scale(rl.Vector2Normalize(direction), ArrowAttackSpeed)
		p.Color = "#C8A064"
		p.Lifetime = 1.6
	case KindNecroSkull:
		p.Damage = NecroAttackDamage
		p.Lifesteal = NecroAttackLifesteal
		p.Radius = NecroAttackRadius
		p.Velocity = rl.Vector2Scale(rl.Vector2Normalize(direction), NecroAttackSpeed)
		p.Color = "#8A2BE2"
	}
	return p
}

// Dir returns the normalized travel direction of the projectile.
func (p *Projectile) Dir() rl.Vector2 {
	return rl.Vector2Normalize(p.Velocity)
}

// Update updates the projectile's position and lifetime.
// Returns true if the projectile is still active, false if it should be removed.
func (p *Projectile) Update(dt float32, bounds world.Bounds) bool {
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

	// Check if out of world bounds
	if p.Position.X < -p.Radius || p.Position.X > bounds.Width+p.Radius ||
		p.Position.Y < -p.Radius || p.Position.Y > bounds.Height+p.Radius {
		p.IsActive = false
		return false
	}

	return true
}

// Draw renders the projectile according to its kind.
func (p *Projectile) Draw() {
	if !p.IsActive {
		return
	}
	d := p.Dir()
	DrawProjectileKindAt(p.Kind, p.Position.X, p.Position.Y, d.X, d.Y)
}

// DrawProjectileAt renders a legacy projectile at a specific position.
func DrawProjectileAt(x, y float32) {
	rl.DrawCircleV(rl.NewVector2(x, y), ProjectileSize, rl.Yellow)
}

// DrawProjectileKindAt renders a basic-attack projectile of the given kind at
// (x, y) traveling toward (dirX, dirY). Used by both the host (live entities)
// and clients (remote snapshots), so visuals stay identical everywhere.
func DrawProjectileKindAt(kind string, x, y, dirX, dirY float32) {
	pos := rl.NewVector2(x, y)
	switch kind {
	case KindFireball:
		drawFireballProjectile(pos, dirX, dirY)
	case KindHoly:
		drawHolyProjectile(pos)
	case KindArrow:
		drawArrowProjectile(pos, dirX, dirY)
	case KindNecroSkull:
		drawNecroSkullProjectile(pos, dirX, dirY)
	default:
		rl.DrawCircleV(pos, ProjectileSize, rl.Yellow)
	}
}

// drawFireballProjectile renders the Mago's basic attack: a big layered fire
// orb with a long, flickering flame tail. Procedural only, no sprites.
func drawFireballProjectile(pos rl.Vector2, dirX, dirY float32) {
	d := safeDir(dirX, dirY)
	perp := rl.NewVector2(-d.Y, d.X)
	t := rl.GetTime()
	flicker := float32(math.Sin(t*28)) * 2.5

	// Long flame tail: fading blobs behind the ball with lateral flicker so
	// the fire visibly dances.
	for i := 1; i <= 6; i++ {
		f := float32(i)
		wob := float32(math.Sin(t*22+float64(i)*1.7)) * (2 + f*1.2)
		tail := rl.NewVector2(
			pos.X-d.X*f*13+perp.X*wob,
			pos.Y-d.Y*f*13+perp.Y*wob,
		)
		alpha := uint8(220 - i*30)
		r := FireballAttackRadius * (1.05 - f*0.14)
		rl.DrawCircleV(tail, r, rl.NewColor(210, 60, 12, alpha))
		rl.DrawCircleV(tail, r*0.55, rl.NewColor(255, 140, 30, alpha))
	}

	// Core: dark ember -> orange -> yellow-white heart.
	rl.DrawCircleV(pos, FireballAttackRadius+flicker, rl.NewColor(190, 45, 10, 255))
	rl.DrawCircleV(pos, (FireballAttackRadius-4)+flicker, rl.NewColor(255, 130, 20, 255))
	rl.DrawCircleV(pos, (FireballAttackRadius-9)+flicker*0.5, rl.NewColor(255, 235, 150, 255))

	// Additive glow halo (strong, wide) plus a hot streak along the tail.
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(pos.X), int32(pos.Y), FireballAttackRadius*2.8,
		rl.NewColor(255, 130, 30, 160), rl.Blank)
	tailEnd := rl.NewVector2(pos.X-d.X*FireballAttackRadius*4.2, pos.Y-d.Y*FireballAttackRadius*4.2)
	rl.DrawLineEx(pos, tailEnd, FireballAttackRadius*0.8, rl.NewColor(255, 90, 20, 70))
	rl.EndBlendMode()
}

// drawHolyProjectile renders the Sacerdotisa's basic attack: a radiant sacred
// orb with a soft golden halo and a light cross sparkle.
func drawHolyProjectile(pos rl.Vector2) {
	t := rl.GetTime()
	pulse := float32(math.Sin(t*10)) * 1.5

	// Bright core.
	rl.DrawCircleV(pos, HolyAttackRadius-4+pulse*0.5, rl.NewColor(255, 250, 230, 255))

	rl.BeginBlendMode(rl.BlendAdditive)
	// Golden halo.
	rl.DrawCircleGradient(int32(pos.X), int32(pos.Y), HolyAttackRadius*2.4+pulse,
		rl.NewColor(255, 226, 130, 150), rl.Blank)
	// Rotating cross sparkle (four rays of light).
	ang := float32(t) * 1.8
	for i := 0; i < 4; i++ {
		a := ang + float32(i)*(math.Pi/2)
		ray := rl.NewVector2(
			pos.X+float32(math.Cos(float64(a)))*(HolyAttackRadius*1.9),
			pos.Y+float32(math.Sin(float64(a)))*(HolyAttackRadius*1.9),
		)
		rl.DrawLineEx(pos, ray, 2.5, rl.NewColor(255, 245, 190, 170))
	}
	rl.EndBlendMode()
}

// drawArrowProjectile renders the Arqueiro's basic attack: a wooden arrow with
// steel head and green fletching, oriented along its travel direction.
func drawArrowProjectile(pos rl.Vector2, dirX, dirY float32) {
	const shaftLen float32 = 42
	d := safeDir(dirX, dirY)
	perp := rl.NewVector2(-d.Y, d.X)
	tip := pos
	tail := rl.NewVector2(tip.X-d.X*shaftLen, tip.Y-d.Y*shaftLen)

	// Wooden shaft.
	rl.DrawLineEx(tail, tip, 3, rl.NewColor(146, 96, 52, 255))

	// Steel head (both winding orders so culling never hides it).
	headBase := rl.NewVector2(tip.X-d.X*11, tip.Y-d.Y*11)
	h1 := rl.NewVector2(headBase.X+perp.X*5, headBase.Y+perp.Y*5)
	h2 := rl.NewVector2(headBase.X-perp.X*5, headBase.Y-perp.Y*5)
	steel := rl.NewColor(210, 214, 222, 255)
	rl.DrawTriangle(tip, h1, h2, steel)
	rl.DrawTriangle(tip, h2, h1, steel)

	// Fletching near the tail.
	feather := rl.NewColor(96, 190, 96, 255)
	for i := 0; i < 2; i++ {
		base := rl.NewVector2(tail.X+d.X*float32(i)*8, tail.Y+d.Y*float32(i)*8)
		back := rl.NewVector2(base.X-d.X*8, base.Y-d.Y*8)
		f1 := rl.NewVector2(back.X+perp.X*5, back.Y+perp.Y*5)
		f2 := rl.NewVector2(back.X-perp.X*5, back.Y-perp.Y*5)
		rl.DrawLineEx(base, f1, 2, feather)
		rl.DrawLineEx(base, f2, 2, feather)
	}
}

// safeDir normalizes (x, y), falling back to pointing right when zero.
func safeDir(x, y float32) rl.Vector2 {
	v := rl.NewVector2(x, y)
	if v.X == 0 && v.Y == 0 {
		return rl.NewVector2(1, 0)
	}
	return rl.Vector2Normalize(v)
}
