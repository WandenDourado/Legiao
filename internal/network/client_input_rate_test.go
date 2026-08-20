package network

import "testing"

func TestInputStateDueAtSteadyCadence(t *testing.T) {
	var s clientInputState
	frameDT := float32(1.0 / 60.0)

	sent := 0
	for i := 0; i < 60; i++ {
		if s.due(frameDT, 0, 0) {
			sent++
		}
	}
	// A 60 fps com dt constante, um acumulador a InputHz (20) deve publicar
	// ~20 vezes por segundo simulado, nao 60.
	if sent < 18 || sent > 22 {
		t.Fatalf("esperava ~%d publicacoes em 1s de quadros a 60 fps, teve %d", InputHz, sent)
	}
}

func TestInputStateSkipsBetweenTicks(t *testing.T) {
	var s clientInputState
	frameDT := float32(1.0 / 60.0)

	// O primeiro quadro sempre publica (o acumulador comeca vazio e o
	// primeiro dt ja pode nao fechar o intervalo — mas nao deve publicar TODO
	// quadro).
	skipped := false
	for i := 0; i < 5; i++ {
		if !s.due(frameDT, 0, 0) {
			skipped = true
			break
		}
	}
	if !skipped {
		t.Fatalf("60 Hz de quadro nao deveria publicar input a cada quadro (InputHz=%d)", InputHz)
	}
}

func TestInputStateForcesSendOnMovementTransition(t *testing.T) {
	var s clientInputState
	frameDT := float32(1.0 / 60.0)

	// Estabelece "parado" como o ultimo estado conhecido.
	s.due(frameDT, 0, 0)

	// Um unico quadro de movimento, bem antes do proximo tique — sem a
	// transicao forcada, isto ficaria preso ate o proximo intervalo de
	// InputHz e o boneco remoto "grudaria" no ultimo ponto.
	if !s.due(frameDT, 200, 0) {
		t.Fatal("parar/comecar a andar deveria forcar uma publicacao imediata, independente da cadencia")
	}
}

func TestInputStateNoTransitionOnSameDirectionMagnitudeChange(t *testing.T) {
	var s clientInputState
	frameDT := float32(1.0 / 60.0)

	s.due(frameDT, 150, 0)
	// Mesma direcao, magnitude diferente: nao e a transicao que a interpolacao
	// deixa de cobrir, entao nao deve forcar envio fora da cadencia.
	if s.due(frameDT, 180, 0) {
		t.Fatal("mudanca de magnitude na mesma direcao nao deveria forcar publicacao fora da cadencia")
	}
}
