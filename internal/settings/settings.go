// Package settings owns the LOCAL, per-machine preferences screen: audio
// buses, frame cap/vsync, touch control geometry, HUD scale and the chosen
// player display name.
//
// NUNCA ENTRA NA REDE. O jogo é host-autoritativo (doc/network.md) e nada
// aqui pode virar estado replicado, regra de jogo ou decisão do host — é por
// isso que este pacote não importa `ui`, `game` nem `network`, e ninguém
// aqui importa `network` para "avisar o grupo" de uma mudança. Cada máquina
// decide o próprio volume, o próprio teto de quadro e o próprio tamanho de
// botão, ponto.
//
// Ler tem de ser barato: `sound.enqueue` chama o equivalente disto a cada
// disparo de efeito, inclusive da goroutine de rede que replica magias. Por
// isso `Get` devolve uma CÓPIA sob um RWMutex — nunca um ponteiro para o
// estado vivo — e por isso os barramentos de áudio propriamente ditos vivem
// em `internal/sound` como valores atômicos (ver doc/audio.md); este pacote
// só guarda os números escolhidos e os persiste.
package settings

import "sync"

// FrameCap enumera as únicas três opções de teto de quadro que a tela expõe.
// Não é um int qualquer: 45 ou 24 não são escolhas do menu, e clamp() rejeita
// qualquer valor fora deste conjunto voltando para o padrão.
type FrameCap int32

const (
	FrameCapUncapped FrameCap = 0
	FrameCap30       FrameCap = 30
	FrameCap60       FrameCap = 60
)

// maxPlayerNameLen limita o nome exibido — não é o PlayerID de rede (esse
// continua sendo `player_<nanotime>`, ver internal/ui/menu.go), é só o rótulo
// local que este round da tela de configurações grava. Ver doc/settings.md,
// seção "Nome do jogador", para por que ele ainda não viaja pela rede.
const maxPlayerNameLen = 20

// HUDScaleMax e o teto da escala de interface. Exportado porque a tela de
// Configuracoes (internal/ui) desenha o slider nessa faixa e os dois nao podem
// discordar: um slider mais curto que o clamp deixa um trecho inalcancavel, e
// um mais longo devolve valores que o clamp corta sem o jogador entender por
// que o texto parou de crescer.
const HUDScaleMax float32 = 1.8

// Settings é o modelo inteiro da tela de configurações. Um campo por opção,
// no tipo mais simples que sirva: float32 0..1 para volume/opacidade, bool
// para chaves, int/FrameCap para escolhas fechadas.
type Settings struct {
	// Áudio. Master multiplica os outros três; Mute* silencia o barramento
	// SEM perder o valor guardado (arrastar para zero e tentar lembrar depois
	// é o jeito ruim de mutar).
	MasterVolume float32
	MusicVolume  float32
	SkillVolume  float32
	AttackVolume float32
	MuteMaster   bool
	MuteMusic    bool
	MuteSkill    bool
	MuteAttack   bool
	// MuteBackground silencia tudo quando a janela perde o foco (desktop) ou
	// o app vai para segundo plano (mobile). Ver internal/game/audio_focus.go.
	MuteBackground bool

	// Vídeo e desempenho.
	TargetFPS FrameCap
	// DeathEffect liga a desintegração dos monstros ao morrer
	// (entity/death_fx.go). Desligável porque no celular cada grão de cinza
	// é uma chamada de desenho a mais — relatos de lentidão e aquecimento em
	// 28/09/2026. Só visual e só local, como todo o resto deste pacote.
	DeathEffect bool
	// GroundDetail desenha a camada `ground_detail` de tiles do mapa: as
	// flores, galhos e pedrinhas do chao. Medido em 28/09/2026: cobre ~10% da
	// tela, entao desligar alivia pouco — entra no preset Baixo porque e de
	// graca, nao porque resolve (plans/plan_qualidade_grafica_mobile.md).
	GroundDetail bool
	// RenderHeight e a altura, em pixels, em que o jogo desenha no CELULAR; o
	// scaler de hardware do aparelho amplia para a tela sem custo de GPU.
	// Zero = resolucao nativa. So vale na PROXIMA abertura (a janela nasce
	// com o tamanho) e e ignorado no desktop, que sempre desenha na resolucao
	// do monitor. Ver quality.go.
	RenderHeight int32
	// DisplayWidth/DisplayHeight sao o tamanho NATIVO da tela, anotado na
	// ultima abertura em resolucao nativa. O Android nao informa isso antes
	// da janela existir, e a janela escalada precisa da PROPORCAO da tela.
	// Zero = ainda nao se sabe (primeira abertura): abre nativo.
	DisplayWidth  int32
	DisplayHeight int32
	// VSync é hint de criação do contexto GL: só vale na PRÓXIMA abertura do
	// jogo (ver internal/game/config.go). A tela avisa isso ao lado da opção
	// em vez de fingir que o efeito é imediato.
	VSync bool
	// Fullscreen é o estado de janela do DESKTOP (rl.ToggleFullscreen). Não
	// confundir com game.Config.FullScreen, que significa "layout Android" —
	// nomes parecidos, coisas diferentes; ver doc/settings.md.
	Fullscreen bool

	// Controles — mobile. Escala e opacidade multiplicam a geometria base de
	// internal/input (attackRadiusFrac etc.), nunca a substituem.
	ActionButtonScale   float32 // 0.8 .. 1.3
	ControlsOpacity     float32 // 0 .. 1
	LeftHanded          bool    // espelha o cluster: joystick à direita, ações à esquerda
	JoystickRadiusScale float32 // 0.7 .. 1.5, multiplica TouchState.MaxRadius

	// Controles — desktop.
	ContinuousMouseAim bool // mira segue o mouse o tempo todo, não só no clique

	// Interface.
	//
	// HUDScale vai a 1.8, e nao a 1.3 como as escalas de controle: desde
	// 20/09/2026 ela rege TODO o texto do jogo (internal/ui/text.go), nao so
	// a barra de vida, e o teto antigo nao dava alcance para quem reclamou de
	// texto pequeno numa TV ou num celular na mao.
	HUDScale     float32 // 0.8 .. HUDScaleMax
	ShowAimHint  bool    // o chevron do internal/ui/aim_indicator.go

	// Jogador.
	PlayerName string
}

var (
	mu      sync.RWMutex
	current = Default()
)

// Get devolve uma CÓPIA do estado atual. Barato de propósito: é chamado de
// qualquer goroutine, a qualquer taxa.
func Get() Settings {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Set substitui o estado inteiro, já com clamp aplicado, e devolve o valor
// efetivamente gravado (pode diferir de s se algum campo estava fora da
// faixa).
func Set(s Settings) Settings {
	s = s.clamped()
	mu.Lock()
	current = s
	mu.Unlock()
	return s
}

// Update aplica fn sobre uma cópia do estado atual e grava o resultado, já
// com clamp. Uso típico: `settings.Update(func(s *settings.Settings) { s.MasterVolume = 0.6 })`.
func Update(fn func(*Settings)) Settings {
	mu.Lock()
	s := current
	fn(&s)
	s = s.clamped()
	current = s
	mu.Unlock()
	return s
}
