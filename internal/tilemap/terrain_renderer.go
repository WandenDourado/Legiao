package tilemap

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	terrainGrass       = 1
	terrainDirt        = 2
	terrainStone       = 3
	terrainDarkGrass   = 4
	terrainSparseGrass = 5
	terrainBareSoil    = 6
	// terrainForestGrass is the healthy forest floor: the FIRST stage of the
	// dark biome's sequence, not a second village grass. It exists because the
	// village grass (1) lives in the green stack, and material from different
	// stacks never paints over each other — so a map that walked from one to
	// the other had a single hard step instead of a ramp, no matter how the
	// boundary was drawn.
	terrainForestGrass = 7
	// terrainForestPath is the trodden path of the forest floor. It exists as
	// its own material, and not as the green stack's dirt (2), for the same
	// reason terrainForestGrass does: a green-stack material inside the dark
	// biome fades against the VILLAGE grass base and prints a bright green halo
	// around itself. In the dark stack it fades against the forest grass, which
	// is the ground actually around it.
	terrainForestPath = 8
	// terrainSiegeGravel and terrainDarkFlagstone are BUILT ground, and they sit
	// in the dark stack rather than in a stack of their own. A stack of their own
	// would have been the obvious reading — pavement is not forest floor — but
	// material from different stacks never paints over each other, so the
	// fortress esplanade would have ended at a single hard step against the dead
	// earth around it, whatever shape it was cut in. In the dark stack the engine
	// ramps it: a flagstone cell draws the gravel under it and the bare soil
	// under that, each layer fading at the border it does not share, and the
	// esplanade reads as stone the dead ground is swallowing at the edge.
	//
	// They inherit the dark stack's edgeWidth of 0.50, which is a biome fade and
	// not the made edge a village pavement gets. That is the intent here: the
	// straight edge of this place comes from the wall art standing on the border,
	// not from the terrain.
	terrainSiegeGravel   = 9
	terrainDarkFlagstone = 10
	// Castle materials are a separate, fully built stack: unlike the fortress
	// esplanade they do not fade into dead forest ground. Map 4 starts inside
	// the castle, so blocks, water and carpet meet on deliberate hard edges.
	terrainCastleBlocks = 11
	terrainCastleWater  = 12
	terrainCastleCarpet = 13
	terrainCastleStone  = 14
)

// terrainOverlays are the materials painted over the grass base, bottom layer
// first. The order across stacks matters as much as within one: the dark biome
// goes down before dirt and stone, which stay the last thing put on the
// ground. See terrain_mask.go for the stacks themselves.
//
// The castle four MUST appear here in the same order they have in their stack
// (stone, blocks, water, carpet). They were listed with stone LAST, and since
// paintedWith gives a cell every material below its own rank, a carpet cell
// paints stone too — drawn last, the stone covered the carpet, the water and
// the blocks, and the whole of map 4 rendered as bare stone floor. The bug was
// dormant only because the first version of world_04 painted zero stone cells.
var terrainOverlays = []int{
	terrainForestPath, terrainForestGrass, terrainDarkGrass, terrainSparseGrass,
	terrainBareSoil, terrainSiegeGravel, terrainDarkFlagstone,
	terrainCastleStone, terrainCastleBlocks, terrainCastleWater, terrainCastleCarpet,
	terrainDirt, terrainStone,
}

// terrainTextureFiles is the texture each material draws with. A material
// missing here is a material that cannot be painted.
var terrainTextureFiles = map[int]string{
	terrainGrass:       "assets/tilesets/terrain_grass.png",
	terrainDirt:        "assets/tilesets/terrain_dirt.png",
	terrainStone:       "assets/tilesets/terrain_stone.png",
	terrainDarkGrass:   "assets/tilesets/terrain_dark_grass.png",
	terrainSparseGrass: "assets/tilesets/terrain_dark_grass_sparse.png",
	terrainBareSoil:    "assets/tilesets/terrain_bare_soil.png",
	terrainForestGrass: "assets/tilesets/terrain_forest_grass.png",
	terrainForestPath:  "assets/tilesets/terrain_dirt.png",

	terrainSiegeGravel:   "assets/tilesets/terrain_siege_gravel.png",
	terrainDarkFlagstone: "assets/tilesets/terrain_dark_flagstone.png",
	terrainCastleBlocks:  "assets/tilesets/terrain_castle_blocks.png",
	terrainCastleWater:   "assets/tilesets/terrain_castle_water.png",
	terrainCastleCarpet:  "assets/tilesets/terrain_castle_carpet.png",
	terrainCastleStone:   "assets/tilesets/terrain_castle_stone.png",
}

