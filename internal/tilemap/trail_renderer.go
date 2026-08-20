package tilemap

import (
	"log"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// trailTexturePath is the ribbon art: RGB from the reference, alpha from the
// path profile measured out of it (work/tiled-assets/build_trail.py).
const trailTexturePath = "assets/tilesets/terrain_trail.png"

// TrailRenderer draws each trail as a strip of textured quads along its curve.
//
// Quads, and not a per-pixel distance field, because of what the field does at
// a bend: asking every pixel for its nearest point on the curve makes a whole
// fan of pixels on the outside of a turn answer with the same vertex, so the
// distance travelled stalls there and the texture pinwheels into rings. With
// geometry the coordinate along the path is one number per vertex, linear
// between them by construction, and the rings cannot happen.
type TrailRenderer struct {
	texture rl.Texture2D
	loaded  bool
	// paths sao as curvas ja reamostradas e suavizadas, uma por trilha.
	//
	// Ficam aqui porque Trail.Path REFAZ o trabalho inteiro: reamostra a
	// polilinha a cada 24 px e suaviza o resultado, alocando dois slices de
	// centenas de rl.Vector2. Isso rodava A CADA QUADRO, para uma curva que
	// nao muda enquanto o mapa nao muda - lixo puro entregue ao GC 60 vezes
	// por segundo, e o suspeito numero um dos picos isolados de 43 ms que o
	// medidor do F3 registrou nos mapas COM trilha.
	paths [][]rl.Vector2
}

func NewTrailRenderer() *TrailRenderer {
	texture := AcquireTexture(trailTexturePath)
	if !rl.IsTextureValid(texture) {
		log.Printf("[Tilemap] %s nao carregou; trilhas nao serao desenhadas", trailTexturePath)
		ReleaseTexture(trailTexturePath)
		return &TrailRenderer{}
	}
	// The ribbon repeats the texture along the path, so v runs past 1.
	rl.SetTextureWrap(texture, rl.WrapRepeat)
	rl.SetTextureFilter(texture, rl.FilterBilinear)
	return &TrailRenderer{texture: texture, loaded: true}
}

func (r *TrailRenderer) Unload() {
	if r != nil && r.loaded {
		ReleaseTexture(trailTexturePath)
		r.loaded = false
	}
	if r != nil {
		r.paths = nil
	}
}

// Prepare resolve as curvas uma vez, no carregamento do mapa. Chamado por
// MapRenderer.Load junto com as texturas, porque a curva e dado do mapa e tem
// o mesmo tempo de vida que ele.
func (r *TrailRenderer) Prepare(trails []Trail) {
	if r == nil {
		return
	}
	r.paths = make([][]rl.Vector2, len(trails))
	for i, trail := range trails {
		r.paths[i] = trail.Path(trailStep)
	}
}

// Draw lays every trail down on the ground, after the terrain and before
// anything standing on it.
func (r *TrailRenderer) Draw(trails []Trail, view Viewport) {
	if r == nil || !r.loaded || len(trails) == 0 {
		return
	}
	rl.SetTexture(r.texture.ID)
	rl.Begin(rl.Quads)
	for i, trail := range trails {
		if i < len(r.paths) {
			r.ribbon(trail, r.paths[i], view)
		}
	}
	rl.End()
	rl.SetTexture(0)
}

// ribbon emits the quads of one trail. Each quad reuses the previous quad's
// far edge, so the two never overlap: an overlap would blend the semi
// transparent art against itself and print a bright rung at every joint.
//
// Um quad fora da tela e PULADO mas o percurso continua sendo percorrido: o
// `travelled` e a aresta anterior tem de avancar de qualquer jeito, senao a
// textura saltaria ao reentrar na tela e a emenda abriria. Vale a pena cullar
// aqui apesar de tudo caber num batch so, porque cada quad custa doze chamadas
// de cgo (cor, coordenada e vertice por canto) e uma trilha longa emite
// centenas deles por quadro.
func (r *TrailRenderer) ribbon(trail Trail, path []rl.Vector2, view Viewport) {
	if len(path) < 2 {
		return
	}
	half := trail.Width / 2
	leftA, rightA := offsets(path[0], direction(path, 0), half)
	travelled := float32(0)

	for i := 0; i+1 < len(path); i++ {
		leftB, rightB := offsets(path[i+1], direction(path, i+1), half)
		next := travelled + distance(path[i], path[i+1])
		// v repeats every ribbon width, so the art keeps its proportions
		// however long the path is.
		vA, vB := travelled/trail.Width, next/trail.Width

		if view.Intersects(quadBounds(leftA, rightA, leftB, rightB)) {
			frameStats.TrailQuads++
			rl.Color4ub(255, 255, 255, 255)
			rl.TexCoord2f(0, vA)
			rl.Vertex2f(leftA.X, leftA.Y)
			rl.TexCoord2f(0, vB)
			rl.Vertex2f(leftB.X, leftB.Y)
			rl.TexCoord2f(1, vB)
			rl.Vertex2f(rightB.X, rightB.Y)
			rl.TexCoord2f(1, vA)
			rl.Vertex2f(rightA.X, rightA.Y)
		}

		leftA, rightA, travelled = leftB, rightB, next
	}
}

// quadBounds is the axis-aligned box around the four corners of one ribbon
// quad. A quad is not axis aligned, so this over-estimates on a diagonal — the
// right way round for a visibility test, which must never drop something that
// is on screen.
func quadBounds(a, b, c, d rl.Vector2) rl.Rectangle {
	minX := math.Min(math.Min(float64(a.X), float64(b.X)), math.Min(float64(c.X), float64(d.X)))
	maxX := math.Max(math.Max(float64(a.X), float64(b.X)), math.Max(float64(c.X), float64(d.X)))
	minY := math.Min(math.Min(float64(a.Y), float64(b.Y)), math.Min(float64(c.Y), float64(d.Y)))
	maxY := math.Max(math.Max(float64(a.Y), float64(b.Y)), math.Max(float64(c.Y), float64(d.Y)))
	return rl.NewRectangle(float32(minX), float32(minY), float32(maxX-minX), float32(maxY-minY))
}

// direction is the tangent at a point, averaged across the joint so the two
// quads meeting there agree on where their shared edge points.
func direction(path []rl.Vector2, i int) rl.Vector2 {
	var dx, dy float32
	if i > 0 {
		dx += path[i].X - path[i-1].X
		dy += path[i].Y - path[i-1].Y
	}
	if i+1 < len(path) {
		dx += path[i+1].X - path[i].X
		dy += path[i+1].Y - path[i].Y
	}
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length == 0 {
		return rl.NewVector2(0, 1)
	}
	return rl.NewVector2(dx/length, dy/length)
}

// offsets are the two edge points of the ribbon at a point of the curve.
func offsets(at, dir rl.Vector2, half float32) (left, right rl.Vector2) {
	nx, ny := -dir.Y, dir.X
	return rl.NewVector2(at.X+nx*half, at.Y+ny*half),
		rl.NewVector2(at.X-nx*half, at.Y-ny*half)
}

func distance(a, b rl.Vector2) float32 {
	return float32(math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y)))
}
