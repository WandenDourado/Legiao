package bot

// Fase A2 (doc/plan_avanco_bots_e_gargula.md §A3, R2/R3): a distant foe must
// not exist for target selection, and a bot with no target should form up
// relative to the humans' front instead of standing on top of them.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestEngageableFoesDropsAFoeFarFromBotAndHumans(t *testing.T) {
	v := View{
		Self:        Ally{Pos: rl.NewVector2(0, 0)},
		HumanCentre: rl.NewVector2(50, 0),
		HasHumans:   true,
		Foes: []Foe{
			{ID: "near", Pos: rl.NewVector2(300, 0)},    // within engageRadius of self
			{ID: "far", Pos: rl.NewVector2(5000, 5000)}, // far from both self and humans
		},
	}
	got := engageableFoes(v)
	if len(got) != 1 || got[0].ID != "near" {
		t.Fatalf("expected only the near foe to survive the filter, got %+v", got)
	}
}

func TestEngageableFoesKeepsAFoeOnlyCloseToHumans(t *testing.T) {
	v := View{
		Self:        Ally{Pos: rl.NewVector2(-5000, 0)}, // far from the foe itself
		HumanCentre: rl.NewVector2(0, 0),
		HasHumans:   true,
		Foes: []Foe{
			{ID: "near-humans", Pos: rl.NewVector2(200, 0)},
		},
	}
	got := engageableFoes(v)
	if len(got) != 1 || got[0].ID != "near-humans" {
		t.Fatalf("expected a foe close to the humans to count as engageable even though it is far from the bot, got %+v", got)
	}
}

func TestFormationPostSitsRelativeToHumanCentreAlongAdvanceDir(t *testing.T) {
	v := View{
		Self:        Ally{Char: entity.CharSacerdotisa},
		HumanCentre: rl.NewVector2(1000, 1000),
		HasHumans:   true,
		AdvanceDir:  rl.NewVector2(1, 0), // advancing due east
	}
	post := formationPost(v)
	// Sacerdotisa sits 350px BEHIND the front: opposite the advance
	// direction, so west of the human centre, same Y (no lateral offset).
	want := rl.NewVector2(1000-350, 1000)
	if rl.Vector2Distance(post, want) > 0.01 {
		t.Fatalf("Sacerdotisa formation post = %+v, want %+v", post, want)
	}
}

func TestFollowDestHoldsPositionWithNoHumans(t *testing.T) {
	v := View{
		Self:      Ally{Pos: rl.NewVector2(42, 42), Char: entity.CharArqueiro},
		HasHumans: false,
	}
	dest := followDest(v)
	if dest != v.Self.Pos {
		t.Fatalf("expected followDest to hold position with no humans alive, got %+v", dest)
	}
}
