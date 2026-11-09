package game

// A seta de borda de tela das gargulas (playtest de 27/09/2026): quem atira de
// fora da tela tem de ser achavel sem andar ate ela. Ver ui/offscreen_marker.go.

import (
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/ui"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// sentryMarkerNear e a distancia (mundo, do centro da camera) ate onde a
	// seta sai com forca total; sentryMarkerFar, onde ela chega ao minimo.
	// Uma tela e meia e o bastante para "logo ali"; 5000 ja e outra sala.
	sentryMarkerNear float32 = 1800
	sentryMarkerFar  float32 = 5000
)

// sentryMarkerBuf e reaproveitado entre quadros: com dez gargulas no mapa 7,
// alocar a lista por quadro seria lixo por nada.
var sentryMarkerBuf []rl.Vector2

// drawSentryMarkers desenha uma seta na borda da tela para cada gargula viva
// que a camera nao mostra. Espaco de TELA: chamado depois do bloco de camera.
func drawSentryMarkers(cam Camera2DState, sw, sh float32) {
	if network.GameOver {
		return
	}
	sentryMarkerBuf = network.LivingSentryCenters(sentryMarkerBuf[:0])
	if len(sentryMarkerBuf) == 0 {
		return
	}
	inset := ui.SentryMarkerInset()
	for _, pos := range sentryMarkerBuf {
		screen := rl.GetWorldToScreen2D(pos, cam.Camera)
		at, dir, onScreen := ui.EdgeMarkerPoint(screen, sw, sh, inset)
		if onScreen {
			continue
		}
		dist := rl.Vector2Distance(pos, cam.Camera.Target)
		ui.DrawSentryMarker(at, dir, markerStrength(dist))
	}
}

// markerStrength e 1 ate sentryMarkerNear e cai linear ate 0 em
// sentryMarkerFar.
func markerStrength(dist float32) float32 {
	switch {
	case dist <= sentryMarkerNear:
		return 1
	case dist >= sentryMarkerFar:
		return 0
	default:
		return 1 - (dist-sentryMarkerNear)/(sentryMarkerFar-sentryMarkerNear)
	}
}
