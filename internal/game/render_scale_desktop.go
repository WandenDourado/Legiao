//go:build !android

package game

import "github.com/WandenDourado/Legiao/internal/settings"

// renderScaleSupported: o PC desenha sempre na resolucao do monitor.
const renderScaleSupported = false

// startupWindowSize no desktop e sempre (0,0): a janela nasce do tamanho do
// monitor, como antes desta opcao existir.
func startupWindowSize(settings.Settings) (int32, int32) { return 0, 0 }
