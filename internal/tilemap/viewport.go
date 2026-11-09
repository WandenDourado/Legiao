package tilemap

// O que a camera realmente mostra.
//
// Ate existir isto, todo passe de desenho percorria o MAPA INTEIRO a cada
// quadro: no world_03 eram 23.044 quads de terreno para mostrar as ~127
// celulas que cabem numa tela de 1080p, e 18.844 deles com troca de shader
// por celula. Nenhum passe consultava a camera.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// viewMarginCells e quanto o viewport cresce alem da tela, em celulas.
//
// Uma celula basta para o quad que atravessa a borda: o desenho de uma celula
// nunca sai da propria celula, e a mascara de borda le a camada `ground`
// direto do mapa, nao o que ja foi desenhado - entao cortar vizinho invisivel
// nao muda um pixel do resultado. Peca de manifesto NAO usa esta margem: ela
// e ancorada longe do proprio desenho (uma arvore ancora no tronco e desenha a
// copa bem acima), entao e testada pelo retangulo desenhado, que e exato.
const viewMarginCells = 1

// Viewport e o retangulo de mundo visivel, ja com a margem, mais o intervalo
// de celulas que ele cobre.
type Viewport struct {
	// World e o retangulo visivel em unidades de mundo.
	World rl.Rectangle
	// MinCellX..MaxCellY e o mesmo retangulo em celulas, inclusivo. Derivado
	// uma vez aqui para os lacos por celula fazerem comparacao de inteiro em
	// vez de remontar um retangulo por celula.
	MinCellX, MinCellY, MaxCellX, MaxCellY int
	// unbounded transforma todo teste em "visivel". E o que MapRenderer.Draw
	// usa: quem nao tem camera nao pode cullar, e desenhar tudo e exatamente o
	// comportamento que existia antes do culling.
	unbounded bool
}

// NewViewport converte a camera no retangulo que ela mostra.
//
// A conversao e a inversa da que o raylib aplica: tela = (mundo - Target) *
// Zoom + Offset, entao o canto superior esquerdo do mundo visivel e
// Target - Offset/Zoom, e a tela mede screen/Zoom em unidades de mundo.
func NewViewport(cam rl.Camera2D, screenW, screenH float32, tileW, tileH int) Viewport {
	zoom := cam.Zoom
	if zoom <= 0 {
		// Camera com zoom zero nao mostra nada, e "nada" aqui seria um mapa
		// invisivel em vez de um erro visivel. Desenhar tudo e a falha segura.
		return EverythingVisible()
	}
	if tileW <= 0 || tileH <= 0 {
		return EverythingVisible()
	}

	marginX := float32(viewMarginCells * tileW)
	marginY := float32(viewMarginCells * tileH)
	world := rl.NewRectangle(
		cam.Target.X-cam.Offset.X/zoom-marginX,
		cam.Target.Y-cam.Offset.Y/zoom-marginY,
		screenW/zoom+2*marginX,
		screenH/zoom+2*marginY,
	)

	return Viewport{
		World:    world,
		MinCellX: cellIndex(world.X, tileW),
		MinCellY: cellIndex(world.Y, tileH),
		MaxCellX: cellIndex(world.X+world.Width, tileW),
		MaxCellY: cellIndex(world.Y+world.Height, tileH),
	}
}

// EverythingVisible e o viewport que nao corta nada.
func EverythingVisible() Viewport {
	return Viewport{unbounded: true}
}

// CellRange intersecta o viewport com os limites da camada e devolve o
// intervalo de celulas a percorrer, ja limitado aos dois.
//
// Devolve o intervalo em vez de um teste por celula de proposito: um laco que
// ja comeca e termina no lugar certo nao paga uma chamada por celula do mapa
// inteiro so para descobrir que 97% delas estao fora.
func (v Viewport) CellRange(width, height int) (minX, minY, maxX, maxY int) {
	minX, minY = 0, 0
	maxX, maxY = width-1, height-1
	if v.unbounded {
		return minX, minY, maxX, maxY
	}
	if v.MinCellX > minX {
		minX = v.MinCellX
	}
	if v.MinCellY > minY {
		minY = v.MinCellY
	}
	if v.MaxCellX < maxX {
		maxX = v.MaxCellX
	}
	if v.MaxCellY < maxY {
		maxY = v.MaxCellY
	}
	return minX, minY, maxX, maxY
}

// Intersects diz se um retangulo de mundo aparece na tela. E o teste das pecas
// de manifesto e dos quads de trilha, que nao moram numa grade.
func (v Viewport) Intersects(r rl.Rectangle) bool {
	if v.unbounded {
		return true
	}
	return r.X < v.World.X+v.World.Width &&
		r.X+r.Width > v.World.X &&
		r.Y < v.World.Y+v.World.Height &&
		r.Y+r.Height > v.World.Y
}

// cellIndex e a celula em que uma coordenada de mundo cai. Usa floor e nao
// truncamento porque a origem do mapa e a unica coisa que um chamador futuro
// pode mover, e truncar devolveria a celula errada do lado negativo.
func cellIndex(v float32, size int) int {
	return int(math.Floor(float64(v) / float64(size)))
}
