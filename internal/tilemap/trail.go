package tilemap

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// trailLayerName is the object layer that holds the walked paths.
const trailLayerName = "trails"

// A trail is a ribbon of art laid along a curve, not tiles and not a material.
//
// Not tiles, because a tile of straight path needs a corner piece drawn for
// every turn — rotating a straight tile gives horizontal from vertical and
// nothing else — and it would pin the route to the 128px grid.
//
// Not a material, because the reference art has no border to cut along: the
// biggest step in greenness between neighbouring columns is 2.12 over a range
// of 8.6 to 19.2, and the middle of the path still has grass in it. That
// gradient IS the art, so the art is drawn as-is and its own measured profile
// became the alpha channel of terrain_trail.png.
const (
	// trailWidth is how many world pixels the ribbon spans across, which is
	// also how often the texture repeats along the path.
	trailWidth = 512
	// trailStep is the spacing of the quads along the curve. Short quads keep
	// the ribbon smooth on a bend; consecutive quads share their edge exactly,
	// so there is no gap and no double-blended overlap.
	trailStep = 24
)

// Trail is one walked path across the map.
type Trail struct {
	Name   string
	Points []rl.Vector2
	// Width is the ribbon width in world pixels, from the object's `width`
	// custom property.
	Width float32
}

// Trails returns every polyline in the map's trail layer. A trail needs at
// least two points; anything shorter is a mis-drawn object, not a path.
func Trails(m *TiledMap) []Trail {
	var trails []Trail
	for _, layer := range m.Layers {
		if layer.Type != "objectgroup" || layer.Name != trailLayerName {
			continue
		}
		for _, object := range layer.Objects {
			if len(object.Polyline) < 2 {
				continue
			}
			points := make([]rl.Vector2, 0, len(object.Polyline))
			for _, p := range object.Polyline {
				points = append(points, rl.NewVector2(object.X+p.X, object.Y+p.Y))
			}
			trails = append(trails, Trail{
				Name:   object.Name,
				Points: points,
				Width:  object.FloatProperty("width", trailWidth),
			})
		}
	}
	return trails
}

// Path resamples the polyline into evenly spaced points along a smoothed
// curve, which is what the ribbon is built on.
//
// Smoothing is not cosmetic. The ribbon folds over itself on the inside of a
// turn whose radius is smaller than half its width, so the corners a hand-drawn
// polyline has must be rounded off before they become geometry.
func (t Trail) Path(step float32) []rl.Vector2 {
	if len(t.Points) < 2 || step <= 0 {
		return nil
	}
	dense := resample(t.Points, step)
	return smooth(dense, int(t.Width/(2*step)))
}

// resample walks the polyline and drops a point every step pixels.
func resample(points []rl.Vector2, step float32) []rl.Vector2 {
	out := []rl.Vector2{points[0]}
	carry := float32(0)
	for i := 0; i+1 < len(points); i++ {
		a, b := points[i], points[i+1]
		dx, dy := b.X-a.X, b.Y-a.Y
		length := float32(math.Hypot(float64(dx), float64(dy)))
		if length <= 0 {
			continue
		}
		for d := step - carry; d < length; d += step {
			s := d / length
			out = append(out, rl.NewVector2(a.X+s*dx, a.Y+s*dy))
		}
		carry = float32(math.Mod(float64(carry+length), float64(step)))
	}
	return append(out, points[len(points)-1])
}

// smooth rounds the curve with a moving average, keeping the ends anchored so
// a trail that starts off-map still starts off-map.
func smooth(points []rl.Vector2, radius int) []rl.Vector2 {
	if radius < 1 || len(points) < 3 {
		return points
	}
	out := make([]rl.Vector2, len(points))
	for i := range points {
		var sx, sy, n float32
		for j := i - radius; j <= i+radius; j++ {
			if j < 0 || j >= len(points) {
				continue
			}
			sx, sy, n = sx+points[j].X, sy+points[j].Y, n+1
		}
		out[i] = rl.NewVector2(sx/n, sy/n)
	}
	out[0], out[len(out)-1] = points[0], points[len(points)-1]
	return out
}
