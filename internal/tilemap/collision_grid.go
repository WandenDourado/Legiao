package tilemap

import rl "github.com/gen2brain/raylib-go/raylib"

// CollisionGrid answers what is solid on a map. It holds two things that are
// deliberately not the same shape:
//
//   - the hand-painted `collision` tile layer, which is authored per cell and
//     so is stored as cells;
//   - manifest collision footprints, which are authored in pixels against the
//     art and so are stored as pixel rectangles (see footprintIndex).
//
// Forcing the second into the first is what used to make a house block open
// ground past its own wall.
type CollisionGrid struct {
	Width, Height         int
	TileWidth, TileHeight int
	solid                 []bool
	footprints            *footprintIndex
}

// NewCollisionGrid reads the invisible Tiled tile layer named "collision".
func NewCollisionGrid(m *TiledMap) *CollisionGrid {
	grid := &CollisionGrid{
		Width: m.Width, Height: m.Height,
		TileWidth: m.TileWidth, TileHeight: m.TileHeight,
		solid:      make([]bool, m.Width*m.Height),
		footprints: newFootprintIndex(m.TileWidth, m.TileHeight),
	}
	for _, layer := range m.Layers {
		if layer.Name != "collision" || layer.Type != "tilelayer" {
			continue
		}
		for i, gid := range layer.Data {
			if i < len(grid.solid) && gid != 0 {
				grid.solid[i] = true
			}
		}
		break
	}
	grid.addManifestFootprints(m)
	return grid
}

// addManifestFootprints registers every manifest's explicit collision
// footprint as the rectangle it was measured as.
func (g *CollisionGrid) addManifestFootprints(m *TiledMap) {
	for _, src := range manifestSources {
		manifest, err := loadAssetManifest(src.Path)
		if err != nil {
			continue
		}
		for _, layer := range m.Layers {
			if layer.Name != src.Layer || layer.Type != "objectgroup" {
				continue
			}
			for _, object := range layer.Objects {
				piece, ok := manifest.Pieces[object.Name]
				if !ok {
					continue
				}
				for _, footprint := range piece.Footprints() {
					g.footprints.Add(rl.NewRectangle(
						object.X+footprint.OffsetX,
						object.Y+footprint.OffsetY,
						footprint.Width,
						footprint.Height,
					))
				}
			}
		}
	}
}

// CollidesCentered reports whether a box centered on pos overlaps solid space,
// checking the painted cells it touches and the prop footprints it overlaps.
func (g *CollisionGrid) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	if g == nil {
		return false
	}
	box := rl.NewRectangle(pos.X-width/2, pos.Y-height/2, width, height)
	return g.collidesCells(box) || g.footprints.Collides(box)
}

func (g *CollisionGrid) collidesCells(box rl.Rectangle) bool {
	left := int(box.X / float32(g.TileWidth))
	right := int((box.X + box.Width - epsilon) / float32(g.TileWidth))
	top := int(box.Y / float32(g.TileHeight))
	bottom := int((box.Y + box.Height - epsilon) / float32(g.TileHeight))
	for y := top; y <= bottom; y++ {
		for x := left; x <= right; x++ {
			if x >= 0 && x < g.Width && y >= 0 && y < g.Height && g.solid[y*g.Width+x] {
				return true
			}
		}
	}
	return false
}

// Rects exposes every solid for projectile obstacle checks: painted cells as
// full cells, prop footprints as the rectangles they are.
func (g *CollisionGrid) Rects() []rl.Rectangle {
	rects := make([]rl.Rectangle, 0)
	if g == nil {
		return rects
	}
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			if g.solid[y*g.Width+x] {
				rects = append(rects, rl.NewRectangle(float32(x*g.TileWidth), float32(y*g.TileHeight), float32(g.TileWidth), float32(g.TileHeight)))
			}
		}
	}
	return append(rects, g.footprints.All()...)
}

// SetFootprintsEnabledOverlapping opens or closes the authored collision
// footprints inside area. It is used for map gates whose art stays in place
// while their passage changes with stage progression.
func (g *CollisionGrid) SetFootprintsEnabledOverlapping(area rl.Rectangle, enabled bool) bool {
	if g == nil {
		return false
	}
	return g.footprints.SetEnabledOverlapping(area, enabled)
}

// DrawDebug outlines every solid while the F3 overlay is active, painted cells
// and prop footprints alike. Safe on a nil grid.
// DrawDebug pinta o espaco solido, e so o que a camera mostra.
//
// O culling aqui nao e economia de quadro comum: e o F3 que LIGA este desenho,
// entao sem ele o proprio ato de medir custava mais de mil retangulos por
// quadro (o world_02 pinta 1.132 celulas de colisao e o world_03 tem 802 mais
// os footprints dos props). O medidor mediria a si mesmo.
func (g *CollisionGrid) DrawDebug(view Viewport) {
	if g == nil || !debugOverlay {
		return
	}
	minX, minY, maxX, maxY := view.CellRange(g.Width, g.Height)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if !g.solid[y*g.Width+x] {
				continue
			}
			px, py := int32(x*g.TileWidth), int32(y*g.TileHeight)
			rl.DrawRectangle(px, py, int32(g.TileWidth), int32(g.TileHeight), rl.NewColor(255, 80, 80, 60))
			rl.DrawRectangleLines(px, py, int32(g.TileWidth), int32(g.TileHeight), rl.Red)
		}
	}
	for _, r := range g.footprints.All() {
		if !view.Intersects(r) {
			continue
		}
		rl.DrawRectangle(int32(r.X), int32(r.Y), int32(r.Width), int32(r.Height), rl.NewColor(255, 80, 80, 60))
		rl.DrawRectangleLines(int32(r.X), int32(r.Y), int32(r.Width), int32(r.Height), rl.Red)
	}
}

// DrawEntityFootprintDebug outlines a centered entity footprint and marks
// whether it currently overlaps solid space. Safe on a nil grid.
func (g *CollisionGrid) DrawEntityFootprintDebug(pos rl.Vector2, width, height float32) {
	if !debugOverlay {
		return
	}
	color := rl.Lime
	if g.CollidesCentered(pos, width, height) {
		color = rl.Orange
	}
	rl.DrawRectangleLines(int32(pos.X-width/2), int32(pos.Y-height/2), int32(width), int32(height), color)
	rl.DrawLine(int32(pos.X-8), int32(pos.Y), int32(pos.X+8), int32(pos.Y), color)
	rl.DrawLine(int32(pos.X), int32(pos.Y-8), int32(pos.X), int32(pos.Y+8), color)
}
