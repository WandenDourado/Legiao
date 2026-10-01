package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// Qual animacao um inimigo esta tocando.
//
// Hoje sao duas e a regra cabe numa comparacao. O arquivo existe separado
// porque este e o lugar onde ataque, flinch e morte vao entrar, e cada um deles
// traz uma regra que uma comparacao nao expressa: um golpe TRAVA o monstro no
// lugar ate o quadro de impacto, um flinch nao pode cancelar um golpe, e uma
// morte nao volta para idle. Misturar isso dentro de updateAnimation faria
// crescer um metodo que ja cuida de quadro e de direcao.
//
// Ver doc/plan_orc_guarnicao.md, Fase 2.

// enemyAnimFor escolhe a animacao a partir do movimento.
//
// Inimigo sem folha de caminhada devolve idle sempre, e isso e o que mantem o
// slime e o lobo intactos: eles nao declaram Anims, entao a pergunta "voce tem
// walk?" e falsa e a maquina nunca sai do lugar.
func enemyAnimFor(def EnemyDef, velocity rl.Vector2) EnemyAnim {
	if !def.HasAnim(AnimWalk) {
		return AnimIdle
	}
	if enemyIsMoving(velocity) {
		return AnimWalk
	}
	return AnimIdle
}

// enemyIsMoving compara com o quadrado do limite para nao tirar raiz por
// inimigo por quadro.
func enemyIsMoving(velocity rl.Vector2) bool {
	return velocity.X*velocity.X+velocity.Y*velocity.Y > enemyWalkThreshold*enemyWalkThreshold
}

// FacingTarget e o ponto do mundo para onde um inimigo direcional deve olhar,
// mais a resposta de se existe alguem para olhar.
//
// Existe como tipo, e nao como um par solto de argumentos, porque o par
// (posicao, existe) so tem sentido junto: passar a posicao sem o booleano faria
// a origem do mundo virar um alvo valido, e todo monstro sem alvo encararia o
// canto superior esquerdo do mapa.
type FacingTarget struct {
	Position rl.Vector2
	OK       bool
}

// NearestFacingTarget escolhe o alvo de atencao mais proximo de uma posicao.
//
// Mora aqui, e nao no renderizador, porque o host e o cliente precisam da
// MESMA regra: o host olha para o jogador mais proximo por FindNearestPlayer, e
// esta e a mesma pergunta feita a partir da lista que o cliente ja tem.
func NearestFacingTarget(from rl.Vector2, candidates []rl.Vector2) FacingTarget {
	best := FacingTarget{}
	bestDist := float32(-1)
	for _, c := range candidates {
		d := rl.Vector2DistanceSqr(from, c)
		if bestDist < 0 || d < bestDist {
			best = FacingTarget{Position: c, OK: true}
			bestDist = d
		}
	}
	return best
}
