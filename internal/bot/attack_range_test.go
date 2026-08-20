package bot

// Fase A3 (doc/plan_avanco_bots_e_gargula.md §A5): a basic attack beyond the
// projectile's real reach spends the cadence on a bolt that expires before
// it can land. Arqueiro is the concrete bug report (ArrowAttackSpeed(700) *
// Lifetime(1.6) = 1120, but the old code fired at any distance).

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestArqueiroDoesNotAttackBeyondUsefulRange(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		Self:      Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		HasHumans: false,
		Foes: []Foe{
			{ID: "far", Pos: rl.NewVector2(arqueiroAttackRange+200, 0), Health: 10, MaxHealth: 10},
		},
	}
	intent := b.Think(v)
	if intent.Attack != nil {
		t.Fatalf("expected no attack beyond arqueiroAttackRange (%.0f), got aim %+v", arqueiroAttackRange, *intent.Attack)
	}
}

func TestArqueiroAttacksWithinUsefulRange(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		Self:      Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		HasHumans: false,
		Foes: []Foe{
			{ID: "near", Pos: rl.NewVector2(400, 0), Health: 10, MaxHealth: 10},
		},
	}
	intent := b.Think(v)
	if intent.Attack == nil {
		t.Fatal("expected an attack within arqueiroAttackRange")
	}
}
