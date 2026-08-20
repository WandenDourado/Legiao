package network

import (
	"os"
	"path/filepath"
	"testing"
)

// A corrida de hordas passou a ser por mapa porque, como variavel de pacote,
// ela vazava: qualquer mapa carregado rodava as tres hordas do world_01. Estes
// testes protegem a tabela nova e os valores que, zerados, quebram o runner de
// um jeito que so aparece em jogo.

func TestWaveDefsForKnownMaps(t *testing.T) {
	for _, path := range []string{"assets/maps/world_01.json", "assets/maps/world_02.json"} {
		if defs := WaveDefsFor(path); len(defs) == 0 {
			t.Errorf("WaveDefsFor(%q) devolveu corrida vazia", path)
		}
	}
}

func TestWaveDefsForUnknownMapIsNil(t *testing.T) {
	// Mapa desconhecido tem que ficar QUIETO, e nao herdar a corrida de outro
	// mapa. Um mapa de validacao de terreno depende disso: com Total 0 o portal
	// nasce liberado e ele nao vira um mapa sem saida.
	if defs := WaveDefsFor("assets/maps/nao_existe.json"); defs != nil {
		t.Errorf("mapa desconhecido devolveu %d hordas; esperado nil", len(defs))
	}
}

func TestWorld02HasThreeWaves(t *testing.T) {
	// Tres e decisao de desenho, nao acaso: a horda 3 e o climax do mapa, e uma
	// horda depois dela esvaziaria a cena do Necromante. A horda 2 existe so
	// para o grupo chegar a matilha sem recarga e sem vida cheia.
	if got := len(WaveDefsFor("assets/maps/world_02.json")); got != 3 {
		t.Errorf("world_02 tem %d hordas; o desenho pede 3", got)
	}
}

// TestClimaxWaveWaitsForTheRescue guarda a propriedade que faz a cena
// acontecer: a horda em que a janela do climax abre nao pode ter ULTIMO.
//
// O defeito relatado no mapa 2 foi exatamente esse: com cinco jogadores o
// grupo limpava a matilha antes de cair a 30% de vida, a corrida terminava e o
// Necromante nunca se erguia — a fase liberava o portal sem entregar a suprema.
// A correcao e `Endless`, e ela e facil de perder sem querer, porque uma horda
// finita maior parece a mesma coisa e nao e.
//
// Vale para todo mapa cuja janela seja `ClimaxWindowWaveIndex`: a horda
// declarada em FromWave tem de ser a ultima da corrida e tem de se repor.
func TestClimaxWaveWaitsForTheRescue(t *testing.T) {
	for mapPath, win := range climaxWindows {
		if win.Kind != ClimaxWindowWaveIndex {
			continue
		}
		defs := WaveDefsFor(mapPath)
		if len(defs) == 0 {
			continue // mapa de emboscada; a corrida dele mora em climaxRuns
		}
		if win.FromWave != len(defs) {
			t.Errorf("%s: a janela abre na horda %d de %d; o climax tem de ser "+
				"a ultima", mapPath, win.FromWave, len(defs))
			continue
		}
		if !defs[win.FromWave-1].Endless {
			t.Errorf("%s: a horda do climax (%s) nao se repoe; o grupo pode "+
				"vencer matando o ultimo e a cena nunca toca",
				mapPath, defs[win.FromWave-1].Name)
		}
	}
}

func TestWorld05LastWaveWaitsForTheClimax(t *testing.T) {
	defs := WaveDefsFor("assets/maps/world_05.json")
	if len(defs) == 0 {
		t.Fatal("world_05 nao tem hordas")
	}
	if !defs[len(defs)-1].Endless {
		t.Fatal("a ultima horda do world_05 precisa se repor ate o climax")
	}
}

func TestEveryWaveIsRunnable(t *testing.T) {
	for path, defs := range waveRuns {
		for i, def := range defs {
			name := def.Name
			if name == "" {
				t.Errorf("%s horda %d: sem Name", path, i)
			}
			if def.Total() == 0 {
				t.Errorf("%s/%s: composicao vazia; a horda nunca termina de nascer", path, name)
			}
			if def.MaxConcurrent <= 0 {
				t.Errorf("%s/%s: MaxConcurrent %d; nada pode nascer", path, name, def.MaxConcurrent)
			}
			if def.BatchSize <= 0 {
				t.Errorf("%s/%s: BatchSize %d; nada pode nascer", path, name, def.BatchSize)
			}
			// SpawnInterval zero solta um lote POR QUADRO: sessenta lobos
			// aparecem num piscar e a horda vira um travamento, nao uma luta.
			if def.SpawnInterval <= 0 {
				t.Errorf("%s/%s: SpawnInterval %v; um lote por quadro", path, name, def.SpawnInterval)
			}
		}
	}
}

func TestWaveRunMapsExist(t *testing.T) {
	// Chave errada aqui nao quebra nada visivel: o mapa so fica quieto, e a
	// causa e um typo que ninguem procura.
	for path := range waveRuns {
		if _, err := os.Stat(filepath.Join("..", "..", path)); err != nil {
			t.Errorf("waveRuns tem chave %s, que nao existe no disco: %v", path, err)
		}
	}
}

func TestRunnerWithoutDefsStaysQuiet(t *testing.T) {
	// Um mapa sem corrida nao pode reportar horda nenhuma, senao o HUD desenha
	// "Horda 1/0" e o portao do portal trava um mapa que deveria estar aberto.
	wr := NewWaveRunner(nil, nil)
	state := wr.State(0)
	if state.Total != 0 {
		t.Errorf("Total = %d sem corrida; esperado 0", state.Total)
	}
	if wr.current() != nil {
		t.Error("current() devolveu horda numa corrida vazia")
	}
}
