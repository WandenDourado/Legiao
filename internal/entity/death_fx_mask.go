package entity

// As mascaras da desintegracao: que pedacos de cada quadro de cada folha de
// inimigo sao corpo, e de que cor.
//
// O efeito corta o quadro que estava na tela em celulas de alguns pixels. Uma
// celula quase transparente nao pode virar grao — viraria um quadrado de
// poeira flutuando onde nao havia monstro nenhum — e a poeira solta e
// desenhada com cor lisa, entao cada celula precisa da cor media dos pixels
// dela. As duas respostas exigem os PIXELS da folha, que so existem na GPU
// depois do carregamento.
//
// Por isso cada folha e decodificada de novo, UMA vez, numa goroutine de
// fundo, e varrida em dois graos (o normal e o dobro, que o orcamento usa
// quando muita coisa morre junto). LoadImage e LoadImageColors sao so CPU —
// stb_image e malloc, nenhum contexto GL — e no Android o fopen do raylib vai
// pelo AAssetManager, que e seguro entre threads. A goroutine nao disputa o
// quadro: o monstro que morrer antes da mascara dele ficar pronta some como
// sempre sumiu, sem efeito, e ninguem trava esperando.
//
// Memoria: 8 bytes por celula. A folha mais cara (idle da Imperatriz, 8
// quadros de 610x590 em grao de 6 px) fica perto de 250 KB.

import (
	"image/color"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// deathCell e um pedaco opaco de um quadro: canto (u,v) e tamanho em px da
// FOLHA, relativos ao quadro, e a cor media (alfa ignorado).
type deathCell struct {
	U, V uint16
	W, H uint8
	Col  color.RGBA
}

// deathSheetMask guarda as celulas de todos os quadros de uma folha, nos dois
// graos. Indice do quadro = linha*colunas + coluna.
type deathSheetMask struct {
	Cols   int
	Fine   [][]deathCell
	Coarse [][]deathCell
}

type deathMaskKey struct {
	Path string
	Cell int
}

var (
	deathMaskMu      sync.Mutex
	deathMasks       = map[deathMaskKey]*deathSheetMask{}
	deathMaskStarted bool
)

// deathCellSrc e o grao em px da FOLHA para um inimigo: o grao de mundo
// dividido pela escala de desenho. Nunca abaixo de 2 px.
func deathCellSrc(def EnemyDef, tune deathTuning) int {
	scale := def.RenderScale
	if scale <= 0 {
		scale = 1
	}
	c := int(math.Round(float64(deathFXCellWorld * tune.CellScale / scale)))
	if c < 2 {
		c = 2
	}
	return c
}

func deathTuneFor(def EnemyDef) deathTuning {
	if def.IsBoss() {
		return deathTuneBoss
	}
	return deathTuneMob
}

// deathMaskFor devolve a mascara pronta, ou nil se ela ainda nao existe.
func deathMaskFor(path string, cell int) *deathSheetMask {
	deathMaskMu.Lock()
	defer deathMaskMu.Unlock()
	return deathMasks[deathMaskKey{path, cell}]
}

// startDeathMaskBuild dispara a varredura de fundo uma vez por processo.
// Chamado do primeiro quadro com o efeito ligado — depois de InitWindow, que
// e quando o AAssetManager do Android ja existe.
func startDeathMaskBuild() {
	deathMaskMu.Lock()
	if deathMaskStarted {
		deathMaskMu.Unlock()
		return
	}
	deathMaskStarted = true
	deathMaskMu.Unlock()
	go buildAllDeathMasks()
}

type deathMaskJob struct {
	ad   EnemyAnimDef
	cell int
	boss bool
}

func buildAllDeathMasks() {
	seen := map[deathMaskKey]bool{}
	var jobs []deathMaskJob
	for _, def := range enemyRegistry {
		cell := deathCellSrc(def, deathTuneFor(def))
		for _, anim := range animationsOf(def) {
			ad := def.AnimDef(anim)
			k := deathMaskKey{ad.SpritePath, cell}
			if ad.SpritePath == "" || seen[k] {
				continue
			}
			seen[k] = true
			jobs = append(jobs, deathMaskJob{ad, cell, def.IsBoss()})
		}
	}
	// Os monstros comuns primeiro: eles morrem desde o primeiro minuto, a
	// chefe so no fim da campanha — e as folhas dela sao as mais pesadas.
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].boss != jobs[j].boss {
			return !jobs[i].boss
		}
		return jobs[i].ad.SpritePath < jobs[j].ad.SpritePath
	})
	built := 0
	for _, j := range jobs {
		if m := loadDeathMask(j.ad, j.cell); m != nil {
			deathMaskMu.Lock()
			deathMasks[deathMaskKey{j.ad.SpritePath, j.cell}] = m
			deathMaskMu.Unlock()
			built++
		}
	}
	log.Printf("[Entity] mascaras de desintegracao prontas: %d de %d folhas", built, len(jobs))
}

func loadDeathMask(ad EnemyAnimDef, cell int) *deathSheetMask {
	img := rl.LoadImage(assets.Path(ad.SpritePath))
	if img == nil || img.Width <= 0 || img.Height <= 0 {
		return nil
	}
	defer rl.UnloadImage(img)
	px := rl.LoadImageColors(img)
	defer rl.UnloadImageColors(px)
	rows := ad.Rows
	if rows <= 0 {
		rows = 1
	}
	return buildDeathSheetMask(px, int(img.Width), int(img.Height), ad.FrameWidth, ad.FrameHeight, ad.Columns, rows, cell)
}

// buildDeathSheetMask varre uma folha inteira ja decodificada. Puro (sem
// raylib) para poder ser testado.
func buildDeathSheetMask(px []color.RGBA, imgW, imgH, fw, fh, cols, rows, cell int) *deathSheetMask {
	if fw <= 0 || fh <= 0 || cols <= 0 || len(px) < imgW*imgH {
		return nil
	}
	m := &deathSheetMask{Cols: cols}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			x0, y0 := c*fw, r*fh
			m.Fine = append(m.Fine, scanDeathCells(px, imgW, imgH, x0, y0, fw, fh, cell))
			m.Coarse = append(m.Coarse, scanDeathCells(px, imgW, imgH, x0, y0, fw, fh, cell*2))
		}
	}
	return m
}

// scanDeathCells corta um quadro em celulas de `cell` px e guarda as opacas.
func scanDeathCells(px []color.RGBA, imgW, imgH, x0, y0, fw, fh, cell int) []deathCell {
	var out []deathCell
	for v := 0; v < fh; v += cell {
		for u := 0; u < fw; u += cell {
			cw, ch := min(cell, fw-u), min(cell, fh-v)
			var a, r, g, b, n float64
			for y := y0 + v; y < y0+v+ch && y < imgH; y++ {
				row := y * imgW
				for x := x0 + u; x < x0+u+cw && x < imgW; x++ {
					p := px[row+x]
					n++
					if p.A > 24 {
						al := float64(p.A)
						a += al
						r += float64(p.R) * al
						g += float64(p.G) * al
						b += float64(p.B) * al
					}
				}
			}
			if n == 0 || a/(255*n) < deathCellCoverage {
				continue
			}
			out = append(out, deathCell{
				U: uint16(u), V: uint16(v), W: uint8(cw), H: uint8(ch),
				Col: color.RGBA{uint8(r / a), uint8(g / a), uint8(b / a), 255},
			})
		}
	}
	return out
}
