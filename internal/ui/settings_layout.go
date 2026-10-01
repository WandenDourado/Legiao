package ui

// A geometria da tela de Configurações: um painel quase cheio de tela (ela
// tem MUITAS linhas — cinco seções, a maior com cinco opções — e o painel do
// menu principal, dimensionado para dois botões, não sobra espaço quando
// espremido a esse tanto de conteúdo). As proporções são todas relativas à
// tela, como o resto do menu (ver main_menu.go), para caber igual num
// celular deitado e numa janela desktop.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// settingsSection é uma aba da tela. A ordem é a sugerida no Passo 5.
type settingsSection int

const (
	sectionAudio settingsSection = iota
	sectionVideo
	// sectionGraphics: qualidade grafica, resolucao do celular, teste
	// automatico e as chaves de efeito (28/09/2026). Aba propria porque Video
	// ja enchia as cinco linhas que o painel comporta.
	sectionGraphics
	sectionControls
	sectionInterface
	sectionPlayer
	sectionCount
)

func (s settingsSection) label() string {
	switch s {
	case sectionAudio:
		return "AUDIO"
	case sectionVideo:
		return "VIDEO"
	case sectionGraphics:
		return "GRAFICOS"
	case sectionControls:
		return "CONTROLES"
	case sectionInterface:
		return "INTERFACE"
	case sectionPlayer:
		return "JOGADOR"
	default:
		return "?"
	}
}

// settingsLayout é toda geometria derivada da tela atual, recalculada todo
// quadro (a tela pode mudar de tamanho — redimensionar a janela no desktop é
// suportado no resto do jogo, e a tela de Configurações não é diferente).
type settingsLayout struct {
	Panel      rl.Rectangle
	TabY       float32
	TabH       float32
	TabW       float32
	ContentTop float32
	RowH       float32
	RowGap     float32
	LabelX     float32
	ControlX   float32
	ControlW   float32
	BackRect   rl.Rectangle
	// RestartRect e o espelho do VOLTAR no canto direito: REINICIAR AGORA,
	// so quando a resolucao escolhida ainda nao vale (render_restart.go).
	RestartRect rl.Rectangle
}

func computeSettingsLayout(sw, sh float32) settingsLayout {
	panelW := min(sw*0.92, Px(900))
	panelH := min(sh*0.90, Px(720))
	panel := rl.NewRectangle((sw-panelW)/2, (sh-panelH)/2, panelW, panelH)

	tabH := max(panelH*0.09, Px(34))
	tabY := panel.Y + panelH*0.14
	tabW := panelW / float32(sectionCount)

	contentTop := tabY + tabH + panelH*0.02
	// rowH/rowGap tem de caber CINCO linhas (a maior secao, Audio) entre
	// contentTop e o botao VOLTAR sem se sobrepor a ele. Os pisos (38/40)
	// foram calculados a mao para o caso mais apertado testado nesta rodada
	// — celular deitado a ~360px de altura util de painel; TELAS AINDA MAIS
	// BAIXAS que isso sao item da lista de verificacao manual (doc/settings.md),
	// porque nenhum teste automatizado aqui abre uma janela de verdade.
	rowH := max(panelH*0.10, Px(38))
	rowGap := rowH * 0.10

	backH := max(panelH*0.09, Px(40))
	back := rl.NewRectangle(panel.X+panelW*0.06, panel.Y+panelH-backH-panelH*0.04, panelW*0.28, backH)
	restartW := panelW * 0.34
	restart := rl.NewRectangle(panel.X+panelW*0.94-restartW, back.Y, restartW, backH)

	return settingsLayout{
		Panel:       panel,
		TabY:        tabY,
		TabH:        tabH,
		TabW:        tabW,
		ContentTop:  contentTop,
		RowH:        rowH,
		RowGap:      rowGap,
		LabelX:      panel.X + panelW*0.06,
		ControlX:    panel.X + panelW*0.42,
		ControlW:    panelW * 0.52,
		BackRect:    back,
		RestartRect: restart,
	}
}

// row devolve o retângulo da linha de índice i dentro do conteúdo da seção
// atual (0 = primeira linha logo abaixo das abas).
func (l settingsLayout) row(i int) rl.Rectangle {
	y := l.ContentTop + float32(i)*(l.RowH+l.RowGap)
	return rl.NewRectangle(l.LabelX, y, l.Panel.X+l.Panel.Width-l.LabelX-l.Panel.Width*0.06, l.RowH)
}

// controlRect é a faixa à direita da linha i, onde o slider/toggle mora — a
// label ocupa a faixa à esquerda dela (ver LabelX/ControlX).
func (l settingsLayout) controlRect(i int) rl.Rectangle {
	r := l.row(i)
	return rl.NewRectangle(l.ControlX, r.Y+r.Height*0.28, l.ControlW, r.Height*0.44)
}

// tabRect é o retângulo da aba de índice idx (0..sectionCount-1).
func (l settingsLayout) tabRect(idx int) rl.Rectangle {
	return rl.NewRectangle(l.Panel.X+float32(idx)*l.TabW, l.TabY, l.TabW, l.TabH)
}

// toggleRect é o retângulo do interruptor de uma linha — bem mais estreito
// que controlRect, que foi desenhado pensando na trilha do slider de
// volume. Um interruptor esticado até a mesma largura virava a mesma barra
// horizontal do slider ao lado (na aba Áudio), fácil de confundir um
// controle com o outro. Um switch só precisa de espaço para os dois
// estados (bolinha à esquerda/direita), então este é um retângulo curto,
// ancorado no mesmo X que o slider (ControlX) para a coluna de controles
// continuar alinhada entre linhas de tipos diferentes.
func (l settingsLayout) toggleRect(i int) rl.Rectangle {
	ctrl := l.controlRect(i)
	return rl.NewRectangle(ctrl.X, ctrl.Y, ctrl.Height*2.1, ctrl.Height)
}

// buttonRect e um botao de acao dentro da coluna de controles da linha i (o
// TESTAR da aba Graficos): mais alto que o interruptor, para o dedo.
func (l settingsLayout) buttonRect(i int) rl.Rectangle {
	r := l.row(i)
	return rl.NewRectangle(l.ControlX, r.Y+r.Height*0.12, l.ControlW*0.45, r.Height*0.76)
}
