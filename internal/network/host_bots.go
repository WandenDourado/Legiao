package network

// A vacant class becomes a bot: an ordinary entry in h.players (see
// doc/plan_bots_de_classe.md §2), identified by a synthetic ID of the form
// "bot_<classe>" — the same shape network/last_stand_heroes.go already uses
// for "npc_<classe>". Because it lives in h.players it gets identity
// broadcast, death/revive, PlaceEveryoneAtSpawn, cooldown gating and
// checkGameOver for free; nothing in this file duplicates any of that.

import (
	"log"
	"strings"

	"github.com/WandenDourado/Legiao/internal/bot"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/nav"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const botIDPrefix = "bot_"

// botIDFor returns the synthetic PlayerID for the bot of a given class.
func botIDFor(char entity.CharacterType) string {
	return botIDPrefix + string(char)
}

// isBotID reports whether a PlayerID names a bot rather than a human.
func isBotID(id string) bool {
	return strings.HasPrefix(id, botIDPrefix)
}

// botCharacterFromID recovers the class from a bot's synthetic ID.
func botCharacterFromID(id string) entity.CharacterType {
	return entity.CharacterType(strings.TrimPrefix(id, botIDPrefix))
}

// botRuntime is per-bot state that never goes on the wire: the AI's memory,
// and the headless animation bookkeeping a bot needs since it has no
// *entity.Player to call updateAnimation on. Guarded by playersMutex, the
// same lock that owns the h.players slot this state belongs to.
type botRuntime struct {
	brain     bot.Brain
	character entity.CharacterType
	animTimer float32
	frame     int
	lastRow   int
	// stuck-detection state for host_bot_move.go.
	lastPos  rl.Vector2
	stuckFor float32
	// detourDir is the world direction ResolveDetour committed to going
	// around whatever is blocking the bot, or the zero vector when the
	// direct path is clear. Passed back in every frame so the bot commits
	// to one way around an obstacle instead of re-deciding — and pushing —
	// every frame (nav plan §1, "remendo imediato"; collision.ResolveDetour).
	detourDir rl.Vector2
	// nav is this bot's route state: the mesh-planned path toward whatever
	// Intent.Dest the brain set, when the straight line is blocked. One
	// Follower per bot, kept across ticks so it can commit to a route
	// instead of re-searching every frame (doc/plan_navegacao_bots_monstros.md §3).
	nav nav.Follower
	// arenaLocked mirrors arenaGate.returnLocked (game/arena_gate.go) but per
	// bot instead of per human crossing: this bot's foot box has been inside
	// the arena's one-way zone at least once since the flag was last cleared
	// (doc/tilemap.md "Arena de mão única"). Reset by ResetBotArenaLocks on
	// stage restart and map load/travel — never touched by ReconcileBots
	// itself, or a human joining mid-run would free every bot still sealed
	// inside the arena.
	arenaLocked bool
}

// ReconcileBots keeps the invariant: exactly one bot per class with no
// human playing it, and none for a class a human already plays. A human
// counts even while ABSENT (host_absence.go) — otherwise the five-minute
// reconnect grace would spawn a bot right next to the parked body of the
// player it is about to replace, and the two would fight over the slot on
// rejoin.
//
// Called from exactly four places: World.ApplyToHost (map load/travel),
// Host.handleJoin (join/reconnect), Host.tickAbsence (slot freed for good),
// Host.ResetStage (F5). Every creation or removal ends in BroadcastRoster —
// identity only travels on the roster (wire.go/slimPlayer), never the
// per-tick snapshot.
func (h *Host) ReconcileBots() {
	h.playersMutex.Lock()

	var created, removedIDs []string
	for _, def := range entity.AllCharacters() {
		char := def.Type
		humanPresent := false
		botKey := ""
		for id, p := range h.players {
			if entity.CharacterType(p.Character) != char {
				continue
			}
			if isBotID(id) {
				botKey = id
			} else {
				humanPresent = true
			}
		}

		switch {
		case humanPresent && botKey != "":
			delete(h.players, botKey)
			removedIDs = append(removedIDs, botKey)
		case !humanPresent && botKey == "":
			id := botIDFor(char)
			state := PlayerState{
				PlayerID:  id,
				X:         int(h.PlayerSpawn.X),
				Y:         int(h.PlayerSpawn.Y),
				Color:     "", // empty on purpose: the class art is the identity (plan §1)
				Character: string(char),
				Health:    100,
				MaxHealth: 100,
			}
			if vacated, ok := h.vacatedBodies[char]; ok {
				// A human of this class just aged out of ReconnectGrace
				// (tickAbsence): the bot inherits where that body was
				// instead of popping up fresh at the map spawn.
				state.X, state.Y = vacated.X, vacated.Y
				state.Health, state.MaxHealth = vacated.Health, vacated.MaxHealth
				state.IsDead = vacated.IsDead
				state.RespawnIn = vacated.RespawnIn
				delete(h.vacatedBodies, char)
			}
			h.players[id] = &state
			created = append(created, id)
		}
	}

	if h.bots == nil {
		h.bots = make(map[string]*botRuntime)
	}
	for _, id := range removedIDs {
		delete(h.bots, id)
	}
	for _, id := range created {
		char := botCharacterFromID(id)
		h.bots[id] = &botRuntime{brain: bot.For(char), character: char}
	}

	h.playersMutex.Unlock()

	for _, id := range created {
		log.Printf("[Host] %s entra em campo (classe vaga)", id)
	}
	for _, id := range removedIDs {
		log.Printf("[Host] %s sai de campo (humano assumiu a classe)", id)
	}
	if len(created) > 0 || len(removedIDs) > 0 {
		h.BroadcastRoster()
	}
}
