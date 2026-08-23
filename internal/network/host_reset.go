package network

// Stage reset: the only way out of a Game Over. Only the host may call it, and
// it puts the run back exactly where it started — first wave, empty field,
// everyone alive on the spawn point.

import (
	"log"
)

// ResetStage restarts the current map. It is safe to call at any time, not
// just after a Game Over, so the host can also use it to start a run over.
func (h *Host) ResetStage() {
	// Field first: enemies, projectiles and every skill visual anchored to a
	// player have to be gone before anybody is put back on the spawn.
	h.EntityManager.Clear()
	h.Skills.Reset()
	// A cena de resgate e por CORRIDA, nao por sessao: quem perdeu a fase tem
	// direito a ela de novo. E a janela de imunidade nao pode sobreviver ao
	// reset, senao a fase recomeca com alguem invencivel.
	ResetLastStand()
	clearInvulnerability()
	// A concessao da corrida (ver progression.go) tambem NAO sobrevive ao
	// reinicio: SetUnlockedUltimates so e chamado numa troca de mapa, e o F5
	// fica no mesmo mapa. Sem isto uma fase perdida DEPOIS do resgate
	// reiniciaria com a suprema do heroi ainda destravada.
	ClearRunGrantedUltimates()
	// A EMBOSCADA DO CLIMAX nao e reiniciada, e sim REMOVIDA. Ela nao pertence
	// ao carregamento do mapa: quem a instala e a chegada do grupo a fortaleza
	// (game/climax_gate.go). Reiniciada como as outras, ela voltaria a nascer
	// no primeiro quadro da nova tentativa, com o grupo ainda na mata — e
	// `StartClimax` recusaria instala-la de novo mais tarde, porque ja haveria
	// corrida em campo.
	if len(GarrisonFor(h.stageMap)) > 0 {
		h.Waves = nil
		SetWaveState(WaveState{})
	} else if h.Waves != nil {
		h.Waves.Reset()
	}
	// E o portal volta a ficar trancado ate a emboscada acontecer de novo. O
	// F5 fica no MESMO mapa, entao SetClimaxMap nunca roda e sem isto a
	// segunda tentativa da fase 3 recomecaria com a saida aberta — que e
	// exatamente a trava que climax_pending.go existe para impedir.
	RearmClimaxPending()
	// E a guarnicao volta ao campo: o EntityManager acabou de ser esvaziado.
	h.RestoreGarrison()
	// As gargulas junto. Sem isto a segunda tentativa do mapa 4 correria sem
	// fogo de longa distancia, e a fase ficaria mais facil a cada derrota.
	h.RestoreSentries()
	// Os canhoes do mapa 6, pela mesma razao: um deles destruido pelo
	// julgamento da Paladina nao pode continuar destruido numa tentativa nova.
	h.RestoreCannons()
	// E o chefe, pelo motivo contrario ao dos outros tres: sem ele a segunda
	// tentativa do mapa 7 nao ficaria mais facil, ficaria IMPOSSIVEL DE
	// TERMINAR — a corrida daquele mapa e infinita e so para quando ele cai.
	h.RestoreBoss(h.stageMap)

	h.cdMutex.Lock()
	h.skillCooldowns = make(map[string]float32)
	h.skillCharges = make(map[string]int)
	h.attackCooldowns = make(map[string]float32)
	h.cdMutex.Unlock()
	ClearCooldowns()

	h.playersMutex.Lock()
	for _, p := range h.players {
		p.Health = p.MaxHealth
		p.IsDead = false
		p.RespawnIn = 0
		p.X = int(h.PlayerSpawn.X)
		p.Y = int(h.PlayerSpawn.Y)
		// Whoever was mid-wait on the portal at the moment of the reset
		// cannot stay invisible and uncontrollable on a stage that just
		// restarted under them.
		p.InPortal = false
	}
	h.playersMutex.Unlock()

	// GameOver is cleared last: tickRespawns and the game-over check both read
	// it, and clearing it earlier would let a stale timer fire mid-reset.
	GameOver = false
	LocalPlayerDead = false
	RequestLocalReset(h.PlayerSpawn)

	log.Printf("[Host] fase reiniciada pelo host")
	h.broadcast(Message{Type: MsgReset, Payload: MustMarshal(ResetPayload{
		SpawnX: int(h.PlayerSpawn.X),
		SpawnY: int(h.PlayerSpawn.Y),
	})})
	// A slot the reset just healed and moved may have been a bot's; a
	// class nobody plays still needs its bot after F5.
	h.ReconcileBots()
	// A bot that was sealed inside the arena when Game Over hit must not
	// stay sealed after the retry starts everyone back at the spawn — see
	// doc/tilemap.md "Arena de mão única". ReconcileBots does not clear
	// this on its own: a bot that already existed keeps its runtime as-is.
	h.ResetBotArenaLocks()
	h.BroadcastRoster()
}

// Reset puts the wave runner back at the first wave, opening on a break so the
// first horde is announced exactly like it is at the start of a run.
func (wr *WaveRunner) Reset() {
	wr.index = 0
	wr.phase = WavePhaseBreak
	wr.breakTime = WaveBreakDuration
	wr.pending = nil
	wr.spawnTimer = 0
	wr.announced = false
	wr.sectorCursor = 0
	wr.sentriesOrdered = false
}