// spanMaterials are the materials whose texture covers SEVERAL cells instead of
// one, with the cell picking its window by position.
//
// Why this exists: drawPlain used to squeeze the whole texture into one 128 px
// cell, so a 1254 px reference lost ~60% of its local contrast on the way to
// the screen and every cell of a material looked identical. Raising the source
// resolution cannot fix that — the bottleneck is the 128 px destination, not
// the origin. Spreading the same texture over N×N cells draws it 1:1 and moves
// the repeat period from one cell to N.
//
// Only the dark biome opts in. The green kit was painted to read at one cell,
// so giving it a span would double the size of every blade in world_01 — a
// visual change to an approved map, and a separate decision from this one.
var spanMaterials = map[int]bool{
	terrainDarkGrass: true, terrainSparseGrass: true, terrainBareSoil: true,
	terrainForestGrass: true,
	// Both fortress grounds are 512 px sheets, so they span 4×4 cells 1:1 like
	// the rest of the dark biome. The flagstone needs it most: at one cell per
	// sheet every slab would land in the same place in every cell and the
	// esplanade would read as a printed pattern instead of laid stone.
	terrainSiegeGravel: true, terrainDarkFlagstone: true,
	terrainCastleBlocks: true, terrainCastleWater: true, terrainCastleCarpet: true, terrainCastleStone: true,
}

// neededMaterials expande os materiais que o mapa cita para tudo que o
// desenho deles exige: cada um mais os que ficam ABAIXO dele na propria pilha,
// mais a grama base.
func neededMaterials(used map[int]bool) map[int]bool {
	needed := map[int]bool{terrainGrass: true}
	for material := range used {
		stack, rank, ok := stackRank(material)
		if !ok {
			continue
		}
		for r, m := range terrainStacks[stack].materials {
			if r <= rank {
				needed[m] = true
			}
		}
	}
	return needed
}

// GroundMaterials sao os materiais que a camada `ground` do mapa cita.
func GroundMaterials(m *TiledMap) map[int]bool {
	used := map[int]bool{}
	if m == nil {
		return used
	}
	for _, layer := range m.Layers {
		if layer.Name != "ground" || layer.Type != "tilelayer" {
			continue
		}
		for _, kind := range layer.Data {
			if kind != 0 {
				used[kind] = true
			}
		}
	}
	return used
}

// TerrainRenderer owns base textures and the optional GPU edge blender.
type TerrainRenderer struct {
	textures map[int]rl.Texture2D
	// paths guarda de onde cada textura veio, para devolver a referencia ao
	// cache no Unload.
	paths                   map[int]string
	shader                  rl.Shader
	enabled                 bool
	edgeLoc                 int32
	cornerLoc, edgeWidthLoc int32
	tileRectLoc             int32

	// Caminho em batch (terrain_batch.go): uma troca de shader por MATERIAL
	// em vez de uma por celula. batched so fica true quando o shader carrega,
	// todos os uniforms existem e a grade cabe em 255 celulas por lado; caso
	// contrario o desenho continua pelo caminho por celula acima.
	batchShader  rl.Shader
	batched      bool
	maskTexLoc   int32
	gridSizeLoc  int32
	batchEdgeLoc int32
	spanFLoc     int32
	// masks tem uma textura por material: um texel por celula, com os oito
	// vizinhos empacotados. Construida uma vez por mapa.
	masks map[int]rl.Texture2D
}

