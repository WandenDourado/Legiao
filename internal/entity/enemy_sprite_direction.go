package entity

import "math"

// Facing for directional enemy sheets.
//
// This is the enemy counterpart of player_sprite_direction.go, and it differs
// in one way that matters: a player sheet has five rows for eight directions,
// while an orc sheet has nine rows for sixteen. The extra angular resolution is
// what a rigid, armoured body needs — five rows on a humanoid this size makes
// the turn read as a snap.

// enemyDirections is how many facings a directional sheet resolves.
const enemyDirections = 16

// enemyDirectionArc is the angle each facing covers.
const enemyDirectionArc = 360.0 / enemyDirections

// EnemyDirectionRows is how many rows a directional sheet stores: the mirror
// axis plus everything on one side of it. Sixteen facings fold onto nine rows
// because the vendor's art is symmetric about the vertical axis — row 0 (away
// from the camera) and row 8 (toward it) are their own mirrors, and the seven
// between them each cover two facings.
const EnemyDirectionRows = enemyDirections/2 + 1

// A CONVENCAO DO PACOTE, e como ela foi estabelecida.
//
// Isto ja esteve errado duas vezes, e as duas vezes o erro passou por uma
// conferencia visual. Entao a evidencia fica escrita:
//
//	dir 000 — nuca. Ombros e costas da cabeca, NENHUM rosto. Anda para longe
//	          da camera, ou seja, para CIMA na tela.
//	dir 090 — perfil com o olho e a focinheira apontando para a DIREITA do
//	          quadro. Anda para a DIREITA.
//	dir 180 — rosto. Olhos e mandibula virados para a camera. Anda para BAIXO.
//	dir 270 — perfil espelhado do 090: olho e focinheira para a ESQUERDA.
//
// Logo o angulo comeca no norte e cresce em sentido HORARIO na tela.
//
// A melhor leitura e a do `Roar`, onde a cabeca levanta e o focinho sai da
// sombra dos ombros; nas poses curvadas de `Idle` e `Walk` o rosto fica
// escondido e as costas musculosas passam por peitoral com facilidade. Se
// alguem precisar reconferir, e essa animacao que decide:
//
//	python3 work/orc-guarnicao/verify_facing.py --anchor
//
// O erro anterior era `atan2(-vx, ...)`, que gira ao contrario. Ele nao produz
// um monstro obviamente quebrado: produz um que encara o reflexo do alvo no
// eixo vertical, o que num corpo curvado parece "de costas" e nao "espelhado".
//
// enemyRowForHeading maps a velocity onto a sheet row and whether that row has
// to be drawn mirrored.
//
// Returns ok=false when the enemy is effectively still, so the caller keeps the
// facing it already had instead of snapping to "away from the camera" every
// time the monster stops.
func enemyRowForHeading(vx, vy float32) (row int, mirror bool, ok bool) {
	if vx*vx+vy*vy < 1e-4 {
		return 0, false, false
	}

	// Zero para cima, crescendo no sentido horario da tela. Y cresce para
	// baixo, entao e -vy que aponta para o norte; X NAO e negado, e e
	// exatamente esse sinal que decide o sentido do giro.
	deg := math.Atan2(float64(vx), float64(-vy)) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}

	index := int(math.Round(deg/enemyDirectionArc)) % enemyDirections
	return enemyRowForIndex(index)
}

// enemyRowForIndex folds one of the sixteen facings onto a stored row.
//
// Facings 0..8 are stored as-is — they are the half that sweeps north, through
// EAST, to south. Facings 9..15 are the western half, which is the mirror of
// 7..1, so they reuse those rows with the horizontal flip. The fold is exact
// only because the sheet's crop window is symmetric about the pivot —
// build_orc.py forces that, and without it a mirrored facing would be drawn
// offset from its own twin.
func enemyRowForIndex(index int) (row int, mirror bool, ok bool) {
	index = ((index % enemyDirections) + enemyDirections) % enemyDirections
	if index <= enemyDirections/2 {
		return index, false, true
	}
	return enemyDirections - index, true, true
}

// validEnemyRow keeps a row inside a sheet that may store fewer rows than the
// fold produces, so a half-installed sheet draws something instead of sampling
// past its own bottom edge.
func validEnemyRow(row, rows int) int {
	if rows <= 0 {
		return 0
	}
	if row < 0 || row >= rows {
		return 0
	}
	return row
}
