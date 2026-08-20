package ui

// Drawing for client-side reconnection: the full-screen overlay shown while
// this machine has lost the host, and the small marker drawn over any OTHER
// player's body while THEIR connection is down. Two different audiences, one
// file because they are the same feature seen from two sides.

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawReconnectOverlay dims the screen and shows the countdown for the local
// player's own lost connection. The mirrored world is drawn BEHIND this by
// the caller (game.DrawFrame) — the overlay only adds the dimming and text on
// top of it, the same layering GameOverPanel uses.
func DrawReconnectOverlay(sw, sh float32, remaining time.Duration) {
	rl.DrawRectangle(0, 0, int32(sw), int32(sh), rl.Fade(rl.Black, 0.55))

	title := "Reconectando..."
	titleSize := int32(sh * 0.06)
	if titleSize < 32 {
		titleSize = 32
	}
	titleWidth := rl.MeasureText(title, titleSize)
	rl.DrawText(title,
		int32(sw/2)-titleWidth/2,
		int32(sh/2)-titleSize-int32(sh*0.02),
		titleSize, rl.White)

	mm := int(remaining.Minutes())
	ss := int(remaining.Seconds()) % 60
	countdown := fmt.Sprintf("%02d:%02d", mm, ss)
	countSize := int32(sh * 0.045)
	if countSize < 24 {
		countSize = 24
	}
	countWidth := rl.MeasureText(countdown, countSize)
	rl.DrawText(countdown,
		int32(sw/2)-countWidth/2,
		int32(sh/2)+int32(sh*0.01),
		countSize, rl.Fade(rl.White, 0.85))
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
