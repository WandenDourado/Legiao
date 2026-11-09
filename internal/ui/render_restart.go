package ui

// Resolucao de desenho: a que esta VALENDO contra a que foi escolhida, e o
// botao que reinicia o jogo para aplicar (29/09/2026).
//
// No celular a resolucao so vale quando a janela nasce (game/render_scale.go).
// Ate esta data a tela so dizia "vale na proxima vez que o jogo abrir", e o
// relato do Gui (Galaxy S25 Ultra) foi o previsivel: trocou, nada mudou, e
// nao havia como saber se a troca pegou. Sair pelo Home e voltar nao abre o
// jogo de novo — o processo e a janela sao os mesmos. Agora a linha mostra o
// tamanho em que o jogo esta desenhando AGORA, e quando ele difere do
// escolhido aparece REINICIAR AGORA (internal/apprestart).

import (
	"fmt"
	"log"

	"github.com/WandenDourado/Legiao/internal/apprestart"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/resume"
	"github.com/WandenDourado/Legiao/internal/settings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// startupWindowH e a altura com que a janela NASCEU nesta abertura, no mesmo
// idioma de settings.WindowSize: 0 = nativa. game.Run anota logo depois de
// InitWindow.
var startupWindowH int32

// SetStartupWindowSize anota o tamanho pedido ao InitWindow nesta abertura
// ((0,0) = nativa).
func SetStartupWindowSize(_, h int32) { startupWindowH = h }

// renderRestartPending diz se a resolucao escolhida ainda nao esta valendo.
// Compara pelo que a janela SERIA (settings.WindowSize), nao pelo numero da
// opcao: 900p numa tela que ja tem 900 de altura e nativa, e nao pede nada.
func renderRestartPending(s settings.Settings) bool {
	if !mobileBuild {
		return false
	}
	_, h := settings.WindowSize(s)
	return h != startupWindowH
}

// renderNowHint e a dica da linha RESOLUCAO no celular.
func renderNowHint(s settings.Settings) string {
	w, h := rl.GetScreenWidth(), rl.GetScreenHeight()
	if renderRestartPending(s) {
		return fmt.Sprintf("agora %dx%d - reinicie para aplicar", w, h)
	}
	if startupWindowH == 0 {
		if s.RenderHeight > 0 && s.DisplayHeight > 0 && s.RenderHeight >= s.DisplayHeight {
			return fmt.Sprintf("a tela ja tem %dp: fica nativa", s.DisplayHeight)
		}
		return fmt.Sprintf("agora: nativa, %dx%d", w, h)
	}
	return fmt.Sprintf("agora: %dx%d, ampliada pela tela", w, h)
}

// updateRestartButton desenha o botao de reinicio no canto do painel quando
// a resolucao escolhida ainda nao vale. Dentro de uma partida ele e
// REINICIAR E VOLTAR: grava o bilhete de volta (resume_session.go) antes de
// o processo morrer. Grava tudo ANTES: o processo morre no meio do clique.
func (scr *SettingsScreen) updateRestartButton(l settingsLayout) {
	if !apprestart.Supported || !renderRestartPending(settings.Get()) {
		return
	}
	if scr.restartFailed {
		// O Android recusou (codigo no logcat, marca [Graficos]). A escolha
		// e o bilhete ja estao gravados; fechar de verdade e abrir de novo
		// aplica e volta a partida.
		r := l.RestartRect
		size := int32(max(r.Height*0.3, Px(12)))
		rl.DrawText("feche e abra o jogo", int32(r.X), int32(r.Y+r.Height/2)-size/2, size, rl.NewColor(246, 216, 142, 255))
		return
	}
	ticket, resuming := captureResume()
	label := "REINICIAR AGORA"
	if resuming {
		label = "REINICIAR E VOLTAR"
		drawHostRestartWarning(l)
	}
	if !SmallButton(l.RestartRect, label) {
		return
	}
	if err := settings.Save(); err != nil {
		log.Printf("[Graficos] falha ao gravar antes de reiniciar: %v", err)
	}
	if resuming {
		if err := resume.Save(ticket); err != nil {
			log.Printf("[Graficos] falha ao gravar a volta a partida: %v", err)
		}
	}
	if err := apprestart.Restart(); err != nil {
		rl.TraceLog(rl.LogWarning, "[Graficos] %v", err)
		scr.restartFailed = true
	}
}

// drawHostRestartWarning avisa o HOST com gente de verdade na partida de que
// ela cai para todos: a partida mora no processo dele. Entre o VOLTAR e o
// botao, alinhado a direita do vao.
func drawHostRestartWarning(l settingsLayout) {
	if network.Role != "host" || network.OtherHumansPresent() == 0 {
		return
	}
	// Vale para rede local e online: a sala online volta com o MESMO codigo
	// (relay.HostAs), e a reconexao dos amigos ja disca esse codigo.
	lines := [2]string{"os outros caem e voltam", "sozinhos em segundos"}
	r := l.RestartRect
	size := int32(max(r.Height*0.22, Px(11)))
	right := int32(r.X - r.Height*0.3)
	color := rl.NewColor(236, 150, 120, 255)
	for i, line := range lines {
		w := rl.MeasureText(line, size)
		y := int32(r.Y+r.Height/2) - size + int32(i)*(size+2)
		rl.DrawText(line, right-w, y, size, color)
	}
}
