package network

// A janela do clímax, por MAPA.
//
// `partyIsFalling` (internal/game/dialogue.go) só perguntava por
// `WaveState.Total > 0` e fase de luta — o que qualquer horda de um mapa com
// corrida satisfaz. No `world_02` e no `world_05` isso deixava o clímax
// disparar na PRIMEIRA horda, bastando o grupo cair de vida ali, quando a
// cena só deveria tocar na horda certa da fase.
//
// Mesmo padrão de `waveRuns` (wave_runs.go), `climaxRuns` (host_garrison.go)
// e `lastStandHeroes` (last_stand_heroes.go): a FASE declara a janela, e o
// resto do sistema só lê a declaração. Uma fase nova entra na tabela e herda
// a regra sozinha; uma fase que não entra aqui nunca dispara `on_last_stand`
// — ver ClimaxWindowOpen.
import (
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ClimaxWindowKind é a NATUREZA da janela: as duas que já existem no jogo.
type ClimaxWindowKind int

const (
	// ClimaxWindowWaveIndex arma a partir da horda FromWave (1-based, a mesma
	// contagem de WaveState.Index): "penúltima" é Total-1, "última" é Total.
	ClimaxWindowWaveIndex ClimaxWindowKind = iota
	// ClimaxWindowAmbush arma enquanto a emboscada do clímax está no ar —
	// `WaveState.Total > 0` e fase de luta. Nos mapas que declaram este tipo
	// (world_03, world_04) a ÚNICA fonte de WaveState é a própria emboscada
	// roteirizada (climaxRuns, host_garrison.go): eles não têm entrada em
	// `waveRuns`, então "corrida no ar" já significa "clímax no ar" — nada de
	// horda comum pode fazer WaveState.Total sair de zero antes disso.
	ClimaxWindowAmbush
	// ClimaxWindowCheckpoint arma quando o grupo alcança a zona
	// `corridor_checkpoint` do mapa (mapa 6, que não tem corrida de horda
	// nenhuma — os canhões atiram desde a chegada).
	ClimaxWindowCheckpoint
)

// ClimaxWindow é a janela declarada por um mapa.
type ClimaxWindow struct {
	Kind ClimaxWindowKind
	// FromWave só vale para ClimaxWindowWaveIndex: a horda a partir da qual a
	// janela arma, 1-based.
	FromWave int
}

// climaxWindows é a janela de cada mapa, pela mesma chave que World.Path,
// campaignMaps, waveRuns e lastStandHeroes usam.
//
// world_01 fica de fora de propósito: não tem cena de clímax (nenhum roteiro
// `on_last_stand` no dele), e um mapa sem entrada aqui nunca dispara a cena —
// ver ClimaxWindowOpen.
var climaxWindows = map[string]ClimaxWindow{
	// A matilha do mapa 2 é a horda 3 de 3: só ela é o clímax, não a 1 nem a 2.
	"assets/maps/world_02.json": {Kind: ClimaxWindowWaveIndex, FromWave: 3},
	// A emboscada da fortaleza. Já implementada como porta de zona
	// (game/climax_gate.go); aqui ela só precisa dizer QUE tipo de janela é,
	// para on_last_stand parar de confiar num WaveState genérico.
	"assets/maps/world_03.json": {Kind: ClimaxWindowAmbush},
	// O salão se fecha: mesma natureza da fortaleza, outra emboscada.
	"assets/maps/world_04.json": {Kind: ClimaxWindowAmbush},
	// A quinta e última horda da senhora.
	"assets/maps/world_05.json": {Kind: ClimaxWindowWaveIndex, FromWave: 5},
	// O corredor final não tem horda — os canhões atiram desde a chegada —,
	// então a janela é o checkpoint, não o WaveState.
	"assets/maps/world_06.json": {Kind: ClimaxWindowCheckpoint},
	// A arena: a corrida e uma horda so e ela nunca termina, entao a janela
	// abre na primeira e fica aberta. E o certo aqui — a fase INTEIRA e o
	// climax da campanha.
	"assets/maps/world_07.json": {Kind: ClimaxWindowWaveIndex, FromWave: 1},
}

// ClimaxWindowFor devolve a janela declarada por um mapa, e se ele declarou
// alguma. Exportado para o diretor de diálogo avisar quando um roteiro
// `on_last_stand` existe sem janela — o mesmo aviso que StartWaveRun já dá
// para um mapa com marcadores e sem corrida.
func ClimaxWindowFor(mapPath string) (ClimaxWindow, bool) {
	w, ok := climaxWindows[mapPath]
	return w, ok
}

// ClimaxWindowOpen reporta se a janela do clímax do mapa está aberta agora.
//
// Mapa sem janela declarada NUNCA abre — é o que corrige o defeito relatado:
// antes, qualquer mapa com WaveState.Total > 0 em fase de luta bastava.
func ClimaxWindowOpen(mapPath string, zones []tilemap.Zone) bool {
	win, ok := climaxWindows[mapPath]
	if !ok {
		return false
	}
	switch win.Kind {
	case ClimaxWindowWaveIndex:
		state := GetWaveState()
		return state.Total > 0 && WavePhase(state.Phase) == WavePhaseFighting &&
			state.Index >= win.FromWave
	case ClimaxWindowAmbush:
		state := GetWaveState()
		return state.Total > 0 && WavePhase(state.Phase) == WavePhaseFighting
	case ClimaxWindowCheckpoint:
		zone, ok := tilemap.CorridorCheckpointZone(zones)
		if !ok {
			return false
		}
		return anyPlayerInZone(zone)
	}
	return false
}

// anyPlayerInZone reporta se ao menos um jogador — de pé ou caído — está
// dentro da zona. `Zone.Contains` lê a posição guardada mesmo de um jogador
// morto de propósito: um corpo que caiu dentro do checkpoint ainda conta como
// "alcançou o ponto" (ver doc/combat_rules.md, "O último suspiro").
func anyPlayerInZone(zone tilemap.Zone) bool {
	for _, p := range GetAllPlayers() {
		if zone.Contains(rl.NewVector2(float32(p.X), float32(p.Y))) {
			return true
		}
	}
	return false
}
