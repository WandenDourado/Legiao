package tilemap

import "testing"

// A pilha diz o que fica embaixo do que; terrainOverlays diz a ordem em que as
// camadas sao efetivamente desenhadas. As duas tem de concordar, e nada no
// codigo obrigava isso.
//
// Quando discordaram, o custo foi um mapa inteiro: os materiais do castelo
// estavam listados como blocks, water, carpet, stone, mas a pilha e stone,
// blocks, water, carpet. Como paintedWith da a uma celula TODOS os materiais
// abaixo do proprio posto, uma celula de tapete tambem pinta pedra - e a pedra,
// desenhada por ultimo, cobria o tapete, a agua e os blocos. O mapa 4 inteiro
// aparecia como chao de pedra nu.
//
// O defeito ficou dormente enquanto o world_04 nao pintava nenhuma celula de
// pedra: a ordem ja estava errada e nao dava sintoma. Por isso a verificacao e
// das TABELAS e nao de um mapa - ela falha no momento em que a ordem quebra, e
// nao no dia em que alguem usa o material que revela.
func TestOverlayOrderMatchesStackOrder(t *testing.T) {
	position := make(map[int]int, len(terrainOverlays))
	for i, material := range terrainOverlays {
		if _, dup := position[material]; dup {
			t.Fatalf("material %d aparece duas vezes em terrainOverlays", material)
		}
		position[material] = i
	}

	for s, stack := range terrainStacks {
		previous := -1
		for rank, material := range stack.materials {
			at, ok := position[material]
			if !ok {
				// Material de pilha que nunca e desenhado: a grama base e o
				// fundo de tudo e nao entra em terrainOverlays.
				if material == terrainGrass {
					continue
				}
				t.Errorf("pilha %d: material %d (posto %d) nao esta em terrainOverlays",
					s, material, rank)
				continue
			}
			if at <= previous {
				t.Errorf("pilha %d: material %d (posto %d) e desenhado na posicao %d, "+
					"antes ou junto do que deveria estar abaixo dele — "+
					"o de baixo vai cobrir o de cima", s, material, rank, at)
			}
			previous = at
		}
	}
}

// Todo material desenhavel precisa de textura, senao ele simplesmente nao
// aparece e o mapa fica com buraco sem erro nenhum.
func TestEveryOverlayHasATexture(t *testing.T) {
	for _, material := range terrainOverlays {
		if terrainTextureFiles[material] == "" {
			t.Errorf("material %d esta em terrainOverlays e nao tem textura", material)
		}
	}
}
