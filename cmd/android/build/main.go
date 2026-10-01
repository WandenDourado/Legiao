package main

import (
	"github.com/WandenDourado/Legiao/internal/game"
)

func main() {
	// Run the shared game loop with Android configuration
	game.Run(game.AndroidConfig())
}
