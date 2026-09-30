package ui

// Drawing for a client that is WAITING FOR THE HOST: the full-screen overlay
// shown while this machine has no usable link — a first connection that has
// not opened yet, or one that opened and dropped — and the small marker drawn
// over any OTHER player's body while THEIR connection is down. Two different
// audiences, one file because they are the same feature seen from two sides.

import (
	"fmt"
	"time"

	"github.com/WandenDourado/Legiao/internal/network"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// drawWaitBackdrop dims the screen and centres the title above the middle.
//
// As duas esperas compartilham isto porque sao a mesma tela: o mundo espelhado
// desenhado atras (game.DrawFrame) e uma frase por cima dizendo que ele nao
// esta sendo governado por ninguem agora. O que as diferencia e so o que vem
// ABAIXO do titulo.
func drawWaitBackdrop(sw, sh float32, title string) {
	rl.DrawRectangle(0, 0, int32(sw), int32(sh), rl.Fade(rl.Black, 0.55))

	titleSize := int32(sh * 0.06)
	if titleSize < int32(Px(32)) {
		titleSize = int32(Px(32))
	}
	titleWidth := rl.MeasureText(title, titleSize)
	rl.DrawText(title,
		int32(sw/2)-titleWidth/2,
		int32(sh/2)-titleSize-int32(sh*0.02),
		titleSize, rl.White)
}

// DrawConnectingOverlay cobre o intervalo entre escolher a sala e a conexao
// existir de verdade.
//
// Sem ela, esse intervalo era desenhado como um JOGO: papel de cliente nao
// simula nada, entao a tela mostrava o mapa sem inimigos e sem os outros
// jogadores, e o jogador andava por um mundo vazio achando que estava na
// partida. Ver network/client_connect.go. Nao tem contagem porque nao ha
// janela para contar: o prazo do dial (dialTimeout) resolve em segundos, e o
// que o jogador precisa saber e que ainda nao entrou.
func DrawConnectingOverlay(sw, sh float32) {
	drawWaitBackdrop(sw, sh, "Conectando ao anfitriao...")

	sub := "Aguardando resposta da sala"
	subSize := int32(sh * 0.035)
	if subSize < int32(Px(20)) {
		subSize = int32(Px(20))
	}
	subWidth := rl.MeasureText(sub, subSize)
	rl.DrawText(sub,
		int32(sw/2)-subWidth/2,
		int32(sh/2)+int32(sh*0.01),
		subSize, rl.Fade(rl.White, 0.85))
}

// DrawReconnectOverlay dims the screen and shows the countdown for the local
// player's own lost connection. The mirrored world is drawn BEHIND this by
// the caller (game.DrawFrame) — the overlay only adds the dimming and text on
// top of it, the same layering GameOverPanel uses.
func DrawReconnectOverlay(sw, sh float32, remaining time.Duration) {
	drawWaitBackdrop(sw, sh, "Reconectando...")

	mm := int(remaining.Minutes())
	ss := int(remaining.Seconds()) % 60
	countdown := fmt.Sprintf("%02d:%02d", mm, ss)
	countSize := int32(sh * 0.045)
	if countSize < int32(Px(24)) {
		countSize = int32(Px(24))
	}
	countWidth := rl.MeasureText(countdown, countSize)
	rl.DrawText(countdown,
		int32(sw/2)-countWidth/2,
		int32(sh/2)+int32(sh*0.01),
		countSize, rl.Fade(rl.White, 0.85))

	// Sala online: o motivo de ainda nao ter voltado (network.ReconnectHint).
	if hint := network.ReconnectHint(); hint != "" {
		size := int32(max(sh*0.024, Px(13)))
		y := int32(sh/2) + int32(sh*0.01) + countSize + size
		rl.DrawText(hint, int32(sw/2)-rl.MeasureText(hint, size)/2, y, size, rl.Fade(rl.NewColor(245, 218, 145, 255), 0.95))
	}
}

// DrawAbsentMarker tags a remote player's body as mid-reconnect: a small
// label above the head, so the rest of the party can tell "not moving
// because the connection dropped" apart from "not moving because dead" (that
// one already greys out via the death shader) or "not moving because
// nobody's pressing a key".
func DrawAbsentMarker(x, y float32) {
	label := "reconectando..."
	size := int32(14)
	width := rl.MeasureText(label, size)
	rl.DrawText(label, int32(x)-width/2, int32(y)-40, size, rl.Fade(rl.Yellow, 0.9))
}
