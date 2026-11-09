package game

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestArenaGateBlocksOnlyTheReturnToTheCorridor(t *testing.T) {
	zone := tilemap.Zone{
		Name:      "arena",
		Rect:      rl.NewRectangle(100, 100, 800, 600),
		ArenaLock: true,
	}
	w := &World{Path: "arena-test", Zones: []tilemap.Zone{zone}}
	p := entity.NewPlayer(rl.NewVector2(500, 300), entity.CharMago)

	// Standing inside the arena arms the one-way boundary.
	w.UpdateArenaGate(p)
	if !w.arenaGate.returnLocked {
		t.Fatal("entrar na arena nao selou o retorno")
	}

	// A subsequent southward move may not put the foot box back in the
	// approach corridor below the arena.
	p.Position.Y = 900
	w.UpdateArenaGate(p)
	center, _, height := PlayerFootprint(p)
	if want := zone.Rect.Y + zone.Rect.Height - height/2; center.Y != want {
		t.Fatalf("centro dos pes = %.1f, esperado limite sul %.1f", center.Y, want)
	}

	// Northward movement remains valid: the seal must not interfere with the
	// exit gate or turn the arena into a trap after the waves are cleared.
	p.Position.Y = 180
	w.UpdateArenaGate(p)
	center, _, _ = PlayerFootprint(p)
	if center.Y >= zone.Rect.Y+zone.Rect.Height-height/2 {
		t.Errorf("movimento para a saida norte foi bloqueado: y = %.1f", center.Y)
	}
}

func TestArenaGateOpensOnlyAfterTheRunIsCleared(t *testing.T) {
	w := &World{
		Path: "arena-test",
		Zones: []tilemap.Zone{{
			Name:      "arena",
			Rect:      rl.NewRectangle(100, 100, 800, 600),
			ArenaLock: true,
		}},
	}

	network.SetWaveState(network.WaveState{Total: 5, Phase: string(network.WavePhaseFighting)})
	defer network.SetWaveState(network.WaveState{})
	w.UpdateArenaGate(entity.NewPlayer(rl.NewVector2(500, 300), entity.CharMago))
	if w.arenaGate.exitOpen {
		t.Fatal("o portao abriu antes de a horda terminar")
	}

	network.SetWaveState(network.WaveState{Total: 5, Phase: string(network.WavePhaseCleared)})
	w.UpdateArenaGate(entity.NewPlayer(rl.NewVector2(500, 300), entity.CharMago))
	if !w.arenaGate.exitOpen {
		t.Fatal("o portao permaneceu fechado depois de a horda ser concluida")
	}
}
