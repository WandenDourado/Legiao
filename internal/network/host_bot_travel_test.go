package network

// HumansAtPortal (buildBotView, plan doc/plan_avanco_bots_e_gargula.md §A2
// cause 4): ignores bots and dead humans, counts a human already InPortal
// directly, and counts a live human within bot.PortalEscortRadius of an
// open portal rectangle.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestHumansAtPortalIgnoresBotsAndDead(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"bot_arqueiro":    {PlayerID: "bot_arqueiro", X: 0, Y: 0, Character: "arqueiro"},
			"human_mago":      {PlayerID: "human_mago", X: 900, Y: 900, Character: "mago", IsDead: true},
			"bot_paladina":    {PlayerID: "bot_paladina", X: 950, Y: 950, Character: "paladina"},
			"human_sacerdote": {PlayerID: "human_sacerdote", X: 5000, Y: 5000, Character: "sacerdotisa"},
		},
		bots: map[string]*botRuntime{
			"bot_arqueiro": {character: entity.CharArqueiro},
			"bot_paladina": {character: entity.CharPaladina},
		},
	}
	h.SetPartyPortals([]rl.Rectangle{{X: 900, Y: 900, Width: 100, Height: 100}}, true)
	defer h.SetPartyPortals(nil, false)

	v, _, _, ok := h.buildBotView("bot_arqueiro", 1.0/60)
	if !ok {
		t.Fatal("expected buildBotView to succeed")
	}
	// Every candidate near the portal rectangle is either a bot or a dead
	// human; the one living human is far away. Bots and the dead human must
	// not count.
	if v.HumansAtPortal {
		t.Fatal("expected HumansAtPortal false: only bots and a dead human are near the portal")
	}
}

func TestHumansAtPortalTrueForALivingHumanInPortal(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"bot_arqueiro": {PlayerID: "bot_arqueiro", X: 0, Y: 0, Character: "arqueiro"},
			"human_mago":   {PlayerID: "human_mago", X: 9999, Y: 9999, Character: "mago", InPortal: true},
		},
		bots: map[string]*botRuntime{
			"bot_arqueiro": {character: entity.CharArqueiro},
		},
	}
	// InPortal counts directly, without measuring distance — the position
	// here is deliberately far from any rectangle to prove that.
	h.SetPartyPortals([]rl.Rectangle{{X: 0, Y: 0, Width: 100, Height: 100}}, true)
	defer h.SetPartyPortals(nil, false)

	v, _, _, ok := h.buildBotView("bot_arqueiro", 1.0/60)
	if !ok {
		t.Fatal("expected buildBotView to succeed")
	}
	if !v.HumansAtPortal {
		t.Fatal("expected HumansAtPortal true: a living human already has InPortal set")
	}
}

func TestHumansAtPortalTrueWithinEscortRadius(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"bot_arqueiro": {PlayerID: "bot_arqueiro", X: 0, Y: 0, Character: "arqueiro"},
			"human_mago":   {PlayerID: "human_mago", X: 1000, Y: 0, Character: "mago"},
		},
		bots: map[string]*botRuntime{
			"bot_arqueiro": {character: entity.CharArqueiro},
		},
	}
	// Portal rectangle centred at (0,0); the human at (1000,0) is within
	// bot.PortalEscortRadius (1200) of it, though not standing inside it.
	h.SetPartyPortals([]rl.Rectangle{{X: -50, Y: -50, Width: 100, Height: 100}}, true)
	defer h.SetPartyPortals(nil, false)

	v, _, _, ok := h.buildBotView("bot_arqueiro", 1.0/60)
	if !ok {
		t.Fatal("expected buildBotView to succeed")
	}
	if !v.HumansAtPortal {
		t.Fatal("expected HumansAtPortal true: the human is within PortalEscortRadius of the portal")
	}
}

func TestHumansAtPortalFalseBeyondEscortRadius(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"bot_arqueiro": {PlayerID: "bot_arqueiro", X: 0, Y: 0, Character: "arqueiro"},
			"human_mago":   {PlayerID: "human_mago", X: 7400, Y: 0, Character: "mago"},
		},
		bots: map[string]*botRuntime{
			"bot_arqueiro": {character: entity.CharArqueiro},
		},
	}
	h.SetPartyPortals([]rl.Rectangle{{X: -50, Y: -50, Width: 100, Height: 100}}, true)
	defer h.SetPartyPortals(nil, false)

	v, _, _, ok := h.buildBotView("bot_arqueiro", 1.0/60)
	if !ok {
		t.Fatal("expected buildBotView to succeed")
	}
	if v.HumansAtPortal {
		t.Fatal("expected HumansAtPortal false: the only human is far beyond PortalEscortRadius")
	}
}
