package skill

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// --- Spectral Legion collection (Necromante ultimate) ---

// LegionEvent reports one combat outcome resolved on the host: either a
// specter bite landing on an enemy, or a specter death (SpecterDied set).
type LegionEvent struct {
	OwnerID     string
	EnemyID     string
	Damage      float32
	EnemyDied   bool
	SpecterDied bool
	SpecterPos  rl.Vector2 // bite/death position (for client sync)
}

// ActivateLegion summons a fresh legion for ownerID (a recast replaces any
// remnant with a full new circle of specters).
func ActivateLegion(m *Manager, ownerID string, pos rl.Vector2) {
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	m.Legions[ownerID] = NewLegion(ownerID, pos)
}

// SetLegionAnchor keeps a legion glued to its owner's position.
func (m *Manager) SetLegionAnchor(ownerID string, pos rl.Vector2) {
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	if l, ok := m.Legions[ownerID]; ok {
		l.Anchor = pos
	}
}

// DissolveLegion marks every remaining specter as dying (owner died/left).
// Returns true if any specter was newly dissolved, so callers can broadcast
// the event exactly once.
// HasLegion reports whether an owner still has specters on the field. The
// scripted rescue uses it to know when its summoned Necromante is done.
func (m *Manager) HasLegion(ownerID string) bool {
	m.necroMutex.RLock()
	defer m.necroMutex.RUnlock()
	_, ok := m.Legions[ownerID]
	return ok
}

func (m *Manager) DissolveLegion(ownerID string) bool {
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	l, ok := m.Legions[ownerID]
	if !ok {
		return false
	}
	any := false
	for _, s := range l.Specters {
		if !s.Dying {
			s.Dying = true
			any = true
		}
	}
	return any
}

// StepLegions advances every legion on the HOST: specters hunt the nearest
// enemy within the owner's leash radius and, while engaged, bite at the
// game's highest attack speed; the enemy fights back at its own (slower)
// attack cadence and can kill the specter. Returns combat events for the
// caller to broadcast (dead enemies are NOT removed here).
func StepLegions(m *Manager, enemies []*entity.Enemy, collisionRects []rl.Rectangle, dt float32) []LegionEvent {
	events := make([]LegionEvent, 0)
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	for owner, l := range m.Legions {
		l.Time += dt
		for _, s := range l.Specters {
			e := nearestHuntable(enemies, l.Anchor, s.Position)
			var target *rl.Vector2
			var radius float32
			if e != nil {
				target = &e.Position
				radius = e.Radius
			}
			if !l.advance(s, target, radius, dt, collisionRects) || e == nil || s.Dying {
				continue
			}
			// Specter bite: fastest attack in the game.
			s.hitTimer -= dt
			if s.hitTimer <= 0 {
				s.hitTimer = SpecterAttackEvery
				s.lungeT = specterLungeTime
				died := e.TakeDamage(SpecterDamage)
				events = append(events, LegionEvent{
					OwnerID: owner, EnemyID: e.ID, Damage: SpecterDamage,
					EnemyDied: died, SpecterPos: s.Position,
				})
			}
			// The enemy retaliates at its own attack speed.
			s.hurtTimer -= dt
			if s.hurtTimer <= 0 {
				s.hurtTimer = e.AttackCooldown
				s.Health -= e.AttackDamage
				if s.Health <= 0 {
					s.Dying = true
					events = append(events, LegionEvent{
						OwnerID: owner, SpecterDied: true, SpecterPos: s.Position,
					})
				}
			}
		}
		l.separate(collisionRects)
		l.prune()
		if l.Spent() {
			delete(m.Legions, owner)
		}
	}
	return events
}

// nearestHuntable returns the closest active enemy to the specter that is
// still inside the owner's leash radius, or nil.
func nearestHuntable(enemies []*entity.Enemy, anchor, from rl.Vector2) *entity.Enemy {
	var best *entity.Enemy
	bestDist := float32(math.MaxFloat32)
	for _, e := range enemies {
		if !e.IsActive {
			continue
		}
		if rl.Vector2Distance(anchor, e.Position) > LegionLeashRadius {
			continue
		}
		d := rl.Vector2Distance(from, e.Position)
		if d < bestDist {
			bestDist = d
			best = e
		}
	}
	return best
}

// AdvanceLegions animates legions on CLIENTS: specters chase the same targets
// and replay the bite lunge cosmetically, but never resolve damage — specter
// deaths arrive as host events via KillLegionSpecterNear. enemyPositions are
// the latest enemy snapshots.
func (m *Manager) AdvanceLegions(dt float32, enemyPositions []rl.Vector2, collisionRects []rl.Rectangle) {
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	for owner, l := range m.Legions {
		l.Time += dt
		for _, s := range l.Specters {
			target := nearestPosition(enemyPositions, l.Anchor, s.Position)
			if l.advance(s, target, 14, dt, collisionRects) && !s.Dying {
				s.hitTimer -= dt
				if s.hitTimer <= 0 {
					s.hitTimer = SpecterAttackEvery
					s.lungeT = specterLungeTime
				}
			}
		}
		l.separate(collisionRects)
		l.prune()
		if l.Spent() {
			delete(m.Legions, owner)
		}
	}
}

// nearestPosition mirrors nearestHuntable for raw snapshot positions.
func nearestPosition(positions []rl.Vector2, anchor, from rl.Vector2) *rl.Vector2 {
	var best *rl.Vector2
	bestDist := float32(math.MaxFloat32)
	for i := range positions {
		p := positions[i]
		if rl.Vector2Distance(anchor, p) > LegionLeashRadius {
			continue
		}
		d := rl.Vector2Distance(from, p)
		if d < bestDist {
			bestDist = d
			best = &positions[i]
		}
	}
	return best
}

// KillLegionSpecterNear mirrors a host specter death on the client.
func (m *Manager) KillLegionSpecterNear(ownerID string, pos rl.Vector2) {
	m.necroMutex.Lock()
	defer m.necroMutex.Unlock()
	if l, ok := m.Legions[ownerID]; ok {
		l.killNearest(pos)
	}
}

// DrawLegions renders every legion in world space.
func (m *Manager) DrawLegions() {
	m.necroMutex.RLock()
	defer m.necroMutex.RUnlock()
	for _, l := range m.Legions {
		l.Draw()
	}
}
