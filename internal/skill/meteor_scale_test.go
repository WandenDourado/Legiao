package skill

// A Chuva de Meteoros mede-se por DENSIDADE (nascimentos por segundo por
// unidade de area do mapa), nao por contagem fixa — ver o comentario de
// MeteorRainIntervalFor em meteor.go e "A chuva mede-se por densidade, nao
// por contagem" em doc/combat_rules.md. Estes testes defendem essa relacao,
// nao um numero escolhido: eles quebram em voz alta se alguem reintroduzir
// um intervalo fixo, um piso de densidade, ou inverter o sinal do
// reescalonamento.

import "testing"

// world_01..world_07 sao os sete mapas da campanha (assets/maps/world_0N.json,
// 128px de tile), na ordem de game.campaignMaps. worldXXArea replica o calculo
// de world.NewBoundsFromMap: largura da grade x altura da grade x 128 x 128.
const (
	world01W, world01H float32 = 60 * 128, 45 * 128
	world02W, world02H float32 = 60 * 128, 50 * 128
	world03W, world03H float32 = 60 * 128, 70 * 128
	world04W, world04H float32 = 64 * 128, 72 * 128
	world05W, world05H float32 = 64 * 128, 90 * 128
	world06W, world06H float32 = 20 * 128, 56 * 128
	world07W, world07H float32 = 26 * 128, 20 * 128
)

// O WORLD_05 E O MAPA DE REFERENCIA: a refatoracao para densidade constante
// nao pode mudar o que ja esta calibrado la (27/08/2026). Desde o reforco de
// 27/09/2026 (MeteorRainDensityBoost) isso vale porque o world_05 esta no teto
// de desempenho, que e o proprio MeteorRainRefInterval: o reforco pediria
// um intervalo menor, e o teto o segura.
func TestMeteorRainKeepsWorld05Untouched(t *testing.T) {
	got := MeteorRainIntervalFor(world05W, world05H)
	want := MeteorRainRefInterval
	if diff := got - want; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("MeteorRainIntervalFor(world_05) = %.6f; esperado o proprio "+
			"MeteorRainRefInterval (%.6f) sem nenhum reescalonamento", got, want)
	}
}

// A DENSIDADE E CONSTANTE ENTRE MAPAS — a relacao que este plano inteiro
// defende. Nao compara os intervalos (que variam por design); compara
// nascimentos-por-segundo dividido pela area, que tem de ser o mesmo numero
// em qualquer mapa da campanha (dentro do teto de desempenho).
func TestMeteorRainDensityIsMapIndependent(t *testing.T) {
	maps := []struct {
		name string
		w, h float32
	}{
		{"world_01", world01W, world01H},
		{"world_02", world02W, world02H},
		{"world_03", world03W, world03H},
		{"world_04", world04W, world04H},
		{"world_05", world05W, world05H},
	}
	// world_06 e world_07 ficam de fora desta comparacao: sao menores que o
	// world_05 e batem no MeteorRainMinInterval? Nao — sao MAIORES em
	// densidade relativa (mapas menores), entao NAO batem no teto (o teto so
	// trava mapas maiores que a referencia). Ainda assim eles tem sua propria
	// checagem de monotonicidade em TestSmallerMapGetsFewerMeteors.
	// A referencia inclui o reforco de 27/09/2026 (MeteorRainDensityBoost).
	// O world_05 sai da comparacao: ele e o unico no teto de desempenho (ver
	// TestMeteorRainKeepsWorld05Untouched), entao o reforco nao o alcanca.
	refDensity := 1.0 / MeteorRainRefInterval / MeteorRainRefArea * MeteorRainDensityBoost
	for _, mp := range maps {
		if mp.name == "world_05" {
			continue
		}
		iv := MeteorRainIntervalFor(mp.w, mp.h)
		area := mp.w * mp.h
		density := 1.0 / iv / area
		diff := (density - refDensity) / refDensity
		if diff > 0.01 || diff < -0.01 {
			t.Errorf("%s: densidade %.10f, referencia (world_05) %.10f — "+
				"divergem mais de 1%%; a chuva parou de escalar por area",
				mp.name, density, refDensity)
		}
	}
}

