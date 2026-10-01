package game

// Config holds platform-specific configuration for the game.
type Config struct {
	Width      int
	Height     int
	Title      string
	FullScreen bool // true = Android (touch controls), false = Desktop (keyboard/mouse)
	// TargetFPS e o teto que Run passa para rl.SetTargetFPS. ZERO significa
	// "sem teto proprio": com VSync ligado, quem cadencia e o monitor.
	//
	// Metade dos quadros e metade do calor no celular (doc/performance.md,
	// item mobile #5), mas baixar o padrao e decisao de produto: os dois
	// devolvem 60 ate a medida em jogo decidir o contrario.
	//
	// ATE 22/08/2026 ESTE CAMPO NAO ERA LIDO: `Run` chamava
	// `rl.SetTargetFPS(60)` com o numero cravado, entao declarar 30 aqui nao
	// teria efeito nenhum. Agora e lido.
	TargetFPS int32
	// VSync pede ao driver para trocar o buffer so no retraco do monitor.
	//
	// SEM ISTO O JOGO RASGA A IMAGEM. `SetTargetFPS` NAO e vsync: ele so faz o
	// laco dormir ate completar o intervalo alvo, e o buffer continua sendo
	// trocado no meio de um desenho de tela — que e exatamente o screen
	// tearing relatado na sessao de teste de 22/08/2026, e que o quadro
	// perfeitamente cravado em 16,7 ms do painel do F3 escondia: a CADENCIA
	// estava certa, o SINCRONISMO nunca existiu.
	//
	// A flag e um hint de criacao do contexto GL, entao `Run` a aplica ANTES
	// de `InitWindow`; depois da janela aberta ela nao tem mais efeito.
	VSync bool
	// CameraZoom e a escala do mundo na tela: 1 = um pixel de mundo por pixel
	// de tela; abaixo de 1 a camera se afasta e mostra mais mapa.
	//
	// O celular usa 0,8 (27/09/2026, playtest): a tela fisica e pequena, e os
	// jogadores nao viam o lobo chegando nem a gargula que precisavam derrubar
	// — o Arqueiro tinha de andar ate ela para saber onde mirar. 0,8 mostra
	// 25% a mais em cada eixo (~56% mais area). O PC fica em 1,0: a tela
	// dele ja e grande, e escala nao inteira muda a nitidez do terreno, que e
	// desenhado com filtro POINT (tilemap/terrain_batch.go).
	//
	// ZERO vale 1: um Config montado a mao sem este campo continua valendo o
	// que sempre valeu.
	CameraZoom float32
}

// DefaultConfig returns a default configuration for desktop.
func DefaultConfig() Config {
	return Config{
		Width:     0,
		Height:    0,
		Title:      "Legião - Survival Shooter",
		TargetFPS:  60,
		VSync:      true,
		CameraZoom: 1.0,
	}
}

// AndroidConfig returns a configuration for Android.
func AndroidConfig() Config {
	return Config{
		Width:      0,
		Height:     0,
		Title:      "Legião",
		FullScreen: true,
		TargetFPS:  60,
		VSync:      true,
		CameraZoom: 0.8,
	}
}
