package network

// updateAdvanceDir (host_bot_tick.go, plan doc/plan_avanco_bots_e_gargula.md
// §A3, R3): the party's smoothed heading ignores bots and holds its last
// value once the humans stop, instead of collapsing to zero.

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestUpdateAdvanceDirIgnoresBotsAndTracksHumans(t *testing.T) {
	h := &Host{
		players: map[string]*PlayerState{
			"human_mago":   {PlayerID: "human_mago", VelX: 300, VelY: 0},
			"bot_arqueiro": {PlayerID: "bot_arqueiro", VelX: 0, VelY: 500}, // would skew the average north if counted
		},
	}
	// A single 1s tick at the default smoothing rate is enough to converge
	// from zero to (within a tolerance) the target direction.
	h.updateAdvanceDir(1.0)

	want := rl.NewVector2(1, 0) // due east, matching the human alone
	if rl.Vector2Distance(h.advanceDir, want) > 0.05 {
		t.Fatalf("advanceDir = %+v, want close to %+v (bots must not skew it)", h.advanceDir, want)
	}
}

func TestUpdateAdvanceDirHoldsLastHeadingWhenPartyStops(t *testing.T) {
	h := &Host{
		players: map[string]*PlayerState{
			"human_mago": {PlayerID: "human_mago", VelX: 300, VelY: 0},
		},
	}
	h.updateAdvanceDir(1.0)
	established := h.advanceDir
	if established.X == 0 && established.Y == 0 {
		t.Fatal("expected a heading to be established from a moving human")
	}

	h.players["human_mago"].VelX = 0
	h.players["human_mago"].VelY = 0
	h.updateAdvanceDir(1.0)

	if h.advanceDir != established {
		t.Fatalf("expected advanceDir to hold its last value once the party stops, got %+v want %+v", h.advanceDir, established)
	}
}

func TestUpdateAdvanceDirHoldsLastHeadingWithNoHumansAlive(t *testing.T) {
	h := &Host{
		players: map[string]*PlayerState{
			"human_mago": {PlayerID: "human_mago", VelX: 300, VelY: 0},
		},
	}
	h.updateAdvanceDir(1.0)
	established := h.advanceDir

	h.players["human_mago"].IsDead = true
	h.updateAdvanceDir(1.0 / 60)

	if h.advanceDir != established {
		t.Fatalf("expected advanceDir to hold once no human is alive, got %+v want %+v", h.advanceDir, established)
	}
}
