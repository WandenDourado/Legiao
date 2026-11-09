package game

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// The motes floating around the portal. They are computed from the clock and an
// index instead of being simulated, so a portal costs no state, no allocation
// and looks identical on every machine of a session.

const (
	// portalSparks is how many motes live around one portal at a time.
	portalSparks = 22
	// portalSparkPull is the share of motes that are drawn INTO the vortex; the
	// rest drift outward and escape. One in three keeps the reading of a
	// doorway that breathes both ways.
	portalSparkPull = 3
)

// portalNoise is a cheap deterministic hash. Motes have to look scattered while
// staying identical frame to frame, and a real RNG would make them flicker
// somewhere new on every draw.
func portalNoise(seed int) float32 {
	value := math.Sin(float64(seed)*12.9898) * 43758.5453
	return float32(value - math.Floor(value))
}

// drawPortalSparks paints the motes: they rise slowly, fade in and out, and
// either spiral into the center or escape past the rim.
func drawPortalSparks(s portalShape, t float32) {
	for spark := 0; spark < portalSparks; spark++ {
		start := portalNoise(spark*3 + 1)
		span := portalNoise(spark*3 + 2)
		size := portalNoise(spark*3 + 3)

		// Each mote has its own lifetime, so they never blink together.
		life := 2.2 + 1.6*span
		phase := float32(math.Mod(float64(t)+float64(start)*float64(life), float64(life))) / life
		// Fading in and out on a sine means a mote is invisible when it is born
		// and when it dies: nothing ever pops into existence.
		fade := float32(math.Sin(float64(phase) * math.Pi))

		var radius float32
		if spark%portalSparkPull == 0 {
			radius = 1.35 - 1.10*phase // pulled toward the center
		} else {
			radius = 0.85 + 0.75*phase // escaping past the rim
		}

		// The angle drifts with the vortex, so the motes belong to the same
		// current as the mist inside.
		angle := float64(start)*2*math.Pi + float64(t)*portalSpin*0.55
		position := s.point(angle, radius)
		position.Y -= (0.35 + 0.50*size) * s.RY * phase
		position.X += 4 * float32(math.Sin(float64(t)*1.3+float64(spark)))

		color := portalTurquoise
		if spark%2 == 0 {
			color = portalRim
		}
		rl.DrawCircleV(position, 1.3+1.1*size, s.fade(color, 0.70*fade))
	}
}
