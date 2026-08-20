package system

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// UpdateEnemyMovement updates all enemies to pursue the nearest player.
// players: map of playerID -> PlayerState from network.
func UpdateEnemyMovement(em *entity.EntityManager, players map[string]network.PlayerState) {
	enemies := em.GetAllEnemies()

	// Convert network.PlayerState to entity.PlayerState for enemy AI
	entityPlayers := make([]entity.PlayerState, 0, len(players))
	for _, p := range players {
		entityPlayers = append(entityPlayers, entity.PlayerState{
			PlayerID:  p.PlayerID,
			X:         p.X,
			Y:         p.Y,
			Color:     p.Color,
			Health:    p.Health,
			MaxHealth: p.MaxHealth,
			IsDead:    p.IsDead,
		})
	}

	// Separation needs the full active set, which GetAllEnemies already
	// returns, so it is passed straight through.
	env := entity.MoveEnv{Solid: em.Solid, Nav: em.Nav}
	for _, e := range enemies {
		if e.IsActive {
			e.Update(0.016, entityPlayers, enemies, env) // Approximate 60fps dt
		}
	}
	entity.ResolveEnemyOverlap(enemies, em.Solid)
}

// MoveEnemiesTowardTarget moves all active enemies toward the nearest player.
// This is the host-authoritative movement logic.
func MoveEnemiesTowardTarget(em *entity.EntityManager, players []entity.PlayerState) {
	enemies := em.GetAllEnemies()

	for _, e := range enemies {
		if !e.IsActive {
			continue
		}

		// Find nearest player
		nearest := e.FindNearestPlayer(players)
		if nearest == nil {
			continue
		}

		// Move toward target
		targetPos := rl.NewVector2(float32(nearest.X), float32(nearest.Y))
		e.MoveTowardTarget(targetPos, 0.016, entity.MoveEnv{Solid: em.Solid, Nav: em.Nav}) // Approximate 60fps dt
	}
}
