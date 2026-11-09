package ui

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func drawDialoguePanel(box rl.Rectangle) {
	// Deslocamentos e espessuras em pixels da tela nativa (Px, text.go).
	shadow := box
	shadow.X += Px(7)
	shadow.Y += Px(8)
	rl.DrawRectangleRounded(shadow, 0.035, 8, rl.Fade(rl.Black, 0.5))
	rl.DrawRectangleRounded(box, 0.035, 8, rl.NewColor(239, 229, 194, 250))
	rl.DrawRectangleRoundedLinesEx(box, 0.035, 8, max(Px(3), 1), rl.NewColor(196, 158, 75, 255))
	inner := box
	inner.X += Px(7)
	inner.Y += Px(7)
	inner.Width -= Px(14)
	inner.Height -= Px(14)
	rl.DrawRectangleRoundedLinesEx(inner, 0.025, 8, 1, rl.Fade(rl.NewColor(84, 68, 38, 255), 0.46))
}

// drawDialoguePortrait desenha o orador solto na tela: sem moldura, sem fundo e
// sem recorte.
//
// As tres ausencias sao a mudanca inteira. A versao anterior punha a arte
// dentro de um retangulo escuro com borda dourada e cortava os 76% de cima de
// uma referencia de corpo inteiro — o que num corpo inteiro deixa o rosto com
// uns 12% da altura do quadro, e a moldura reafirmava que aquilo era uma
// figurinha colada. A arte de retrato ja vem enquadrada e com alfa, entao aqui
// so resta coloca-la no lugar.
//
// O destino vem pronto de dialogueLayout e a arte tem a proporcao dele, entao
// nao ha conta de encaixe: a textura preenche o retangulo. Se um dia entrar
// arte fora da proporcao, o certo e corrigir a arte, nao esticar aqui.
func drawDialoguePortrait(tex rl.Texture2D, dest rl.Rectangle) {
	source := rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height))
	rl.DrawTexturePro(tex, source, dest, rl.NewVector2(0, 0), 0, rl.White)
}

func drawDialogueNameplate(name string, box rl.Rectangle, size int32) {
	width := float32(rl.MeasureText(name, size)) + Px(54)
	plate := rl.NewRectangle(box.X+Px(dialoguePadding), box.Y-float32(size)-Px(18), width, float32(size)+Px(18))
	rl.DrawRectangleRounded(plate, 0.11, 8, rl.NewColor(25, 42, 42, 255))
	rl.DrawRectangleRoundedLinesEx(plate, 0.11, 8, max(Px(2), 1), rl.NewColor(203, 166, 79, 255))
	rl.DrawText(name, int32(plate.X+Px(27)), int32(plate.Y+Px(8)), size, rl.NewColor(250, 218, 133, 255))
}

// drawDialogueHint distinguishes the host, who advances the line, from a
// client, which only receives the host's current dialogue state.
func drawDialogueHint(d network.DialogueState, box rl.Rectangle, size int32) {
	hint := "Enter ou toque para continuar"
	if network.Role == "client" {
		hint = "aguardando o narrador..."
	}
	if d.Total > 0 {
		hint = fmt.Sprintf("%s   %d/%d", hint, d.Index, d.Total)
	}
	width := rl.MeasureText(hint, size)
	rl.DrawText(hint,
		int32(box.X+box.Width-Px(dialoguePadding))-width,
		int32(box.Y+box.Height-Px(dialoguePadding)-float32(size)),
		size, rl.Fade(rl.NewColor(60, 66, 61, 255), 0.78))
}
