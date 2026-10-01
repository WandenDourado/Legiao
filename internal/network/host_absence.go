package network

// A dropped connection is not a lost player.
//
// Before this, handleClient's defer deleted the player the instant its
// decode loop errored — there was no state between "in the match" and
// "gone". A phone whose screen locks does not necessarily break the TCP
// connection (the OS pauses the process, the socket can sit idle for
// minutes), so a player who could still come back was erased the moment the
// host noticed silence, and the only way back in was restarting the whole
// run.
//
// Now a dropped connection marks the player ABSENT instead: it keeps its
// slot, its position, its health, its cooldowns (those already live in maps
// keyed by playerID and are untouched by any of this). It is still drawn,
// still vulnerable — if it dies while absent, it dies. Only two things
// change while absent: it is excluded from every "did the whole party..."
// decision (PresentPlayers, below), and after ReconnectGrace with no rejoin
// the slot is finally freed for good.

import (
	"log"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// ReconnectGrace is how long an absent player's slot is held before the host
// gives up and removes it for good.
//
// Five minutes, not a TCP timeout: a locked Android screen pauses the app
// rather than killing it, and the socket can look alive for a long while with
// nothing flowing through it. This is a deliberately generous window for a
// player to unlock their phone and come back, not a network tuning constant.
const ReconnectGrace = 5 * time.Minute

// markAbsent flags a connected player as disconnected without touching
// anything else about it. Caller must hold playersMutex.
func (h *Host) markAbsent(playerID string) {
	p, ok := h.players[playerID]
	if !ok || p.Absent {
		return
	}
	p.Absent = true
	p.absentSince = time.Now()
	log.Printf("[Host] Player %s ausente; janela de reconexao de %s", playerID, ReconnectGrace)
}

// clearAbsence marks a rejoined player present again. Caller must hold
// playersMutex.
func (h *Host) clearAbsence(p *PlayerState) {
	p.Absent = false
	p.absentSince = time.Time{}
}

// tickAbsence removes any player whose absence has outlasted ReconnectGrace.
// Called once per simulation tick from UpdateSimulation, the same place every
// other host-owned timer advances (tickRespawns, tickInvulnerability).
//
// Once removed, the slot is gone for good: a later reconnect with the same ID
// arrives as a fresh MsgJoin and starts over as a new player, same as anyone
// joining for the first time.
func (h *Host) tickAbsence() {
	var removed []string
	h.playersMutex.Lock()
	for id, p := range h.players {
		if !p.Absent {
			continue
		}
		if time.Since(p.absentSince) >= ReconnectGrace {
			if !isBotID(id) {
				// The replacement bot (ReconcileBots, below) inherits where
				// this body was instead of popping up fresh at the map
				// spawn — the symmetric operation to a human taking over a
				// bot's class (host_bot_takeover.go).
				if h.vacatedBodies == nil {
					h.vacatedBodies = make(map[entity.CharacterType]PlayerState)
				}
				h.vacatedBodies[entity.CharacterType(p.Character)] = *p
			}
			delete(h.players, id)
			removed = append(removed, id)
		}
	}
	h.playersMutex.Unlock()

	if len(removed) == 0 {
		return
	}
	for _, id := range removed {
		log.Printf("[Host] Player %s excedeu a janela de reconexao (%s); removido", id, ReconnectGrace)
	}
	// ReconcileBots broadcasts the roster itself when a bot is created or
	// removed, but not when a class still has another human present after
	// this one is gone — that case still needs the roster refreshed so
	// every peer drops the removed player.
	h.ReconcileBots()
	h.BroadcastRoster()
}

// PresentPlayers is GetAllPlayers filtered to players who are not currently
// absent.
//
// Every "did the whole party..." decision has to read this instead of
// GetAllPlayers: the host's own Game Over check, partyIsFalling (the last
// stand rescue), partyArrived (the climax gate) and the portal door. An
// absent player parked mid-field would otherwise hold the group hostage for
// up to five minutes — a Game Over that never comes, a portal that never
// opens.
//
// GetAllPlayers stays what it was: the full snapshot, absent players
// included, which is what drawing and the HUD need — an absent player is
// still on screen with its "reconectando" marker.
func PresentPlayers() map[string]PlayerState {
	all := GetAllPlayers()
	for id, p := range all {
		if p.Absent {
			delete(all, id)
		}
	}
	return all
}
