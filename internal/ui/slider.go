package ui

// Slider e o controle de volume: uma trilha horizontal que responde a cinco
// coisas ao mesmo tempo (ver doc/settings.md, "O controle de volume") —
// tocar na trilha salta para o valor, a trilha tem alvo de toque maior que o
// desenho (reusa hitPad(), o mesmo do resto do menu), o valor anda em passos
// de 5%, e soltar (nao arrastar) e o momento certo para uma previa sonora.
// Mudo por barramento fica FORA daqui: e um botao a parte, ao lado da
// trilha, desenhado por quem monta a secao (settings_audio.go) — o slider
// so sabe de um numero 0..1.
//
// A matemática (posição ↔ valor, clamp, degrau) está em SliderValueAt e
// SliderStepUp/Down, funções puras sem chamada de raylib que precise de
// janela — é o que slider_test.go exercita sem abrir nada.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// SliderStep e o degrau de 5% pedido no Passo 4: sem ele "50%" nunca se
// repete, sempre fica 48% ou 53%.
const SliderStep float32 = 0.05

// SliderValueAt mapeia uma posição X (mesmo espaço de coordenadas de track)
// para um valor 0..1, já clampado e arredondado para o degrau de 5% mais
// próximo. Usada tanto por um toque na trilha (salta direto) quanto por um
// arraste (persegue o dedo/mouse quadro a quadro).
func SliderValueAt(track rl.Rectangle, x float32) float32 {
	if track.Width <= 0 {
		return 0
	}
	t := (x - track.X) / track.Width
	return snapStep(clamp01f(t))
}

// SliderStepUp and SliderStepDown move value one 5% step, clamped at both
// ends — the roda do mouse sobre a trilha usa isto (ver Slider.Update).
func SliderStepUp(value float32) float32   { return snapStep(clamp01f(value + SliderStep)) }
func SliderStepDown(value float32) float32 { return snapStep(clamp01f(value - SliderStep)) }

func snapStep(v float32) float32 {
	steps := float32(math.Round(float64(v / SliderStep)))
	return clamp01f(steps * SliderStep)
}

func clamp01f(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// sliderNoTouch mirrors input.NoTouch, kept local so this file has no
// dependency on the input package for one constant.
const sliderNoTouch int32 = -1

// sliderMouseID marks the drag as owned by the mouse rather than a real
// touch, exactly like TouchState's own -2 convention (internal/input/touch.go).
const sliderMouseID int32 = -2

// Slider holds the one bit of state a drag needs across frames: which
// touch (or the mouse) currently owns it. Everything else is recomputed
// every frame from the rectangle and the current value.
type Slider struct {
	touchID int32
}

// NewSlider creates a slider with no active drag.
func NewSlider() Slider { return Slider{touchID: sliderNoTouch} }

// SliderResult is what Update reports for this frame.
type SliderResult struct {
	Value    float32 // the (possibly unchanged) value after this frame
	Changed  bool    // Value moved this frame — apply it live (volume, etc.)
	Released bool    // a drag/tap/wheel-step just ENDED — the cue for the sound preview and for persisting to disk
}
