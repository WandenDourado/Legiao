package network

// buildBotView's HumanCentre/HasHumans (plan doc/plan_avanco_bots_e_gargula.md
// §A3, R1): the "follow the group" reference must average only living
// humans, or an advancing bot pulls its own position into the average the
// rest of the party (bots included) chases — the marching-off-together bug
// the plan documents as cause 2.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestBuildBotViewHumanCentreIgnoresBotsAndDead(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			// The human centre must average ONLY this one: alive and human.
			"human_paladina": {PlayerID: "human_paladina", X: 100, Y: 0, Character: "paladina"},
			// A dead human does not get to anchor the party either.
			"human_mago": {PlayerID: "human_mago", X: 900, Y: 900, Character: "mago", IsDead: true},
			// Both bots sit far from the human; if they leaked into the
			// average the assertion below would fail.
			"bot_arqueiro":   {PlayerID: "bot_arqueiro", X: 500, Y: 500, Character: "arqueiro"},
			"bot_necromante": {PlayerID: "bot_necromante", X: 700, Y: 700, Character: "necromante"},
		},
		bots: map[string]*botRuntime{
			"bot_arqueiro":   {character: entity.CharArqueiro},
			"bot_necromante": {character: entity.CharNecromante},
		},
	}

	v, _, _, ok := h.buildBotView("bot_arqueiro", 1.0/60)
	if !ok {
		t.Fatal("expected buildBotView to succeed")
	}
	if !v.HasHumans {
		t.Fatal("expected HasHumans true with one living human in the party")
	}
	want := rl.NewVector2(100, 0)
	if v.HumanCentre != want {
		t.Fatalf("HumanCentre = %+v, want %+v (the one living human, bots and the dead human excluded)", v.HumanCentre, want)
	}
}

func TestBuildBotViewHasHumansFalseWithNoLivingHuman(t *testing.T) {
	h := &Host{
		EntityManager: entity.NewEntityManager(),
		players: map[string]*PlayerState{
			"human_mago":   {PlayerID: "human_mago", X: 10, Y: 10, Character: "mago", IsDead: true},
			"bot_arqueiro": {PlayerID: "bot_arqueiro", X: 500, Y: 500, Character: "arqueiro"},
		},
		bots: map[string]*botRuntime{
			"bot_arqueiro": {character: entity.CharArqueiro},
		},
	}

	v, _, _, ok := h.buildBotView("bot_arqueiro", 1.0/60)
	if !ok {
		t.Fatal("expected buildBotView to succeed")
	}
	if v.HasHumans {
		t.Fatal("expected HasHumans false: the only human in the party is dead")
	}
}
