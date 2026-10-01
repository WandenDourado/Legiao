package skill

// A geometria do estouro da bola de canhao, do lado do desenho.
//
// O que este arquivo NAO guarda mais: a igualdade entre o anel desenhado e a
// area do dano. Elas foram amarradas em 27/08/2026 e desamarradas em 28/08 a
// pedido do Gui — o anel expande ate ~4,6x o raio da bola como ONDA DE CHOQUE,
// bem alem dos 630 px que machucam, e ele leu bem assim em jogo. Anel de
// choque maior que a zona letal e convencao, nao promessa quebrada.
//
// O defeito que pareceu ser disso era outro: a bola detonava 664 px antes de
// encostar em alguem. Ver network.cannonBallImpactPoint.

import "testing"

// O estouro tem de passar do corpo da bola, senao encostar na bola de fogo e
// nao sangrar volta a ser possivel.
func TestBlastRadiusIsWiderThanTheBallItself(t *testing.T) {
	if CannonBlastRadius <= CannonBallRadius {
		t.Errorf("o estouro (%.0f) nao passa do corpo da bola (%.0f)",
			CannonBlastRadius, CannonBallRadius)
	}
}
