package game

// Config holds platform-specific configuration for the game.
type Config struct {
	Width      int
	Height     int
	Title      string
	FullScreen bool // true = Android (touch controls), false = Desktop (keyboard/mouse)
	// TargetFPS e o que Run passa para rl.SetTargetFPS. Metade dos quadros e
	// metade do calor no celular (doc/performance.md, item mobile #5), mas
	// baixar o padrao e decisao de produto, nao deste commit: os dois
	// devolvem 60 ate a medida em jogo decidir o contrario.
	TargetFPS int32
}

// DefaultConfig returns a default configuration for desktop.
func DefaultConfig() Config {
	return Config{
		Width:     0,
		Height:    0,
		Title:     "Legião - Survival Shooter",
		TargetFPS: 60,
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
	}
}
