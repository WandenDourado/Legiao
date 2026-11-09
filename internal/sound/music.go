// Package sound owns the campaign's stage music: one looping track per map,
// started when the map loads and swapped whenever a portal (or the F8 stage
// skip) takes the party to a different one.
package sound

import (
	"path/filepath"

	"github.com/WandenDourado/Legiao/internal/assets"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// mapTracks maps a stage's map filename to the music that plays while that
// map is loaded. Keyed by the base name (e.g. "world_01.json") rather than
// the full "assets/maps/..." path so a caller does not have to match the
// prefix exactly.
var mapTracks = map[string]string{
	"world_01.json": "assets/sounds/5-10. Splendid Dreams.mp3",
	"world_02.json": "assets/sounds/1-16. Labyrinth.mp3",
	"world_03.json": "assets/sounds/5-12. Daytime in Manuk.mp3",
	"world_04.json": "assets/sounds/4-20. Kingdom Memories.mp3",
	"world_05.json": "assets/sounds/5-06. Into The Arena!.mp3",
	"world_06.json": "assets/sounds/5-13. Dread and Bold.mp3",
	"world_07.json": "assets/sounds/4-13. Invisible Invasion.mp3",
}

// victoryTrack e a trilha que fecha a campanha. Ela NAO esta em `mapTracks`
// de proposito: nao pertence a um mapa, pertence a um MOMENTO — o fim da
// ultima cena, depois que a Senhora das Trevas caiu. O mapa 7 continua tendo
// a sua propria trilha de luta ("Invisible Invasion"), e e ela que esta
// tocando quando isto entra por cima.
const victoryTrack = "assets/sounds/1-09. Great Honor.mp3"

// musicBufferFrames e o tamanho do sub-buffer de cada AudioStream, em quadros
// de amostra.
//
// O PADRAO DO RAYLIB E sampleRate/30 — cerca de 33 ms. `Update` alimenta o
// stream UMA vez por quadro do jogo, entao com o padrao a musica so nao engasga
// enquanto o jogo mantiver todo quadro abaixo de 33 ms: um unico quadro de
// 40 ms deixa o dispositivo sem dado antes do proximo `UpdateMusicStream`, e o
// que se ouve e a travada relatada nos mapas 2 e 3.
//
// 4096 quadros sao ~93 ms a 44,1 kHz: a musica passa a atravessar quadros
// longos sem falhar. ISTO NAO CONSERTA O QUADRO LONGO — quem faz isso e a
// oclusao do terreno (internal/tilemap/terrain_renderer.go, `hidden`). Conserta
// o audio DEPENDER de o quadro nunca passar de 33 ms, que e um acoplamento que
// nao deveria existir: uma queda de fps tem de custar fluidez de imagem, nao
// audio picotado.
//
// Constante sem tipo de proposito: ela e o argumento de uma funcao do binding,
// e assim serve tanto a uma assinatura `int` quanto a uma `int32`.
const musicBufferFrames = 4096

var (
	ready        bool
	current      rl.Music
	currentTrack string
	playing      bool
)

// Init opens the audio device. Call once, after InitWindow and before the
// game loop starts. Safe to call more than once; only the first call does
// anything, so a session restart inside the same process does not try to
// reopen an already-open device.
func Init() {
	if ready {
		return
	}
	// ANTES de qualquer LoadMusicStream: o tamanho e lido na criacao de cada
	// AudioStream, entao mudar depois nao alcanca a trilha ja carregada.
	rl.SetAudioStreamBufferSizeDefault(musicBufferFrames)
	rl.InitAudioDevice()
	// Os efeitos das magias entram junto e pelo mesmo motivo: um dispositivo
	// por processo, e o catalogo carregado uma vez (ver sfx_bank.go).
	//
	// Eles NAO sao stream: `rl.Sound` carrega o arquivo inteiro na memoria, e o
	// sub-buffer acima nao os toca.
	loadSFX()
	ready = true
}

// Shutdown stops and unloads whatever track is loaded and closes the audio
// device. Call once, when the process is exiting.
func Shutdown() {
	if !ready {
		return
	}
	stopCurrent()
	unloadSFX()
	rl.CloseAudioDevice()
	ready = false
}

// PlayForMap starts the track mapped to mapPath's stage, looping it forever.
// Calling it again while that same stage's track is already playing is a
// no-op — a portal that leads back into a map already visited, or a new
// session starting again on map 1, must not restart the track from zero. A
// map with no configured track (a sandbox or terrain-validation map) just
// silences the music.
func PlayForMap(mapPath string) {
	if !ready {
		return
	}
	track, ok := mapTracks[filepath.Base(mapPath)]
	if !ok {
		stopCurrent()
		return
	}
	startTrack(track)
}

// PlayVictory troca a trilha do mapa pela musica de encerramento da campanha.
//
// Chamada TODO QUADRO por quem observa `BossState.VictoryReady`
// (game/victory_music.go) e nao uma unica vez no instante da virada: assim
// nao existe um "evento de vitoria" que possa se perder num quadro
// congelado por dialogo ou reconexao, e o cliente — que so descobre a
// vitoria quando o sinalizador chega pela rede — cai na mesma regra do host
// sem nenhum pacote novo. E um no-op enquanto a trilha ja for esta, exatamente
// como PlayForMap.
func PlayVictory() {
	if !ready {
		return
	}
	startTrack(victoryTrack)
}

// startTrack carrega e toca track em loop, trocando o que estiver no ar. Se
// track ja e o que esta tocando, nao faz nada — reiniciar do zero uma musica
// que ja esta correndo e sempre um defeito, seja um portal que volta ao mesmo
// mapa ou a chamada por quadro de PlayVictory.
func startTrack(track string) {
	if playing && track == currentTrack {
		return
	}
	stopCurrent()

	m := rl.LoadMusicStream(assets.Path(track))
	// LoadMusicStream already comes back with Looping true by default; set
	// explicitly anyway because the replay-after-it-ends behaviour is the
	// actual requirement here, not an accident of raylib's default.
	m.Looping = true
	rl.SetMusicVolume(m, effectiveMusicVolume())
	rl.PlayMusicStream(m)

	current = m
	currentTrack = track
	playing = true
}

// Update pumps the currently loaded track's audio buffer. Call once per
// frame, even on a frame where a dialogue or a reconnect overlay freezes the
// rest of the game — the music keeps playing through both, so its stream
// still needs to be fed every frame or it stutters.
func Update() {
	if !ready {
		return
	}
	// Os efeitos primeiro, e fora do teste de `playing`: um mapa sem trilha
	// configurada continua tendo magias. dt sai de GetFrameTime porque e
	// exatamente o mesmo valor que o laco ja usa neste quadro.
	updateSFX(rl.GetFrameTime())
	if !playing {
		return
	}
	rl.UpdateMusicStream(current)
}

func stopCurrent() {
	if !playing {
		return
	}
	rl.StopMusicStream(current)
	rl.UnloadMusicStream(current)
	playing = false
	currentTrack = ""
}

// applyMusicVolume re-aplica o volume efetivo (barramento × master, mudos
// incluídos) na faixa que estiver tocando agora. Chamada pelos setters dos
// barramentos em bus.go: a música é contínua, então quem arrasta o slider
// tem de ouvir a mudança na hora, não só no próximo disparo — ao contrário
// de um efeito de curta duração, que só muda no próximo `PlayAt`.
func applyMusicVolume() {
	if !playing {
		return
	}
	rl.SetMusicVolume(current, effectiveMusicVolume())
}
