package game

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/world"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Camera2DState manages the rl.Camera2D that follows the player.
type Camera2DState struct {
	Camera rl.Camera2D
}

// NewCamera creates a Camera2DState with default values.
func NewCamera(sw, sh float32) Camera2DState {
	return Camera2DState{
		Camera: rl.Camera2D{
			Offset:   rl.NewVector2(sw/2, sh/2),
			Target:   rl.NewVector2(0, 0),
			Rotation: 0,
			Zoom:     1.0,
		},
	}
}

// Update moves the camera directly to the target position,
// then clamps the result to world bounds so the viewport never shows outside the map.
func (c *Camera2DState) Update(target rl.Vector2, sw, sh float32, bounds world.Bounds) {
	c.Camera.Offset = rl.NewVector2(sw/2, sh/2)
	c.Camera.Zoom = 1.0

	// Set camera target directly to player position
	c.Camera.Target.X = target.X
	c.Camera.Target.Y = target.Y

	// Clamp to world bounds, handling maps smaller than the screen
	halfW := sw / 2
	halfH := sh / 2
	if bounds.Width < sw {
		// Map narrower than screen – keep camera centered horizontally
		c.Camera.Target.X = bounds.Width / 2
	} else {
		c.Camera.Target.X = clamp(c.Camera.Target.X, halfW, bounds.Width-halfW)
	}
	if bounds.Height < sh {
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
	c.Camera.Target.X = roundPx(c.Camera.Target.X)
	c.Camera.Target.Y = roundPx(c.Camera.Target.Y)
	c.Camera.Offset.X = roundPx(c.Camera.Offset.X)
	c.Camera.Offset.Y = roundPx(c.Camera.Offset.Y)
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
