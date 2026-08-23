package tilemap

import (
	"log"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// MapRenderer holds the parsed map data and loaded tileset textures.
type MapRenderer struct {
	Map       *TiledMap
	Textures  map[string]rl.Texture2D
	Terrain   *TerrainRenderer
	Trail     *TrailRenderer
	trails    []Trail
	Manifests []*ManifestRenderer
	// Collision is optional and used only by the F3 debug overlay.
	Collision *CollisionGrid
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
		// Dois tilesets do mesmo mapa podem citar a MESMA imagem. `Acquire`
		// conta cada chamada, mas `mr.Textures` e indexado por caminho e
		// guardaria uma entrada so — o `Unload` devolveria uma referencia para
		// duas tomadas, e a textura ficaria presa na VRAM para o resto da
		// sessao. Uma referencia por caminho, entao.
		if _, already := mr.Textures[ts.ImagePath]; already {
			continue
		}
		tex := AcquireTexture(ts.ImagePath)
		if !rl.IsTextureValid(tex) {
			ReleaseTexture(ts.ImagePath)
			continue
		}
		mr.Textures[ts.ImagePath] = tex
	}
	// SO os materiais que este mapa cita, mais as camadas sob eles.
	mr.Terrain = NewTerrainRenderer(GroundMaterials(mr.Map))
	// As mascaras de vizinhanca sao dado do MAPA: construidas aqui, uma vez,
	// em vez de recalculadas a cada quadro dentro do desenho.
	for _, layer := range mr.Map.Layers {
		if layer.Name == "ground" && layer.Type == "tilelayer" {
			mr.Terrain.PrepareMasks(layer)
		}
	}
	mr.trails = Trails(mr.Map)
	if len(mr.trails) > 0 {
		mr.Trail = NewTrailRenderer()
		// A curva e dado do MAPA: resolvida uma vez aqui em vez de a cada
		// quadro dentro do desenho, que era onde ela estava.
		mr.Trail.Prepare(mr.trails)
	}
	// SO os manifestos cujas pecas este mapa realmente coloca.
	mr.Manifests = NewManifestRenderers(mr.Map)
	log.Printf("[Tilemap] mapa carregado: %d tilesets, %d manifestos, %d texturas na VRAM",
		len(mr.Textures), len(mr.Manifests), TextureCacheSize())
}

// Unload releases all loaded tileset textures.
func (mr *MapRenderer) Unload() {
	for path := range mr.Textures {
		ReleaseTexture(path)
	}
	mr.Textures = make(map[string]rl.Texture2D)
	if mr.Terrain != nil {
		mr.Terrain.Unload()
	}
	mr.Trail.Unload()
	mr.Trail = nil
	mr.trails = nil
	for _, m := range mr.Manifests {
		m.Unload()
	}
	// Zerados depois de devolvidos: um `Unload` chamado duas vezes no mesmo
	// renderer (um destino que nao carrega e deixa o mundo antigo de pe, por
	// exemplo) devolveria referencias que ele ja nao tem, e ai a textura sairia
	// da VRAM com alguem ainda desenhando com ela.
	mr.Terrain = nil
	mr.Manifests = nil
}

// viewport is what the camera shows, in world units and in cells. Every draw
// pass is restricted to it: see viewport.go for why.
func (mr *MapRenderer) viewport(camera rl.Camera2D) Viewport {
	return NewViewport(camera,
		float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight()),
		mr.Map.TileWidth, mr.Map.TileHeight)
}

// Draw renders all visible layers in order (bottom to top) inside a single
// BeginMode2D/EndMode2D block.
func (mr *MapRenderer) Draw(camera rl.Camera2D) {
	beginFrameStats()
	rl.BeginMode2D(camera)
	mr.drawVisibleLayers(mr.viewport(camera))
	rl.EndMode2D()
	endFrameStats()
}

// DrawWithCamera renders bottom layers, then calls drawEntities, then renders
// top layers — all inside a single BeginMode2D/EndMode2D block.
// drawEntities is the callback where the caller draws players, enemies, etc.
func (mr *MapRenderer) DrawWithCamera(camera rl.Camera2D, stopBefore, startFrom string, drawEntities func()) {
	beginFrameStats()
	view := mr.viewport(camera)
	rl.BeginMode2D(camera)

	start := time.Now()
	mr.drawLayersUpTo(stopBefore, view)
	mapTime := time.Since(start)

	if rl.IsKeyPressed(rl.KeyF3) {
		ToggleDebug()
	}

	start = time.Now()
	drawEntities()
	frameStats.EntityMS = milliseconds(time.Since(start))

	start = time.Now()
	mr.drawLayersFrom(startFrom, view)
	frameStats.MapMS = milliseconds(mapTime + time.Since(start))

	rl.EndMode2D()
	endFrameStats()
}

func (mr *MapRenderer) drawVisibleLayers(view Viewport) {
	for _, layer := range mr.Map.Layers {
		mr.drawLayer(layer, view)
	}
	mr.drawManifests("structures_back", view)
	mr.drawManifests("foreground", view)
	mr.Collision.DrawDebug(view)
}

func (mr *MapRenderer) drawLayersUpTo(stopBefore string, view Viewport) {
	for _, layer := range mr.Map.Layers {
		if layer.Name == stopBefore {
			break
		}
		mr.drawLayer(layer, view)
	}
	mr.drawManifests("structures_back", view)
	// Collision cells are drawn before entities so the player footprint,
	// drawn by the caller, stays readable on top of them.
	mr.Collision.DrawDebug(view)
}

func (mr *MapRenderer) drawLayersFrom(startFrom string, view Viewport) {
	started := false
	for _, layer := range mr.Map.Layers {
		if layer.Name == startFrom {
			started = true
		}
		if started {
			mr.drawLayer(layer, view)
		}
	}
	mr.drawManifests("foreground", view)
}

func (mr *MapRenderer) drawManifests(role string, view Viewport) {
	for _, m := range mr.Manifests {
		m.Draw(mr.Map, role, view)
	}
}

func (mr *MapRenderer) drawLayer(layer Layer, view Viewport) {
	if !layer.Visible || layer.Type != "tilelayer" {
		return
	}
	if layer.Name == "ground" && mr.Terrain != nil {
		mr.Terrain.Draw(layer, mr.Map.TileWidth, mr.Map.TileHeight, view)
		// Trails belong to the ground: on top of the terrain, under the
		// detail scattered over it and under everything standing on it.
		mr.Trail.Draw(mr.trails, view)
		return
	}
	mr.drawTileLayer(layer, view)
}

func (mr *MapRenderer) drawTileLayer(layer Layer, view Viewport) {
	tileW := float32(mr.Map.TileWidth)
	tileH := float32(mr.Map.TileHeight)

	minX, minY, maxX, maxY := view.CellRange(layer.Width, layer.Height)
	for row := minY; row <= maxY; row++ {
		for col := minX; col <= maxX; col++ {
			idx := row*layer.Width + col
			if idx >= len(layer.Data) || idx < 0 {
				continue
			}
			frameStats.CellsVisited++
			gid := layer.Data[idx]
			if gid == 0 {
				continue
			}
			if mr.manifestOwnsGID(gid) {
				continue // Manifest-owned art is drawn through its explicit manifest.
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
			frameStats.Tiles++
			rl.DrawTextureRec(tex, srcRect, rl.NewVector2(dstX, dstY), rl.White)
		}
	}
}

func (mr *MapRenderer) manifestOwnsGID(gid int) bool {
	for _, m := range mr.Manifests {
		if m.UsesGID(mr.Map, gid) {
			return true
		}
	}
	return false
}
