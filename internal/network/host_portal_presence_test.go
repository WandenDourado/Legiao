package network

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// bigRectAround is a rectangle generous enough to swallow any character's
// foot-box offset from its raw position — the tests care about inside vs.
// outside, not the exact few pixels of GroundBoxAt's own offset.
func bigRectAround(pos rl.Vector2) rl.Rectangle {
	return rl.NewRectangle(pos.X-500, pos.Y-500, 1000, 1000)
}

func TestTickPortalPresenceMarksInsideAndClearsOutside(t *testing.T) {
	h := &Host{players: map[string]*PlayerState{
		"inside":  {PlayerID: "inside", X: 100, Y: 100, Character: "mago"},
		"outside": {PlayerID: "outside", X: 9000, Y: 9000, Character: "mago"},
	}}
	h.SetPartyPortals([]rl.Rectangle{bigRectAround(rl.NewVector2(100, 100))}, true)

	h.tickPortalPresence()

	if !h.players["inside"].InPortal {
		t.Error("expected the player standing in the rectangle to be marked InPortal")
	}
	if h.players["outside"].InPortal {
		t.Error("expected the player far from the rectangle to stay InPortal=false")
	}
}

func TestTickPortalPresenceNeverMarksTheDead(t *testing.T) {
	h := &Host{players: map[string]*PlayerState{
		"corpse": {PlayerID: "corpse", X: 100, Y: 100, Character: "mago", IsDead: true},
	}}
	h.SetPartyPortals([]rl.Rectangle{bigRectAround(rl.NewVector2(100, 100))}, true)

	h.tickPortalPresence()

	if h.players["corpse"].InPortal {
		t.Error("a dead player must never enter portal-wait — it travels with the party, still fallen")
	}
}

func TestTickPortalPresenceInactivePortalMarksNobody(t *testing.T) {
	h := &Host{players: map[string]*PlayerState{
		"p1": {PlayerID: "p1", X: 100, Y: 100, Character: "mago", InPortal: true},
	}}
	// active=false mirrors both "not fully materialised" and "current.arrived"
	// from game/portal.go — either way, nobody should be waiting.
	h.SetPartyPortals([]rl.Rectangle{bigRectAround(rl.NewVector2(100, 100))}, false)

	h.tickPortalPresence()

	if h.players["p1"].InPortal {
		t.Error("an inactive portal must clear InPortal, not just refuse to set it")
	}
}

func TestTickBotsSkipsAPlayerWaitingInThePortal(t *testing.T) {
	h := &Host{
		EntityManager: nil,
		players: map[string]*PlayerState{
			"bot_mago": {PlayerID: "bot_mago", X: 100, Y: 100, Character: "mago", InPortal: true},
		},
		bots: map[string]*botRuntime{
			"bot_mago": {character: "mago"},
		},
	}

	h.tickBots(1.0 / 60)

	p := h.players["bot_mago"]
	if p.X != 100 || p.Y != 100 {
		t.Fatalf("expected a bot waiting in the portal to not move, got (%d,%d)", p.X, p.Y)
	}
}

func TestPlaceEveryoneAtSpawnClearsPortalWait(t *testing.T) {
	h := &Host{
		players: map[string]*PlayerState{
			"p1": {PlayerID: "p1", InPortal: true},
			"p2": {PlayerID: "p2", InPortal: true},
		},
		peers:       map[string]*ClientConn{},
		PlayerSpawn: rl.NewVector2(50, 60),
	}

	h.PlaceEveryoneAtSpawn()

	for id, p := range h.players {
		if p.InPortal {
			t.Errorf("expected %s.InPortal cleared after arriving on the new map", id)
		}
	}
}

func TestResetStageClearsPortalWait(t *testing.T) {
	h := &Host{
		EntityManager:   entity.NewEntityManager(),
		Skills:          skill.NewManager(),
		players:         map[string]*PlayerState{"p1": {PlayerID: "p1", InPortal: true, Health: 10, MaxHealth: 100}},
		peers:           map[string]*ClientConn{},
		PlayerSpawn:     rl.NewVector2(0, 0),
		skillCooldowns:  map[string]float32{},
		skillCharges:    map[string]int{},
		attackCooldowns: map[string]float32{},
	}

	h.ResetStage()

	if h.players["p1"].InPortal {
		t.Error("expected InPortal cleared by a stage reset")
	}
}
