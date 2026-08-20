package network

// Death and revival are host-authoritative: the host arms the countdown when a
// player dies, ticks it down, and brings the player back. Clients only display
// the number that rides along in PlayerState.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// markPlayerDead flags a player as dead and arms its revive countdown.
// Caller must hold playersMutex.
func (h *Host) markPlayerDead(p *PlayerState) {
	p.Health = 0
	p.IsDead = true
	p.RespawnIn = entity.RespawnDelay
	log.Printf("[Host] Player %s died, revive em %.0fs", p.PlayerID, p.RespawnIn)
}

// tickRespawns advances every dead player's countdown and revives whoever
// reaches zero, at RespawnHealthPercent of max health.
//
// The body gets up where it fell — X/Y are deliberately left untouched. A dead
// player stops sending input, so the position the host holds is still the spot
// where it died. Reviving at the map spawn instead would teleport the player
// across the map, undoing the walk back to the fight.
//
// While the run is over the timers are frozen: reviving into a Game Over would
// quietly undo it, and the only way back is the host restarting the stage.
func (h *Host) tickRespawns(dt float32) {
	if GameOver {
		return
	}

	var revived []PlayerState
	h.playersMutex.Lock()
	for _, p := range h.players {
		if !p.IsDead {
			continue
		}
		p.RespawnIn -= dt
		if p.RespawnIn > 0 {
			continue
		}
		p.RespawnIn = 0
		p.IsDead = false
		p.Health = p.MaxHealth * entity.RespawnHealthPercent
		revived = append(revived, *p)
	}
	h.playersMutex.Unlock()

	for _, p := range revived {
		log.Printf("[Host] Player %s ressuscitou com %.0f de vida", p.PlayerID, p.Health)
		h.broadcastRespawn(p)
	}
}

// broadcastRespawn tells every client that a player is back on their feet.
func (h *Host) broadcastRespawn(p PlayerState) {
	h.broadcast(Message{Type: MsgRespawn, Payload: MustMarshal(RespawnPayload{
		PlayerID: p.PlayerID,
		Health:   p.Health,
		X:        p.X,
		Y:        p.Y,
	})})
}
