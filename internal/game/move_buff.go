package game

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
)

// ApplyMoveBuff ajusta a velocidade do jogador local ao que as magias ativas
// nele mandam, ANTES do passo deste quadro.
//
// Recalculado do zero toda vez, a partir de entity.PlayerSpeed: um bonus que
// fosse aplicado por multiplicacao sobre o valor corrente se acumularia a 60
// vezes por segundo, e a Sacerdotisa sairia da tela em dois segundos. Aqui a
// velocidade base e a verdade e o buff e sempre um fator sobre ela — sem
// estado a zerar quando a magia acaba.
func ApplyMoveBuff(p *entity.Player) {
	p.Speed = entity.PlayerSpeed * network.MoveSpeedMultiplier(network.LocalPlayerID)
}
