package game

// "2/3" sobre o portal: quantos do grupo já entraram.
//
// Sem isto, um portal aberto que não leva ninguém é indistinguível de um portal
// quebrado — o jogador que chegou primeiro fica parado na luz vendo nada
// acontecer. O número é a diferença entre "a fase travou" e "a fase está
// esperando o seu time".
//
// Ele só aparece quando há ALGUÉM dentro e o grupo ainda não está completo:
// ninguém dentro não tem o que anunciar, e todos dentro já virou viagem no
// mesmo quadro.

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// portalCounterSize is measured against the doorway, not against the
	// screen: this is drawn in world space, so a fixed screen size would grow
	// and shrink with the camera zoom.
	portalCounterSize = 34
	// portalCounterLift keeps the text clear of the oval's rim.
	portalCounterLift = 26
)

// drawPortalCounter paints the party count above one portal.
func drawPortalCounter(s portalShape, tally portalTally) {
	if !tally.waiting() {
		return
	}
	label := fmt.Sprintf("%d/%d", tally.Inside, tally.Alive)
	width := rl.MeasureText(label, portalCounterSize)
	x := int32(s.Center.X) - width/2
	y := int32(s.Center.Y-s.RY) - portalCounterLift - portalCounterSize

	// Escrito duas vezes: uma sombra escura atrás e o texto claro por cima. O
	// portal é a coisa mais luminosa do mapa, e texto claro sobre luz clara
	// desaparece exatamente onde ele precisa ser lido.
	rl.DrawText(label, x+2, y+2, portalCounterSize, rl.Fade(rl.Black, 0.55*s.Reveal))
	rl.DrawText(label, x, y, portalCounterSize, s.fade(portalRim, 1))
}
