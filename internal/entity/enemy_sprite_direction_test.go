package entity

import "testing"

// A tabela abaixo e a unica coisa que garante que o orc olha para onde anda.
//
// Ela vale a pena porque o erro provavel aqui e silencioso: uma troca de sinal
// no atan2 nao quebra nada, nao gera panico e nao aparece em nenhum log. Ela
// so faz o monstro andar de costas, e isso se descobre olhando a tela — depois
// de a IA, a rede e o ataque ja estarem construidos em cima do defeito.
//
// As direcoes esperadas vem da convencao do fornecedor: o angulo comeca nas
// COSTAS do personagem e cresce em sentido HORARIO na tela. 000 anda para longe
// da camera, 090 anda para a DIREITA, 180 anda para a camera, 270 anda para a
// ESQUERDA. A evidencia de cada um desses quatro esta escrita em
// enemy_sprite_direction.go; a leitura decisiva e o focinho no `Roar`.
//
// Esta tabela ja esteve espelhada, e o sintoma nao era um monstro visivelmente
// invertido: era um monstro que parecia perseguir de costas.
func TestEnemyRowForHeading(t *testing.T) {
	tests := []struct {
		name       string
		vx, vy     float32
		wantRow    int
		wantMirror bool
	}{
		// Os quatro eixos.
		{"norte: de costas para a camera", 0, -1, 0, false},
		{"leste: perfil com o focinho para a direita", 1, 0, 4, false},
		{"sul: de frente para a camera", 0, 1, 8, false},
		{"oeste: perfil esquerdo, espelhado do leste", -1, 0, 4, true},

		// As quatro diagonais.
		{"nordeste", 1, -1, 2, false},
		{"sudeste", 1, 1, 6, false},
		{"sudoeste: espelho do sudeste", -1, 1, 6, true},
		{"noroeste: espelho do nordeste", -1, -1, 2, true},

		// Os passos intermediarios de 22,5 graus, que sao a razao de existirem
		// nove linhas em vez das cinco que um personagem jogavel usa.
		{"nor-nordeste", 0.3827, -0.9239, 1, false},
		{"les-nordeste", 0.9239, -0.3827, 3, false},
		{"les-sudeste", 0.9239, 0.3827, 5, false},
		{"sul-sudeste", 0.3827, 0.9239, 7, false},
		{"sul-sudoeste", -0.3827, 0.9239, 7, true},
		{"oes-sudoeste", -0.9239, 0.3827, 5, true},
		{"oes-noroeste", -0.9239, -0.3827, 3, true},
		{"nor-noroeste", -0.3827, -0.9239, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, mirror, ok := enemyRowForHeading(tt.vx, tt.vy)
			if !ok {
				t.Fatalf("velocidade (%.4f, %.4f) foi tratada como parado", tt.vx, tt.vy)
			}
			if row != tt.wantRow || mirror != tt.wantMirror {
				t.Errorf("(%.4f, %.4f) = linha %d espelho %v; queria linha %d espelho %v",
					tt.vx, tt.vy, row, mirror, tt.wantRow, tt.wantMirror)
			}
		})
	}
}

// Parado nao e uma direcao. Sem isto, o orc que para de andar viraria de costas
// para a camera no mesmo quadro, porque zero cai no primeiro setor.
func TestEnemyRowForHeadingIgnoresStillness(t *testing.T) {
	if _, _, ok := enemyRowForHeading(0, 0); ok {
		t.Error("velocidade zero devolveu uma direcao; o chamador precisa manter a anterior")
	}
	if _, _, ok := enemyRowForHeading(0.001, -0.001); ok {
		t.Error("deriva de sub-pixel devolveu uma direcao; o orc vai tremer entre linhas")
	}
}

// A dobra do espelho tem que ser exata: cada setor de um lado do eixo vertical
// e o mesmo setor do outro lado, invertido. Se esta propriedade falhar, metade
// das direcoes desenha a linha errada e a outra metade parece certa — que e o
// tipo de defeito que passa por uma inspecao visual rapida.
func TestEnemyMirrorFoldIsSymmetric(t *testing.T) {
	for index := 1; index < enemyDirections/2; index++ {
		row, mirror, ok := enemyRowForIndex(index)
		if !ok || mirror {
			t.Fatalf("setor %d devia ser guardado sem espelho", index)
		}
		twin := enemyDirections - index
		twinRow, twinMirror, ok := enemyRowForIndex(twin)
		if !ok || !twinMirror {
			t.Fatalf("setor %d devia ser espelhado", twin)
		}
		if twinRow != row {
			t.Errorf("setores %d e %d deviam dividir a linha; deram %d e %d",
				index, twin, row, twinRow)
		}
	}
}

// Os dois setores do proprio eixo do espelho nao tem par: eles sao o proprio
// espelho, entao nunca podem pedir a inversao.
func TestEnemyMirrorAxisIsNeverFlipped(t *testing.T) {
	for _, index := range []int{0, enemyDirections / 2} {
		row, mirror, ok := enemyRowForIndex(index)
		if !ok {
			t.Fatalf("setor %d nao resolveu", index)
		}
		if mirror {
			t.Errorf("setor %d (eixo do espelho) pediu inversao; linha %d", index, row)
		}
	}
}

// Toda linha que a dobra pode produzir tem que existir na folha. Este teste e
// a ponte entre a matematica e a arte: se alguem mudar o numero de direcoes
// guardadas sem regerar a folha, e aqui que se descobre.
func TestEnemyFoldStaysInsideStoredRows(t *testing.T) {
	for index := 0; index < enemyDirections; index++ {
		row, _, ok := enemyRowForIndex(index)
		if !ok {
			t.Fatalf("setor %d nao resolveu", index)
		}
		if row < 0 || row >= EnemyDirectionRows {
			t.Errorf("setor %d deu linha %d, fora das %d guardadas",
				index, row, EnemyDirectionRows)
		}
	}
}
