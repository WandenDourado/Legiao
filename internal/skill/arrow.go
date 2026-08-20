package skill

import (
	"fmt"
	"math"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Arrow constants tune the Arqueiro's Rajada de Flechas (arrow volley) skill.
const (
	// ArrowCount is how many arrows a single fan/wave launches.
	ArrowCount int = 10
	// ArrowVolleyWaves is how many fans one cast fires in sequence.
	ArrowVolleyWaves int = 3
	// ArrowVolleyWaveDelay separates consecutive waves of the same cast.
	ArrowVolleyWaveDelay float32 = 0.18
	// ArrowSpreadDeg is the TOTAL cone angle covered by the volley (fan).
	ArrowSpreadDeg float32 = 50
	// ArrowSpeed is the arrow travel speed (world units/sec).
	ArrowSpeed float32 = 950
	// ArrowRange is the max distance an arrow travels before despawning.
	ArrowRange float32 = 950
	// ArrowTTL is a safety lifetime cap.
	ArrowTTL float32 = 2.0
	// ArrowDamage is the damage dealt per arrow (single target).
	ArrowDamage float32 = 15
	// ArrowHitRadius is the collision radius used against enemies.
	ArrowHitRadius float32 = 16
	// ArrowLength is the visual shaft length.
	ArrowLength float32 = 58
	// ArrowVolleyCooldown is the per-caster cooldown in seconds.
	//
	// A MESMA REGUA DA BOLA DE FOGO (FireballCooldown, 6 s), e nao um numero
	// proprio: uma habilidade Q que volta em 1,5 s deixa de ser habilidade e
	// vira o ataque basico do Arqueiro com outro botao. Com o bot jogando a
	// classe isso ficou impossivel de nao ver — ele lanca a rajada assim que a
	// recarga permite, e ela quase sempre permitia.
	ArrowVolleyCooldown float32 = 6.0
)

// arrowSeq generates unique arrow IDs (several arrows spawn the same frame,
// so the tiny random generateID() would collide).
var arrowSeq uint64

func nextArrowID() string {
	return fmt.Sprintf("ar%d", atomic.AddUint64(&arrowSeq, 1))
}

// Arrow is a procedurally rendered projectile: wooden shaft, steel head and
// green fletching drawn with primitives, plus a faint additive energy trail.
// No sprites are used.
type Arrow struct {
	ID       string
	OwnerID  string
	Position rl.Vector2
	Velocity rl.Vector2
	Origin   rl.Vector2
	Traveled float32
	TTL      float32
	Trail    *ParticleEmitter
}

// NewArrow creates an arrow launched from start toward dir (any length).
func NewArrow(ownerID string, start, dir rl.Vector2) *Arrow {
	d := rl.Vector2Normalize(dir)
	return &Arrow{
		ID:       nextArrowID(),
		OwnerID:  ownerID,
		Position: start,
		Velocity: rl.Vector2Scale(d, ArrowSpeed),
		Origin:   start,
		TTL:      ArrowTTL,
	}
}

// Update advances the arrow (host simulation). Returns false when it should be
// removed (TTL expired or max range reached).
func (a *Arrow) Update(dt float32) bool {
	a.advance(dt)
	a.TTL -= dt
	return a.TTL > 0 && a.Traveled < ArrowRange
}

// AdvanceVisual moves the arrow WITHOUT deciding life (client-side visuals).
// Expired reports true once range/TTL is exceeded so the client can prune it.
func (a *Arrow) AdvanceVisual(dt float32) {
	a.advance(dt)
	a.TTL -= dt
}

// Expired reports whether the arrow exceeded its range or lifetime.
func (a *Arrow) Expired() bool {
	return a.TTL <= 0 || a.Traveled >= ArrowRange
}

// advance integrates position and emits the energy trail.
func (a *Arrow) advance(dt float32) {
	a.Position = rl.Vector2Add(a.Position, rl.Vector2Scale(a.Velocity, dt))
	a.Traveled = rl.Vector2Distance(a.Origin, a.Position)
	if a.Trail == nil {
		a.Trail = NewParticleEmitter()
	}
	// Faint wind/energy streak behind the shaft: a few small fading motes.
	d := rl.Vector2Normalize(a.Velocity)
	tail := rl.Vector2Subtract(a.Position, rl.Vector2Scale(d, ArrowLength))
	for i := 0; i < 2; i++ {
		jitter := rl.NewVector2(
			float32(rl.GetRandomValue(-6, 6)),
			float32(rl.GetRandomValue(-6, 6)),
		)
		pos := rl.Vector2Add(tail, jitter)
		a.Trail.Emit(pos, rl.Vector2Scale(d, -40), 0.22, 7, rl.NewColor(150, 255, 170, 255))
	}
	a.Trail.Update(dt)
}

// Draw renders the arrow with primitives: solid shaft/head/fletching in normal
// blending, plus an additive glow so the volley reads as a magical skill.
func (a *Arrow) Draw() {
	d := rl.Vector2Normalize(a.Velocity)
	perp := rl.NewVector2(-d.Y, d.X)
	tip := a.Position
	tail := rl.Vector2Subtract(tip, rl.Vector2Scale(d, ArrowLength))

	// --- solid arrow body (normal blending so it reads as a real arrow) ---
	// Wooden shaft.
	rl.DrawLineEx(tail, tip, 3.5, rl.NewColor(146, 96, 52, 255))
	// Steel arrowhead: triangle at the tip (drawn in both winding orders so it
	// is visible regardless of raylib's CCW culling).
	headBase := rl.Vector2Subtract(tip, rl.Vector2Scale(d, 14))
	h1 := rl.Vector2Add(headBase, rl.Vector2Scale(perp, 6))
	h2 := rl.Vector2Subtract(headBase, rl.Vector2Scale(perp, 6))
	steel := rl.NewColor(210, 214, 222, 255)
	rl.DrawTriangle(tip, h1, h2, steel)
	rl.DrawTriangle(tip, h2, h1, steel)
	// Green fletching: two feather pairs near the tail.
	feather := rl.NewColor(96, 190, 96, 255)
	for i := 0; i < 2; i++ {
		base := rl.Vector2Add(tail, rl.Vector2Scale(d, float32(i)*9))
		back := rl.Vector2Subtract(base, rl.Vector2Scale(d, 9))
		f1 := rl.Vector2Add(back, rl.Vector2Scale(perp, 5.5))
		f2 := rl.Vector2Subtract(back, rl.Vector2Scale(perp, 5.5))
		rl.DrawLineEx(base, f1, 2.5, feather)
		rl.DrawLineEx(base, f2, 2.5, feather)
	}

	// --- additive glow: enchanted edge + trail ---
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(tip.X), int32(tip.Y), 12,
		rl.NewColor(140, 255, 170, 130), rl.Blank)
	if a.Trail != nil {
		for _, p := range a.Trail.particles {
			drawParticle(p)
		}
	}
	rl.EndBlendMode()
}

// VolleyDirections returns the ArrowCount unit vectors of a fan centered on
// dir, evenly covering ArrowSpreadDeg (the cone's tip is the archer).
func VolleyDirections(dir rl.Vector2) []rl.Vector2 {
	d := rl.Vector2Normalize(dir)
	base := math.Atan2(float64(d.Y), float64(d.X))
	total := float64(ArrowSpreadDeg) * math.Pi / 180
	dirs := make([]rl.Vector2, 0, ArrowCount)
	for i := 0; i < ArrowCount; i++ {
		// t in [-0.5, 0.5] spreads arrows symmetrically around the aim axis.
		t := float64(i)/float64(ArrowCount-1) - 0.5
		ang := base + t*total
		dirs = append(dirs, rl.NewVector2(
			float32(math.Cos(ang)), float32(math.Sin(ang))))
	}
	return dirs
}
