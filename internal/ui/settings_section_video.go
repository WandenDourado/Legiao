package ui

// Seção Vídeo e desempenho: teto de quadros, vsync, tela cheia (desktop) e
// o painel de desempenho (F3) — a opção que dá acesso a ele no celular.
// Qualidade grafica, resolucao do celular e as chaves de efeito moraram aqui
// ate 28/09/2026 e foram para a aba Graficos (settings_section_graphics.go).

import (
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/tilemap"
)

var frameCapLabels = []string{"30", "60", "SEM TETO"}

func frameCapIndex(fc settings.FrameCap) int {
	switch fc {
	case settings.FrameCap30:
		return 0
	case settings.FrameCap60:
		return 1
	default:
		return 2
	}
}

func frameCapAt(idx int) settings.FrameCap {
	switch idx {
	case 0:
		return settings.FrameCap30
	case 1:
		return settings.FrameCap60
	default:
		return settings.FrameCapUncapped
	}
}

func (scr *SettingsScreen) updateVideoSection(l settingsLayout) {
	s := settings.Get()

	rowLabel(l, 0, "TETO DE QUADROS")
	if idx := ChoiceRow(l.controlRect(0), frameCapLabels, frameCapIndex(s.TargetFPS)); idx >= 0 {
		fc := frameCapAt(idx)
		settings.Update(func(s *settings.Settings) { s.TargetFPS = fc })
		applyTargetFPS(fc)
	}

	rowLabel(l, 1, "VSYNC")
	rowHint(l, 1, "so vale na proxima vez que o jogo abrir")
	if v, changed := ToggleRow(l.toggleRect(1), s.VSync); changed {
		settings.Update(func(s *settings.Settings) { s.VSync = v })
	}

	scr.drawFullscreenRow(l, 2, s)

	rowLabel(l, 3, "PAINEL DE DESEMPENHO (F3)")
	rowHint(l, 3, "fps, memoria, contagem de desenho por quadro")
	if _, changed := ToggleRow(l.toggleRect(3), tilemap.DebugEnabled()); changed {
		tilemap.ToggleDebug()
	}
}

// drawFullscreenRow is desktop-only in EFFECT (rl.ToggleFullscreen on
// Android has nothing to toggle — the window already fills the screen by
// FullScreen config), but the row itself is drawn on every platform so the
// tab does not shift layout between them; see settings_controls for the
// pattern this file does NOT use (a build-tagged section) because this is a
// single row, not a whole section.
func (scr *SettingsScreen) drawFullscreenRow(l settingsLayout, i int, s settings.Settings) {
	rowLabel(l, i, "TELA CHEIA (DESKTOP)")
	if v, changed := ToggleRow(l.toggleRect(i), s.Fullscreen); changed {
		settings.Update(func(s *settings.Settings) { s.Fullscreen = v })
		toggleFullscreenIfDesktop()
	}
}

// applyTargetFPS and toggleFullscreenIfDesktop are declared in
// settings_apply_desktop.go / settings_apply_android.go: the frame cap
// applies at runtime via rl.SetTargetFPS everywhere, but fullscreen means
// nothing on Android (FullScreen config already fills the screen), so the
// android build gets a no-op instead of calling rl.ToggleFullscreen on a
// window that was never in windowed mode to begin with.