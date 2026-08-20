package tilemap

// Uma troca de shader por MATERIAL, em vez de uma por CELULA.
//
// O caminho antigo (drawBlended) embrulha cada celula num
// Begin/EndShaderMode proprio, porque a mascara de vizinhos vai em uniform e
// uniform so muda entre draw calls. Cada um desses esvazia o batch do rlgl,
// entao no world_03 eram 1.104 draw calls por quadro so de chao.
//
// Aqui a mascara sai do uniform e vai para uma TEXTURA do tamanho da grade —
// um texel por celula, R = as quatro bordas, G = os quatro cantos — e o indice
// da celula viaja no TINT do DrawTexturePro, que o rlgl guarda na cor do
// vertice e deixa variar por quad sem quebrar o batch. Resultado: um
// BeginShaderMode por material, ~10 por quadro.
//
// O caminho antigo continua no codigo e continua sendo usado quando este nao
// puder rodar. Isso nao e indecisao: e a mesma politica que o terreno ja
// adotava com `t.enabled` (shader indisponivel cai para borda dura), e sem
// toolchain para compilar e testar aqui, a alternativa seria embarcar uma
// reescrita do sistema visualmente mais delicado do projeto sem rede.

import (
	"image"
	"image/color"
	"log"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// maxBatchGrid e o maior mapa que o caminho em batch aceita.
//
// O indice da celula viaja em dois canais de 8 bits do tint, entao 255 e o
// teto por lado. Nenhum mapa chega perto (o maior e 64x72), mas passar disso
// silenciosamente desenharia o chao com as coordenadas erradas — a checagem
// existe para cair no caminho antigo em vez de imprimir lixo.
const maxBatchGrid = 255

// initBatch carrega o shader de batch e resolve os uniforms. Falhar aqui nao e
// erro fatal: o terreno continua desenhando pelo caminho por celula.
func (t *TerrainRenderer) initBatch() {
	vs, fs := terrainBatchShaderPaths()
	shader := rl.LoadShader(assets.Path(vs), assets.Path(fs))
	if !rl.IsShaderValid(shader) {
		log.Printf("[Tilemap] shader de batch indisponivel (%s); terreno segue por celula", fs)
		return
	}
	t.batchShader = shader
	t.maskTexLoc = rl.GetShaderLocation(shader, "maskTex")
	t.gridSizeLoc = rl.GetShaderLocation(shader, "gridSize")
	t.batchEdgeLoc = rl.GetShaderLocation(shader, "edgeWidth")
	t.spanFLoc = rl.GetShaderLocation(shader, "spanF")
	t.batched = t.maskTexLoc >= 0 && t.gridSizeLoc >= 0 &&
		t.batchEdgeLoc >= 0 && t.spanFLoc >= 0
	if !t.batched {
		log.Printf("[Tilemap] shader de batch sem uniforms; terreno segue por celula")
		rl.UnloadShader(shader)
	}
}

// PrepareMasks constroi uma textura de mascara por material presente na camada.
//
// A mascara depende so da camada `ground`, que nao muda em runtime — por isso
// ela e construida UMA vez no carregamento do mapa em vez de recalculada a
// cada quadro, que e o que edgeMask/cornerMask faziam: oito consultas de
// vizinho por celula pintada, cada uma varrendo as pilhas linearmente.
func (t *TerrainRenderer) PrepareMasks(layer Layer) {
	if !t.batched {
		return
	}
	if layer.Width <= 0 || layer.Height <= 0 ||
		layer.Width > maxBatchGrid || layer.Height > maxBatchGrid {
		log.Printf("[Tilemap] grade %dx%d fora do alcance do batch; terreno segue por celula",
			layer.Width, layer.Height)
		t.batched = false
		rl.UnloadShader(t.batchShader)
		return
	}
	t.masks = make(map[int]rl.Texture2D, len(terrainOverlays))
	for _, material := range terrainOverlays {
		if _, ok := t.textures[material]; !ok {
			continue // material que este mapa nao carrega, logo nao pinta
		}
		if tex, ok := buildMaskTexture(layer, material); ok {
			t.masks[material] = tex
		}
	}
}

// buildMaskTexture empacota, para cada celula, os oito vizinhos deste material.
//
// R guarda as quatro bordas na ordem de edgeMask (N, L, S, O) e G os quatro
// cantos na de cornerMask (NO, NE, SE, SO). Manter a ORDEM identica a de
// terrain_mask.go e o que permite os dois caminhos desenharem igual; trocar
// dois bits aqui produziria um desbotamento espelhado, que e o tipo de defeito
// que passa despercebido numa captura estatica.
//
// Devolve false quando o material nao pinta nenhuma celula: uma textura de
// mascara toda zerada seria VRAM gasta para dizer "nada aqui".
func buildMaskTexture(layer Layer, material int) (rl.Texture2D, bool) {
	img := image.NewRGBA(image.Rect(0, 0, layer.Width, layer.Height))
	painted := false
	for y := 0; y < layer.Height; y++ {
		for x := 0; x < layer.Width; x++ {
			i := y*layer.Width + x
			if i >= len(layer.Data) || !paintedWith(layer.Data[i], material) {
				continue
			}
			painted = true
			img.Set(x, y, color.RGBA{
				R: packBits(edgeMask(layer, x, y, material)),
				G: packBits(cornerMask(layer, x, y, material)),
				B: 0,
				A: 255,
			})
		}
	}
	if !painted {
		return rl.Texture2D{}, false
	}

	rlImg := rl.NewImageFromImage(img)
	texture := rl.LoadTextureFromImage(rlImg)
	rl.UnloadImage(rlImg)
	if !rl.IsTextureValid(texture) {
		return rl.Texture2D{}, false
	}
	// POINT e obrigatorio: a mascara e um dado indexado por celula, nao uma
	// imagem. Interpolar entre dois texels misturaria a vizinhanca de duas
	// celulas diferentes e desbotaria bordas que nao existem.
	rl.SetTextureFilter(texture, rl.FilterPoint)
	return texture, true
}

// packBits transforma quatro flags 0/1 num byte, bit 0 primeiro.
func packBits(flags []float32) uint8 {
	var out uint8
	for i, f := range flags {
		if i >= 8 {
			break
		}
		if f > 0.5 {
			out |= 1 << uint(i)
		}
	}
	return out
}

func (t *TerrainRenderer) unloadMasks() {
	for _, texture := range t.masks {
		if rl.IsTextureValid(texture) {
			rl.UnloadTexture(texture)
		}
	}
	t.masks = nil
}

// drawBatchedOverlays desenha todas as camadas de overlay com um
// BeginShaderMode por material. Devolve false quando o caminho nao esta
// disponivel, e ai o chamador usa o de sempre.
func (t *TerrainRenderer) drawBatchedOverlays(layer Layer, tileW, tileH int, view Viewport) bool {
	if !t.batched || len(t.masks) == 0 {
		return false
	}
	gridSize := []float32{float32(layer.Width), float32(layer.Height)}

	for _, material := range terrainOverlays {
		mask, ok := t.masks[material]
		if !ok {
			continue
		}
		texture := t.textureFor(material)
		span := t.spanFor(material, texture, tileW)

		frameStats.ShaderBinds++
		rl.BeginShaderMode(t.batchShader)
		rl.SetShaderValue(t.batchShader, t.gridSizeLoc, gridSize, rl.ShaderUniformVec2)
		rl.SetShaderValue(t.batchShader, t.batchEdgeLoc,
			[]float32{edgeWidthFor(material)}, rl.ShaderUniformFloat)
		rl.SetShaderValue(t.batchShader, t.spanFLoc,
			[]float32{float32(span)}, rl.ShaderUniformFloat)
		// A mascara e ligada DENTRO do bloco: fora dele o shader nao esta
		// ativo e o raylib nao teria onde registrar o sampler.
		rl.SetShaderValueTexture(t.batchShader, t.maskTexLoc, mask)

		t.eachCell(layer, view, func(x, y, kind int) {
			if !paintedWith(kind, material) {
				return
			}
			t.drawCellTinted(texture, x, y, tileW, tileH, span)
		})
		rl.EndShaderMode()
	}
	return true
}

// drawCellTinted desenha uma celula com o INDICE dela no tint.
//
// O tint normalmente e cor; aqui ele e endereco. E o unico canal que o
// DrawTexturePro deixa variar por quad sem quebrar o batch, e e por isso que o
// shader de batch nao multiplica fragColor no resultado — se multiplicasse,
// pintaria o chao com as proprias coordenadas.
func (t *TerrainRenderer) drawCellTinted(texture rl.Texture2D, x, y, tileW, tileH, span int) {
	frameStats.TerrainQuads++
	src, _ := tileWindow(texture, x, y, span)
	dest := rl.NewRectangle(float32(x*tileW), float32(y*tileH), float32(tileW), float32(tileH))
	rl.DrawTexturePro(texture, src, dest, rl.Vector2{}, 0,
		rl.NewColor(uint8(x), uint8(y), 0, 255))
}
