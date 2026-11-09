package ui

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// referenceArtAspect e a proporcao da arte de selecao padronizada
// (1024x1536). Ela mora aqui, e nao no pacote entity, porque e um fato de
// LAYOUT: quem precisa dela e o card, para dimensionar o slot e a moldura.
const referenceArtAspect = 1024.0 / 1536.0

// drawCharacterSelectScreen renders the choice as a hero card. It returns the
// controls separately so ShowCharacterSelect remains responsible for input.
func drawCharacterSelectScreen(def entity.CharacterDef, preview characterPreview) (rl.Rectangle, rl.Rectangle, rl.Rectangle, rl.Rectangle) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	drawMenuBackdrop(sw, sh)

	headingSize := int32(max(sh*0.032, Px(19)))
	drawMenuCentered("ESCOLHA SEU HEROI", sw/2, sh*0.205, headingSize, rl.NewColor(239, 230, 192, 255))
	panelW := min(sw*0.66, Px(740))
	panelH := min(sh*0.68, Px(600))
	panel := rl.NewRectangle((sw-panelW)/2, sh*0.265, panelW, panelH)
	drawMenuPanel(panel)

	// O slot da arte, e a moldura dele.
	//
	// Os dois saem de um retangulo FIXO, calculado antes de desenhar. Antes a
	// moldura era derivada do retangulo que o desenho devolvia — e como cada
	// personagem tinha uma tela diferente (1122x1402, 1183x1330, 1254x1254),
	// ela mudava de largura a cada seta apertada e o card inteiro pulava. A
	// arte padronizada resolveria isso sozinha; o retangulo fixo garante que
	// continue resolvido se um dia entrar arte fora do padrao.
	//
	// O slot tem a proporcao da propria arte, entao a arte o preenche e a
	// moldura a abraca. Um slot de outra proporcao deixaria faixas vazias
	// dentro da moldura, porque o encaixe preserva a proporcao.
	// A pilha vertical do card e apertada: moldura, nome, dica das setas e os
	// dois botoes dividem a altura do painel. Aumentar o slot sem subir o topo
	// empurra o nome para cima da dica — foi o que aconteceu na primeira
	// tentativa, com 0.60 comecando em 0.09.
	previewTop := panel.Y + panel.Height*0.06
	slotH := panel.Height * 0.58
	slotW := slotH * referenceArtAspect
	frame := rl.NewRectangle(sw/2-slotW/2-10, previewTop-10, slotW+20, slotH+20)
	rl.DrawRectangleRounded(frame, 0.04, 8, rl.Fade(rl.Black, 0.28))
	rl.DrawRectangleRoundedLinesEx(frame, 0.04, 8, 2, rl.NewColor(212, 181, 101, 255))
	// Depois da moldura, para a arte nunca ficar escondida pelo proprio fundo.
	drawCharacterPreview(preview, def, sw/2, previewTop, slotW, slotH)

	buttonH := max(panel.Height*0.12, Px(48))
	buttonY := panel.Y + panel.Height - buttonH - panel.Height*0.065

	nameSize := int32(max(panel.Height*0.07, Px(18)))
	// O nome ancora na MOLDURA, nao na arte: com a moldura fixa, ele para de
	// subir e descer conforme a silhueta do personagem.
	nameY := frame.Y + frame.Height + panel.Height*0.02
	drawMenuCentered(def.Name, sw/2, nameY, nameSize, rl.NewColor(244, 218, 145, 255))
	countSize := int32(max(panel.Height*0.035, Px(12)))
	nameBottom := nameY + float32(nameSize)/2
	buttonTop := buttonY

	instructionY := (nameBottom + buttonTop) / 2
	drawMenuCentered("USE AS SETAS PARA EXPLORAR OS HEROIS", sw/2, instructionY, countSize, rl.Fade(rl.White, 0.68))

	navSize := max(panel.Height*0.14, Px(48))
	previewMid := frame.Y + frame.Height/2
	prev := rl.NewRectangle(panel.X+panel.Width*0.08, previewMid-navSize/2, navSize, navSize)
	next := rl.NewRectangle(panel.X+panel.Width-panel.Width*0.08-navSize, previewMid-navSize/2, navSize, navSize)
	drawCharacterNav(prev, false, hit(prev, rl.GetMousePosition()))
	drawCharacterNav(next, true, hit(next, rl.GetMousePosition()))

	gap := panel.Width * 0.035
	back := rl.NewRectangle(panel.X+panel.Width*0.1, buttonY, panel.Width*0.31, buttonH)
	confirm := rl.NewRectangle(back.X+back.Width+gap, buttonY, panel.Width*0.49, buttonH)
	drawCharacterAction(back, "VOLTAR", false, hit(back, rl.GetMousePosition()))
	drawCharacterAction(confirm, "ESCOLHER HEROI", true, hit(confirm, rl.GetMousePosition()))
	return prev, next, back, confirm
}

func drawCharacterNav(rect rl.Rectangle, forward, hovered bool) {
	fill := rl.NewColor(39, 70, 69, 255)
	if hovered {
		fill = rl.NewColor(57, 91, 86, 255)
	}
	rl.DrawRectangleRounded(rect, 0.18, 8, fill)
	rl.DrawRectangleRoundedLinesEx(rect, 0.18, 8, 2, rl.NewColor(151, 192, 179, 255))
	cx, cy := rect.X+rect.Width/2, rect.Y+rect.Height/2
	if forward {
		rl.DrawTriangle(rl.NewVector2(cx+rect.Width*0.18, cy), rl.NewVector2(cx-rect.Width*0.1, cy-rect.Height*0.2), rl.NewVector2(cx-rect.Width*0.1, cy+rect.Height*0.2), rl.NewColor(220, 238, 226, 255))
		return
	}
	rl.DrawTriangle(rl.NewVector2(cx-rect.Width*0.18, cy), rl.NewVector2(cx+rect.Width*0.1, cy+rect.Height*0.2), rl.NewVector2(cx+rect.Width*0.1, cy-rect.Height*0.2), rl.NewColor(220, 238, 226, 255))
}

func drawCharacterAction(rect rl.Rectangle, label string, primary, hovered bool) {
	fill := rl.NewColor(38, 69, 68, 255)
	border := rl.NewColor(135, 176, 162, 255)
	text := rl.NewColor(226, 236, 223, 255)
	if primary {
		fill = rl.NewColor(98, 75, 38, 255)
		border = rl.NewColor(225, 188, 99, 255)
		text = rl.NewColor(250, 224, 150, 255)
	}
	if hovered {
		fill = rl.NewColor(uint8(min(int(fill.R)+17, 255)), uint8(min(int(fill.G)+17, 255)), uint8(min(int(fill.B)+12, 255)), 255)
	}
	rl.DrawRectangleRounded(rect, 0.09, 8, fill)
	rl.DrawRectangleRoundedLinesEx(rect, 0.09, 8, 2, border)
	size := int32(max(rect.Height*0.31, Px(16)))
	drawMenuCentered(label, rect.X+rect.Width/2, rect.Y+(rect.Height-float32(size))/2, size, text)
}
