package bot

// The portal shortcut in tickOneBot used to skip the Brain entirely whenever
// the portal was active, which meant a bot never fought its way across a
// garrison map (world_03 has no waves, so the portal is unlocked from the
// first frame while 83 monsters are still in the field). travelDest puts the
// portal back as a destination the Brain itself may choose — only when there
// is nothing to fight AND a human is already at the door
// (doc/plan_avanco_bots_e_gargula.md §A2, cause 4).

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestTravelDestRefusedWithAnEngageableFoeNearby(t *testing.T) {
	v := View{
		Self:           Ally{Pos: rl.NewVector2(0, 0)},
		PortalActive:   true,
		Portal:         rl.NewVector2(5000, 5000),
		HumansAtPortal: true,
		Foes: []Foe{
			{ID: "orc", Pos: rl.NewVector2(300, 0)}, // well within engageRadius
		},
	}
	if _, ok := travelDest(v); ok {
		t.Fatal("expected no travel destination with an engageable foe nearby")
	}
}

func TestTravelDestRefusedWithoutAHumanAtThePortal(t *testing.T) {
	v := View{
		Self:           Ally{Pos: rl.NewVector2(0, 0)},
		PortalActive:   true,
		Portal:         rl.NewVector2(5000, 5000),
		HumansAtPortal: false,
	}
	if _, ok := travelDest(v); ok {
		t.Fatal("expected no travel destination when no human is at the portal — a bot must not cross the map alone")
	}
}

func TestTravelDestGrantedWithAHumanAtThePortalAndNoFoes(t *testing.T) {
	v := View{
		Self:           Ally{Pos: rl.NewVector2(0, 0)},
		PortalActive:   true,
		Portal:         rl.NewVector2(5000, 5000),
		HumansAtPortal: true,
	}
	dest, ok := travelDest(v)
	if !ok || dest != v.Portal {
		t.Fatalf("expected the portal as destination, got %+v ok=%v", dest, ok)
	}
}

// TestArqueiroDoesNotTravelWithAnEngageableFoeNearby proves the whole chain
// through a real brain, not just travelDest in isolation: an engageable orc
// near the spawn must make the bot fight it instead of marching to the door,
// even with a human already standing in the portal.
func TestArqueiroDoesNotTravelWithAnEngageableFoeNearby(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		Self:           Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		HasHumans:      true,
		HumanCentre:    rl.NewVector2(0, 0),
		PortalActive:   true,
		Portal:         rl.NewVector2(7400, 7400),
		HumansAtPortal: true,
		Foes: []Foe{
			{ID: "orc", Pos: rl.NewVector2(300, 0), Health: 100, MaxHealth: 100},
		},
	}
	intent := b.Think(v)
	if !intent.HasDest || intent.Dest == v.Portal {
		t.Fatalf("expected the bot to engage the nearby orc instead of heading to the portal, got %+v", intent)
	}
}

// TestArqueiroFormsUpInsteadOfTravelingWithNoHumanAtThePortal is the
// concrete regression case from the bug report: a garrison map's portal is
// open from the first frame (no waves at all), but nobody has left the spawn
// yet. The bot must fall back to its formation post, not the door.
func TestArqueiroFormsUpInsteadOfTravelingWithNoHumanAtThePortal(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		// Self starts well clear of its own formation post (the humans are
		// far south, like a group still at world_03's spawn), so followDest
		// actually returns the post instead of "already there, hold".
		Self:           Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		HasHumans:      true,
		HumanCentre:    rl.NewVector2(0, 5000),
		AdvanceDir:     rl.NewVector2(0, -1), // advancing north, toward the portal
		PortalActive:   true,
		Portal:         rl.NewVector2(0, -7400), // far north, like world_03's fortress gate
		HumansAtPortal: false,
	}
	intent := b.Think(v)
	if !intent.HasDest {
		t.Fatal("expected a destination (the formation post)")
	}
	if intent.Dest == v.Portal {
		t.Fatal("expected the bot to hold its formation post, not march to the portal alone")
	}
	want := formationPost(v)
	if rl.Vector2Distance(intent.Dest, want) > 0.01 {
		t.Fatalf("Dest = %+v, want the formation post %+v", intent.Dest, want)
	}
}

// TestArqueiroTravelsWithoutPushWhenAHumanIsAtThePortal is the third
// required case: door open, nothing to fight, a human already there — the
// bot must head straight for the portal with Push left at zero, since
// separation is exactly what would shove it back out of the small rectangle
// (doc/tilemap.md "Quem entra no portal some e espera").
func TestArqueiroTravelsWithoutPushWhenAHumanIsAtThePortal(t *testing.T) {
	b := &arqueiroBrain{}
	v := View{
		Self:           Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		HasHumans:      true,
		HumanCentre:    rl.NewVector2(0, -7000),
		PortalActive:   true,
		Portal:         rl.NewVector2(0, -7400),
		HumansAtPortal: true,
		Allies: []Ally{
			{ID: "human_1", Pos: rl.NewVector2(0, -7000), Health: 100, MaxHealth: 100},
		},
	}
	intent := b.Think(v)
	if !intent.HasDest || intent.Dest != v.Portal {
		t.Fatalf("expected the bot to head straight for the portal, got %+v", intent)
	}
	if intent.Push.X != 0 || intent.Push.Y != 0 {
		t.Fatalf("expected zero Push while traveling to the portal, got %+v", intent.Push)
	}
}

// TestArqueiroAttackIsNotSuppressedInAPortalTravelEligibleScene is the
// "andar e atirar" requirement (plan §A2, point 3): a stale committed
// target — picked a moment ago while it was still engageable, now past
// engageRadius (900) but still inside the Arqueiro's real attack range
// (arqueiroAttackRange, 1120) — must not stop firing just because every
// OTHER condition for heading to the portal is met.
//
// The bot correctly keeps chasing the target instead of the door here
// (Dest != Portal): "alvo engajado" outranks travelDest in the chain, and
// Arqueiro's own kiting logic (untouched by this change) reassigns Dest the
// moment hasTarget is true, for any distance. What this test actually
// guards is the bug the diagnosis called out by name — a `return` in the
// travel branch landing before the attack decision — which would zero
// Intent.Attack instead.
func TestArqueiroAttackIsNotSuppressedInAPortalTravelEligibleScene(t *testing.T) {
	b := &arqueiroBrain{
		// A committed target from a "previous tick": decideIn > 0 skips
		// re-selection this Think() call, so hasTarget resolves true via
		// findFoe (which scans ALL of v.Foes, engage radius or not) even
		// though the foe below no longer shows up in engageableFoes.
		targetID: "orc",
		decideIn: decideEvery,
	}
	foePos := rl.NewVector2(1000, 0) // > engageRadius (900), <= arqueiroAttackRange (1120)
	v := View{
		Self:           Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		HasHumans:      true,
		HumanCentre:    rl.NewVector2(-2000, -2000), // far from the foe too, so it truly is not engageable
		PortalActive:   true,
		Portal:         rl.NewVector2(7400, 7400),
		HumansAtPortal: true,
		Foes: []Foe{
			{ID: "orc", Pos: foePos, Health: 100, MaxHealth: 100},
		},
	}
	if len(engageableFoes(v)) != 0 {
		t.Fatal("test setup is wrong: the foe must NOT be engageable, or this is not the scenario being tested")
	}
	intent := b.Think(v)
	if intent.Attack == nil {
		t.Fatal("expected the basic attack to keep firing at the in-range stale target even though the portal is otherwise available")
	}
}
