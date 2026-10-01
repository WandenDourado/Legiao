package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// drawOnlineRoomSetup desenha a tela de nome da sala online e devolve o que o
// jogador tocou neste quadro. Com busy (registrando no relay) nada responde.
func drawOnlineRoomSetup(name, status string, busy bool) (create, back bool, key rune) {
	sw := float32(rl.GetScreenWidth())
	sh := float32(rl.GetScreenHeight())
	drawMenuBackdrop(sw, sh)
	drawMenuCentered("CRIAR PARTIDA ONLINE", sw/2, sh*0.195, int32(max(sh*0.03, Px(18))), rl.NewColor(239, 230, 192, 255))

	pw := min(sw*0.82, Px(900))
	p := rl.NewRectangle((sw-pw)/2, sh*0.25, pw, sh*0.72)
	drawMenuPanel(p)

	drawMenuCentered("NOME DA SALA", sw/2, p.Y+p.Height*0.04, int32(max(p.Height*0.04, Px(12))), rl.NewColor(245, 218, 145, 255))
	box := rl.NewRectangle(p.X+p.Width*0.12, p.Y+p.Height*0.1, p.Width*0.76, p.Height*0.12)
	rl.DrawRectangleRounded(box, 0.2, 8, rl.NewColor(16, 31, 31, 255))
	rl.DrawRectangleRoundedLinesEx(box, 0.2, 8, 2, rl.NewColor(224, 187, 97, 255))
	caret := ""
	if !busy && int(rl.GetTime()*2)%2 == 0 {
		caret = "_"
	}
	size := int32(max(box.Height*0.45, Px(16)))
	drawMenuCentered(name+caret, sw/2, box.Y+(box.Height-float32(size))/2, size, rl.NewColor(238, 242, 248, 255))

	kb := rl.NewRectangle(p.X+p.Width*0.05, p.Y+p.Height*0.26, p.Width*0.9, p.Height*0.46)
	switch {
	case useOnScreenKeyboard && !busy:
		key = drawOnScreenKeyboard(kb)
	case !useOnScreenKeyboard:
		hint := int32(max(p.Height*0.04, Px(13)))
		drawMenuCentered("Digite o nome no teclado", sw/2, kb.Y+kb.Height*0.35, hint, rl.Fade(rl.White, 0.72))
		drawMenuCentered("ENTER cria a sala  -  ESC volta", sw/2, kb.Y+kb.Height*0.35+float32(hint)*1.8, int32(float32(hint)*0.8), rl.Fade(rl.White, 0.5))
	}

	msg, color := status, rl.NewColor(240, 150, 120, 255)
	if busy {
		msg, color = "Abrindo sala no servidor...", rl.NewColor(219, 235, 225, 255)
	}
	if msg != "" {
		drawMenuCentered(msg, sw/2, p.Y+p.Height*0.76, int32(max(p.Height*0.035, Px(12))), color)
	}

	mouse := rl.GetMousePosition()
	bh := max(p.Height*0.11, Px(44))
	backRect := rl.NewRectangle(p.X+p.Width*0.1, p.Y+p.Height*0.84, p.Width*0.36, bh)
	createRect := rl.NewRectangle(p.X+p.Width*0.54, p.Y+p.Height*0.84, p.Width*0.36, bh)
	drawCharacterAction(backRect, "VOLTAR", false, hit(backRect, mouse))
	drawCharacterAction(createRect, "CRIAR SALA", true, hit(createRect, mouse))
	if !busy && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		back, create = hit(backRect, mouse), hit(createRect, mouse)
	}
	return create, back, key
}
