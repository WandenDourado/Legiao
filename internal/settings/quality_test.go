package settings

import "testing"

func TestPresetsRoundTrip(t *testing.T) {
	for _, p := range []Preset{PresetLow, PresetMedium, PresetHigh} {
		s := Default()
		ApplyPreset(&s, p)
		if got := PresetOf(s.clamped()); got != p {
			t.Errorf("preset %d aplicado e lido como %d", p, got)
		}
		s.RenderHeight = 540 // resolucao e eixo proprio
		if got := PresetOf(s); got != p {
			t.Errorf("trocar so a resolucao desfez o preset %d (virou %d)", p, got)
		}
	}
	if PresetOf(Default()) != PresetHigh {
		t.Errorf("o padrao de fabrica deveria ser o preset Alto")
	}
	s := Default()
	s.DeathEffect = false // 60 fps sem desintegracao: nenhum nivel e assim
	if PresetOf(s) != PresetCustom {
		t.Errorf("campo mexido a mao deveria virar personalizado")
	}
}

func TestWindowSizeKeepsAspectAndNeverUpscales(t *testing.T) {
	s := Default()
	s.DisplayWidth, s.DisplayHeight = 2400, 1080
	s.RenderHeight = 720
	if w, h := WindowSize(s); w != 1600 || h != 720 {
		t.Errorf("2400x1080 em 720p deveria dar 1600x720, deu %dx%d", w, h)
	}
	s.RenderHeight = 900
	s.DisplayWidth, s.DisplayHeight = 1600, 720 // tela menor que o pedido
	if w, h := WindowSize(s); w != 0 || h != 0 {
		t.Errorf("tela de 720 nao pode abrir em 900: %dx%d", w, h)
	}
	s.DisplayWidth, s.DisplayHeight = 0, 0 // primeira abertura
	if w, h := WindowSize(s); w != 0 || h != 0 {
		t.Errorf("sem tamanho de tela anotado tem de abrir nativo: %dx%d", w, h)
	}
}

func TestClampRejectsUnknownRenderHeight(t *testing.T) {
	s := Default()
	s.RenderHeight = 1234
	if s.clamped().RenderHeight != 0 {
		t.Errorf("resolucao fora da lista deveria voltar a nativa")
	}
}
