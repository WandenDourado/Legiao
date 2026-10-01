package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Teclado desenhado pelo proprio jogo.
//
// Existe porque o raylib nao abre o teclado do Android (ver
// settings_textfield.go): sem ele, dar nome a uma sala online no celular seria
// impossivel. So maiusculas, digitos e espaco - e o que a fonte padrao desenha
// sem surpresa e o que um nome de sala precisa. No desktop o teclado fisico
// continua valendo ao mesmo tempo (online_room_setup.go).

var keyboardRows = []string{"1234567890", "QWERTYUIOP", "ASDFGHJKL", "ZXCVBNM"}

// Teclas especiais devolvidas por drawOnScreenKeyboard.
const (
	keyNone      rune = 0
	keyBackspace rune = -1
	keySpace     rune = ' '
)

// drawOnScreenKeyboard desenha o teclado dentro de area e devolve a tecla
// tocada neste quadro, ou keyNone.
func drawOnScreenKeyboard(area rl.Rectangle) rune {
	rows := float32(len(keyboardRows) + 1)
	gap := max(area.Width*0.008, Px(4))
	keyW := (area.Width - gap*9) / 10
	keyH := (area.Height - gap*(rows-1)) / rows
	mouse := rl.GetMousePosition()
	pressed := rl.IsMouseButtonPressed(rl.MouseLeftButton)
	got := keyNone

	for i, row := range keyboardRows {
		n := float32(len(row))
		rowW := n*keyW + (n-1)*gap
		x := area.X + (area.Width-rowW)/2
		y := area.Y + float32(i)*(keyH+gap)
		for j, ch := range row {
			rect := rl.NewRectangle(x+float32(j)*(keyW+gap), y, keyW, keyH)
			if drawKey(rect, string(ch), mouse) && pressed {
				got = ch
			}
		}
	}

	y := area.Y + float32(len(keyboardRows))*(keyH+gap)
	space := rl.NewRectangle(area.X+keyW+gap, y, keyW*5+gap*4, keyH)
	back := rl.NewRectangle(space.X+space.Width+gap, y, keyW*3+gap*2, keyH)
	if drawKey(space, "ESPACO", mouse) && pressed {
		got = keySpace
	}
	if drawKey(back, "APAGAR", mouse) && pressed {
		got = keyBackspace
	}
	return got
}

// drawKey desenha uma tecla e diz se o ponteiro esta sobre ela.
func drawKey(rect rl.Rectangle, label string, mouse rl.Vector2) bool {
	over := hit(rect, mouse)
	fill := rl.NewColor(39, 70, 69, 255)
	if over {
		fill = rl.NewColor(62, 98, 92, 255)
	}
	rl.DrawRectangleRounded(rect, 0.18, 6, fill)
	rl.DrawRectangleRoundedLinesEx(rect, 0.18, 6, 1.5, rl.NewColor(126, 163, 154, 255))
	size := int32(max(rect.Height*0.42, Px(12)))
	if len(label) > 1 {
		size = int32(max(rect.Height*0.3, Px(11)))
	}
	drawMenuCentered(label, rect.X+rect.Width/2, rect.Y+(rect.Height-float32(size))/2, size, rl.NewColor(228, 239, 228, 255))
	return over
}
