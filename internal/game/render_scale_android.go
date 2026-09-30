//go:build android

package game

import "github.com/WandenDourado/Legiao/internal/settings"

// renderScaleSupported: no celular a resolucao de desenho e escolhivel.
const renderScaleSupported = true

// startupWindowSize e o tamanho com que a janela nasce: (0,0) = nativa.
func startupWindowSize(s settings.Settings) (int32, int32) {
	return settings.WindowSize(s)
}
