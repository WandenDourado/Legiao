package tilemap

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Um footprint de manifesto e medido em PIXELS contra a arte. Antes ele era
// arredondado para a grade de 128 px, e era isso que fazia a colisao deixar de
// bater com o desenho: uma celula virava solida com meia celula de cobertura,
// entao uma casa bloqueava ate 64 px de chao livre alem da propria parede e o
// jogador esbarrava no nada. Estes testes travam a medida em pixels.

func TestFootprintDoesNotSpillIntoTheRestOfTheCell(t *testing.T) {
	index := newFootprintIndex(128, 128)
	// Um footprint no canto de uma celula de 128: cobre 80x80 dela.
	index.Add(rl.NewRectangle(0, 0, 80, 80))

	inside := rl.NewRectangle(10, 10, 20, 20)
	if !index.Collides(inside) {
		t.Fatalf("caixa dentro do footprint nao colidiu")
	}
	// O resto da mesma celula continua livre. Com quantizacao esta caixa
	// batia, porque a celula inteira virava solida.
	outside := rl.NewRectangle(90, 90, 20, 20)
	if index.Collides(outside) {
		t.Errorf("caixa em %v colidiu: a celula inteira foi bloqueada por um footprint de 80x80", outside)
	}
}

func TestFootprintIsFoundAcrossCellBoundaries(t *testing.T) {
	index := newFootprintIndex(128, 128)
	// Uma casa nao respeita a grade: ela comeca no meio de uma celula e
	// termina no meio de outra. O indice tem que achar o retangulo a partir
	// de qualquer celula que ele toque.
	house := rl.NewRectangle(70, 70, 308, 192)
	index.Add(house)

	for _, box := range []rl.Rectangle{
		{X: 60, Y: 100, Width: 20, Height: 20},  // celula 0,0
		{X: 200, Y: 150, Width: 20, Height: 20}, // celula 1,1
		{X: 370, Y: 250, Width: 20, Height: 20}, // celula 2,1
	} {
		if !index.Collides(box) {
			t.Errorf("caixa %v nao achou a casa %v", box, house)
		}
	}
	// Logo abaixo da parede o chao e livre, ainda que na mesma celula.
	if free := (rl.Rectangle{X: 200, Y: 270, Width: 20, Height: 20}); index.Collides(free) {
		t.Errorf("caixa %v colidiu: a colisao passa da arte por baixo", free)
	}
}

func TestPieceAcceptsBothFootprintForms(t *testing.T) {
	single := manifestPiece{Collision: true}
	single.CollisionFootprint = manifestFootprint{OffsetX: -10, OffsetY: -10, Width: 20, Height: 20}
	if got := single.Footprints(); len(got) != 1 || got[0].Width != 20 {
		t.Fatalf("forma singular: %v", got)
	}

	// Um canto de cerca e um L e um portao aberto tem vao: nenhum dos dois
	// cabe num retangulo so.
	many := manifestPiece{Collision: true, CollisionFootprints: []manifestFootprint{
		{Width: 300, Height: 48}, {Width: 46, Height: 180},
	}}
	if got := many.Footprints(); len(got) != 2 {
		t.Fatalf("forma plural: esperado 2 retangulos, veio %d", len(got))
	}

	if got := (manifestPiece{Collision: false}).Footprints(); got != nil {
		t.Errorf("peca sem colisao devolveu %v", got)
	}
}

func TestCollisionGridKeepsPaintedCellsAndFootprintsApart(t *testing.T) {
	grid := &CollisionGrid{
		Width: 4, Height: 4, TileWidth: 128, TileHeight: 128,
		solid:      make([]bool, 16),
		footprints: newFootprintIndex(128, 128),
	}
	grid.solid[1*4+1] = true // celula pintada a mao: solida por inteiro
	grid.footprints.Add(rl.NewRectangle(300, 40, 60, 40))

	// Celula pintada: qualquer ponto dela bloqueia.
	if !grid.CollidesCentered(rl.NewVector2(200, 200), 10, 10) {
		t.Errorf("celula pintada 1,1 nao bloqueou")
	}
	// Footprint: so o retangulo bloqueia, nao a celula 2,0 inteira.
	if !grid.CollidesCentered(rl.NewVector2(330, 60), 10, 10) {
		t.Errorf("footprint nao bloqueou")
	}
	if grid.CollidesCentered(rl.NewVector2(280, 110), 10, 10) {
		t.Errorf("celula 2,0 bloqueou fora do footprint")
	}
}

func TestCollisionGridCanOpenAndCloseFootprintGate(t *testing.T) {
	grid := &CollisionGrid{
		Width: 4, Height: 4, TileWidth: 128, TileHeight: 128,
		solid:      make([]bool, 16),
		footprints: newFootprintIndex(128, 128),
	}
	gate := rl.NewRectangle(128, 128, 128, 32)
	grid.footprints.Add(gate)
	center := rl.NewVector2(192, 144)
	if !grid.CollidesCentered(center, 20, 20) {
		t.Fatal("portao fechado nao bloqueou")
	}
	if !grid.SetFootprintsEnabledOverlapping(gate, false) {
		t.Fatal("abrir o portao nao alterou o footprint")
	}
	if grid.CollidesCentered(center, 20, 20) {
		t.Error("portao aberto continuou bloqueando")
	}
	if !grid.SetFootprintsEnabledOverlapping(gate, true) {
		t.Fatal("fechar o portao nao restaurou o footprint")
	}
	if !grid.CollidesCentered(center, 20, 20) {
		t.Error("portao fechado nao voltou a bloquear")
	}
}
