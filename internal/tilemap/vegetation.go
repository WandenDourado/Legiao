package tilemap

import (
	"encoding/json"
	"log"
	"path/filepath"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// manifestSources lists every asset manifest and the Tiled objectgroup layer
// whose named objects it renders. Adding a category is one entry here plus
// its manifest file; no code may infer atlas rectangles from grid arithmetic.
var manifestSources = []struct {
	Path  string
	Layer string
}{
	{"assets/vegetation_manifest.json", "vegetation"},
	{"assets/forest_manifest.json", "vegetation"},
	{"assets/forest_pine_manifest.json", "vegetation"},
	{"assets/forest_pine_dark_manifest.json", "vegetation"},
	{"assets/forest_dark_props_manifest.json", "vegetation"},
	// A muralha da fortaleza (mapa 3) vive na camada `buildings`, e nao em
	// `vegetation`: ela e construcao, e a auditoria de layout trata as duas
	// camadas por regras diferentes — nada brota em chao construido, mas
	// construcao em cima dele e justamente o esperado.
	{"assets/fortress_wall_manifest.json", "buildings"},
	// Castelo da Senhora das Trevas (mapa 4): atlas preparado a partir das
	// folhas-fonte no pipeline work/castle-assets.
	{"assets/castle_manifest.json", "buildings"},
	// As defesas de campo (mapa 3) vivem em `props`: nao sao vegetacao nem
	// construcao fixa, e a camada separada deixa a auditoria tratar as tres
	// por regras diferentes.
	{"assets/siege_defenses_manifest.json", "props"},
	{"assets/buildings_manifest.json", "buildings"},
	{"assets/fences_manifest.json", "fences"},
}

type assetManifest struct {
	Atlas  string                   `json:"atlas"`
	Pieces map[string]manifestPiece `json:"pieces"`
}

// manifestFootprint is one blocked rectangle, in pixels relative to the
// object's world anchor.
type manifestFootprint struct {
	OffsetX, OffsetY, Width, Height float32
}

type manifestPiece struct {
	Source    struct{ X, Y, Width, Height float32 } `json:"source"`
	Anchor    struct{ X, Y float32 }                `json:"anchor"`
	Role      string                                `json:"role"`
	Collision bool                                  `json:"collision"`
	// CollisionFootprint is the single-rectangle form, which fits anything
	// whose blocked ground is one box: a house, a trunk, a stump.
	CollisionFootprint manifestFootprint `json:"collisionFootprint"`
	// CollisionFootprints is the list form, for pieces whose blocked ground
	// is not one box. A fence corner is an L and an open gate is two posts
	// with a walkable gap; describing either as a single rectangle would
	// block the empty inside of the corner and seal the gate.
	CollisionFootprints []manifestFootprint `json:"collisionFootprints"`
}

// Footprints returns every blocked rectangle of a piece, accepting either
// manifest form. The singular field stays valid so existing manifests are
// untouched by the arrival of the plural one.
func (p manifestPiece) Footprints() []manifestFootprint {
	if !p.Collision {
		return nil
	}
	if len(p.CollisionFootprints) > 0 {
		return p.CollisionFootprints
	}
	return []manifestFootprint{p.CollisionFootprint}
}

// debugOverlay is the single F3 toggle shared by every debug drawing routine
// (manifest pieces, collision grid, entity footprints).
var debugOverlay bool

// DebugEnabled reports whether the F3 debug overlay is active, so callers
// outside this package can draw their own debug shapes in the same pass.
func DebugEnabled() bool { return debugOverlay }

// ToggleDebug flips the debug overlay.
func ToggleDebug() { debugOverlay = !debugOverlay }

// ManifestRenderer draws named map objects using an explicit asset manifest.
type ManifestRenderer struct {
	layer    string
	manifest assetManifest
	texture  rl.Texture2D
	// atlasPath e de onde a textura veio, para devolver a referencia ao cache.
	atlasPath string
}

func loadAssetManifest(path string) (*assetManifest, error) {
	data, err := readFile(assets.Path(path))
	if err != nil {
		return nil, err
	}
	var manifest assetManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// NewManifestRenderers builds one renderer per manifest THIS MAP actually uses.
//
// Carregava os nove atlas em todo mapa: 71,9 MB de VRAM, dos quais o world_01
// (uma vila verde) usa 27,5. Ele pagava pela muralha da fortaleza (16 MB), pelas
// defesas de cerco (16 MB) e pelos props de mata escura, que nunca desenha.
//
// O filtro e "o mapa cita alguma peca deste manifesto pelo nome". Nao da para
// filtrar por camada: `vegetation` e dividida por cinco manifestos, e Draw ja
// ignora objeto cujo nome nao esta no dele.
//
// A missing or invalid manifest is logged and skipped.
func NewManifestRenderers(m *TiledMap) []*ManifestRenderer {
	renderers := make([]*ManifestRenderer, 0, len(manifestSources))
	for _, src := range manifestSources {
		manifest, err := loadAssetManifest(src.Path)
		if err != nil {
			log.Printf("[Tilemap] manifest unavailable (%s): %v", src.Path, err)
			continue
		}
		if !manifestUsedBy(m, src.Layer, *manifest) {
			continue
		}
		r := &ManifestRenderer{layer: src.Layer, manifest: *manifest, atlasPath: manifest.Atlas}
		r.texture = AcquireTexture(r.atlasPath)
		if !rl.IsTextureValid(r.texture) {
			log.Printf("[Tilemap] manifest atlas unavailable: %s", r.manifest.Atlas)
			ReleaseTexture(r.atlasPath)
			continue
		}
		renderers = append(renderers, r)
	}
	return renderers
}

// manifestUsedBy reports whether the map places any piece of this manifest.
func manifestUsedBy(m *TiledMap, layerName string, manifest assetManifest) bool {
	if m == nil {
		// Sem mapa nao da para filtrar, e carregar tudo e o comportamento
		// antigo — a falha segura e desenhar demais, nao de menos.
		return true
	}
	for _, layer := range m.Layers {
		if layer.Name != layerName || layer.Type != "objectgroup" {
			continue
		}
		for _, object := range layer.Objects {
			if _, ok := manifest.Pieces[object.Name]; ok {
				return true
			}
		}
	}
	return false
}

// UsesGID identifies manifest-owned tiles by resolved atlas path, never a GID range.
func (r *ManifestRenderer) UsesGID(m *TiledMap, gid int) bool {
	if r == nil {
		return false
	}
	ts, ok := m.TilesetForGID(gid)
	return ok && filepath.Clean(ts.ImagePath) == filepath.Clean(r.manifest.Atlas)
}

func (r *ManifestRenderer) Unload() {
	if r != nil && r.atlasPath != "" {
		ReleaseTexture(r.atlasPath)
		r.atlasPath = ""
	}
}

// Draw desenha as pecas deste manifesto que estao no papel pedido E dentro do
// viewport.
//
// O teste e feito contra o retangulo DESENHADO e nao contra a celula ancora,
// e essa distincao e a regra: uma peca e ancorada longe do proprio desenho -
// a arvore ancora no tronco e desenha a copa centenas de pixels acima, a casa
// ancora no pe da parede. Cullar pela ancora faria a copa sumir enquanto o
// tronco ainda esta abaixo da tela. O retangulo desenhado e exato, entao nao
// precisa de margem nenhuma.
func (r *ManifestRenderer) Draw(m *TiledMap, role string, view Viewport) {
	if r == nil {
		return
	}
	for _, layer := range m.Layers {
		if layer.Name != r.layer || layer.Type != "objectgroup" {
			continue
		}
		for _, object := range layer.Objects {
			piece, ok := r.manifest.Pieces[object.Name]
			if !ok || piece.Role != role {
				continue
			}
			dst := rl.NewRectangle(object.X-piece.Anchor.X, object.Y-piece.Anchor.Y, piece.Source.Width, piece.Source.Height)
			if !view.Intersects(dst) {
				continue
			}
			src := rl.NewRectangle(piece.Source.X, piece.Source.Y, piece.Source.Width, piece.Source.Height)
			frameStats.Props++
			rl.DrawTexturePro(r.texture, src, dst, rl.Vector2{}, 0, rl.White)
			if debugOverlay {
				r.drawDebug(object, dst)
			}
		}
	}
}

func (r *ManifestRenderer) drawDebug(object Object, dst rl.Rectangle) {
	rl.DrawRectangleLines(int32(dst.X), int32(dst.Y), int32(dst.Width), int32(dst.Height), rl.Magenta)
	rl.DrawRectangleLines(int32(object.X-64), int32(object.Y-64), 128, 128, rl.SkyBlue)
	rl.DrawLine(int32(object.X-8), int32(object.Y), int32(object.X+8), int32(object.Y), rl.Red)
	rl.DrawLine(int32(object.X), int32(object.Y-8), int32(object.X), int32(object.Y+8), rl.Red)
}