// NewTerrainRenderer carrega SO as texturas que este mapa pode pintar.
//
// Antes carregava as treze, sempre: o world_01, que e uma vila verde, pagava
// pela muralha da fortaleza, pelo cascalho de cerco e pelo chao de castelo do
// mapa 4 — 6,8 MB de VRAM presos em textura que aquele mapa nunca desenha. No
// desktop isso e invisivel; no Android e parte do que decide se o jogo abre.
//
// used sao os materiais que a camada `ground` do mapa cita. A PILHA INTEIRA
// abaixo de cada um entra junto, e isso nao e excesso de zelo: o terreno e
// desenhado empilhado, entao uma celula de laje negra pinta o cascalho e a
// terra nua por baixo dela. Faltar uma dessas seria buraco no chao.
//
// A grama base entra sempre, porque o passe de fundo a desenha sob todas as
// celulas de qualquer mapa.
func NewTerrainRenderer(used map[int]bool) *TerrainRenderer {
	needed := neededMaterials(used)
	t := &TerrainRenderer{
		textures: make(map[int]rl.Texture2D, len(needed)),
		paths:    make(map[int]string, len(needed)),
	}
	for material := range needed {
		file, ok := terrainTextureFiles[material]
		if !ok {
			continue
		}
		t.textures[material] = AcquireTexture(file)
		t.paths[material] = file
	}
	vs, fs := terrainShaderPaths()
	t.shader = rl.LoadShader(assets.Path(vs), assets.Path(fs))
	if !rl.IsShaderValid(t.shader) {
		log.Printf("[Tilemap] terrain shader unavailable (%s); using hard-edge terrain fallback", fs)
		return t
	}
	t.edgeLoc = rl.GetShaderLocation(t.shader, "edge")
	t.cornerLoc = rl.GetShaderLocation(t.shader, "corner")
	t.edgeWidthLoc = rl.GetShaderLocation(t.shader, "edgeWidth")
	t.tileRectLoc = rl.GetShaderLocation(t.shader, "tileRect")
	t.enabled = t.edgeLoc >= 0 && t.cornerLoc >= 0 && t.edgeWidthLoc >= 0 && t.tileRectLoc >= 0
	if !t.enabled {
		log.Printf("[Tilemap] terrain shader missing uniforms; using hard-edge terrain fallback")
		rl.UnloadShader(t.shader)
		return t
	}
	// O caminho rapido so entra depois que o de sempre esta de pe: ele e uma
	// otimizacao do mesmo desenho, nao um substituto independente.
	t.initBatch()
	return t
}

func (t *TerrainRenderer) Unload() {
	// Devolve a referencia em vez de descarregar: o mapa de destino pode usar
	// a mesma textura, e nesse caso ela nem chega a sair da VRAM.
	for _, path := range t.paths {
		ReleaseTexture(path)
	}
	t.textures = make(map[int]rl.Texture2D)
	t.paths = make(map[int]string)
	t.unloadMasks()
	if t.enabled {
		rl.UnloadShader(t.shader)
	}
	if t.batched {
		rl.UnloadShader(t.batchShader)
	}
}

// Draw paints the terrain in layers: grass under everything, then each
// overlay material over every cell that carries it OR anything above it.
// Painting in layers (instead of one self-contained quad per cell) is what
// lets stone dissolve INTO the dirt it touches, because the dirt is already on
// screen when the stone is drawn.
// view restricts every pass to the cells the camera shows. It is passed down
// instead of being consulted per cell because Draw makes TEN passes over the
// grid (the grass base plus one per overlay material): with the whole map in
// range those ten passes visited 42.000 cells per frame in world_03 to draw
// the ~127 that fit on screen.
//
// Culling here cannot change what appears: a cell's quad never leaves its own
// cell, and the border mask reads the `ground` layer straight from the map
// rather than what is already on screen, so a neighbour that was skipped still
// counts as linked.
func (t *TerrainRenderer) Draw(layer Layer, tileW, tileH int, view Viewport) {
	grass := t.textureFor(terrainGrass)
	grassSpan := t.spanFor(terrainGrass, grass, tileW)
	t.eachCell(layer, view, func(x, y, kind int) {
		t.drawPlain(grass, x, y, tileW, tileH, grassSpan)
	})
	if !t.enabled {
		t.eachCell(layer, view, func(x, y, kind int) {
			if kind != terrainGrass {
				texture := t.textureFor(kind)
				t.drawPlain(texture, x, y, tileW, tileH, t.spanFor(kind, texture, tileW))
			}
		})
		return
	}
	// Caminho rapido primeiro; ele devolve false quando nao esta disponivel e
	// o de sempre assume, desenhando exatamente o mesmo resultado por um preco
	// maior.
	if t.drawBatchedOverlays(layer, tileW, tileH, view) {
		return
	}
	for _, material := range terrainOverlays {
		t.eachCell(layer, view, func(x, y, kind int) {
			if paintedWith(kind, material) {
				t.drawBlended(layer, x, y, tileW, tileH, material)
			}
		})
	}
}

