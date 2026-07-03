package game

// Config holds platform-specific configuration for the game.
type Config struct {
	Width      int
	Height     int
	Title      string
	FullScreen bool // true = Android (touch controls), false = Desktop (keyboard/mouse)
}

// DefaultConfig returns a default configuration for desktop.
func DefaultConfig() Config {
	return Config{
		Width:  0,
		Height: 0,
		Title:  "Legião - Survival Shooter",
	}
}

// AndroidConfig returns a configuration for Android.
func AndroidConfig() Config {
	return Config{
		Width:      0,
		Height:     0,
		Title:      "Legião",
		FullScreen: true,
	}
}
