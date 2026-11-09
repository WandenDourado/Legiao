package ui

// A tela de Configurações inteira (Passo 4): reusa o pano de fundo e o
// painel do menu principal (drawMenuBackdrop/drawMenuPanel, main_menu.go)
// para parecer a mesma família de tela, com cinco abas (Passo 5) e um botão
// VOLTAR sempre visível — o único jeito de sair no celular.
//
// Aplica toda mudança NA HORA (sem Salvar/Cancelar, ver doc/settings.md):
// cada seção chama settings.Update diretamente e, quando o campo tem efeito
// fora deste pacote (áudio, geometria de toque, teto de quadro...), o setter
// correspondente também — ver settings_apply.go para os que cruzam pacote.
// O que fica só para SAIR é gravar em disco (settings.Save), porque isso sim
// custaria caro a cada quadro de arraste.
//
// Reaproveitada tanto pelo menu inicial (ShowMenu, menu.go) quanto pela
// sobreposição dentro da partida (internal/game/settings_overlay.go) — as
// duas só chamam Update/Draw numa instância que guardam.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// SettingsScreen holds every widget's cross-frame state (slider drags, the
// name field's editing flag) plus which tab is showing.
type SettingsScreen struct {
	section settingsSection

	masterSlider, musicSlider, skillSlider, attackSlider Slider
	actionScaleSlider, opacitySlider, radiusSlider        Slider
	hudScaleSlider                                        Slider
	nameField                                              TextField

	// bench e o teste automatico de desempenho em andamento (nil = parado), e
	// benchResult a frase que ele deixou na tela. Ver perf_bench.go.
	bench       *perfBench
	benchResult string
	// restartFailed: o REINICIAR AGORA foi recusado pelo Android
	// (render_restart.go); o canto passa a dizer para fechar e abrir.
	restartFailed bool

	// suppressEscOnce evita que o MESMO toque de ESC que ABRIU a
	// sobreposicao (internal/game/settings_overlay.go) tambem a FECHE no
	// mesmo quadro: rl.IsKeyPressed fica true pelo quadro inteiro em que a
	// tecla foi pressionada, e quem abriu ja rodou antes de Update ser
	// chamado (a leitura de abrir acontece em ProcessInput, antes de
	// DrawFrame chamar esta funcao) — sem a supressao, updateExit veria o
	// MESMO ESC e fecharia a tela no instante em que ela abriu.
	suppressEscOnce bool
}

// NewSettingsScreen builds one instance of the screen. Keep it around
// (a package-level var, like uiSkillButton in game/input_handler.go) instead
// of constructing one per frame — Slider/TextField carry drag/edit state
// across frames.
func NewSettingsScreen() *SettingsScreen {
	return &SettingsScreen{
		masterSlider:      NewSlider(),
		musicSlider:       NewSlider(),
		skillSlider:       NewSlider(),
		attackSlider:      NewSlider(),
		actionScaleSlider: NewSlider(),
		opacitySlider:     NewSlider(),
		radiusSlider:      NewSlider(),
		hudScaleSlider:    NewSlider(),
	}
}

// Update draws and updates the whole screen for one frame, and reports
// whether the player asked to leave it (ESC on desktop, or the VOLTAR
// button — see updateExit). Callers are responsible for calling
// settings.Save() once when this returns true (see settings_apply.go); this
// function only applies changes live, never touches disk itself.
func (scr *SettingsScreen) Update(sw, sh float32) bool {
	drawMenuBackdrop(sw, sh)
	l := computeSettingsLayout(sw, sh)
	drawMenuPanel(l.Panel)
	drawMenuCentered("CONFIGURACOES", l.Panel.X+l.Panel.Width/2, l.Panel.Y+l.Panel.Height*0.045,
		int32(max(l.Panel.Height*0.05, Px(20))), rl.NewColor(235, 208, 135, 255))

	scr.updateTabs(l)
	scr.drawTabs(l)

	switch scr.section {
	case sectionAudio:
		scr.updateAudioSection(l)
	case sectionVideo:
		scr.updateVideoSection(l)
	case sectionGraphics:
		scr.updateGraphicsSection(l)
	case sectionControls:
		scr.updateControlsSection(l)
	case sectionInterface:
		scr.updateInterfaceSection(l)
	case sectionPlayer:
		scr.updatePlayerSection(l)
	}

	leaving := scr.updateExit(l)
	if leaving && scr.bench != nil {
		// Sair no meio do teste cancela o teste: sem isto o jogo continuaria
		// sem desenhar o mundo (PerfBenchRunning) depois da tela fechar.
		scr.bench.cancel()
		scr.bench = nil
	}
	return leaving
}

