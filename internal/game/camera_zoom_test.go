package game

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// A zoom 1 (PC) a camera faz exatamente o que sempre fez.
func TestCameraZoomOneIsUnchanged(t *testing.T) {
	c := NewCamera(1920, 1080, 1)
	c.Update(rl.NewVector2(4000.4, 3000.6), 1920, 1080, world.Bounds{Width: 7680, Height: 5760})
	if c.Camera.Zoom != 1 || c.Camera.Target.X != 4000 || c.Camera.Target.Y != 3001 {
		t.Errorf("zoom 1 mudou: zoom %.2f alvo (%.2f, %.2f)", c.Camera.Zoom, c.Camera.Target.X, c.Camera.Target.Y)
	}
}

// Com zoom, o clamp mede a tela em unidades de MUNDO (sw/zoom): na borda do
// mapa a camera para meia tela de MUNDO para dentro, nao meia tela de pixel.
func TestCameraClampUsesWorldUnits(t *testing.T) {
	c := NewCamera(2400, 1080, 0.8)
	b := world.Bounds{Width: 7680, Height: 5760}
	c.Update(rl.NewVector2(0, 0), 2400, 1080, b)
	wantX, wantY := float32(2400/0.8/2), float32(1080/0.8/2)
	if d := c.Camera.Target.X - wantX; d > 1.25 || d < -1.25 {
		t.Errorf("alvo X %.2f na borda; esperado ~%.0f (meia tela em mundo)", c.Camera.Target.X, wantX)
	}
	if d := c.Camera.Target.Y - wantY; d > 1.25 || d < -1.25 {
		t.Errorf("alvo Y %.2f na borda; esperado ~%.0f", c.Camera.Target.Y, wantY)
	}
}

// A ancoragem de pixel vale em espaco de TELA: Target*zoom e inteiro.
func TestCameraAnchorsInScreenPixels(t *testing.T) {
	c := NewCamera(2400, 1080, 0.8)
	for _, x := range []float32{3001.3, 3002.9, 3777.77} {
		c.Update(rl.NewVector2(x, 2500.5), 2400, 1080, world.Bounds{Width: 7680, Height: 5760})
		sx := c.Camera.Target.X * c.Camera.Zoom
		if d := sx - float32(int(sx+0.5)); d > 0.001 || d < -0.001 {
			t.Errorf("Target.X*zoom = %.4f nao e inteiro", sx)
		}
	}
}

// O corredor do mapa 6 (2560 px) num celular de 2400 px: a 0,8 sobraria preto
// dos dois lados, entao a camera so se afasta ate o mapa encher a largura.
func TestCameraNeverZoomsOutPastTheMap(t *testing.T) {
	corridor := world.Bounds{Width: 20 * 128, Height: 56 * 128}
	if z := effectiveZoom(0.8, 2400, 1080, corridor); z < 2400.0/2560-0.0001 || z > 2400.0/2560+0.0001 {
		t.Errorf("mapa 6: zoom %.4f; esperado 2400/2560", z)
	}
	big := world.Bounds{Width: 7680, Height: 5760}
	if z := effectiveZoom(0.8, 2400, 1080, big); z != 0.8 {
		t.Errorf("mapa grande: zoom %.4f; esperado 0,8 inteiro", z)
	}
	// A regra so limita o afastamento: um monitor maior que o mapa, a zoom 1,
	// continua a zoom 1.
	if z := effectiveZoom(1, 3840, 2160, corridor); z != 1 {
		t.Errorf("PC num mapa menor que a tela: zoom %.4f; esperado 1", z)
	}
}

func TestSentryMarkerFadesWithDistance(t *testing.T) {
	if markerStrength(100) != 1 || markerStrength(sentryMarkerFar+1) != 0 {
		t.Error("forca fora dos extremos")
	}
	if a, b := markerStrength(2500), markerStrength(4000); !(a > b && b > 0) {
		t.Errorf("forca nao cai com a distancia: %.2f, %.2f", a, b)
	}
}
