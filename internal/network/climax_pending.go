package network

// O PORTAL NAO PODE ABRIR ANTES DA EMBOSCADA.
//
// `game.PortalsUnlocked` sempre leu uma coisa so: o estado da corrida de
// hordas. "Total 0" queria dizer "mapa quieto, nao tranque a saida" — e nos
// mapas de terreno isso esta certo.
//
// Nos mapas de EMBOSCADA esta errado, e foi o defeito relatado na fase 3. O
// world_03 nao tem um unico marcador `enemy_spawn_*`, de proposito: a
// jogabilidade dele e de guarnicao, e a unica horda que ele tem e o climax,
// que so e instalado quando o grupo inteiro alcanca a fortaleza
// (game/climax_gate.go -> Host.StartClimax). Ate la o WaveState fica zerado, o
// portal se materializava no primeiro quadro do mapa, e o resultado era o pior
// tipo de trava:
//
//	um jogador entra no portal -> host_portal_presence.go o congela e o
//	renderer para de desenha-lo -> a porta do climax exige TODOS os vivos
//	dentro da zona da fortaleza -> aquele corpo esta parado no portal, longe
//	dela -> a emboscada nunca arma, e o grupo fica olhando uma fortaleza vazia.
//
// A correcao e dizer a verdade sobre o mapa: enquanto ele ainda DEVE a
// emboscada, ele nao e um mapa quieto — ele e um mapa cuja luta ainda nao
// comecou, e a saida fica trancada como ficaria em qualquer corrida de hordas.
//
// Depois que a emboscada entra em campo nada disto e consultado: `WaveState`
// passa a ter Total > 0 e a regra normal (so abre com a fase limpa) volta a
// valer sozinha. E por isso que o cliente nao precisa receber mensagem nenhuma
// para acompanhar — ele descobre o mapa no carregamento, igual ao host, e o
// resto chega pelo WaveState que ja viaja junto do snapshot de inimigos.

import "sync"

var (
	climaxMu sync.RWMutex
	// climaxMapPath e o mapa que esta carregado nesta maquina.
	climaxMapPath string
	// climaxSprung marca que a emboscada ja foi instalada nesta corrida.
	climaxSprung bool
)

// MapOwesClimax reports whether a map declares a scripted climax ambush at
// all. A map without one never locks anything.
func MapOwesClimax(mapPath string) bool { return len(climaxRuns[mapPath]) > 0 }

// SetClimaxMap records which map is loaded here and re-arms the tracking.
// Called on every map load, by host and client alike (game.World.ApplyToHost).
func SetClimaxMap(mapPath string) {
	climaxMu.Lock()
	climaxMapPath, climaxSprung = mapPath, false
	climaxMu.Unlock()
}

// NoteClimaxSprung records that the ambush is on the field. Host only —
// Host.StartClimax is the single place that installs it.
func NoteClimaxSprung() {
	climaxMu.Lock()
	climaxSprung = true
	climaxMu.Unlock()
}

// RearmClimaxPending puts the tracking back to "the ambush is still owed",
// for a fresh run of the same map. Called from ResetStage (host) and from the
// client's mirror of it — the F5 stays on the same map, so SetClimaxMap never
// runs and the flag would otherwise stay spent.
func RearmClimaxPending() {
	climaxMu.Lock()
	climaxSprung = false
	climaxMu.Unlock()
}

// ClimaxPending reports whether the loaded map still owes its ambush.
func ClimaxPending() bool {
	climaxMu.RLock()
	defer climaxMu.RUnlock()
	return !climaxSprung && len(climaxRuns[climaxMapPath]) > 0
}
