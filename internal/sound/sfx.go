package sound

// EFEITOS SONOROS DAS MAGIAS.
//
// A REGRA QUE DEFINE ESTE ARQUIVO: um efeito so e ouvido por quem esta PERTO
// de onde ele nasceu. Nao ha "som do meu jogador" e "som do jogador remoto" —
// ha um evento com uma posicao no mundo e um ouvinte, e a distancia entre os
// dois decide o volume (ate silencio). E por isso que as magias disparam som
// la no `internal/skill`, no ponto em que o efeito NASCE: aquele codigo roda
// igual no host e em todo cliente, entao cada maquina resolve sozinha o que
// o dono dela ouve, sem uma unica mensagem de rede a mais.
//
// POR QUE UMA FILA. `PlayAt` e chamada de qualquer goroutine — o cliente
// replica magias na goroutine que le a rede, o host as resolve no laco
// principal. Chamadas de audio do raylib ficam todas em `updateSFX`, que roda
// uma vez por quadro na thread do laco (a mesma que ja bombeia a musica).
// Aqui so entra na fila.

import (
	"sync"
	"sync/atomic"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// SFX identifica um efeito do catalogo abaixo.
type SFX string

// O catalogo, por personagem e slot.
const (
	SFXSanctuary   SFX = "sacerdotisa_q" // Santuario
	SFXAngelic     SFX = "sacerdotisa_r" // Area Angelical
	SFXFireLaunch  SFX = "mago_q_lancar" // Bola de Fogo saindo
	SFXFireImpact  SFX = "mago_q_impacto"
	SFXFireGround  SFX = "mago_q_chamas" // o chao que fica queimando
	SFXMeteor      SFX = "mago_r_impacto"
	SFXGraveyard   SFX = "necromante_q" // Cemiterio
	SFXLegion      SFX = "necromante_r" // Legiao Espectral
	SFXArrowVolley SFX = "arqueiro_q"   // uma rajada (soa 3x por lancamento)
	SFXCelestial   SFX = "arqueiro_r"   // Flechas Celestiais
	SFXShield      SFX = "paladina_q"   // Escudo Sagrado
	SFXAvatar      SFX = "paladina_r"   // Avatar dos Deuses

	// Ataque basico, um por personagem. Quem dispara e
	// entity.PlayAttackSound, que e onde o personagem escolhe o proprio som
	// (CharacterDef.AttackSound) — aqui so ficam o arquivo e a mixagem.
	SFXAttackMago        SFX = "mago_ataque"
	SFXAttackSacerdotisa SFX = "sacerdotisa_ataque"
	SFXAttackPaladina    SFX = "paladina_ataque"
	SFXAttackArqueiro    SFX = "arqueiro_ataque"
	SFXAttackNecromante  SFX = "necromante_ataque"
)

// O alcance da audicao, em unidades de MUNDO. No PC o zoom da camera e 1.0
// (game/camera.go), entao uma unidade de mundo e um pixel de tela: 1600 e um
// pouco alem da borda da tela num monitor 1080p, ou seja, ouve-se o que se ve
// e um tanto do que ainda vai aparecer. No celular (zoom 0,8, desde
// 27/09/2026) um aparelho de 2400 px enxerga 1500 de mundo para cada lado — a
// borda da tela continua dentro do raio. Ate SFXFullRadius o volume e cheio;
// dali ate SFXHearRadius ele cai; passou disso, o evento nao entra na fila.
const (
	SFXFullRadius float32 = 400
	SFXHearRadius float32 = 1600
	// sfxFadeOut e a janela de desvanecimento de um som com prazo (ver
	// PlayAtFor). Um corte seco no fim da magia soa como falha de audio.
	sfxFadeOut float32 = 0.6
)

// sfxDef descreve como um efeito se comporta na mixagem. `voices` e quantas
// copias do arquivo ficam carregadas: uma voz so toca um disparo por vez, e
// um segundo disparo cortaria o primeiro. `minGap` e o intervalo minimo entre
// dois disparos do MESMO efeito — a chuva de meteoros derruba ~67 pedras por
// segundo e sem isto o impacto viraria ruido branco.
type sfxDef struct {
	file   string
	voices int
	minGap float32
	// bus decide qual barramento de volume (internal/sound/bus.go) governa
	// este efeito: as doze magias vao para busSkill, os cinco ataques
	// basicos para busAttack. E uma coluna a mais nesta tabela, nao uma
	// segunda tabela (Passo 2 de doc/audio.md).
	bus sfxBus
}

var sfxDefs = map[SFX]sfxDef{
	SFXSanctuary:   {"assets/sounds/Sacer_Q.mp3", 2, 0.10, busSkill},
	SFXAngelic:     {"assets/sounds/Sacer_R.mp3", 1, 0.10, busSkill},
	SFXFireLaunch:  {"assets/sounds/Fireball-lauch.mp3", 3, 0.05, busSkill},
	SFXFireImpact:  {"assets/sounds/fireball-impact.mp3", 3, 0.06, busSkill},
	SFXFireGround:  {"assets/sounds/flames.mp3", 2, 0.20, busSkill},
	SFXMeteor:      {"assets/sounds/meteor.mp3", 4, 0.16, busSkill},
	SFXGraveyard:   {"assets/sounds/Necro_Q.mp3", 2, 0.10, busSkill},
	SFXLegion:      {"assets/sounds/Necro_R.mp3", 1, 0.10, busSkill},
	SFXArrowVolley: {"assets/sounds/Arch_Q.mp3", 3, 0.05, busSkill},
	SFXCelestial:   {"assets/sounds/Arch_R.mp3", 2, 0.05, busSkill},
	SFXShield:      {"assets/sounds/Pala_Q.mp3", 2, 0.10, busSkill},
	SFXAvatar:      {"assets/sounds/Pala_R.mp3", 1, 0.10, busSkill},

	// O intervalo minimo do ataque basico tem de caber DENTRO da cadencia
	// mais rapida do elenco (o Arqueiro, 2,5 golpes por segundo = 0,4 s), ou
	// o proprio ritmo do personagem engoliria golpes em silencio. 0,05 s
	// ainda serve para o que ele existe: dois jogadores da mesma classe
	// atirando no mesmo quadro soam uma vez, nao duas em cima uma da outra.
	SFXAttackMago:        {"assets/sounds/Mago_attack.mp3", 2, 0.05, busAttack},
	SFXAttackSacerdotisa: {"assets/sounds/Sacer_attack.mp3", 2, 0.05, busAttack},
	SFXAttackPaladina:    {"assets/sounds/Pala_attack.mp3", 2, 0.05, busAttack},
	SFXAttackArqueiro:    {"assets/sounds/Arch_attack.mp3", 3, 0.04, busAttack},
	SFXAttackNecromante:  {"assets/sounds/Necro_attack.mp3", 2, 0.05, busAttack},
}

// sfxRequest e um disparo ja aprovado (distancia e intervalo resolvidos)
// esperando a thread do laco para virar som.
type sfxRequest struct {
	id   SFX
	vol  float32
	life float32
}

// sfxReady e atomico, e nao o `ready` da musica, porque quem le e a goroutine
// de rede: antes de Init (e nos testes, que nunca abrem dispositivo de audio)
// PlayAt tem de ser um no-op barato e sem corrida.
var sfxReady atomic.Bool

var (
	sfxMutex    sync.Mutex
	listener    rl.Vector2
	listenerSet bool
	// Duas listas trocadas de lugar a cada quadro: enfileirar nunca aloca e
	// o laco nunca segura o mutex enquanto fala com o raylib.
	pending  []sfxRequest
	flushing []sfxRequest
)

// pendingCap trava a fila caso o laco pare de drenar (menu, sessao trocando).
// Perder um disparo velho e melhor do que crescer sem fim.
const pendingCap = 64

// SetListener publica onde esta o ouvinte — o jogador local. Chamada uma vez
// por quadro pelo laco do jogo.
func SetListener(pos rl.Vector2) {
	sfxMutex.Lock()
	listener, listenerSet = pos, true
	sfxMutex.Unlock()
}

// PlayAt dispara o efeito na posicao de mundo pos, tocando o arquivo inteiro.
// Silencioso se o ouvinte estiver longe demais.
func PlayAt(id SFX, pos rl.Vector2) { enqueue(id, pos, 0) }

// PlayAtFor e PlayAt para um efeito que dura o que a magia dura: o som toca
// normalmente e comeca a desvanecer perto de `life` segundos, terminando com
// ela. Existe porque varios arquivos sao mais longos que o efeito (flames tem
// 10,4 s para uma zona de 5 s), e fogo crepitando sobre chao apagado anuncia
// uma magia que ja acabou.
func PlayAtFor(id SFX, pos rl.Vector2, life float32) { enqueue(id, pos, life) }

func enqueue(id SFX, pos rl.Vector2, life float32) {
	if !sfxReady.Load() {
		return
	}
	def, ok := sfxDefs[id]
	if !ok {
		return
	}
	sfxMutex.Lock()
	defer sfxMutex.Unlock()
	if !listenerSet || lastTrigger == nil || len(pending) >= pendingCap {
		return
	}
	bus := busVolume(def.bus)
	if bus <= 0 {
		return
	}
	vol := attenuation(rl.Vector2Distance(listener, pos))
	if vol <= 0 {
		return
	}
	if t, seen := lastTrigger[id]; seen && sfxClock-t < def.minGap {
		return
	}
	lastTrigger[id] = sfxClock
	pending = append(pending, sfxRequest{id: id, vol: vol * bus, life: life})
}


// attenuation devolve o fator de volume para uma distancia. A queda e
// quadratica de proposito: linear deixa tudo dentro do raio soando quase
// igualmente alto, e a graca da regra e justamente conseguir dizer, de ouvido,
// se a magia caiu ao seu lado ou na outra ponta da tela.
func attenuation(d float32) float32 {
	if d <= SFXFullRadius {
		return 1
	}
	if d >= SFXHearRadius {
		return 0
	}
	t := (d - SFXFullRadius) / (SFXHearRadius - SFXFullRadius)
	return (1 - t) * (1 - t)
}
