package skill

import (
	"fmt"
	"math"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Sword constants tune the Paladina's basic melee attack (sword sweep).
const (
	// SwordRange is the blade reach from the paladin's center (short range).
	SwordRange float32 = 140
	// SwordInnerRadius is where the blade/arc begins, pushed away from the
	// character's body so the sword is projected in FRONT of her, never over
	// her sprite.
	SwordInnerRadius float32 = 34
	// SwordArcDeg is the TOTAL angle covered by one sweep.
	SwordArcDeg float32 = 120
	// SwordSweepTime is how long the blade takes to travel edge to edge.
	SwordSweepTime float32 = 0.28
	// SwordDamage is applied once to every enemy inside the arc.
	SwordDamage float32 = 30
	// swordFadeTime is the extra time the arc trail lingers after the sweep.
	swordFadeTime float32 = 0.12

	// While the Paladina is a Divine Avatar her basic attack transfigures
	// too: a much larger golden greatsword with empowered damage.
	// SwordAvatarReachScale multiplies the blade reach.
	SwordAvatarReachScale float32 = 1.9
	// SwordAvatarDamage replaces SwordDamage while the avatar lasts.
	SwordAvatarDamage float32 = 75
)

// swordSeq generates unique sweep IDs (a player can chain swings quickly).
var swordSeq uint64

func nextSwordID() string {
	return fmt.Sprintf("sw%d", atomic.AddUint64(&swordSeq, 1))
}

// SwordSweep is the Paladina's basic attack: a blade that sweeps a 120° arc
// edge-to-edge around the owner. The anchor is re-synced to the owner's
// position every frame, so the sword follows the character while she moves
// (the character never walks "off" her own blade). Procedural visuals only.
type SwordSweep struct {
	ID       string
	OwnerID  string
	Position rl.Vector2 // anchor, synced to the owner's position every frame
	// BaseAngle is the aim direction in radians; the blade sweeps from
	// BaseAngle-arc/2 to BaseAngle+arc/2.
	BaseAngle float32
	Elapsed   float32
	// Empowered marks a Divine Avatar swing: a far larger golden greatsword
	// with SwordAvatarDamage instead of SwordDamage.
	Empowered bool
}

// NewSwordSweep creates a sweep anchored at pos aimed toward dir.
func NewSwordSweep(ownerID string, pos, dir rl.Vector2) *SwordSweep {
	d := dir
	if d.X == 0 && d.Y == 0 {
		d = rl.NewVector2(0, 1)
	}
	return &SwordSweep{
		ID:        nextSwordID(),
		OwnerID:   ownerID,
		Position:  pos,
		BaseAngle: float32(math.Atan2(float64(d.Y), float64(d.X))),
	}
}

// Update advances the sweep. Returns false when it (including the fading
// trail) has finished and should be removed.
func (s *SwordSweep) Update(dt float32) bool {
	s.Elapsed += dt
	return s.Elapsed < SwordSweepTime+swordFadeTime
}

// progress reports sweep completion in [0, 1].
func (s *SwordSweep) progress() float32 {
	p := s.Elapsed / SwordSweepTime
	if p > 1 {
		p = 1
	}
	return p
}

// Reach returns the blade range: the normal sword's, or the avatar
// greatsword's while empowered.
func (s *SwordSweep) Reach() float32 {
	if s.Empowered {
		return SwordRange * SwordAvatarReachScale
	}
	return SwordRange
}

// Damage returns the damage this swing applies to each enemy in the arc.
func (s *SwordSweep) Damage() float32 {
	if s.Empowered {
		return SwordAvatarDamage
	}
	return SwordDamage
}

// InArc reports whether target is inside the sweep's damage area: within
// Reach() of the anchor and inside the 120° cone centered on BaseAngle.
// extraRadius extends the reach by the target's own radius.
func (s *SwordSweep) InArc(target rl.Vector2, extraRadius float32) bool {
	delta := rl.Vector2Subtract(target, s.Position)
	if rl.Vector2Length(delta) > s.Reach()+extraRadius {
		return false
	}
	ang := float32(math.Atan2(float64(delta.Y), float64(delta.X)))
	diff := angleDiff(ang, s.BaseAngle)
	half := SwordArcDeg * math.Pi / 180 / 2
	return diff <= half
}

// angleDiff returns the absolute smallest difference between two angles.
func angleDiff(a, b float32) float32 {
	d := float64(a - b)
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	for d < -math.Pi {
		d += 2 * math.Pi
	}
	if d < 0 {
		d = -d
	}
	return float32(d)
}

// Draw renders the sweep: a fading golden arc trail covering the area already
// swept, plus the sword itself (blade, guard, grip) at the current angle. All
// geometry is computed from the live anchor, so it tracks the moving owner.
func (s *SwordSweep) Draw() {
	half := float64(SwordArcDeg) / 2
	startDeg := float64(s.BaseAngle)*180/math.Pi - half
	prog := float64(s.progress())
	curDeg := startDeg + float64(SwordArcDeg)*prog

	// Trail opacity fades out after the sweep completes.
	fade := float32(1)
	if s.Elapsed > SwordSweepTime {
		fade = 1 - (s.Elapsed-SwordSweepTime)/swordFadeTime
		if fade < 0 {
			fade = 0
		}
	}

	// Swept arc trail (additive golden light). Drawn as a ring so the area
	// over the character's own body stays clear — the slash reads as
	// happening in front of her. The avatar greatsword burns brighter.
	reach := s.Reach()
	trailA, edgeA := float32(80), float32(150)
	if s.Empowered {
		trailA, edgeA = 120, 220
	}
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawRing(s.Position, SwordInnerRadius, reach,
		float32(startDeg), float32(curDeg), 24,
		rl.NewColor(255, 220, 120, uint8(trailA*fade)))
	rl.DrawRingLines(s.Position, SwordInnerRadius, reach,
		float32(startDeg), float32(curDeg), 24,
		rl.NewColor(255, 236, 160, uint8(edgeA*fade)))
	rl.EndBlendMode()

	// The sword is only drawn while sweeping (not during the fade).
	if s.Elapsed > SwordSweepTime {
		return
	}

	ang := float64(s.BaseAngle) - half*math.Pi/180 + float64(SwordArcDeg)*math.Pi/180*prog
	dir := rl.NewVector2(float32(math.Cos(ang)), float32(math.Sin(ang)))
	perp := rl.NewVector2(-dir.Y, dir.X)

	// Proportions scale up for the avatar greatsword.
	sc := float32(1)
	if s.Empowered {
		sc = 1.7
	}

	// The whole sword lives beyond SwordInnerRadius, in front of the body.
	gripStart := rl.Vector2Add(s.Position, rl.Vector2Scale(dir, SwordInnerRadius))
	guardPos := rl.Vector2Add(s.Position, rl.Vector2Scale(dir, SwordInnerRadius+22*sc))
	bladeTip := rl.Vector2Add(s.Position, rl.Vector2Scale(dir, reach))

	// Grip (brown) between the hand and the guard.
	rl.DrawLineEx(gripStart, guardPos, 7*sc, rl.NewColor(110, 70, 40, 255))
	// Pommel at the grip's base.
	rl.DrawCircleV(gripStart, 5*sc, rl.NewColor(230, 190, 80, 255))
	// Golden crossguard.
	g1 := rl.Vector2Add(guardPos, rl.Vector2Scale(perp, 14*sc))
	g2 := rl.Vector2Subtract(guardPos, rl.Vector2Scale(perp, 14*sc))
	rl.DrawLineEx(g1, g2, 6*sc, rl.NewColor(230, 190, 80, 255))

	if s.Empowered {
		// Divine greatsword: a huge GOLDEN blade with a white-gold core.
		rl.DrawLineEx(guardPos, bladeTip, 16, rl.NewColor(218, 165, 40, 255))
		rl.DrawLineEx(guardPos, bladeTip, 9, rl.NewColor(255, 215, 90, 255))
		rl.DrawLineEx(guardPos, bladeTip, 3.5, rl.NewColor(255, 248, 214, 255))
	} else {
		// Steel blade with a bright edge highlight.
		rl.DrawLineEx(guardPos, bladeTip, 9, rl.NewColor(200, 205, 215, 255))
		rl.DrawLineEx(guardPos, bladeTip, 3, rl.NewColor(245, 248, 255, 255))
	}

	// Blade tip point.
	t1 := rl.Vector2Add(rl.Vector2Subtract(bladeTip, rl.Vector2Scale(dir, 14*sc)), rl.Vector2Scale(perp, 6*sc))
	t2 := rl.Vector2Subtract(rl.Vector2Subtract(bladeTip, rl.Vector2Scale(dir, 14*sc)), rl.Vector2Scale(perp, 6*sc))
	tipEnd := rl.Vector2Add(bladeTip, rl.Vector2Scale(dir, 6*sc))
	tipColor := rl.NewColor(220, 224, 232, 255)
	if s.Empowered {
		tipColor = rl.NewColor(255, 215, 90, 255)
	}
	rl.DrawTriangle(tipEnd, t1, t2, tipColor)
	rl.DrawTriangle(tipEnd, t2, t1, tipColor)

	// Glint following the blade edge (the greatsword blazes golden light
	// along its whole length).
	rl.BeginBlendMode(rl.BlendAdditive)
	if s.Empowered {
		mid := rl.Vector2Add(guardPos, rl.Vector2Scale(dir, (reach-SwordInnerRadius-22*sc)*0.5))
		rl.DrawCircleGradient(int32(mid.X), int32(mid.Y), 46,
			rl.NewColor(255, 210, 100, 90), rl.Blank)
		rl.DrawCircleGradient(int32(bladeTip.X), int32(bladeTip.Y), 34,
			rl.NewColor(255, 240, 180, 180), rl.Blank)
	} else {
		rl.DrawCircleGradient(int32(bladeTip.X), int32(bladeTip.Y), 18,
			rl.NewColor(255, 240, 180, 160), rl.Blank)
	}
	rl.EndBlendMode()
}