// eachCell visits every painted cell of the layer INSIDE the viewport, top-left
// to bottom-right.
func (t *TerrainRenderer) eachCell(layer Layer, view Viewport, visit func(x, y, kind int)) {
	minX, minY, maxX, maxY := view.CellRange(layer.Width, layer.Height)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			i := y*layer.Width + x
			if i < 0 || i >= len(layer.Data) || layer.Data[i] == 0 {
				continue
			}
			frameStats.CellsVisited++
			visit(x, y, layer.Data[i])
		}
	}
}

// spanFor is how many cells the material's texture covers on a side.
//
// It is derived from the texture itself rather than declared: a 512 px sheet
// against a 128 px cell is 4×4, a 256 px sheet is 2×2. So dropping in higher
// resolution art raises the span by itself, and art that is not big enough
// never gets magnified — span only grows while whole cells fit.
func (t *TerrainRenderer) spanFor(material int, texture rl.Texture2D, tileW int) int {
	if !spanMaterials[material] || tileW <= 0 {
		return 1
	}
	span := int(texture.Width) / tileW
	if span < 1 {
		return 1
	}
	return span
}

// tileWindow is the source rectangle this cell samples, plus the same window in
// UV for the shader. With span 1 it is the whole texture, which is what every
// caller did before spans existed.
func tileWindow(texture rl.Texture2D, x, y, span int) (rl.Rectangle, []float32) {
	w := float32(texture.Width) / float32(span)
	h := float32(texture.Height) / float32(span)
	// Go's % keeps the sign of the dividend, and a cell index is never negative
	// here — but the map origin is the one thing a future caller could move, so
	// normalise instead of trusting it.
	wx := float32(((x % span) + span) % span)
	wy := float32(((y % span) + span) % span)
	src := rl.NewRectangle(wx*w, wy*h, w, h)
	uv := []float32{wx / float32(span), wy / float32(span), 1 / float32(span), 1 / float32(span)}
	return src, uv
}

func (t *TerrainRenderer) drawPlain(texture rl.Texture2D, x, y, tileW, tileH, span int) {
	frameStats.TerrainQuads++
	src, _ := tileWindow(texture, x, y, span)
	dest := rl.NewRectangle(float32(x*tileW), float32(y*tileH), float32(tileW), float32(tileH))
	rl.DrawTexturePro(texture, src, dest, rl.Vector2{}, 0, rl.White)
}

// drawBlended draws one cell of material with its borders faded to
// transparent, so it blends with whatever was painted underneath.
// Begin/EndShaderMode wraps each cell because the uniforms change per cell and
// the batch has to be flushed between them.
func (t *TerrainRenderer) drawBlended(layer Layer, x, y, tileW, tileH, material int) {
	texture := t.textureFor(material)
	span := t.spanFor(material, texture, tileW)
	_, uv := tileWindow(texture, x, y, span)
	rl.SetShaderValue(t.shader, t.edgeLoc, edgeMask(layer, x, y, material), rl.ShaderUniformVec4)
	rl.SetShaderValue(t.shader, t.cornerLoc, cornerMask(layer, x, y, material), rl.ShaderUniformVec4)
	rl.SetShaderValue(t.shader, t.edgeWidthLoc, []float32{edgeWidthFor(material)}, rl.ShaderUniformFloat)
	// Without this the edge fade would read fragTexCoord as if it spanned the
	// whole cell, and any material with a span would lose its border blend.
	rl.SetShaderValue(t.shader, t.tileRectLoc, uv, rl.ShaderUniformVec4)
	frameStats.ShaderBinds++
	rl.BeginShaderMode(t.shader)
	t.drawPlain(texture, x, y, tileW, tileH, span)
	rl.EndShaderMode()
}

// textureFor returns the material's texture, falling back to grass so an
// unmapped material draws as plain ground instead of nothing at all.
func (t *TerrainRenderer) textureFor(material int) rl.Texture2D {
	if texture, ok := t.textures[material]; ok {
		return texture
	}
	return t.textures[terrainGrass]
}
