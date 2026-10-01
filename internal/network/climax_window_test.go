package network

import "testing"

// A janela do climax e a correcao do defeito relatado: antes, qualquer mapa
// com WaveState.Total > 0 em fase de luta bastava para `on_last_stand`
// disparar, entao a horda 1 de 3 do world_02 abria a cena do Necromante tao
// bem quanto a horda 3 - a unica que deveria.

func TestClimaxWindowClosedBeforeTheDeclaredWave(t *testing.T) {
	defer SetWaveState(WaveState{})
	// world_02 so arma na horda 3 (a matilha). A horda 1 nao e o climax.
	SetWaveState(WaveState{Total: 3, Index: 1, Phase: string(WavePhaseFighting)})
	if ClimaxWindowOpen("assets/maps/world_02.json", nil) {
		t.Fatal("a janela abriu na horda 1 de 3; so a horda declarada (3) deveria abrir")
	}
}

func TestClimaxWindowOpensOnTheDeclaredWave(t *testing.T) {
	defer SetWaveState(WaveState{})
	SetWaveState(WaveState{Total: 3, Index: 3, Phase: string(WavePhaseFighting)})
	if !ClimaxWindowOpen("assets/maps/world_02.json", nil) {
		t.Fatal("a janela nao abriu na horda declarada (3 de 3)")
	}
}

func TestClimaxWindowStaysOpenPastTheDeclaredWave(t *testing.T) {
	// world_05 arma a partir da horda 5 (a ultima); FromWave e um PISO, nao um
	// numero exato, entao um mapa que so declarasse "a partir de" quebraria se
	// alguem trocasse >= por ==.
	defer SetWaveState(WaveState{})
	SetWaveState(WaveState{Total: 5, Index: 5, Phase: string(WavePhaseFighting)})
	if !ClimaxWindowOpen("assets/maps/world_05.json", nil) {
		t.Fatal("a janela do world_05 nao abriu na quinta horda")
	}
}

func TestClimaxWindowRequiresFightingPhase(t *testing.T) {
	defer SetWaveState(WaveState{})
	// A horda certa, mas em pausa entre ondas: a janela nao pode abrir fora do
	// combate.
	SetWaveState(WaveState{Total: 3, Index: 3, Phase: string(WavePhaseBreak)})
	if ClimaxWindowOpen("assets/maps/world_02.json", nil) {
		t.Fatal("a janela abriu fora da fase de luta")
	}
}

func TestClimaxWindowAmbushFollowsTheRun(t *testing.T) {
	defer SetWaveState(WaveState{})
	// world_03 e world_04 sao emboscadas: a janela e "a corrida esta no ar".
	SetWaveState(WaveState{Total: 0})
	if ClimaxWindowOpen("assets/maps/world_03.json", nil) {
		t.Fatal("a emboscada do mapa 3 abriu sem corrida nenhuma no ar")
	}
	SetWaveState(WaveState{Total: 1, Index: 1, Phase: string(WavePhaseFighting)})
	if !ClimaxWindowOpen("assets/maps/world_03.json", nil) {
		t.Fatal("a emboscada do mapa 3 nao abriu com a corrida no ar")
	}
	if !ClimaxWindowOpen("assets/maps/world_04.json", nil) {
		t.Fatal("a emboscada do mapa 4 nao abriu com a corrida no ar")
	}
}

func TestClimaxWindowUndeclaredMapNeverOpens(t *testing.T) {
	defer SetWaveState(WaveState{})
	// world_01 nao tem cena de climax: nenhuma horda deveria abrir a janela,
	// por maior que seja o WaveState.
	SetWaveState(WaveState{Total: 3, Index: 3, Phase: string(WavePhaseFighting)})
	if ClimaxWindowOpen("assets/maps/world_01.json", nil) {
		t.Fatal("um mapa sem janela declarada abriu a janela do climax")
	}
	if ClimaxWindowOpen("assets/maps/sandbox.json", nil) {
		t.Fatal("um mapa fora da tabela abriu a janela do climax")
	}
}
