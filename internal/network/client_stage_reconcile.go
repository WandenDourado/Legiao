package network

// Self-healing for the two one-shot events a client can miss: MsgGameOver and
// MsgReset. Both still exist for the instant reaction on the screen that
// receives them; this is the second, redundant layer for whoever does not —
// a phone whose screen locked through the moment the host sent it, a socket
// that dropped and had to reconnect. See doc/plan_correcoes_30-08-2026.md,
// item 2 ("cliente mobile preso na tela de Game Over").
//
// Every StateUpdatePayload now carries StageGen and GameOver (protocol.go),
// so "the next snapshot" is at most one simulation tick away — the host
// publishes it whether or not anything was missed, and this file is what a
// client does with it.

import (
	"log"
	"sync"
)

var (
	stageReconcileMu sync.Mutex
	// lastSeenStageGen is the host's StageGen from the last StateUpdatePayload
	// this client actually processed. It is host-reported, not this client's
	// own StageGeneration() counter: the two can legitimately disagree while
	// a client is catching up, and it is the host's number that matters here.
	lastSeenStageGen int
	// stageGenSeen is false until the first StateUpdatePayload of this
	// client's connection lands. Nothing is "missed" before there is a
	// baseline to miss it against — a first snapshot right after an ordinary
	// join must never read as a stage change and force a needless reset.
	stageGenSeen bool
)

// resetStageReconcileBaseline forgets the last-seen generation. Called only
// where there is truly nothing yet to compare against — a brand-new
// connection in this process (ConnectClient) or a fresh session after giving
// up on reconnecting (ResetClientSession) — so THAT first snapshot is taken
// as the baseline instead of misread as a stage change.
//
// A reconnect (rejoin, reconnect.go) deliberately does NOT call this: keeping
// lastSeenStageGen from before the connection dropped is exactly what lets
// the first post-reconnect snapshot reveal whether a MsgReset happened while
// this client was gone.
func resetStageReconcileBaseline() {
	stageReconcileMu.Lock()
	stageGenSeen = false
	stageReconcileMu.Unlock()
}

// reconcileStageState is called from handleMessage on every StateUpdatePayload.
// If the host's StageGen moved since the last snapshot this client processed,
// a MsgReset was missed: catch up with the same applyStageReset a live
// MsgReset would have triggered, landing on the spawn position the host
// already reported for this player in THIS SAME snapshot (players, keyed by
// LocalPlayerID — after a real reset the host's PlayerState.X/Y IS the spawn
// point, ResetStage in host_reset.go). Otherwise, a lone GameOver mismatch
// means only MsgGameOver (or the MsgReset that would have cleared it) was
// missed, without a generation change — for instance a client that was
// already caught up on StageGen and only failed to receive the one-shot
// game-over announcement itself.
func (c *Client) reconcileStageState(state StateUpdatePayload, players map[string]PlayerState) {
	stageReconcileMu.Lock()
	first := !stageGenSeen
	previous := lastSeenStageGen
	changed := stageGenSeen && state.StageGen != previous
	lastSeenStageGen = state.StageGen
	stageGenSeen = true
	stageReconcileMu.Unlock()

	if first {
		return
	}
	if changed {
		spawn := players[LocalPlayerID]
		log.Printf("[Client] geracao da fase avancou sem MsgReset (%d -> %d); "+
			"recuperando pelo snapshot", previous, state.StageGen)
		c.applyStageReset(ResetPayload{SpawnX: spawn.X, SpawnY: spawn.Y})
		return
	}
	if state.GameOver != GameOver {
		log.Printf("[Client] GameOver do host (%v) divergia do local (%v); adotando o do host",
			state.GameOver, GameOver)
		GameOver = state.GameOver
	}
}
