package sound

// O BANCO DE VOZES: onde os arquivos ficam carregados e onde a fila de
// sfx.go vira som. Tudo aqui roda na thread do laco do jogo — `loadSFX` e
// `unloadSFX` de dentro de Init/Shutdown, `updateSFX` de dentro de Update.
//
// POR QUE VARIAS COPIAS DO MESMO ARQUIVO. Um `rl.Sound` toca UM disparo por
// vez: mandar tocar de novo reinicia o que estava tocando. Dois meteoros
// caindo com meio segundo de diferenca precisam soar sobrepostos, entao cada
// efeito carrega tantas copias quanto `sfxDef.voices` pedir e os disparos
// giram entre elas.

import (
	"github.com/WandenDourado/Legiao/internal/assets"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// voice e uma copia carregada de um efeito, e o estado do disparo que ela
// esta tocando agora.
type voice struct {
	s    rl.Sound
	busy bool
	age  float32
	life float32 // 0 = toca ate o fim do arquivo
	fade float32 // janela final de desvanecimento (so quando life > 0)
	vol  float32 // volume decidido pela distancia no momento do disparo
}

// bank sao as vozes de um efeito, com o rodizio de qual usar em seguida.
type bank struct {
	voices []*voice
	next   int
}

var (
	banks map[SFX]*bank
	// sfxClock e o relogio do audio (segundos desde Init) contra o qual
	// lastTrigger mede o intervalo minimo de cada efeito.
	sfxClock    float32
	lastTrigger map[SFX]float32
)

// loadSFX carrega o catalogo inteiro. Chamada uma vez, logo depois de abrir o
// dispositivo de audio: sao arquivos curtos, e carregar sob demanda colocaria
// a leitura de disco dentro do quadro em que a magia e lancada.
func loadSFX() {
	banks = make(map[SFX]*bank, len(sfxDefs))
	lastTrigger = make(map[SFX]float32, len(sfxDefs))
	for id, def := range sfxDefs {
		b := &bank{}
		for i := 0; i < def.voices; i++ {
			b.voices = append(b.voices, &voice{s: rl.LoadSound(assets.Path(def.file))})
		}
		banks[id] = b
	}
	sfxReady.Store(true)
}

// unloadSFX para e descarrega todas as vozes.
func unloadSFX() {
	sfxReady.Store(false)
	for _, b := range banks {
		for _, v := range b.voices {
			rl.StopSound(v.s)
			rl.UnloadSound(v.s)
		}
	}
	banks = nil
	sfxMutex.Lock()
	// lastTrigger sai sob o mutex, e nao junto de banks: quem escreve nele e
	// enqueue, de outra goroutine. Zera-lo por fora deixaria uma janela para
	// uma escrita em mapa nil — enqueue trata o nil como "audio fechado".
	lastTrigger = nil
	pending, flushing = pending[:0], flushing[:0]
	listenerSet = false
	sfxMutex.Unlock()
}

// updateSFX drena a fila e envelhece as vozes ocupadas. Uma vez por quadro.
func updateSFX(dt float32) {
	if !sfxReady.Load() {
		return
	}
	sfxMutex.Lock()
	sfxClock += dt
	// Troca de listas: o laco percorre `flushing` sem o mutex enquanto
	// `pending` volta vazia para receber os disparos deste quadro.
	pending, flushing = flushing[:0], pending
	sfxMutex.Unlock()

	for _, r := range flushing {
		startVoice(r)
	}
	ageVoices(dt)
}

func startVoice(r sfxRequest) {
	b := banks[r.id]
	if b == nil || len(b.voices) == 0 {
		return
	}
	v := b.pick()
	rl.StopSound(v.s)
	rl.SetSoundVolume(v.s, r.vol)
	rl.PlaySound(v.s)
	v.busy, v.age, v.vol, v.life, v.fade = true, 0, r.vol, r.life, 0
	if r.life > 0 {
		v.fade = sfxFadeOut
		// Um som de prazo curto nao pode passar quase a vida inteira
		// desvanecendo: o teto de 40% mantem o corpo do som audivel.
		if lim := r.life * 0.4; v.fade > lim {
			v.fade = lim
		}
	}
}

// pick devolve uma voz livre, ou a proxima do rodizio se todas estiverem
// ocupadas (o disparo novo corta o mais antigo daquele efeito).
func (b *bank) pick() *voice {
	for _, v := range b.voices {
		if !v.busy {
			return v
		}
	}
	v := b.voices[b.next]
	b.next = (b.next + 1) % len(b.voices)
	return v
}

func ageVoices(dt float32) {
	for _, b := range banks {
		for _, v := range b.voices {
			if !v.busy {
				continue
			}
			v.age += dt
			if v.life > 0 {
				rem := v.life - v.age
				if rem <= 0 {
					rl.StopSound(v.s)
					v.busy = false
					continue
				}
				if rem < v.fade {
					rl.SetSoundVolume(v.s, v.vol*rem/v.fade)
				}
			}
			if !rl.IsSoundPlaying(v.s) {
				v.busy = false
			}
		}
	}
}
