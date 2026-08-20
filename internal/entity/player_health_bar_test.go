package entity

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func luminance(c rl.Color) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

func TestAllyBarColorBands(t *testing.T) {
	cases := []struct {
		percent float32
		want    rl.Color
	}{
		{1.0, allyBarColor(1.0)},
		{0.51, rl.NewColor(150, 226, 170, 255)},
		{0.50, rl.NewColor(240, 208, 138, 255)},
		{0.26, rl.NewColor(240, 208, 138, 255)},
		{0.25, rl.NewColor(235, 145, 148, 255)},
		{0.0, rl.NewColor(235, 145, 148, 255)},
	}
	for _, c := range cases {
		got := allyBarColor(c.percent)
		if got != c.want {
			t.Errorf("allyBarColor(%v) = %v, want %v", c.percent, got, c.want)
		}
	}
}

// TestAllyPaletteStaysDistinctFromMonster keeps the two bars from ever being
// visually reconciled by accident: the ally is always the lighter, softer
// color in the same threshold band, so a player can tell them apart without
// reading the number.
func TestAllyPaletteStaysDistinctFromMonster(t *testing.T) {
	bands := []struct {
		name    string
		percent float32
		monster rl.Color
	}{
		{"healthy", 1.0, rl.Green},
		{"hurt", 0.4, rl.Orange},
		{"critical", 0.1, rl.Red},
	}
	for _, b := range bands {
		ally := allyBarColor(b.percent)
		if ally == b.monster {
			t.Errorf("%s: ally color equals monster color %v", b.name, b.monster)
		}
		if luminance(ally) <= luminance(b.monster) {
			t.Errorf("%s: ally luminance %.1f not brighter than monster luminance %.1f",
				b.name, luminance(ally), luminance(b.monster))
		}
	}
}

func TestAllyBarLayoutClearsCharacterFrame(t *testing.T) {
	for _, def := range AllCharacters() {
		above, halfWidth := allyBarLayout(def)
		scale := def.RenderScale
		if scale <= 0 {
			scale = 1
		}
		frameTop := float32(def.FrameHeight) / 2 * scale
		if above <= frameTop {
			t.Errorf("%s: above %v does not clear frame top %v", def.Type, above, frameTop)
		}
		if halfWidth <= 0 {
			t.Errorf("%s: halfWidth must be positive, got %v", def.Type, halfWidth)
		}
	}
}

func TestAllyBarFraction(t *testing.T) {
	if _, ok := allyBarFraction(50, 0); ok {
		t.Fatal("MaxHealth <= 0 should report ok=false")
	}
	if _, ok := allyBarFraction(50, -10); ok {
		t.Fatal("negative MaxHealth should report ok=false")
	}
	if frac, ok := allyBarFraction(150, 100); !ok || frac != 1 {
		t.Fatalf("health above max should saturate at 1, got %v, ok=%v", frac, ok)
	}
	if frac, ok := allyBarFraction(25, 100); !ok || frac != 0.25 {
		t.Fatalf("expected 0.25, got %v, ok=%v", frac, ok)
	}
}
