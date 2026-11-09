//go:build !android

package ui

import (
	"github.com/WandenDourado/Legiao/internal/settings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// applyTargetFPS re-cadencia o laço na hora — ao contrário do vsync (hint de
// criação de contexto GL, só vale na próxima abertura), o teto de quadro é
// só rl.SetTargetFPS, que aceita ser chamado a qualquer momento.
func applyTargetFPS(fc settings.FrameCap) {
	rl.SetTargetFPS(int32(fc))
}

// toggleFullscreenIfDesktop alterna a janela entre tela cheia e janela. No
// desktop é exatamente isso: um alternador real, porque a janela pode estar
// nos dois estados.
//
// ToggleBorderlessWindowed, NÃO rl.ToggleFullscreen: a janela já nasce do
// tamanho do monitor (game.Run chama InitWindow(0, 0, ...)), então "tela
// cheia" aqui é só tirar a borda — ToggleFullscreen troca de modo de vídeo
// de verdade (recria o contexto, muda a resolução do monitor), e é o que
// piscava preto ao ligar esta opção (sessão de teste de 04/09/2026). Ver o
// mesmo comentário em game/loop.go, no `if loaded.Fullscreen` de startup —
// os dois caminhos têm de usar a mesma chamada, ou o estado salvo (tela
// cheia por ToggleBorderlessWindowed) reabriria por um modo diferente
// (ToggleFullscreen) da próxima vez.
func toggleFullscreenIfDesktop() {
	rl.ToggleBorderlessWindowed()
}

// mobileBuild: no PC a resolucao e sempre a do monitor.
const mobileBuild = false
