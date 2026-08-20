package network

// Handoff for a stage reset that has to touch the local player entity. The
// network layer does not own the player (the game package does), so a reset —
// whether the host triggered it or it arrived from the host — is queued here
// and applied by the game loop on its next frame.

import (
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	pendingReset      bool
	pendingResetSpawn rl.Vector2
	resetMutex        sync.Mutex
	// stageGeneration conta as corridas da fase. Quem guarda estado POR
	// CORRIDA compara este numero com o que viu por ultimo e sabe que comecou
	// tudo de novo.
	//
	// Ele mora aqui porque RequestLocalReset e o unico ponto por onde os dois
	// papeis passam: o host ao reiniciar, o cliente ao receber o reinicio.
	// Contar em qualquer outro lugar teria que ser contado duas vezes.
	stageGeneration int
)

// RequestLocalReset queues a reset of the local player at the given spawn.
func RequestLocalReset(spawn rl.Vector2) {
	resetMutex.Lock()
	pendingReset = true
	pendingResetSpawn = spawn
	stageGeneration++
	resetMutex.Unlock()

	// Os dois snapshots guardados para interpolar sao do mundo que acabou de
	// deixar de existir. Interpolar do campo de batalha para o ponto de spawn
	// faria todo mundo DESLIZAR pelo mapa inteiro ao reiniciar a fase.
	//
	// Aqui, e nao em applyStageReset, pelo mesmo motivo que stageGeneration
	// esta aqui: este e o unico ponto por onde os DOIS papeis passam.
	ResetInterpolation()
}

// StageGeneration is how many times the stage has been restarted. State that
// belongs to a single run watches it for changes.
func StageGeneration() int {
	resetMutex.Lock()
	defer resetMutex.Unlock()
	return stageGeneration
}

// ConsumeLocalReset returns the queued spawn point and clears the request.
// The second result is false when there is nothing pending.
func ConsumeLocalReset() (rl.Vector2, bool) {
	resetMutex.Lock()
	defer resetMutex.Unlock()
	if !pendingReset {
		return rl.Vector2{}, false
	}
	pendingReset = false
	return pendingResetSpawn, true
}
