package tilemap

import rl "github.com/gen2brain/raylib-go/raylib"

// footprintIndex stores manifest collision footprints as the rectangles they
// actually are, bucketed by grid cell so a lookup stays local.
//
// They used to be quantised into the 128 px cell grid, and that is what made
// collision stop matching the art: a cell became solid once a footprint
// covered half of it, so a house blocked up to half a cell (64 px) of open
// ground past its own wall, and the player bumped into nothing. Props are
// authored in pixels, so they are stored in pixels; only the hand-painted
// `collision` tile layer is a grid, because there it is one.
type footprintIndex struct {
	rects                 []rl.Rectangle
	byCell                map[cellKey][]int32
	disabled              map[int32]bool
	tileWidth, tileHeight float32
}

// cellKey identifies one bucket. It is a struct rather than packed bits
// because a footprint may hang off the map edge into a negative cell, and
// packing would have to be 32-bit-safe for the Android builds.
type cellKey struct{ col, row int32 }

func newFootprintIndex(tileWidth, tileHeight int) *footprintIndex {
	return &footprintIndex{
		byCell: map[cellKey][]int32{}, disabled: map[int32]bool{},
		tileWidth: float32(tileWidth), tileHeight: float32(tileHeight),
	}
}

// Add registers one footprint rectangle and indexes every cell it touches.
func (f *footprintIndex) Add(rect rl.Rectangle) {
	if f == nil || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	id := int32(len(f.rects))
	f.rects = append(f.rects, rect)
	left, top := int(rect.X/f.tileWidth), int(rect.Y/f.tileHeight)
	right := int((rect.X + rect.Width - epsilon) / f.tileWidth)
	bottom := int((rect.Y + rect.Height - epsilon) / f.tileHeight)
	for row := top; row <= bottom; row++ {
		for col := left; col <= right; col++ {
			key := cellKey{col: int32(col), row: int32(row)}
			f.byCell[key] = append(f.byCell[key], id)
		}
	}
}

// epsilon keeps a box whose edge lands exactly on a cell boundary from
// claiming the next cell along.
const epsilon = 0.01

// Collides reports whether a box overlaps any footprint. Only the cells the
// box touches are consulted, so cost is proportional to the box, not to how
// many props the map has.
func (f *footprintIndex) Collides(box rl.Rectangle) bool {
	if f == nil || len(f.rects) == 0 {
		return false
	}
	left, top := int(box.X/f.tileWidth), int(box.Y/f.tileHeight)
	right := int((box.X + box.Width - epsilon) / f.tileWidth)
	bottom := int((box.Y + box.Height - epsilon) / f.tileHeight)
	for row := top; row <= bottom; row++ {
		for col := left; col <= right; col++ {
			for _, id := range f.byCell[cellKey{col: int32(col), row: int32(row)}] {
				if f.disabled[id] {
					continue
				}
				if rl.CheckCollisionRecs(box, f.rects[id]) {
					return true
				}
			}
		}
	}
	return false
}

// SetEnabledOverlapping toggles every footprint that overlaps area. It returns
// true only when a footprint actually changed state, so callers can avoid
// rebuilding dependent collision snapshots every frame.
func (f *footprintIndex) SetEnabledOverlapping(area rl.Rectangle, enabled bool) bool {
	if f == nil {
		return false
	}
	changed := false
	for id, rect := range f.rects {
		if !rl.CheckCollisionRecs(rect, area) {
			continue
		}
		key := int32(id)
		if enabled == !f.disabled[key] {
			continue
		}
		f.disabled[key] = !enabled
		changed = true
	}
	return changed
}

// All exposes the footprints for projectile checks and the F3 overlay.
func (f *footprintIndex) All() []rl.Rectangle {
	if f == nil {
		return nil
	}
	result := make([]rl.Rectangle, 0, len(f.rects))
	for id, rect := range f.rects {
		if !f.disabled[int32(id)] {
			result = append(result, rect)
		}
	}
	return result
}
