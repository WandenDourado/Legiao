package network

// Host-side simulation for the Necromante: graveyard damage/slow ticks,
// spectral legion hunting, and basic-attack lifesteal. All health mutations
// are authoritative here and synced to clients via combat/ultimate events.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// handleNecroTick advances graveyards and spectral legions for this frame.
func (h *Host) handleNecroTick(dt float32) {
	enemies := h.EntityManager.GetAllEnemies()

	// Cursed ground: damage-over-time + slow.
	for _, ev := range skill.StepGraveyards(h.Skills, enemies, dt) {
		if ev.Died {
			h.EntityManager.RemoveEnemy(ev.EnemyID)
			h.broadcastCombatEvent("death", ev.EnemyID, "enemy", 0, "")
		} else {
			h.broadcastCombatEvent("damage", ev.EnemyID, "enemy", ev.Damage, "")
		}
	}

	// Legions follow their owners; a dead owner's legion dissolves.
	h.playersMutex.RLock()
	for id, p := range h.players {
		if p.IsDead {
			continue
		}
		h.Skills.SetLegionAnchor(id, rl.NewVector2(float32(p.X), float32(p.Y)))
	}
	dissolved := make([]string, 0)
	for id, p := range h.players {
		if p.IsDead {
			dissolved = append(dissolved, id)
		}
	}
	h.playersMutex.RUnlock()
	for _, id := range dissolved {
		if h.Skills.DissolveLegion(id) {
			h.broadcastUltimate("legion_end", id, rl.Vector2{}, rl.Vector2{})
		}
	}

	// Specter combat: rapid bites on enemies; enemies strike back and can
	// kill specters (mirrored on clients via specter_die events).
	for _, ev := range skill.StepLegions(h.Skills, enemies, h.collisionRects, dt) {
		if ev.SpecterDied {
			h.broadcastUltimate("specter_die", ev.OwnerID, ev.SpecterPos, rl.Vector2{})
			continue
		}
		if ev.EnemyDied {
			h.EntityManager.RemoveEnemy(ev.EnemyID)
			h.broadcastCombatEvent("death", ev.EnemyID, "enemy", 0, "")
		} else {
			h.broadcastCombatEvent("damage", ev.EnemyID, "enemy", ev.Damage, "")
		}
	}
}

// applyProjectileLifesteal restores health to a projectile's owner after it
// hit an enemy (Necromante shadow skulls). Synced via "heal" combat events
// (value = new absolute health), matching the holy-bolt pattern.
func (h *Host) applyProjectileLifesteal(proj *entity.Projectile) {
	if proj.Lifesteal <= 0 {
		return
	}
	h.playersMutex.Lock()
	defer h.playersMutex.Unlock()
	p, ok := h.players[proj.OwnerID]
	if !ok || p.IsDead || p.Health >= p.MaxHealth {
		return
	}
	p.Health += proj.Lifesteal
	if p.Health > p.MaxHealth {
		p.Health = p.MaxHealth
	}
	h.broadcastCombatEvent("heal", proj.OwnerID, "player", p.Health, proj.OwnerID)
}
