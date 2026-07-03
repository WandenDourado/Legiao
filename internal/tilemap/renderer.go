package tilemap

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// MapRenderer holds the parsed map data and loaded tileset textures.
type MapRenderer struct {
	Map      *TiledMap
	Textures map[string]rl.Texture2D
}

// NewMapRenderer creates a new renderer for the given TiledMap.
func NewMapRenderer(tiledMap *TiledMap) *MapRenderer {
	return &MapRenderer{
		Map:      tiledMap,
		Textures: make(map[string]rl.Texture2D),
	}
}

// Load loads all tileset textures referenced by the map.
// Textures that fail to load are logged and skipped.
func (mr *MapRenderer) Load() {
	for _, ts := range mr.Map.Tilesets {
		if ts.ImagePath == "" {
			continue
		}
		tex := rl.LoadTexture(assets.Path(ts.ImagePath))
		if !rl.IsTextureValid(tex) {
			log.Printf("[Tilemap] WARNING: failed to load tileset texture: %s", ts.ImagePath)
			continue
		}
		mr.Textures[ts.ImagePath] = tex
		log.Printf("[Tilemap] Loaded tileset texture: %s", ts.ImagePath)
	}
}

// Unload releases all loaded tileset textures.
func (mr *MapRenderer) Unload() {
	for _, tex := range mr.Textures {
		rl.UnloadTexture(tex)
	}
	mr.Textures = make(map[string]rl.Texture2D)
}

// Draw renders all visible layers in order (bottom to top) inside a single
// BeginMode2D/EndMode2D block.
func (mr *MapRenderer) Draw(camera rl.Camera2D) {
	rl.BeginMode2D(camera)
	mr.drawVisibleLayers()
	rl.EndMode2D()
}

// DrawWithCamera renders bottom layers, then calls drawEntities, then renders
// top layers — all inside a single BeginMode2D/EndMode2D block.
// drawEntities is the callback where the caller draws players, enemies, etc.
func (mr *MapRenderer) DrawWithCamera(camera rl.Camera2D, stopBefore, startFrom string, drawEntities func()) {
	rl.BeginMode2D(camera)
	mr.drawLayersUpTo(stopBefore)
	drawEntities()
	mr.drawLayersFrom(startFrom)
	rl.EndMode2D()
}

func (mr *MapRenderer) drawVisibleLayers() {
	for _, layer := range mr.Map.Layers {
		mr.drawLayer(layer)
	}
}

func (mr *MapRenderer) drawLayersUpTo(stopBefore string) {
	for _, layer := range mr.Map.Layers {
		if layer.Name == stopBefore {
			break
		}
		mr.drawLayer(layer)
	}
}

func (mr *MapRenderer) drawLayersFrom(startFrom string) {
	started := false
	for _, layer := range mr.Map.Layers {
		if layer.Name == startFrom {
			started = true
		}
		if started {
			mr.drawLayer(layer)
		}
	}
}

func (mr *MapRenderer) drawLayer(layer Layer) {
	if !layer.Visible || layer.Type != "tilelayer" {
		return
	}
	mr.drawTileLayer(layer)
}

func (mr *MapRenderer) drawTileLayer(layer Layer) {
	tileW := float32(mr.Map.TileWidth)
	tileH := float32(mr.Map.TileHeight)

	for row := 0; row < layer.Height; row++ {
		for col := 0; col < layer.Width; col++ {
			idx := row*layer.Width + col
			if idx >= len(layer.Data) {
				continue
			}
			gid := layer.Data[idx]
			if gid == 0 {
				continue
			}
			ts, ok := mr.Map.TilesetForGID(gid)
			if !ok {
				continue
			}
			localID := gid - ts.FirstGID
			srcX := float32((localID % ts.Columns) * ts.TileWidth)
			srcY := float32((localID / ts.Columns) * ts.TileHeight)
			srcRect := rl.NewRectangle(srcX, srcY, tileW, tileH)
			dstX := float32(col) * tileW
			dstY := float32(row) * tileH
			tex, ok := mr.Textures[ts.ImagePath]
			if !ok {
				continue
			}
			rl.DrawTextureRec(tex, srcRect, rl.NewVector2(dstX, dstY), rl.White)
		}
	}
}
