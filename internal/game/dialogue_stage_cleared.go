package game

import "github.com/WandenDourado/Legiao/internal/network"

// stageFightOver e a condicao do gatilho on_waves_cleared: a luta que fecha a
// fase acabou.
//
// Em mapa de hordas e o que o nome do gatilho diz — a ultima horda morreu
// (inclusive a emboscada dos mapas 3 e 4, que e uma horda de verdade).
//
// O corredor da fase 6 nao tem horda nenhuma: nao tem entrada em waveRuns e o
// WaveState fica em Total 0 o tempo todo. O que fecha aquela fase e calar os
// dois canhoes — a mesma regra que ja abre o portal dela (PortalsUnlocked).
// Sem esta segunda metade, a cena "depois de destruir os canhoes" da
// narrativa de 10/09/2026 existiria no roteiro e nunca tocaria, em silencio.
//
// A pergunta e "ha canhao declarado e todos cairam?", nunca "que mapa e
// este?", o mesmo principio de PortalsUnlocked. Mapa quieto (sem horda e sem
// canhao) responde falso: ele nao terminou luta nenhuma.
func stageFightOver() bool {
	if network.CannonsStanding() {
		return false
	}
	state := network.GetWaveState()
	if state.Total > 0 {
		return network.WavePhase(state.Phase) == network.WavePhaseCleared
	}
	return network.CannonsSilenced()
}