func (scr *SettingsScreen) updateTabs(l settingsLayout) {
	if scr.bench != nil {
		return // o teste so avanca na aba Graficos; trocar de aba o congelaria
	}
	for i := 0; i < int(sectionCount); i++ {
		if clicked(l.tabRect(i)) {
			scr.section = settingsSection(i)
		}
	}
}

func (scr *SettingsScreen) drawTabs(l settingsLayout) {
	for i := 0; i < int(sectionCount); i++ {
		sec := settingsSection(i)
		rect := l.tabRect(i)
		fill := rl.Fade(rl.NewColor(39, 70, 69, 255), 0.9)
		textColor := rl.Fade(rl.White, 0.65)
		if sec == scr.section {
			fill = rl.NewColor(91, 70, 37, 255)
			textColor = rl.NewColor(246, 216, 142, 255)
		}
		rl.DrawRectangleRec(rl.NewRectangle(rect.X, rect.Y, rect.Width-2, rect.Height), fill)
		size := int32(max(rect.Height*0.32, Px(11)))
		label := sec.label()
		w := rl.MeasureText(label, size)
		rl.DrawText(label, int32(rect.X+rect.Width/2)-w/2, int32(rect.Y+rect.Height/2)-size/2, size, textColor)
	}
	rl.DrawLineEx(rl.NewVector2(l.Panel.X, l.TabY+l.TabH), rl.NewVector2(l.Panel.X+l.Panel.Width, l.TabY+l.TabH), 2, rl.Fade(rl.Gold, 0.4))
}

// updateExit handles the two ways out asked for in the Passo 4/6: ESC on
// desktop, and a VOLTAR button that is ALWAYS visible (it is the only way
// out on a phone, which has no ESC key at all).
func (scr *SettingsScreen) updateExit(l settingsLayout) bool {
	back := SmallButton(l.BackRect, "VOLTAR")
	escPressed := rl.IsKeyPressed(rl.KeyEscape) && !scr.suppressEscOnce
	scr.suppressEscOnce = false
	return back || escPressed
}

// SuppressNextEsc makes the next Update() call ignore an ESC press it would
// otherwise see. Called by internal/game/settings_overlay.go the instant it
// opens the overlay FROM an ESC press — see the field comment on
// suppressEscOnce for why that specific case needs it.
func (scr *SettingsScreen) SuppressNextEsc() { scr.suppressEscOnce = true }

// rowLabel draws a row's left-hand caption in the neutral menu text colour.
func rowLabel(l settingsLayout, i int, text string) {
	r := l.row(i)
	size := int32(max(r.Height*0.28, Px(14)))
	rl.DrawText(text, int32(r.X), int32(r.Y+r.Height*0.08), size, rl.Fade(rl.White, 0.88))
}

// hint draws a small secondary line under a row's label — used for the
// VSync "só na próxima abertura" warning and similar caveats.
func rowHint(l settingsLayout, i int, text string) {
	r := l.row(i)
	size := int32(max(r.Height*0.18, Px(11)))
	rl.DrawText(text, int32(r.X), int32(r.Y+r.Height*0.55), size, rl.Fade(rl.NewColor(222, 202, 138, 255), 0.85))
}
