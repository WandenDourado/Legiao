package entity

import "testing"

func TestSacerdotisaRenderDefinition(t *testing.T) {
	def := GetCharacter(CharSacerdotisa)
	if def.SpritePath != "assets/sprites/sacerdotisa/sacerdotisa.png" {
		t.Fatalf("unexpected sprite path: %q", def.SpritePath)
	}
	if def.RenderScale != 1.15 {
		t.Fatalf("expected Sacerdotisa render scale 1.15, got %v", def.RenderScale)
	}
	if def.FrameWidth != 128 || def.FrameHeight != 192 || def.Columns != 8 || def.Rows != 5 {
		t.Fatalf("unexpected sprite contract: %#v", def)
	}
}
