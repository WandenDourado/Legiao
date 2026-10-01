package game

// A porta do climax e a fase 3 inteira: se ela nao abre, a horda infinita da
// fortaleza nunca comeca e a fase fica parada sem dizer por que. Ver
// climax_gate.go.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// fortress e a esplanada: um retangulo em (1000,1000) de 500x500.
func fortress() climaxGate {
	return climaxGate{
		zone: tilemap.Zone{
			Name:       "fortaleza",
			ClimaxGate: true,
			Rect:       rl.NewRectangle(1000, 1000, 500, 500),
		},
	}
}

func TestClimaxNeedsEverybodyInside(t *testing.T) {
	g := fortress()
	if g.partyArrived(map[string]network.PlayerState{
		"dentro": {PlayerID: "dentro", X: 1100, Y: 1100},
		"fora":   {PlayerID: "fora", X: 200, Y: 200},
	}) {
		t.Error("a emboscada armou com metade do grupo ainda na mata")
	}
	if !g.partyArrived(map[string]network.PlayerState{
		"a": {PlayerID: "a", X: 1100, Y: 1100},
		"b": {PlayerID: "b", X: 1400, Y: 1400},
	}) {
		t.Error("o grupo inteiro chegou e a emboscada nao armou")
	}
}

// O defeito relatado na fase 3: o portal abria antes da emboscada, alguem
// entrava nele, congelava fora da esplanada — e a porta esperava para sempre
// por um corpo que nem desenhado estava.
func TestSomebodyWaitingInAPortalDoesNotHoldTheGateShut(t *testing.T) {
	g := fortress()
	if !g.partyArrived(map[string]network.PlayerState{
		"chegou":  {PlayerID: "chegou", X: 1100, Y: 1100},
		"noPorta": {PlayerID: "noPorta", X: 50, Y: 50, InPortal: true},
	}) {
		t.Error("um jogador congelado dentro do portal travou a porta do climax")
	}
}

func TestAnEmptyPartyHasNotArrived(t *testing.T) {
	g := fortress()
	if g.partyArrived(map[string]network.PlayerState{}) {
		t.Error("a cena tocaria no primeiro quadro, antes de existir jogador")
	}
	// So mortos tambem nao e chegada: nao ha ninguem de pe para ser emboscado.
	if g.partyArrived(map[string]network.PlayerState{
		"corpo": {PlayerID: "corpo", X: 1100, Y: 1100, IsDead: true},
	}) {
		t.Error("um grupo inteiro caido armou a emboscada")
	}
}

func TestADeadBodyLeftBehindDoesNotHoldTheGateShut(t *testing.T) {
	g := fortress()
	if !g.partyArrived(map[string]network.PlayerState{
		"vivo":  {PlayerID: "vivo", X: 1100, Y: 1100},
		"morto": {PlayerID: "morto", X: 50, Y: 50, IsDead: true},
	}) {
		t.Error("um cadaver deixado para tras travou a porta do climax")
	}
}