// O REFORCO DE 27/09/2026 CHEGA A ARENA. O Mago so tem a Chuva no world_06 e
// no world_07 (game.UltimatesGrantedOn); o pedido foi por ELES. Nos dois a
// chuva tem de nascer MeteorRainDensityBoost vezes mais rapido que a
// densidade calibrada pediria sem reforco.
func TestMeteorRainBoostReachesTheMagoMaps(t *testing.T) {
	for _, mp := range []struct {
		name string
		w, h float32
	}{{"world_06", world06W, world06H}, {"world_07", world07W, world07H}} {
		plain := MeteorRainRefInterval * (MeteorRainRefArea / (mp.w * mp.h))
		got := MeteorRainIntervalFor(mp.w, mp.h)
		ratio := plain / got
		if diff := ratio - MeteorRainDensityBoost; diff > 0.0001 || diff < -0.0001 {
			t.Errorf("%s: a chuva nasce %.4fx mais rapido que a densidade de "+
				"referencia; esperado %.4fx (MeteorRainDensityBoost)",
				mp.name, ratio, MeteorRainDensityBoost)
		}
	}
}

// MAPA MENOR RECEBE MENOS METEOROS: monotonicidade da relacao area->intervalo.
// Pega uma inversao de sinal no reescalonamento (por exemplo, multiplicar por
// area/RefArea em vez de RefArea/area).
func TestSmallerMapGetsFewerMeteors(t *testing.T) {
	maps := []struct {
		name string
		w, h float32
	}{
		{"world_07", world07W, world07H}, // menor mapa do jogo
		{"world_06", world06W, world06H},
		{"world_01", world01W, world01H},
		{"world_02", world02W, world02H},
		{"world_03", world03W, world03H},
		{"world_04", world04W, world04H},
		{"world_05", world05W, world05H}, // referencia, o maior
	}
	prevArea := float32(0)
	prevInterval := float32(0)
	for i, mp := range maps {
		area := mp.w * mp.h
		iv := MeteorRainIntervalFor(mp.w, mp.h)
		if i > 0 {
			if area <= prevArea {
				t.Fatalf("tabela de mapas fora de ordem: %s (%.0f) nao e maior que o anterior (%.0f)",
					mp.name, area, prevArea)
			}
			if iv > prevInterval {
				t.Errorf("%s tem area maior (%.0f > %.0f) mas intervalo TAMBEM maior "+
					"(%.4f > %.4f); mapa maior deveria nascer meteoros MAIS rapido, "+
					"nao mais devagar", mp.name, area, prevArea, iv, prevInterval)
			}
		}
		prevArea = area
		prevInterval = iv
	}
}

// O TETO DE DESEMPENHO SEGURA MAPAS MAIORES QUE O WORLD_05. Densidade
// constante pediria um intervalo menor que 0,015 para um mapa maior que a
// referencia; MeteorRainMinInterval e o regime medido (doc/performance.md
// §8) que barra isso.
func TestMeteorRainRespectsThePerformanceCeiling(t *testing.T) {
	bigW, bigH := world05W*2, world05H*2 // 4x a area do world_05
	got := MeteorRainIntervalFor(bigW, bigH)
	if got < MeteorRainMinInterval {
		t.Errorf("mapa 4x maior que o world_05 produziu intervalo %.6f, abaixo "+
			"do teto de desempenho %.6f", got, MeteorRainMinInterval)
	}
	if got != MeteorRainMinInterval {
		t.Errorf("mapa maior que o world_05 deveria travar EXATAMENTE no teto "+
			"(%.6f); obteve %.6f", MeteorRainMinInterval, got)
	}
}

// MUNDO DEGENERADO NAO DIVIDE POR ZERO. (0, 0) nao e um mapa real, mas
// devolver o intervalo de referencia em vez de +Inf/NaN evita travar a chuva
// silenciosamente se algo passar bounds vazios.
func TestMeteorRainSurvivesADegenerateWorld(t *testing.T) {
	got := MeteorRainIntervalFor(0, 0)
	if got != MeteorRainRefInterval {
		t.Errorf("mundo degenerado (0,0) devolveu %.6f; esperado o intervalo de "+
			"referencia (%.6f), nao uma divisao por zero", got, MeteorRainRefInterval)
	}
}
