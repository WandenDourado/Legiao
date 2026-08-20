package nav

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// world05Bounds mirrors the largest real map (plan §3.1): 64x90 tiles of
// 128px, the biggest mesh the game builds.
func world05Bounds() world.Bounds {
	return world.Bounds{Width: 64 * 128, Height: 90 * 128}
}

func BenchmarkBuildWorld05Scale(b *testing.B) {
	bounds := world05Bounds()
	for i := 0; i < b.N; i++ {
		Build(nil, bounds, CellSize, AgentBox)
	}
}

func BenchmarkFindPathWorld05Scale(b *testing.B) {
	// A few scattered obstacles so the search does real work instead of a
	// straight line, without approaching "no route exists".
	walls := testWalls{
		{x: 1000, y: 1000, width: 600, height: 40},
		{x: 3000, y: 4000, width: 40, height: 600},
		{x: 5000, y: 8000, width: 800, height: 40},
	}
	g := Build(walls, world05Bounds(), CellSize, AgentBox)
	from := rl.NewVector2(100, 100)
	to := rl.NewVector2(8000, 11000)

	b.ResetTimer()
	var out []rl.Vector2
	for i := 0; i < b.N; i++ {
		out, _ = g.FindPath(from, to, out)
	}
}
