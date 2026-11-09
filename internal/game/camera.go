package game

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Camera2DState manages the rl.Camera2D that follows the player.
type Camera2DState struct {
	Camera rl.Camera2D
	// Zoom e o Config.CameraZoom da plataforma, fixo pela sessao inteira.
	Zoom float32
}

// NewCamera creates a Camera2DState for the platform's zoom (Config.CameraZoom;
// zero or negative falls back to 1).
func NewCamera(sw, sh, zoom float32) Camera2DState {
	if zoom <= 0 {
		zoom = 1
	}
	return Camera2DState{
		Camera: rl.Camera2D{
			Offset:   rl.NewVector2(sw/2, sh/2),
			Target:   rl.NewVector2(0, 0),
			Rotation: 0,
			Zoom:     zoom,
		},
		Zoom: zoom,
	}
}

// Update moves the camera directly to the target position,
// then clamps the result to world bounds so the viewport never shows outside the map.
//
// Everything below is measured in WORLD units: with zoom z the screen covers
// sw/z x sh/z of the map, so that is what the clamp compares against. At zoom
// 1 (desktop) the two are the same number, which is why this used to read sw.
func (c *Camera2DState) Update(target rl.Vector2, sw, sh float32, bounds world.Bounds) {
	zoom := effectiveZoom(c.Zoom, sw, sh, bounds)
	c.Camera.Offset = rl.NewVector2(sw/2, sh/2)
	c.Camera.Zoom = zoom

	// Set camera target directly to player position
	c.Camera.Target.X = target.X
	c.Camera.Target.Y = target.Y

	// Clamp to world bounds, handling maps smaller than the screen
	viewW, viewH := sw/zoom, sh/zoom
	halfW := viewW / 2
	halfH := viewH / 2
	if bounds.Width < viewW {
		// Map narrower than screen – keep camera centered horizontally
		c.Camera.Target.X = bounds.Width / 2
	} else {
		c.Camera.Target.X = clamp(c.Camera.Target.X, halfW, bounds.Width-halfW)
	}
	if bounds.Height < viewH {
		// Map shorter than screen – keep camera centered vertically
		c.Camera.Target.Y = bounds.Height / 2
	} else {
		c.Camera.Target.Y = clamp(c.Camera.Target.Y, halfH, bounds.Height-halfH)
	}

	// ANCORAGEM EM PIXEL (C2 de doc/performance.md), depois do clamp.
	//
	// O terreno nao define filtro de textura, entao usa o padrao do raylib:
	// POINT, vizinho mais proximo. Com a camera em coordenada fracionaria — e
	// ela ficava, porque `Target` recebia a posicao float do jogador e
	// `Offset` e `sw/2`, meio pixel em largura impar — o chao era amostrado
	// com um deslocamento que mudava continuamente: ele SALTAVA de texel em
	// texel enquanto arvores, monstros e o jogador deslizavam suavemente por
	// cima. As duas camadas discordavam sobre onde o mundo esta, e o olho le
	// isso como a imagem se partindo enquanto se caminha — facil de confundir
	// com screen tearing, mas nao e: tearing acontece parado tambem, e este
	// artefato so aparece em movimento.
	//
	// Arredondar as duas pontas faz o deslocamento de amostragem ficar
	// CONSTANTE. O preco e o mundo andar em passos de 1 px, que a 200 u/s e
	// imperceptivel e e o que jogo 2D faz.
	//
	// COM ZOOM, O PIXEL QUE IMPORTA E O DA TELA (27/09/2026, zoom 0,8 no
	// celular). Um ponto de mundo w cai em (w - Target)*zoom + Offset; para a
	// parte fracionaria disso nao mudar quando a camera anda, e
	// `Target*zoom` que tem de ser inteiro, nao `Target`. Por isso o alvo e
	// arredondado em ESPACO DE TELA e trazido de volta: o mundo continua
	// andando em passos de 1 px de tela (1,25 de mundo a 0,8). A zoom 1 a
	// conta e exatamente a de antes.
	c.Camera.Target.X = roundPx(c.Camera.Target.X*zoom) / zoom
	c.Camera.Target.Y = roundPx(c.Camera.Target.Y*zoom) / zoom
	c.Camera.Offset.X = roundPx(c.Camera.Offset.X)
	c.Camera.Offset.Y = roundPx(c.Camera.Offset.Y)
}

// effectiveZoom e o zoom deste quadro: o da plataforma, mas nunca afastado a
// ponto de mostrar ALEM do mapa.
//
// O mapa 6 e um corredor de 2560 px de largura. Um celular de 2400 px a 0,8
// enxerga 3000 — o corredor inteiro e mais 220 px de preto de cada lado. Em
// vez disso a camera so se afasta ate o mapa encher a tela naquele eixo
// (2400/2560 = 0,94 ali), e em todo mapa maior que a tela o 0,8 vale inteiro.
//
// So LIMITA o afastamento, nunca aproxima alem de 1: a zoom 1 (PC) nada muda,
// e um mapa menor que um monitor grande continua centrado como sempre foi.
func effectiveZoom(base, sw, sh float32, bounds world.Bounds) float32 {
	if base <= 0 {
		base = 1
	}
	if base >= 1 || bounds.Width <= 0 || bounds.Height <= 0 {
		return base
	}
	fit := sw / bounds.Width
	if fy := sh / bounds.Height; fy > fit {
		fit = fy
	}
	if fit <= base {
		return base
	}
	if fit > 1 {
		return 1
	}
	return fit
}

// roundPx arredonda para o pixel mais proximo. Existe separado porque as
// quatro chamadas acima tem de usar exatamente a mesma regra: arredondar alvo
// e offset de formas diferentes reintroduz a fracao que a ancoragem veio tirar.
func roundPx(v float32) float32 { return float32(math.Round(float64(v))) }

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
