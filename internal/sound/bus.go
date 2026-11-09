package sound

// OS QUATRO BARRAMENTOS DE VOLUME: master, música, habilidades, ataque
// básico. Antes deste arquivo, `Volume` (música) e `SFXVolume` (efeitos)
// eram CONSTANTES (ver doc/audio.md, versão antiga) — a tela de
// configurações (internal/settings, internal/ui) precisa poder mudá-los em
// tempo real, então viraram valores atômicos.
//
// Atômicos e não um mutex porque `enqueue` (sfx.go) roda a partir de
// QUALQUER goroutine — o cliente replica magias na goroutine que lê a rede —
// e o volume de um disparo tem de ser lido sem lock, no meio de um caminho
// que já é sensível a latência.
//
// Um efeito JÁ TOCANDO não muda de volume no meio do caminho: o próximo
// disparo já sai no valor novo (é `enqueue`, chamado uma vez por disparo,
// que lê o barramento). A EXCEÇÃO é a música — contínua, e é o que o
// jogador está ouvindo enquanto arrasta o slider — por isso
// `SetMusicBusVolume`/`SetMuteMusic`/`SetMuteMaster` chamam
// `applyMusicVolume` (music.go) na hora.

import (
	"math"
	"sync/atomic"
)

// sfxBus é a categoria de um efeito, usada só para escolher o barramento —
// é uma coluna a mais em sfxDef (ver sfx.go), não uma segunda tabela.
type sfxBus int

const (
	busSkill sfxBus = iota
	busAttack
)

var (
	masterVolBits atomic.Uint32
	musicVolBits  atomic.Uint32
	skillVolBits  atomic.Uint32
	attackVolBits atomic.Uint32

	atomicMuteMaster atomic.Bool
	atomicMuteMusic  atomic.Bool
	atomicMuteSkill  atomic.Bool
	atomicMuteAttack atomic.Bool

	// atomicFocusSuppressed e "Silenciar em segundo plano" (Passo 5, Audio):
	// SEPARADO de atomicMuteMaster, que e a escolha manual do jogador. O
	// laco do jogo (internal/game) chama SetFocusSuppressed a cada quadro
	// com o estado de foco da janela/app; desmutar manualmente nao pode
	// reativar o audio enquanto o jogo continua em segundo plano, e
	// recuperar o foco nao pode desfazer um mudo que o jogador pediu.
	atomicFocusSuppressed atomic.Bool
)

func init() {
	// Os mesmos números que eram as constantes antigas: quem já jogava não
	// deve notar volume nenhum diferente até abrir Configurações e mexer.
	storeF32(&masterVolBits, 1.0)
	storeF32(&musicVolBits, 0.5)
	storeF32(&skillVolBits, 0.85)
	storeF32(&attackVolBits, 0.85)
}

func storeF32(a *atomic.Uint32, v float32) { a.Store(math.Float32bits(v)) }
func loadF32(a *atomic.Uint32) float32     { return math.Float32frombits(a.Load()) }

func clampVol(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SetMasterVolume/MasterVolume: o barramento que multiplica todos os outros.
func SetMasterVolume(v float32) { storeF32(&masterVolBits, clampVol(v)); applyMusicVolume() }
func MasterVolume() float32     { return loadF32(&masterVolBits) }

// SetMusicBusVolume/MusicBusVolume: a trilha de fase (music.go).
func SetMusicBusVolume(v float32) { storeF32(&musicVolBits, clampVol(v)); applyMusicVolume() }
func MusicBusVolume() float32     { return loadF32(&musicVolBits) }

// SetSkillVolume/SkillVolume: os doze efeitos de magia do catálogo.
func SetSkillVolume(v float32) { storeF32(&skillVolBits, clampVol(v)) }
func SkillVolume() float32     { return loadF32(&skillVolBits) }

// SetAttackVolume/AttackVolume: os cinco `*_attack.mp3`.
func SetAttackVolume(v float32) { storeF32(&attackVolBits, clampVol(v)) }
func AttackVolume() float32     { return loadF32(&attackVolBits) }

// SetMuteMaster/MuteMaster e as demais Mute*: silenciam o barramento SEM
// perder o valor guardado. Master muda a música na hora pelo mesmo motivo
// que SetMasterVolume; os outros dois bastam no próximo disparo.
func SetMuteMaster(m bool) { atomicMuteMaster.Store(m); applyMusicVolume() }
func MuteMaster() bool     { return atomicMuteMaster.Load() }

// SetFocusSuppressed é "Silenciar em segundo plano": internal/game chama
// isto todo quadro com o estado de foco atual. Independente de MuteMaster —
// ver o comentário de atomicFocusSuppressed.
func SetFocusSuppressed(v bool) { atomicFocusSuppressed.Store(v); applyMusicVolume() }
func SetMuteMusic(m bool)  { atomicMuteMusic.Store(m); applyMusicVolume() }
func MuteMusic() bool      { return atomicMuteMusic.Load() }
func SetMuteSkill(m bool)  { atomicMuteSkill.Store(m) }
func MuteSkill() bool      { return atomicMuteSkill.Load() }
func SetMuteAttack(m bool) { atomicMuteAttack.Store(m) }
func MuteAttack() bool     { return atomicMuteAttack.Load() }

// masterFactor é 0 quando o master está mudo, senão o próprio volume master.
func masterFactor() float32 {
	if atomicMuteMaster.Load() || atomicFocusSuppressed.Load() {
		return 0
	}
	return loadF32(&masterVolBits)
}

// busVolume devolve o fator FINAL (barramento × master, mudo incluído) para
// um efeito daquela categoria. enqueue() multiplica isto pela atenuação de
// distância; PlayUI usa o valor puro, sem espacialização.
func busVolume(b sfxBus) float32 {
	switch b {
	case busSkill:
		if atomicMuteSkill.Load() {
			return 0
		}
		return loadF32(&skillVolBits) * masterFactor()
	case busAttack:
		if atomicMuteAttack.Load() {
			return 0
		}
		return loadF32(&attackVolBits) * masterFactor()
	default:
		return 0
	}
}

// effectiveMusicVolume é o volume final da trilha: barramento de música ×
// master, honrando os dois mudos.
func effectiveMusicVolume() float32 {
	if atomicMuteMusic.Load() {
		return 0
	}
	return loadF32(&musicVolBits) * masterFactor()
}
