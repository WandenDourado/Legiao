package network

// O portal de um mapa de EMBOSCADA nao pode abrir antes dela. Estes testes
// travam a maquina de estado que game.PortalsUnlocked consulta; o defeito que
// ela corrige esta contado em climax_pending.go.

import "testing"

func TestClimaxPendingOnlyForMapsThatDeclareAnAmbush(t *testing.T) {
	defer SetClimaxMap("")

	// Mapa de emboscada: deve a cena desde o carregamento.
	SetClimaxMap("assets/maps/world_03.json")
	if !ClimaxPending() {
		t.Error("world_03 carregou sem dever a emboscada")
	}

	// Mapa de horda comum: nunca deve nada, entao o portal continua sendo
	// decidido so pelo WaveState, como sempre foi.
	SetClimaxMap("assets/maps/world_01.json")
	if ClimaxPending() {
		t.Error("world_01 nao tem emboscada e mesmo assim trancaria o portal")
	}
}

func TestClimaxPendingClearsWhenTheAmbushIsInstalled(t *testing.T) {
	defer SetClimaxMap("")

	SetClimaxMap("assets/maps/world_04.json")
	NoteClimaxSprung()
	if ClimaxPending() {
		t.Error("a emboscada entrou em campo e o portal continuaria trancado por ela")
	}
}

func TestStageRestartOwesTheAmbushAgain(t *testing.T) {
	defer SetClimaxMap("")

	SetClimaxMap("assets/maps/world_03.json")
	NoteClimaxSprung()

	// O F5 fica no MESMO mapa, entao SetClimaxMap nunca roda: sem o rearme a
	// segunda tentativa comecaria com a saida aberta.
	RearmClimaxPending()
	if !ClimaxPending() {
		t.Error("a tentativa nova comecou com o portal ja liberado")
	}
}

func TestLoadingAnotherMapReArmsByItself(t *testing.T) {
	defer SetClimaxMap("")

	SetClimaxMap("assets/maps/world_03.json")
	NoteClimaxSprung()
	// A travessia do portal chama SetClimaxMap, e ela sozinha tem de devolver
	// a divida do mapa novo — nada mais roda entre os dois mapas.
	SetClimaxMap("assets/maps/world_04.json")
	if !ClimaxPending() {
		t.Error("o mapa 4 herdou a emboscada ja gasta do mapa 3")
	}
}

// Toda fase que declara emboscada tem de aparecer em climaxRuns, que e a
// tabela que MapOwesClimax e ClimaxPending leem. Se um mapa novo entrar em
// `climaxWindows` como ClimaxWindowAmbush e esquecer desta, o portal dele
// abriria no primeiro quadro — exatamente o defeito da fase 3.
func TestEveryAmbushWindowHasAClimaxRun(t *testing.T) {
	for mapPath, win := range climaxWindows {
		if win.Kind != ClimaxWindowAmbush {
			continue
		}
		if !MapOwesClimax(mapPath) {
			t.Errorf("%s declara janela de emboscada e nao tem entrada em climaxRuns",
				mapPath)
		}
	}
}
