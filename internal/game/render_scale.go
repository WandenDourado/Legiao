package game

// Resolucao de desenho no celular (Configuracoes > Graficos > Resolucao).
//
// O jogo pede ao raylib uma janela MENOR que a tela; no Android isso cai no
// ramo "Upscaling" de SetupFramebuffer, que chama
// ANativeWindow_setBuffersGeometry com o tamanho pedido. Quem amplia para a
// tela e o scaler de hardware do aparelho, sem custo de GPU — a recomendacao
// oficial do Android para jogo limitado por preenchimento de pixel. O toque
// e reescalado pelo proprio raylib (AndroidInputCallback). Estudo em
// plans/plan_qualidade_grafica_mobile.md.
//
// No desktop nada disto acontece: startupWindowSize devolve (0,0) e o jogo
// abre na resolucao do monitor, como sempre.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/settings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// renderZoomScale e quanto o zoom da camera tem de encolher quando a janela
// nasceu escalada. Sem isto, desenhar em 720p numa tela de 1080 mostraria
// 33% MENOS mapa — trocar resolucao nao pode mudar o que o jogador enxerga,
// so a nitidez. Um (1) quando a janela e nativa.
func renderZoomScale(w, h int32, s settings.Settings) float32 {
	if w <= 0 || h <= 0 || s.DisplayHeight <= 0 {
		return 1
	}
	return float32(rl.GetScreenHeight()) / float32(s.DisplayHeight)
}

// noteDisplaySize anota o tamanho nativo da tela quando a janela nasceu
// nativa, para a proxima abertura saber a proporcao. Grava na hora: o
// jogador pode escolher 720p e fechar o app sem nunca passar pela tela de
// Configuracoes de novo.
func noteDisplaySize(w, h int32, s settings.Settings) {
	if !renderScaleSupported || w > 0 || h > 0 {
		return
	}
	dw, dh := int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	if dw <= 0 || dh <= 0 || (dw == s.DisplayWidth && dh == s.DisplayHeight) {
		return
	}
	settings.Update(func(s *settings.Settings) { s.DisplayWidth, s.DisplayHeight = dw, dh })
	if err := settings.Save(); err != nil {
		log.Printf("[Graficos] falha ao anotar o tamanho da tela: %v", err)
	}
}
