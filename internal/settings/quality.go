package settings

// Qualidade grafica: os tres presets e a conta da resolucao do celular.
// Estudo e numeros em plans/plan_qualidade_grafica_mobile.md (28/09/2026).
//
// Um preset nao e um campo gravado: e um NOME para uma combinacao de campos
// que continuam editaveis um a um. A tela mostra o preset cuja combinacao bate
// com o que esta ligado agora, ou "personalizado" quando nenhum bate.

import "math"

// Preset e um nivel de qualidade grafica.
type Preset int

const (
	PresetLow Preset = iota
	PresetMedium
	PresetHigh
	// PresetCustom e "nenhum dos tres": algum campo foi mexido a mao.
	PresetCustom
)

// RenderHeights sao as resolucoes oferecidas no celular, da maior para a
// menor. Zero e a nativa. O "4K" pedido para o PC nao entra: o jogo ja
// desenha na resolucao do monitor, e acima dela so se gasta.
var RenderHeights = []int32{0, 900, 720, 540}

func clampRenderHeight(h int32) int32 {
	for _, v := range RenderHeights {
		if v == h {
			return h
		}
	}
	return 0
}

// presetFields e o que cada preset liga. A ordem dos campos e a do preset:
// resolucao (so celular), teto de quadro, desintegracao, detalhe do chao.
type presetFields struct {
	RenderHeight int32
	TargetFPS    FrameCap
	DeathEffect  bool
	GroundDetail bool
}

var presets = map[Preset]presetFields{
	PresetHigh:   {0, FrameCap60, true, true},
	PresetMedium: {900, FrameCap60, true, false},
	// Baixo e o celular antigo: 720p pelo scaler (56% menos pixel numa tela
	// 2400x1080) e 30 fps — metade do trabalho e bem menos calor, que e o
	// que faz o jogo "comecar bem e piorar".
	PresetLow: {720, FrameCap30, false, false},
}

// ApplyPreset liga os campos do preset em s. Custom nao muda nada.
func ApplyPreset(s *Settings, p Preset) {
	f, ok := presets[p]
	if !ok {
		return
	}
	s.RenderHeight = f.RenderHeight
	s.TargetFPS = f.TargetFPS
	s.DeathEffect = f.DeathEffect
	s.GroundDetail = f.GroundDetail
}

// PresetOf diz qual preset s e. A RESOLUCAO nao entra na comparacao: ela e
// um eixo proprio (linha propria na tela, e o teste automatico escolhe a
// resolucao pela medida, nao pela tabela). Escolher um preset sugere uma
// resolucao; trocar so a resolucao depois nao desfaz o nivel escolhido.
func PresetOf(s Settings) Preset {
	for _, p := range []Preset{PresetHigh, PresetMedium, PresetLow} {
		f := presets[p]
		if f.TargetFPS == s.TargetFPS && f.DeathEffect == s.DeathEffect && f.GroundDetail == s.GroundDetail {
			return p
		}
	}
	return PresetCustom
}

// WindowSize e o tamanho com que a janela do celular deve nascer: (0,0) =
// nativa. Escala so quando a resolucao pedida e MENOR que a tela anotada, e
// a largura sai da proporcao da tela — com outra proporcao o raylib poria
// tarjas pretas dos lados.
func WindowSize(s Settings) (w, h int32) {
	if s.RenderHeight <= 0 || s.DisplayWidth <= 0 || s.DisplayHeight <= 0 || s.RenderHeight >= s.DisplayHeight {
		return 0, 0
	}
	h = s.RenderHeight
	w = int32(math.Round(float64(h) * float64(s.DisplayWidth) / float64(s.DisplayHeight)))
	return w, h
}
