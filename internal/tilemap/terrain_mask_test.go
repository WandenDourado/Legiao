package tilemap

import "testing"

// smoothstep(edge0, edge1, x) e INDEFINIDO em GLSL quando edge0 >= edge1: a
// implementacao calcula (x-edge0)/(edge1-edge0), e no mobile (GLSL ES 2.0,
// mediump) essa divisao por zero vira NaN em vez de +-infinito, e
// clamp(NaN, 0, 1) apaga a celula inteira. Os quatro shaders de terreno
// (terrain_blend_100/330.fs, terrain_batch_100/330.fs) derivam os limites da
// rampa de edgeWidth com estas mesmas formulas, exatamente para que uma
// pilha de borda dura (edgeWidth pequeno, como a do castelo, 0.01) nunca
// produza edge0 >= edge1. Este teste replica as formulas do lado do Go e
// prova que elas nunca degeneram para nenhuma pilha declarada em
// terrainStacks — e continua provando isso automaticamente se alguem
// adicionar uma pilha nova ou mudar um edgeWidth existente.
func rampLo(w float32) float32 {
	if 0.01 < w*0.5 {
		return 0.01
	}
	return w * 0.5
}

func rampHi(w float32) float32 {
	lo := rampLo(w)
	if w > lo+0.002 {
		return w
	}
	return lo + 0.002
}

func cornerLo(w float32) float32 {
	if 0.02 < w*0.5 {
		return 0.02
	}
	return w * 0.5
}

func cornerHi(w float32) float32 {
	lo := cornerLo(w)
	if w > lo+0.002 {
		return w
	}
	return lo + 0.002
}

func TestStackEdgeWidthNeverDegeneratesTheShaderRamp(t *testing.T) {
	for _, stack := range terrainStacks {
		w := stack.edgeWidth
		if w <= 0 {
			t.Fatalf("stack %v: edgeWidth must be positive, got %v", stack.materials, w)
		}
		if lo, hi := rampLo(w), rampHi(w); !(lo < hi) {
			t.Fatalf("stack %v: side() ramp degenerates for edgeWidth=%v: rampLo=%v rampHi=%v", stack.materials, w, lo, hi)
		}
		if lo, hi := cornerLo(w), cornerHi(w); !(lo < hi) {
			t.Fatalf("stack %v: corner smoothstep degenerates for edgeWidth=%v: cornerLo=%v cornerHi=%v", stack.materials, w, lo, hi)
		}
	}
}

// TestEdgeWidthForNeverDegeneratesTheShaderRamp cobre o mesmo contrato pelo
// caminho que o shader realmente usa a cada quadro: edgeWidthFor(material),
// para todo material que alguma pilha declara.
func TestEdgeWidthForNeverDegeneratesTheShaderRamp(t *testing.T) {
	for _, stack := range terrainStacks {
		for _, material := range stack.materials {
			w := edgeWidthFor(material)
			if lo, hi := rampLo(w), rampHi(w); !(lo < hi) {
				t.Fatalf("material %d: side() ramp degenerates for edgeWidth=%v: rampLo=%v rampHi=%v", material, w, lo, hi)
			}
			if lo, hi := cornerLo(w), cornerHi(w); !(lo < hi) {
				t.Fatalf("material %d: corner smoothstep degenerates for edgeWidth=%v: cornerLo=%v cornerHi=%v", material, w, lo, hi)
			}
		}
	}
}
