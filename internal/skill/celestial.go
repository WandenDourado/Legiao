package skill

import (
	"fmt"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Celestial Arrow constants tune the Arqueiro's ultimate (Flechas Celestiais).
const (
	// CelestialCharges is how many arrows one activation grants (each aimed
	// and fired separately; cooldown starts after the last one).
	CelestialCharges int = 2
	// CelestialSpeed is the arrow travel speed (world units/sec).
	CelestialSpeed float32 = 1600 * gameSpeed
	// CelestialRange crosses the entire map.
	CelestialRange float32 = 4800
	// CelestialDamage e o dano causado a CADA inimigo perfurado.
	//
	// 40 -> 600 (31/08/2026, a pedido do Gui: "uma flecha celestial deveria
	// derrubar um orc"). O numero e derivado, nao escolhido: o orc
	// (EnemyTypeGarrison) tem 600 de vida em entity/enemy_sprite.go, e e
	// exatamente essa vida que a flecha tem de cobrir — nem menos (a suprema
	// nao derruba o alvo que veio derrubar), nem mais (a diferenca entre a
	// suprema do Arqueiro e a do Mago e precisao contra area, nao quantidade
	// de dano). Se a vida do orc mudar, este numero muda junto. Ver
	// TestOneCelestialArrowFellsAnOrc (network/balance_test.go) e
	// doc/combat_rules.md.
	//
	// Os 40 vinham de quando o slime de 100 de vida era o elenco inteiro — a
	// mesma origem que os 100 de MeteorImpactDamage tinham antes de 27/08. So
	// o meteoro foi reajustado quando o orc entrou; a flecha ficou para tras.
	CelestialDamage float32 = 600
	// CelestialHitRadius is the piercing collision radius.
	CelestialHitRadius float32 = 70
	// CelestialLength is the visual shaft length: a GIGANTIC heavenly arrow,
	// a spear of light several characters long.
	CelestialLength float32 = 420
	// CelestialCooldown starts after the second arrow is fired.
	CelestialCooldown float32 = 30.0
)

// Celestial palette: light blue body, white details (per the spell design).
var (
	celestialBlue  = rl.NewColor(150, 215, 255, 255)
	celestialWhite = rl.NewColor(245, 252, 255, 255)
)

var celestialSeq uint64

func nextCelestialID() string {
	return fmt.Sprintf("ce%d", atomic.AddUint64(&celestialSeq, 1))
}

// CelestialArrow is a giant heavenly arrow that crosses the whole map,
// piercing every enemy it touches (each damaged once). Drawn with primitives:
// light-blue shaft, white core and head, ethereal wing fletching, long
// luminous wake.
type CelestialArrow struct {
	ID       string
	OwnerID  string
	Position rl.Vector2
	Velocity rl.Vector2
	Origin   rl.Vector2
	Traveled float32
	Time     float32
	// HitIDs records enemies already pierced so each is damaged only once.
	HitIDs map[string]bool
	Trail  *ParticleEmitter
}

// NewCelestialArrow creates a celestial arrow launched from start toward dir.
func NewCelestialArrow(ownerID string, start, dir rl.Vector2) *CelestialArrow {
	d := rl.Vector2Normalize(dir)
	return &CelestialArrow{
		ID:       nextCelestialID(),
		OwnerID:  ownerID,
		Position: start,
		Velocity: rl.Vector2Scale(d, CelestialSpeed),
		Origin:   start,
		HitIDs:   make(map[string]bool),
		Trail:    NewParticleEmitter(),
	}
}

// Advance moves the arrow and feeds its luminous wake. Returns false once the
// arrow exceeded its range (callers prune it).
func (c *CelestialArrow) Advance(dt float32) bool {
	c.Time += dt
	c.Position = rl.Vector2Add(c.Position, rl.Vector2Scale(c.Velocity, dt))
	c.Traveled = rl.Vector2Distance(c.Origin, c.Position)
	d := rl.Vector2Normalize(c.Velocity)
	tail := rl.Vector2Subtract(c.Position, rl.Vector2Scale(d, CelestialLength))
	// Wake: layered white/blue motes streaming off the shaft.
	for i := 0; i < 8; i++ {
		t := float32(i) / 8.0
		pos := rl.Vector2Add(tail, rl.Vector2Scale(d, CelestialLength*t))
		jitter := rl.NewVector2(
			float32(rl.GetRandomValue(-18, 18)),
			float32(rl.GetRandomValue(-18, 18)),
		)
		c.Trail.Emit(rl.Vector2Add(pos, jitter), rl.Vector2Scale(d, -80), 0.45, 24, celestialBlue)
	}
	c.Trail.Emit(c.Position, rl.NewVector2(0, 0), 0.32, 18, celestialWhite)
	c.Trail.Update(dt)
	return c.Traveled < CelestialRange
}

// PierceFlash bursts sparks at an enemy the arrow just passed through.
func (c *CelestialArrow) PierceFlash(at rl.Vector2) {
	c.Trail.Burst(at, 12, 140, 320, 0.35, 10, celestialWhite)
	c.Trail.Burst(at, 8, 80, 200, 0.4, 8, celestialBlue)
}

// Draw renders the celestial arrow (world space).
func (c *CelestialArrow) Draw() {
	d := rl.Vector2Normalize(c.Velocity)
	perp := rl.NewVector2(-d.Y, d.X)
	tip := c.Position
	tail := rl.Vector2Subtract(tip, rl.Vector2Scale(d, CelestialLength))

	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	// Luminous shaft: wide blue beam with a thin white core (the "detail").
	rl.DrawLineEx(tail, tip, 44, rl.Fade(celestialBlue, 0.3))
	rl.DrawLineEx(tail, tip, 26, rl.Fade(celestialBlue, 0.6))
	rl.DrawLineEx(tail, tip, 10, rl.Fade(celestialWhite, 0.95))

	// Radiant head: a great white blade over a blue glow.
	headBase := rl.Vector2Subtract(tip, rl.Vector2Scale(d, 96))
	h1 := rl.Vector2Add(headBase, rl.Vector2Scale(perp, 36))
	h2 := rl.Vector2Subtract(headBase, rl.Vector2Scale(perp, 36))
	rl.DrawTriangle(tip, h1, h2, rl.Fade(celestialWhite, 0.95))
	rl.DrawTriangle(tip, h2, h1, rl.Fade(celestialWhite, 0.95))
	rl.DrawCircleGradient(int32(tip.X), int32(tip.Y), 120, rl.Fade(celestialBlue, 0.5), rl.Blank)
	rl.DrawCircleGradient(int32(tip.X), int32(tip.Y), 56, rl.Fade(celestialWhite, 0.9), rl.Blank)

	// Ethereal wing fletching: two swept blades of light at the tail.
	for s := float32(-1); s <= 1; s += 2 {
		w1 := rl.Vector2Add(tail, rl.Vector2Scale(perp, 18*s))
		w2 := rl.Vector2Add(tail, rl.Vector2Add(
			rl.Vector2Scale(perp, 70*s), rl.Vector2Scale(d, -88)))
		w3 := rl.Vector2Add(tail, rl.Vector2Scale(d, 48))
		wing := rl.Fade(celestialWhite, 0.7)
		rl.DrawTriangle(w1, w2, w3, wing)
		rl.DrawTriangle(w1, w3, w2, wing)
	}

	for _, p := range c.Trail.particles {
		drawParticle(p)
	}
}
