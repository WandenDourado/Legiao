package tilemap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A troca de peca so e segura porque a variante em ruinas casa com a de pe em
// DUAS coisas: mesmo footprint de colisao e mesma linha de chao sob a ancora.
// Se alguem redesenhar a peca destruida com um plinto mais estreito e remedir
// o footprint, o jogo nao falha — o canhao cai e a parede invisivel muda de
// tamanho no meio da fase, o que e pior do que falhar.
func TestWreckedVariantKeepsFootprintAndGroundLine(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "castle_cannons_manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sem castle_cannons_manifest.json (%v)", err)
	}
	var m assetManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("castle_cannons_manifest.json ilegivel: %v", err)
	}

	checked := 0
	for name, intact := range m.Pieces {
		wrecked, ok := m.Pieces[name+wreckedSuffix]
		if !ok {
			continue
		}
		checked++
		if intact.CollisionFootprint != wrecked.CollisionFootprint {
			t.Errorf("%s: footprint da versao em ruinas difere da de pe (%+v vs %+v)",
				name, intact.CollisionFootprint, wrecked.CollisionFootprint)
		}
		// A ancora Y e a linha do chao: nas duas ela tem de estar na base da
		// arte, senao o plinto sobe ou afunda no instante da troca.
		if intact.Anchor.Y != intact.Source.Height || wrecked.Anchor.Y != wrecked.Source.Height {
			t.Errorf("%s: ancora fora da linha do chao (de pe %v/%v, ruinas %v/%v)",
				name, intact.Anchor.Y, intact.Source.Height, wrecked.Anchor.Y, wrecked.Source.Height)
		}
	}
	if checked == 0 {
		t.Fatal("nenhum par de pe/em-ruinas encontrado; a troca de peca ficou sem uso")
	}
}

// A troca e por POSICAO e nao por nome, e a tolerancia e de meia celula. Este
// teste guarda os dois lados disso: o posto certo troca, o vizinho a uma
// celula de distancia nao.
func TestPieceNameAtSwapsOnlyTheWreckedSpot(t *testing.T) {
	ClearWreckedPieces()
	defer ClearWreckedPieces()

	r := &ManifestRenderer{manifest: assetManifest{Pieces: map[string]manifestPiece{
		"cannon_west":         {},
		"cannon_west_wrecked": {},
		"statue":              {},
	}}}

	if got := r.pieceNameAt("cannon_west", 896, 768); got != "cannon_west" {
		t.Fatalf("sem posto destruido, esperava a peca de pe; veio %q", got)
	}

	WreckPieceAt(896, 768)
	if got := r.pieceNameAt("cannon_west", 896, 768); got != "cannon_west_wrecked" {
		t.Errorf("o posto caiu e a peca nao trocou; veio %q", got)
	}
	// Uma celula inteira ao lado nao e o mesmo posto.
	if got := r.pieceNameAt("cannon_west", 896+128, 768); got != "cannon_west" {
		t.Errorf("trocou a peca de um objeto a uma celula de distancia; veio %q", got)
	}
	// Peca sem variante em ruinas continua desenhando ela mesma, ainda que
	// esteja em cima do posto: um mapa sem a variante desenha a peca inteira,
	// e nao um buraco.
	if got := r.pieceNameAt("statue", 896, 768); got != "statue" {
		t.Errorf("peca sem variante em ruinas foi trocada; veio %q", got)
	}
}
