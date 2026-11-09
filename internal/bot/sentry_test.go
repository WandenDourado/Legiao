package bot

// Fase B2/B4 (doc/plan_avanco_bots_e_gargula.md §B4): a sentinela entra em
// Foes com IsSentry, mas nenhuma seleção de alvo comum pode vê-la — só a
// suprema do Arqueiro pode, e só quando ela está pronta.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestNearestFoeNeverReturnsASentry(t *testing.T) {
	foes := []Foe{
		{ID: "sentry", Pos: rl.NewVector2(10, 0), IsSentry: true}, // closer
		{ID: "orc", Pos: rl.NewVector2(500, 0)},
	}
	got, ok := nearestFoe(rl.NewVector2(0, 0), foes)
	if !ok || got.ID != "orc" {
		t.Fatalf("expected nearestFoe to skip the sentry and return the orc, got %+v ok=%v", got, ok)
	}
}

func TestNearestSentryOnlyReturnsSentries(t *testing.T) {
	foes := []Foe{
		{ID: "orc", Pos: rl.NewVector2(10, 0)}, // closer, but not a sentry
		{ID: "sentry", Pos: rl.NewVector2(500, 0), IsSentry: true, HitCentre: rl.NewVector2(500, 0)},
	}
	got, ok := nearestSentry(rl.NewVector2(0, 0), foes)
	if !ok || got.ID != "sentry" {
		t.Fatalf("expected nearestSentry to skip the orc and return the sentry, got %+v ok=%v", got, ok)
	}
}

func TestArqueiroIgnoresSentryWithoutUltimateReady(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		Self:          Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		UltimateReady: false,
		Foes: []Foe{
			{ID: "sentry", Pos: rl.NewVector2(300, 0), IsSentry: true, HitCentre: rl.NewVector2(300, 0)},
		},
	}
	intent := b.Think(v)
	if intent.Skill != nil && intent.Skill.SkillID == "celestial_arrows" {
		t.Fatal("expected no sentry-hunting cast while the ultimate is not ready")
	}
	if intent.Attack != nil {
		t.Fatal("expected no basic attack against a sentry ever")
	}
}

func TestArqueiroFiresCelestialArrowsAtASentryInRange(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		Self:          Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		UltimateReady: true,
		UltimateRange: 4800,
		Foes: []Foe{
			{ID: "sentry", Pos: rl.NewVector2(1000, 0), IsSentry: true, HitCentre: rl.NewVector2(1000, 0)},
		},
	}
	intent := b.Think(v)
	if intent.Skill == nil || intent.Skill.SkillID != "celestial_arrows" {
		t.Fatalf("expected a celestial_arrows cast at a sentry well within range, got %+v", intent.Skill)
	}
}

func TestArqueiroApproachesADistantSentryBeforeFiring(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		Self:          Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		UltimateReady: true,
		UltimateRange: 4800,
		Foes: []Foe{
			// Far beyond UltimateRange - celestialApproachMargin: must walk
			// first, not fire a shot that will expire on the way.
			{ID: "sentry", Pos: rl.NewVector2(9000, 0), IsSentry: true, HitCentre: rl.NewVector2(9000, 0)},
		},
	}
	intent := b.Think(v)
	if intent.Skill != nil {
		t.Fatalf("expected no cast while still out of the approach range, got %+v", intent.Skill)
	}
	if !intent.HasDest || intent.Dest.X <= 0 {
		t.Fatalf("expected the bot to move toward the distant sentry, got %+v", intent)
	}
}
