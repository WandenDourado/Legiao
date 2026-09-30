package tilemap

import "testing"

// A oclusao de terreno (TerrainRenderer.hidden) e a otimizacao mais perigosa do
// pacote, porque ela nao muda o que e desenhado — ela decide o que NAO e. Um
// erro aqui nao aparece como excecao nem como borda estranha: aparece como
// buraco no chao, e so em algumas celulas de alguns mapas.
//
// Por isso estes testes verificam a REGRA, e nao o resultado: a unica condicao
// que autoriza pular uma camada e "os oito vizinhos carregam o mesmo material",
// que e o que faz o shader devolver alfa 1,0 exato.

// layerOf monta uma camada `ground` a partir de linhas de materiais, na ordem
// em que aparecem escritas — a primeira linha e o norte do mapa.
func layerOf(rows ...[]int) Layer {
	if len(rows) == 0 {
		return Layer{Type: "tilelayer", Name: "ground"}
	}
	w := len(rows[0])
	data := make([]int, 0, w*len(rows))
	for _, row := range rows {
		if len(row) != w {
			panic("layerOf: linhas de larguras diferentes")
		}
		data = append(data, row...)
	}
	return Layer{Type: "tilelayer", Name: "ground", Visible: true,
		Width: w, Height: len(rows), Data: data}
}

func TestFullyLinkedNeedsAllEightNeighbours(t *testing.T) {
	// Um campo de grama com UMA celula de pedra no meio. O centro do campo
	// (1,1) toca a pedra na diagonal, entao a grama dele nao esta cercada de
	// grama; (1,3) esta longe o bastante e esta.
	layer := layerOf(
		[]int{terrainGrass, terrainGrass, terrainGrass, terrainGrass, terrainGrass},
		[]int{terrainGrass, terrainGrass, terrainGrass, terrainGrass, terrainGrass},
		[]int{terrainGrass, terrainGrass, terrainGrass, terrainGrass, terrainGrass},
		[]int{terrainGrass, terrainGrass, terrainGrass, terrainGrass, terrainGrass},
		[]int{terrainGrass, terrainGrass, terrainGrass, terrainGrass, terrainGrass},
	)
	if !fullyLinked(layer, 2, 2, terrainGrass) {
		t.Fatal("celula no meio de um campo uniforme deveria estar cercada")
	}
	// Fora do mapa conta como ligado (mesma regra de `linked`), entao o canto
	// de um mapa uniforme tambem esta cercado — se nao estivesse, a orla de
	// todo mapa desenharia camadas invisiveis para sempre.
	if !fullyLinked(layer, 0, 0, terrainGrass) {
		t.Fatal("canto de mapa uniforme deveria contar como cercado")
	}

	// A pedra e RANK MAIS ALTO que a grama na mesma pilha, entao ela "carrega"
	// grama: paintedWith(pedra, grama) e verdadeiro, e a vizinhanca continua
	// ligada para a GRAMA. O que quebra o cerco e o contrario — a grama nao
	// carrega pedra.
	withStone := layerOf(
		[]int{terrainGrass, terrainGrass, terrainGrass},
		[]int{terrainGrass, terrainStone, terrainGrass},
		[]int{terrainGrass, terrainGrass, terrainGrass},
	)
	if !fullyLinked(withStone, 1, 1, terrainGrass) {
		t.Fatal("vizinho de rank mais alto conta como ligado para o material de baixo")
	}
	if fullyLinked(withStone, 1, 1, terrainStone) {
		t.Fatal("pedra isolada em campo de grama nao pode estar cercada de pedra")
	}
}

// TestHiddenOnlyEverHidesWhatAnOpaqueCellCovers e o teste central: varre uma
// camada com as tres pilhas e todos os materiais de overlay, e exige que TODA
// vez que `hidden` disser para pular um quad, a celula esteja de fato cercada
// pelo proprio material — a verdade calculada aqui de forma independente, sem
// consultar `covered`.
func TestHiddenOnlyEverHidesWhatAnOpaqueCellCovers(t *testing.T) {
	layer := layerOf(
		[]int{terrainGrass, terrainDirt, terrainStone, terrainStone, terrainDirt},
		[]int{terrainDirt, terrainStone, terrainStone, terrainStone, terrainGrass},
		[]int{terrainForestGrass, terrainDarkGrass, terrainDarkGrass, terrainBareSoil, terrainForestPath},
		[]int{terrainDarkGrass, terrainDarkGrass, terrainDarkGrass, terrainSiegeGravel, terrainDarkFlagstone},
		[]int{terrainCastleStone, terrainCastleBlocks, terrainCastleWater, terrainCastleCarpet, terrainCastleStone},
	)
	r := &TerrainRenderer{}
	r.prepareCoverage(layer)

	for y := 0; y < layer.Height; y++ {
		for x := 0; x < layer.Width; x++ {
			kind := layer.Data[y*layer.Width+x]
			for _, material := range append([]int{terrainGrass}, terrainOverlays...) {
				if !r.hidden(x, y, kind, material) {
					continue
				}
				if material == kind {
					t.Fatalf("(%d,%d): escondeu o material PROPRIO da celula (%d) — e ele que e o quad opaco", x, y, kind)
				}
				if !paintedWith(kind, material) {
					t.Fatalf("(%d,%d): escondeu %d, que esta celula nem pintaria", x, y, material)
				}
				if !fullyLinked(layer, x, y, kind) {
					t.Fatalf("(%d,%d): escondeu %d sob um material (%d) que NAO esta cercado — a camada de cima tem borda desbotada e deixaria ver o buraco", x, y, material, kind)
				}
			}
		}
	}
}

// TestHiddenIsInertWithoutCoverage garante a falha segura: um renderer que
// nunca teve PrepareMasks chamado (mapa sem camada `ground`, ou um teste que
// desenha direto) desenha TUDO, como antes da oclusao existir.
func TestHiddenIsInertWithoutCoverage(t *testing.T) {
	r := &TerrainRenderer{}
	if r.hidden(0, 0, terrainStone, terrainGrass) {
		t.Fatal("sem cobertura calculada, nada pode ser escondido")
	}
	// E uma celula fora da grade para a qual a cobertura foi construida
	// tambem nao pode ser escondida.
	r.prepareCoverage(layerOf([]int{terrainGrass, terrainGrass}, []int{terrainGrass, terrainGrass}))
	if r.hidden(9, 9, terrainStone, terrainGrass) {
		t.Fatal("celula fora da grade da cobertura nao pode ser escondida")
	}
}

// TestCoverageIgnoresEmptyAndUnknownCells: celula vazia (gid 0) e material que
// nenhuma pilha declara nunca sao opacos, entao nunca escondem nada.
func TestCoverageIgnoresEmptyAndUnknownCells(t *testing.T) {
	const unknownMaterial = 99
	layer := layerOf(
		[]int{terrainGrass, terrainGrass, terrainGrass},
		[]int{terrainGrass, 0, terrainGrass},
		[]int{terrainGrass, unknownMaterial, terrainGrass},
	)
	r := &TerrainRenderer{}
	r.prepareCoverage(layer)
	if r.covered[1*layer.Width+1] {
		t.Fatal("celula vazia nao desenha quad nenhum, logo nao esconde nada")
	}
	if r.covered[2*layer.Width+1] {
		t.Fatal("material fora de toda pilha nunca e pintado, logo nao esconde nada")
	}
}
