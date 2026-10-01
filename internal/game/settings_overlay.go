package game

// Acesso às Configurações DE DENTRO da partida (Passo 6): ESC no desktop, e
// em toda plataforma um ícone de engrenagem ao lado do contador de horda
// (ui.DrawSettingsAccessButton, wired em renderer.go) — os dois caminhos
// abrem a mesma tela. A PARTIDA NÃO PAUSA — nada aqui congela o laço principal
// (game/loop.go); a tela só é desenhada por cima e a simulação continua,
// exatamente como a sobreposição de reconexão já faz.
//
// ESC já tem um significado dentro do jogo: cancelar a espera de portal
// (portal_cancel.go). Resolvido assim, sem duas leituras na mesma tecla no
// mesmo quadro: RequestOpenSettingsOverlay só é chamada FORA do galho
// LocalPlayerInPortal em ProcessInput (input_handler.go) — enquanto o
// jogador espera um portal, ESC continua sendo só "sair do portal".
//
// SPLIT EM DOIS PASSOS DE PROPOSITO. Abrir (ler ESC/toque) só precisa de
// input e roda em ProcessInput, ANTES de DrawFrame chamar rl.BeginDrawing.
// Atualizar a tela em si (uiSettingsScreen.Update) DESENHA — sliders,
// painel, texto — e por isso só pode rodar dentro do bloco de desenho de
// DrawFrame (renderer.go), como todo o resto da UI imediata deste jogo.
// Chamar a tela de dentro de ProcessInput desenharia fora de
// BeginDrawing/EndDrawing e não apareceria em lugar nenhum.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	uiSettingsScreen    = ui.NewSettingsScreen()
	settingsOverlayOpen bool
)

// SettingsOverlayOpen reports whether the overlay currently owns local
// input this frame — ProcessInput freezes movement/attack/skill reading
// while it does, without freezing the simulation.
func SettingsOverlayOpen() bool { return settingsOverlayOpen }

// RequestOpenSettingsOverlay checks this frame's open gesture (ESC on
// desktop, a tap on the mobile button) and, on a hit, opens the overlay and
// releases whatever touch gesture the action cluster was mid-hold on — the
// overlay is about to start reading touches of its own (sliders), and a
// held aim left armed underneath it would fire the moment the player closes
// the screen. No-op while already open; call once per frame from
// ProcessInput, outside the LocalPlayerInPortal branch.
func RequestOpenSettingsOverlay(cfg Config, aj *input.AimJoystick, sw, sh float32) {
	if settingsOverlayOpen {
		return
	}
	openedByEsc := false
	opened := false
	if !cfg.FullScreen && rl.IsKeyPressed(rl.KeyEscape) {
		opened, openedByEsc = true, true
	} else if ui.SettingsAccessButtonTapped(sw, sh) {
		// A engrenagem abre em qualquer plataforma agora: no desktop ela e
		// so um segundo caminho ao lado do ESC (o clique que a acerta e
		// consumido aqui mesmo — ProcessInput retorna antes do galho de
		// "clique ataca" mais abaixo, entao a engrenagem nunca dispara um
		// ataque de brinde).
		opened = true
	}
	if opened {
		settingsOverlayOpen = true
		cancelAndroidAim(aj)
		if openedByEsc {
			uiSettingsScreen.SuppressNextEsc()
		}
	}
}

// UpdateAndDrawSettingsOverlay draws and updates the settings screen for one
// frame, and saves+closes it once the player leaves. No-op when the overlay
// is not open. Called from DrawFrame's screen-space pass — see the file
// comment for why it cannot run from ProcessInput.
func UpdateAndDrawSettingsOverlay(sw, sh float32) {
	if !settingsOverlayOpen {
		return
	}
	if uiSettingsScreen.Update(sw, sh) {
		// Gravar em disco so ao SAIR da tela (Passo 4), nunca a cada quadro
		// de arraste — o mesmo ponto de saida que ui.ShowMenu usa fora da
		// partida.
		if err := settings.Save(); err != nil {
			log.Printf("[Configuracoes] falha ao gravar settings.json: %v", err)
		}
		settingsOverlayOpen = false
	}
}

// DrawSettingsOverlayButton paints the gear-icon entry point (every
// platform now, not just mobile), hidden while the overlay itself is
// already open (it would sit uselessly behind it).
func DrawSettingsOverlayButton(cfg Config, sw, sh float32) {
	if !settingsOverlayOpen {
		ui.DrawSettingsAccessButton(sw, sh)
	}
}
