package skill

// Legion: the Necromante's ultimate (Legião Espectral). Summons LegionCount
// specters in a circle around the caster. They orbit him and dart at enemies
// that come inside the leash radius.
//
// A specter is NOT a one-hit kamikaze: it stays on the field and bites at the
// highest attack speed in the game, while the enemy fights back at its own,
// slower cadence and can kill it (StepLegions in legion_manager.go). That is
// what gives the skill its shape — strong against many fragile enemies, weak
// against a single high-health one, because there the DPS race is lost before
// the health bar empties. The legion ends when every specter has died.
//
// This comment used to say each specter consumed itself on striking, which was
// never what the code did. It is worth keeping accurate: read literally, it
// puts the ultimate's damage ceiling at LegionCount × SpecterDamage, which is
// about a thirtieth of the truth and turns any balance reasoning built on it
// upside down.

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/collision"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Legion owns the specters summoned by one caster.
type Legion struct {
	OwnerID  string
	Anchor   rl.Vector2 // owner position, synced every frame
	Time     float32
	Specters []*Specter
	// push e o acumulador reaproveitado por `separate`. Fica no Legion, e nao
	// numa alocacao por quadro, porque `separate` roda a cada quadro para cada
	// legiao em campo.
	push []rl.Vector2
}

// NewLegion summons a full circle of specters around pos.
func NewLegion(ownerID string, pos rl.Vector2) *Legion {
	l := &Legion{OwnerID: ownerID, Anchor: pos}
	for i := 0; i < LegionCount; i++ {
		ang := float32(i) / LegionCount * 2 * math.Pi
		slot := l.slotPoint(ang)
		l.Specters = append(l.Specters, &Specter{
			HomeAngle: ang,
			Position:  slot,
			Facing:    rl.Vector2Normalize(rl.Vector2Subtract(slot, pos)),
			Phase:     float32(i) * 0.37,
			Health:    SpecterMaxHealth,
			hurtTimer: 1.0, // enemies need a moment before their first blow lands
		})
	}
	return l
}

// slotPoint is the specter's idle position on the slowly rotating circle.
func (l *Legion) slotPoint(homeAngle float32) rl.Vector2 {
	a := float64(homeAngle + l.Time*0.7)
	return rl.NewVector2(
		l.Anchor.X+float32(math.Cos(a))*LegionOrbitRadius,
		l.Anchor.Y+float32(math.Sin(a))*LegionOrbitRadius*0.8, // squashed pseudo-3D
	)
}

// Spent reports whether every specter has been consumed (legion over).
func (l *Legion) Spent() bool { return len(l.Specters) == 0 }

// moveSpecter displaces a specter by delta, respecting solid obstacles with
// axis-sliding (like players). A specter that somehow starts inside a solid
// (e.g., summoned over one) moves freely until it escapes.
func moveSpecter(s *Specter, delta rl.Vector2, solid collision.Solid) {
	if delta.X == 0 && delta.Y == 0 {
		return
	}
	size := SpecterRadius * 1.6
	if blocked(solid, s.Position, size) {
		s.Position = rl.Vector2Add(s.Position, delta)
		return
	}
	next := rl.Vector2Add(s.Position, delta)
	if !blocked(solid, next, size) {
		s.Position = next
		return
	}
	nx := rl.NewVector2(s.Position.X+delta.X, s.Position.Y)
	if delta.X != 0 && !blocked(solid, nx, size) {
		s.Position = nx
		return
	}
	ny := rl.NewVector2(s.Position.X, s.Position.Y+delta.Y)
	if delta.Y != 0 && !blocked(solid, ny, size) {
		s.Position = ny
	}
}

