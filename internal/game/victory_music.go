package game

// A musica que fecha a campanha.

import (
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/sound"
)

// updateVictoryMusic troca a trilha do mapa pela musica de encerramento
// assim que a vitoria e ANUNCIADA — isto e, quando a ultima fala da cena
// final ja fechou.
//
// O gatilho e `BossState.VictoryReady`, e nao a morte da chefe
// (`Health <= 0`) nem o fim do dialogo lido direto do diretor. Tres razoes,
// e cada uma descarta uma alternativa que parece mais direta:
//
//   - VictoryReady e exatamente "a cena final terminou de falar": o host so
//     o levanta em MarkBossVictoryReady, chamado quando a ultima linha do
//     roteiro `on_waves_cleared` fecha (game/dialogue.go). E o mesmo instante
//     que os fogos e o "Missao Concluida" esperam, entao a musica entra junto
//     com eles em vez de disputar um segundo relogio.
//   - Ele viaja para o cliente no mesmo pacote da vida da chefe
//     (network/boss_state.go): o cliente nao dirige a narrativa e nunca sabe
//     que uma cena acabou, mas ve este sinalizador virar. Sem isto a musica
//     tocaria so na maquina do host, e a mesa inteira ouviria coisas
//     diferentes no fim do jogo.
//   - Um mapa sem chefe tem `Present` falso, entao a campanha inteira passa
//     por aqui sem que nada aconteca — nao e preciso perguntar qual mapa esta
//     carregado.
//
// Chamada por quadro, ao lado de sound.Update no laco principal: PlayVictory
// e no-op depois da primeira vez, e ser um ESTADO observado todo quadro (em
// vez de um evento disparado uma vez) e o que faz a troca sobreviver a um
// quadro congelado por dialogo ou reconexao.
func updateVictoryMusic() {
	boss := network.GetBossState()
	if !boss.Present || !boss.VictoryReady {
		return
	}
	sound.PlayVictory()
}
