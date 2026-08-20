package network

// Join handling, split from host.go because a join is now two different
// things: a brand-new player, or a reconnect. A MsgJoin whose PlayerID is
// already in h.players is the second — see connectToHost in ui/menu.go,
// which now reuses the same ID for the lifetime of the process instead of
// minting a new one on every connection attempt.

import (
	"encoding/json"
	"log"
)

// handleJoin processes a MsgJoin on connection c. A known PlayerID is a
// reconnect: the existing PlayerState (position, health, IsDead, RespawnIn)
// is preserved, only identity fields that can legitimately change (color,
// character was already fixed on first join but resent) are refreshed and
// the absence flag is cleared. An unknown PlayerID is a fresh spawn, exactly
// as before.
func (h *Host) handleJoin(c *ClientConn, payload json.RawMessage) {
	var join JoinPayload
	if err := json.Unmarshal(payload, &join); err != nil {
		log.Printf("failed to unmarshal join payload: %v", err)
		return
	}

	h.playersMutex.Lock()
	existing, isReconnect := h.players[join.PlayerID]
	tookOverBot := false
	if isReconnect {
		// Position, health, IsDead and RespawnIn are left untouched on
		// purpose: a reconnect must land back exactly where the connection
		// dropped, not spawn fresh and healed. Cooldowns are untouched too —
		// skillCooldowns/attackCooldowns/skillCharges are keyed by playerID
		// in maps this handler never reaches.
		existing.Color = join.Color
		existing.Character = join.Character
		h.clearAbsence(existing)
		h.playersMutex.Unlock()
	} else {
		h.playersMutex.Unlock()
		// A fresh join with the class of a bot in the field TAKES IT OVER
		// (the body continues, the owner changes) rather than spawning a
		// second, separate character — see plan §3.
		tookOverBot = h.takeOverBot(join.PlayerID, join.Color, join.Character)
		if !tookOverBot {
			h.playersMutex.Lock()
			h.players[join.PlayerID] = &PlayerState{
				PlayerID:  join.PlayerID,
				X:         int(h.PlayerSpawn.X),
				Y:         int(h.PlayerSpawn.Y),
				Color:     join.Color,
				Character: join.Character,
				Health:    100,
				MaxHealth: 100,
				IsDead:    false,
			}
			h.playersMutex.Unlock()
		}
	}
	c.playerID = join.PlayerID

	// A conexao ANTIGA com o mesmo ID (se ainda nao caiu sozinha) tem que
	// fechar ANTES do roster ir para a nova, ou os dois clientes responderiam
	// pela mesma pessoa por um instante.
	h.supersedeConnection(c)

	if isReconnect {
		log.Printf("[Host] Player %s reconectou", join.PlayerID)
	} else {
		log.Printf("[Host] Player %s joined with color %s", join.PlayerID, join.Color)
	}
	// ROSTER, e nao o snapshot magro: os peers que ja estavam aqui nunca
	// viram a cor nem o personagem de quem acabou de entrar (ou voltar), e o
	// snapshot de tique nao os carrega mais.
	h.BroadcastRoster()
	// Send current state to the new client specifically.
	h.sendStateToClient(c)
	if isReconnect {
		// Quem reconecta pode ter caido num mapa que ja nao existe mais para
		// o grupo: sem isto ele ficaria sozinho no mapa antigo depois de um
		// portal atravessado durante a queda. Reaproveita MsgTravel, so que
		// para um destinatario so.
		h.sendCurrentMapTo(c)
	}
	// A client joining mid-scene has to land already paused, or it would run
	// around alone while everyone else reads.
	h.sendDialogueTo(c)
}

// supersedeConnection closes any OTHER live connection already registered
// for c.playerID. Its deferred cleanup in handleClient checks c.superseded
// and skips marking the (now reassigned) slot absent.
func (h *Host) supersedeConnection(c *ClientConn) {
	h.peersMutex.Lock()
	var stale []*ClientConn
	for _, peer := range h.peers {
		if peer != c && peer.playerID == c.playerID {
			stale = append(stale, peer)
		}
	}
	h.peersMutex.Unlock()

	for _, peer := range stale {
		peer.superseded.Store(true)
		peer.conn.Close()
		log.Printf("[Host] conexao antiga de %s fechada por reconexao", c.playerID)
	}
}

// sendCurrentMapTo tells one client which map the party is on right now, by
// reusing the same message any portal travel sends — the recipient reads
// spawn, bounds, collision and portals from the file itself, exactly like
// travelTo already does for everyone. Empty TargetSpawn means the map's own
// player_spawn.
func (h *Host) sendCurrentMapTo(c *ClientConn) {
	if h.stageMap == "" {
		return
	}
	h.sendTo(c, Message{Type: MsgTravel, Payload: MustMarshal(TravelPayload{
		TargetMap: h.stageMap,
		Reconnect: true,
	})})
}