// separate pushes overlapping specters apart so the pack never stacks up on
// itself (pushes also respect solid obstacles).
//
// Os empurroes de TODOS os pares sao somados primeiro e aplicados UMA vez por
// espectro no fim. Antes, cada par movia os dois espectros na hora: com trinta
// espectros sao 435 pares, ou seja ate 870 `moveSpecter` por quadro contra os
// 30 de agora. Alem de 29x mais barato, somar antes e mais estavel — aplicando
// par a par, o empurrao de A contra B mudava a posicao que o par seguinte lia,
// e a ordem do laco virava parte do resultado.
func (l *Legion) separate(solid collision.Solid) {
	minDist := SpecterRadius * 1.7
	if cap(l.push) < len(l.Specters) {
		l.push = make([]rl.Vector2, len(l.Specters))
	}
	push := l.push[:len(l.Specters)]
	for i := range push {
		push[i] = rl.Vector2{}
	}

	for i := 0; i < len(l.Specters); i++ {
		a := l.Specters[i]
		if a.Dying || a.Age < specterSpawnTime {
			continue
		}
		for j := i + 1; j < len(l.Specters); j++ {
			b := l.Specters[j]
			if b.Dying || b.Age < specterSpawnTime {
				continue
			}
			to := rl.Vector2Subtract(b.Position, a.Position)
			dist := rl.Vector2Length(to)
			if dist >= minDist {
				continue
			}
			if dist < 0.1 { // perfectly stacked: nudge apart deterministically
				to = rl.NewVector2(
					float32(math.Cos(float64(a.HomeAngle))),
					float32(math.Sin(float64(a.HomeAngle))))
			} else {
				to = rl.Vector2Scale(to, 1/dist)
			}
			amount := (minDist - dist) / 2
			push[i] = rl.Vector2Subtract(push[i], rl.Vector2Scale(to, amount))
			push[j] = rl.Vector2Add(push[j], rl.Vector2Scale(to, amount))
		}
	}

	for i, s := range l.Specters {
		moveSpecter(s, push[i], solid)
	}
	l.push = push
}

// advance moves one specter for this frame: dying specters dissolve, hunters
// dart at their target, idle specters glide back to their circle slot.
// target is nil when there is nothing to hunt. Returns true while the specter
// is ENGAGED (in biting range of its target) — the caller runs the combat
// timers (bites out, enemy blows in).
func (l *Legion) advance(s *Specter, target *rl.Vector2, targetRadius, dt float32, solid collision.Solid) bool {
	s.Age += dt
	if s.lungeT > 0 {
		s.lungeT -= dt
	}
	if s.Dying {
		s.DieAge += dt
		return false
	}
	if s.Age < specterSpawnTime {
		return false // still rising from the ground
	}
	if target != nil {
		to := rl.Vector2Subtract(*target, s.Position)
		dist := rl.Vector2Length(to)
		if dist > 1 {
			s.Facing = rl.Vector2Scale(to, 1/dist)
		}
		if dist <= targetRadius+SpecterRadius*0.7 {
			return true // engaged: hold position and bite
		}
		step := SpecterSpeed * dt
		if step > dist {
			step = dist
		}
		moveSpecter(s, rl.Vector2Scale(to, step/dist), solid)
		return false
	}
	// No prey: ease back toward the orbit slot, facing outward (on guard).
	slot := l.slotPoint(s.HomeAngle)
	to := rl.Vector2Subtract(slot, s.Position)
	dist := rl.Vector2Length(to)
	if dist > 1 {
		step := SpecterSpeed * 0.75 * dt
		if step > dist {
			step = dist
		}
		moveSpecter(s, rl.Vector2Scale(to, step/dist), solid)
	}
	out := rl.Vector2Subtract(s.Position, l.Anchor)
	if rl.Vector2Length(out) > 1 {
		s.Facing = rl.Vector2Normalize(out)
	}
	return false
}

// prune drops fully dissolved specters.
func (l *Legion) prune() {
	kept := l.Specters[:0]
	for _, s := range l.Specters {
		if !s.gone() {
			kept = append(kept, s)
		}
	}
	l.Specters = kept
}

// killNearest marks the living specter closest to pos as dying (used by
// clients to mirror host-authoritative specter deaths).
func (l *Legion) killNearest(pos rl.Vector2) {
	var best *Specter
	bestDist := float32(math.MaxFloat32)
	for _, s := range l.Specters {
		if s.Dying {
			continue
		}
		d := rl.Vector2Distance(s.Position, pos)
		if d < bestDist {
			bestDist = d
			best = s
		}
	}
	if best != nil {
		best.Dying = true
		best.Position = pos
	}
}

// Draw renders the legion: a faint summoning circle under the owner while
// specters remain, then every ghost.
func (l *Legion) Draw() {
	if l.Spent() {
		return
	}
	// Necro-purple summoning ring on the ground, counter-rotating arcs.
	rl.BeginBlendMode(rl.BlendAdditive)
	spin := l.Time * 40
	for i := 0; i < 3; i++ {
		start := spin + float32(i)*120
		rl.DrawRing(l.Anchor, LegionOrbitRadius-4, LegionOrbitRadius-1,
			start, start+70, 24, rl.Fade(gravePurple, 0.22))
	}
	rl.EndBlendMode()
	for _, s := range l.Specters {
		s.Draw()
	}
}
