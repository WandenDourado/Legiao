package bot

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestSacerdotisaAimsThroughTheMostWoundedAlly(t *testing.T) {
	self := rl.NewVector2(0, 0)
	wounded := rl.NewVector2(500, 0)
	v := View{
		Self:        Ally{ID: "s", Pos: self, Health: 100, MaxHealth: 100},
		PartyCentre: self,
		Allies: []Ally{
			{ID: "a1", Pos: wounded, Health: 40, MaxHealth: 100},
		},
		// Present, but nowhere near the self->wounded segment: keeps calm
		// mode off without blocking the heal line.
		Foes: []Foe{{ID: "f1", Pos: rl.NewVector2(800, 50), Health: 10, MaxHealth: 10}},
		Dt:   1.0 / 60,
	}

	b := &sacerdotisaBrain{}
	intent := b.Think(v)

	if intent.Attack == nil {
		t.Fatal("expected an attack aimed through the wounded ally")
	}
	got := direction(self, *intent.Attack)
	want := direction(self, wounded)
	if rl.Vector2Distance(got, want) > 0.01 {
		t.Fatalf("aim direction = %+v, want %+v (through the wounded ally)", got, want)
	}
}

func TestSacerdotisaRefusesTheLineWhenAFoeBlocksIt(t *testing.T) {
	self := rl.NewVector2(0, 0)
	wounded := rl.NewVector2(500, 0)
	blocker := rl.NewVector2(250, 0)
	v := View{
		Self:        Ally{ID: "s", Pos: self, Health: 100, MaxHealth: 100},
		PartyCentre: self,
		Allies: []Ally{
			{ID: "a1", Pos: wounded, Health: 40, MaxHealth: 100},
		},
		Foes: []Foe{{ID: "blocker", Pos: blocker, Radius: 20, Health: 10, MaxHealth: 10}},
		Dt:   1.0 / 60,
	}

	b := &sacerdotisaBrain{}
	intent := b.Think(v)

	if intent.Attack == nil {
		t.Fatal("expected an attack on the blocking foe")
	}
	if rl.Vector2Distance(*intent.Attack, blocker) > 1 {
		t.Fatalf("aim = %+v, want the blocking foe at %+v, not the wounded ally", *intent.Attack, blocker)
	}
}

func TestSacerdotisaTargetsThreatWhenEveryoneIsFull(t *testing.T) {
	self := rl.NewVector2(0, 0)
	foe := rl.NewVector2(300, 0)
	v := View{
		Self:        Ally{ID: "s", Pos: self, Health: 100, MaxHealth: 100},
		PartyCentre: self,
		Allies: []Ally{
			{ID: "a1", Pos: rl.NewVector2(100, 100), Health: 100, MaxHealth: 100},
		},
		Foes: []Foe{{ID: "f1", Pos: foe, Health: 10, MaxHealth: 10}},
		Dt:   1.0 / 60,
	}

	b := &sacerdotisaBrain{}
	intent := b.Think(v)

	if intent.Attack == nil {
		t.Fatal("expected an attack on the nearest threat")
	}
	if rl.Vector2Distance(*intent.Attack, foe) > 1 {
		t.Fatalf("aim = %+v, want the foe at %+v", *intent.Attack, foe)
	}
}

func TestSacerdotisaRecoversAndKeepsFiringWithNoFoesNearby(t *testing.T) {
	self := rl.NewVector2(0, 0)
	wounded := rl.NewVector2(200, 0)
	v := View{
		Self:        Ally{ID: "s", Pos: self, Health: 100, MaxHealth: 100},
		PartyCentre: self,
		Allies: []Ally{
			{ID: "a1", Pos: wounded, Health: 50, MaxHealth: 100},
		},
		Dt: 1.0 / 60,
	}

	b := &sacerdotisaBrain{}
	intent := b.Think(v)

	if intent.Attack == nil {
		t.Fatal("expected recovery mode to keep firing at the wounded ally")
	}
}

func TestSacerdotisaDoesNotFireWithNoFoesAndEveryoneFull(t *testing.T) {
	self := rl.NewVector2(0, 0)
	v := View{
		Self:        Ally{ID: "s", Pos: self, Health: 100, MaxHealth: 100},
		PartyCentre: self,
		Allies: []Ally{
			{ID: "a1", Pos: rl.NewVector2(50, 50), Health: 100, MaxHealth: 100},
		},
		Dt: 1.0 / 60,
	}

	b := &sacerdotisaBrain{}
	intent := b.Think(v)

	if intent.Attack != nil {
		t.Fatalf("expected no attack with nothing to fight and nobody hurt, got aim %+v", *intent.Attack)
	}
}
