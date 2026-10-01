package entity

import (
	"fmt"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// PresetColors contains a set of distinct colors for players.
var PresetColors = []string{
	"#FF5733", // red-orange
	"#33FF57", // green
	"#3357FF", // blue
	"#FF33A8", // pink
	"#FFC133", // orange
	"#33FFF6", // cyan
	"#A833FF", // purple
}

// hexToColor converts a hex color string like "#FF5733" to rl.Color.
func hexToColor(hex string) rl.Color {
	if hex == "" {
		return rl.SkyBlue
	}
	hex = strings.TrimPrefix(hex, "#")
	r, g, b := uint8(0), uint8(0), uint8(0)
	if len(hex) >= 2 {
		fmt.Sscanf(hex[0:2], "%02x", &r)
	}
	if len(hex) >= 4 {
		fmt.Sscanf(hex[2:4], "%02x", &g)
	}
	if len(hex) >= 6 {
		fmt.Sscanf(hex[4:6], "%02x", &b)
	}
	return rl.NewColor(r, g, b, 255)
}
