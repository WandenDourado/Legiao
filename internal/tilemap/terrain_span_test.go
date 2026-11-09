package tilemap

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A textura de terreno pode cobrir varias celulas, com cada celula usando a
// janela da sua posicao. O que estes testes protegem e a relacao entre as duas
// saidas de tileWindow: o retangulo em PIXELS que raylib recorta e o mesmo
// retangulo em UV que o shader recebe. Se as duas discordarem, a mascara de
// borda passa a ser calculada sobre a coordenada errada e o desbotamento entre
// materiais some — que e um defeito visual sutil, facil de nao notar num
// commit e caro de rastrear depois.

func TestTileWindowSpanOneIsTheWholeTexture(t *testing.T) {
	texture := rl.Texture2D{Width: 256, Height: 256}
	// Span 1 tem que reproduzir exatamente o que o renderer fazia antes de
	// existir span: textura inteira, e tileRect neutro para o shader.
	for _, cell := range [][2]int{{0, 0}, {3, 7}, {59, 49}} {
		src, uv := tileWindow(texture, cell[0], cell[1], 1)
		want := rl.NewRectangle(0, 0, 256, 256)
		if src != want {
			t.Errorf("celula %v: src = %v, esperado %v", cell, src, want)
		}
		for i, v := range []float32{0, 0, 1, 1} {
			if uv[i] != v {
				t.Fatalf("celula %v: tileRect = %v, esperado [0 0 1 1]", cell, uv)
			}
		}
	}
}

func TestTileWindowMatchesUV(t *testing.T) {
	const span = 4
	texture := rl.Texture2D{Width: 512, Height: 512}
	for y := 0; y < 9; y++ {
		for x := 0; x < 9; x++ {
			src, uv := tileWindow(texture, x, y, span)

			// raylib normaliza o src pela dimensao da textura; o shader tem que
			// receber exatamente essa janela, senao ele reconstroi a coordenada
			// local da celula errada.
			if got, want := uv[0], src.X/float32(texture.Width); got != want {
				t.Errorf("(%d,%d): uv.x = %v, src normalizado = %v", x, y, got, want)
			}
			if got, want := uv[1], src.Y/float32(texture.Height); got != want {
				t.Errorf("(%d,%d): uv.y = %v, src normalizado = %v", x, y, got, want)
			}
			if got, want := uv[2], src.Width/float32(texture.Width); got != want {
				t.Errorf("(%d,%d): uv.w = %v, src normalizado = %v", x, y, got, want)
			}

			// A janela nunca pode sair da folha: sair significa amostrar lixo ou
			// grudar na borda, e as duas coisas aparecem como faixa no chao.
			if src.X < 0 || src.Y < 0 ||
				src.X+src.Width > float32(texture.Width) ||
				src.Y+src.Height > float32(texture.Height) {
				t.Errorf("(%d,%d): janela %v sai da textura", x, y, src)
			}
		}
	}
}

func TestTileWindowTilesWithoutGapOrOverlap(t *testing.T) {
	const span = 4
	texture := rl.Texture2D{Width: 512, Height: 512}

	// Celulas vizinhas tem que pegar janelas vizinhas, e o conjunto de span x
	// span celulas tem que cobrir a folha inteira exatamente uma vez. E isso
	// que faz uma textura seamless continuar seamless depois de fatiada.
	seen := map[[2]float32]int{}
	for y := 0; y < span; y++ {
		for x := 0; x < span; x++ {
			src, _ := tileWindow(texture, x, y, span)
			seen[[2]float32{src.X, src.Y}]++
		}
	}
	if len(seen) != span*span {
		t.Fatalf("%d janelas distintas em %d celulas; esperado %d",
			len(seen), span*span, span*span)
	}
	for corner, count := range seen {
		if count != 1 {
			t.Errorf("janela %v usada %d vezes", corner, count)
		}
	}

	// E o padrao repete a cada span celulas, nao antes: era o ganho todo da
	// mudanca, entao vale travar.
	a, _ := tileWindow(texture, 1, 2, span)
	b, _ := tileWindow(texture, 1+span, 2+span, span)
	if a != b {
		t.Errorf("celulas a span de distancia usaram janelas diferentes: %v e %v", a, b)
	}
	c, _ := tileWindow(texture, 2, 2, span)
	if a == c {
		t.Error("celulas vizinhas usaram a mesma janela; o padrao ainda repete a cada celula")
	}
}

func TestSpanForOnlyGrowsWhileWholeCellsFit(t *testing.T) {
	renderer := &TerrainRenderer{}
	cases := []struct {
		name     string
		material int
		width    int32
		want     int
	}{
		// O bioma verde nao entra: dar span a ele dobraria o tamanho de cada
		// folha do world_01, que e mapa aprovado.
		{"grama clara fica em 1 mesmo com folha grande", terrainGrass, 512, 1},
		{"512px contra celula de 128 da 4x4", terrainDarkGrass, 512, 4},
		{"256px da 2x2", terrainSparseGrass, 256, 2},
		{"128px da 1x1", terrainBareSoil, 128, 1},
		// Arte menor que a celula nunca pode ser ampliada: span < 1 daria
		// divisao por zero na UV e uma janela maior que a folha.
		{"64px nao vira span 0", terrainBareSoil, 64, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			texture := rl.Texture2D{Width: tc.width, Height: tc.width}
			if got := renderer.spanFor(tc.material, texture, 128); got != tc.want {
				t.Errorf("spanFor = %d, esperado %d", got, tc.want)
			}
		})
	}
}
