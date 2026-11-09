//go:build android

package ui

import (
	"github.com/WandenDourado/Legiao/internal/settings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// applyTargetFPS: igual ao desktop, o teto de quadro sempre pode ser
// recadenciado na hora — é o próprio motivo de doc/performance.md (mobile
// #5) recomendar expor 30 fps como opção.
func applyTargetFPS(fc settings.FrameCap) {
	rl.SetTargetFPS(int32(fc))
}

// toggleFullscreenIfDesktop e um NO-OP no Android: a janela ja preenche a
// tela inteira por causa de game.Config.FullScreen (que aqui significa
// "layout Android", nao "modo de janela" — ver internal/game/config.go).
// Chamar rl.ToggleFullscreen num contexto que nunca teve modo janela nao
// tem o que alternar; a linha de UI que oferece esta opcao so aparece
// rotulada DESKTOP, mas o build Android continua desenhando a mesma tela
// (ver settings_section_video.go) para as abas nao mudarem de layout entre
// plataformas.
func toggleFullscreenIfDesktop() {}

// mobileBuild liga as linhas que so fazem sentido no celular (resolucao de
// desenho) e o teste automatico com varias resolucoes.
const mobileBuild = true
