package entity

// A VELOCIDADE DO JOGO INTEIRO, EM UM NUMERO.
//
// Os testadores de 20/09/2026 nao disseram "o personagem anda devagar em
// relacao aos monstros" — disseram que o JOGO e lento. A queixa nao e de
// balanceamento (as proporcoes entre lobo, orc e jogador estavam certas), e
// sim de ritmo: tudo se desloca devagar ao mesmo tempo. Por isso a correcao e
// um fator unico sobre tudo que se move, e nao uma revisao numero a numero,
// que mudaria as proporcoes sem querer.
//
// ONDE ELE ENTRA (quatro pontos, de proposito — espalhar seria perder o fio):
//
//  1. PlayerSpeed (player.go). Os bots vem junto, porque botSpeed deriva dele
//     (network/host_bot_move.go).
//  2. RegisterEnemy (enemy_sprite.go): multiplica def.Speed. Os numeros
//     escritos em cada registro continuam sendo os originais, legiveis ao lado
//     dos comentarios que os explicam ("o orc e o mais lento do elenco, 130
//     contra os 200 do jogador"). Speed == 0 continua 0 sozinho, entao a
//     Sentinela e a Imperatriz nao saem do lugar por acidente.
//  3. O TEMPO POR QUADRO da animacao de caminhada, dividido pelo mesmo fator,
//     no mesmo RegisterEnemy e em RegisterCharacter. Sem isto o orc PATINA: o
//     FrameTime dele nao e gosto, e a unica solucao de uma equacao que envolve
//     a Speed (ver o comentario do AnimWalk em enemy_sprite.go).
//  4. Velocidade de projetil (projectile.go, skill/), com o TEMPO DE VIDA
//     dividido pelo mesmo fator. O produto dos dois e o ALCANCE, e o alcance
//     nao pode mudar: `entity.AttackReach` e o contrato numerico das supremas
//     dependem dele. O tiro chega mais rapido no mesmo lugar.
//
// O QUE ELE DELIBERADAMENTE NAO TOCA: recarga, dano, duracao de magia de area,
// cadencia de ataque, tempo de ressurreicao. O pedido era fluidez de
// deslocamento. Multiplicar a cadencia junto transformaria isto numa mudanca
// de dificuldade que ninguem pediu — se o combate ficar facil demais com todo
// mundo mais rapido, o ajuste e em AttackCooldown, monstro a monstro, e nao
// aqui.
//
// E UMA CONSTANTE DE COMPILACAO, E NAO UMA CONFIGURACAO. O jogo e
// host-autoritativo, mas o jogador LOCAL e simulado no proprio quadro
// (network/interpolation.go): com fatores diferentes nas duas pontas, o
// cliente preve 270 e o host corrige para 200 — rubber-banding continuo. Por
// isso ela nunca entra na tela de Configuracoes, e por isso host e clientes
// precisam da MESMA build.
const GameSpeed float32 = 1.20

// scaledSpeed aplica o fator preservando o zero: "parado" e uma decisao de
// design (a Sentinela, a Imperatriz), nao um numero pequeno.
func scaledSpeed(v float32) float32 {
	if v <= 0 {
		return v
	}
	return v * GameSpeed
}

// scaledFrameTime encurta um ciclo de animacao de LOCOMOCAO na mesma
// proporcao em que o corpo dele passou a andar mais rapido. Zero continua
// zero: e o que diz "esta definicao nao anima".
func scaledFrameTime(v float32) float32 {
	if v <= 0 {
		return v
	}
	return v / GameSpeed
}
