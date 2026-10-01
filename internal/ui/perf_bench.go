package ui

// O teste automatico de desempenho (Configuracoes > Graficos > TESTAR).
//
// Desenha uma cena sintetica PESADA — terreno, monstros, brilho aditivo e
// poeira, ~30% acima de um momento cheio de luta — em cada resolucao
// candidata, da melhor para a pior, e mede quanto a CENA custa. Fica com a
// melhor combinacao que o aparelho aguenta:
//
//   - a primeira resolucao cuja cena cabe no orcamento de 60 fps -> 60 fps
//     (ALTA se for a nativa, MEDIA se for menor);
//   - nenhuma cabe -> a melhor que cabe no de 30, a 30 fps (BAIXA);
//   - nem isso -> a menor resolucao, a 30 fps (BAIXA).
//
// Cada resolucao e desenhada numa RenderTexture do tamanho dela: o custo de
// pixel e o mesmo que o scaler de hardware vai ter (render_scale.go, pacote
// game), sem precisar reabrir o jogo para medir.
//
// A MEDIDA (29/09/2026): tempo entre duas cercas de GPU (perf_bench_gpu.go),
// uma antes e outra depois da cena — nao o tempo do quadro. O tempo do
// quadro carregava o vsync, a simulacao da partida e a tela de
// Configuracoes, e fazia um Galaxy S25 Ultra "nao segurar 60" nem a 540p.
// Mediana por resolucao, porque a pergunta e o custo tipico da cena; os
// picos do jogo de verdade ja estao pagos pela cena ser 30% mais pesada.
//
// O que ele NAO sabe, e a tela diz: e um teste de alguns segundos com o
// aparelho frio, e mede a GPU, nao a CPU da simulacao. Um celular que
// esquenta depois de 20 minutos pode precisar de um nivel abaixo; um que
// trava por CPU (F3: "simulacao" alta) pede 30 fps na mao.
//
// Enquanto ele roda, o jogo nao desenha o mundo (game/renderer.go): o custo
// medido e so o da cena. A partida continua simulando atras.

import (
	"fmt"
	"time"

	"github.com/WandenDourado/Legiao/internal/settings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	benchWarmup  = 0.6 // descartado: primeira subida de textura, clock subindo
	benchMeasure = 1.4 // segundos medidos por resolucao
	// Orcamento da CENA, em segundos de GPU (mediana). Nao e o tempo do
	// quadro: CPU e GPU trabalham em paralelo no jogo, entao a GPU pode usar
	// quase o quadro inteiro — 12 de 16,7 ms deixa folga para o HUD, a
	// copia final e a variacao de clock. 30 fps: o dobro.
	benchGPU60 = 0.012
	benchGPU30 = 0.024
)

var perfBenchRunning bool

// PerfBenchRunning diz se o teste esta rodando neste quadro — o renderer do
// jogo deixa de desenhar o mundo enquanto isso.
func PerfBenchRunning() bool { return perfBenchRunning }

type benchCandidate struct {
	height int32   // settings.RenderHeight: 0 = nativa
	w, h   int32   // tamanho da RenderTexture
	cost   float32 // mediana do custo da cena, em segundos; 0 = nao medido
}

type perfBench struct {
	cands   []benchCandidate
	idx     int
	t       float32
	samples []float32
	rt      rl.RenderTexture2D
	fence   *gpuFence
	scene   *benchScene
}

func newPerfBench(s settings.Settings) *perfBench {
	dw, dh := s.DisplayWidth, s.DisplayHeight
	if dw <= 0 || dh <= 0 || !mobileBuild {
		dw, dh = int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight())
	}
	b := &perfBench{scene: newBenchScene(), fence: newGPUFence()}
	b.cands = append(b.cands, benchCandidate{height: 0, w: dw, h: dh})
	if mobileBuild {
		for _, h := range settings.RenderHeights[1:] {
			if h < dh {
				b.cands = append(b.cands, benchCandidate{height: h, w: int32(float32(h) * float32(dw) / float32(dh)), h: h})
			}
		}
	}
	b.rt = rl.LoadRenderTexture(b.cands[0].w, b.cands[0].h)
	perfBenchRunning = true
	// Sem teto proprio durante o teste: nao muda a medida (ela e entre
	// cercas), mas encurta o teste — com 30 fps de teto ele teria metade das
	// amostras.
	rl.SetTargetFPS(0)
	return b
}

// updateBench roda um quadro do teste e desenha o progresso por cima.
func (scr *SettingsScreen) updateBench(l settingsLayout) {
	b := scr.bench
	c := &b.cands[b.idx]
	b.t += rl.GetFrameTime()

	// Cerca, cena, cerca: tudo o que o quadro ja tinha pedido (painel, fundo,
	// quadros anteriores) termina ANTES do relogio comecar.
	b.fence.wait(rl.Texture2D{})
	start := time.Now()
	rl.BeginTextureMode(b.rt)
	b.scene.draw(float32(c.w), float32(c.h), b.t)
	rl.EndTextureMode()
	b.fence.wait(b.rt.Texture)
	if b.t > benchWarmup {
		b.samples = append(b.samples, float32(time.Since(start).Seconds()))
	}
	b.drawProgress(l)

	if b.t < benchWarmup+benchMeasure {
		return
	}
	c.cost = median(b.samples)
	rl.TraceLog(rl.LogInfo, "[Graficos] teste: %s %dx%d cena %.2f ms (%d amostras)",
		renderHeightLabel(c.height), c.w, c.h, c.cost*1000, len(b.samples))
	if c.cost <= benchGPU60 || b.idx == len(b.cands)-1 {
		scr.benchResult = b.finish()
		scr.bench = nil
		return
	}
	b.idx++
	b.t, b.samples = 0, b.samples[:0]
	rl.UnloadRenderTexture(b.rt)
	b.rt = rl.LoadRenderTexture(b.cands[b.idx].w, b.cands[b.idx].h)
}

func (b *perfBench) drawProgress(l settingsLayout) {
	c := b.cands[b.idx]
	// A cena aparece pequena no painel so para o jogador ver que algo esta
	// acontecendo; o custo medido e o da RenderTexture inteira.
	r := l.row(1)
	thumbH := r.Height * 3
	thumbW := thumbH * float32(c.w) / float32(c.h)
	dst := rl.NewRectangle(l.Panel.X+(l.Panel.Width-thumbW)/2, l.row(0).Y, thumbW, thumbH)
	rl.DrawTexturePro(b.rt.Texture, rl.NewRectangle(0, 0, float32(c.w), -float32(c.h)), dst, rl.Vector2{}, 0, rl.White)
	msg := fmt.Sprintf("Testando %s (%d de %d)... nao feche", renderHeightLabel(c.height), b.idx+1, len(b.cands))
	size := int32(max(r.Height*0.3, Px(14)))
	w := rl.MeasureText(msg, size)
	rl.DrawText(msg, int32(l.Panel.X+l.Panel.Width/2)-w/2, int32(dst.Y+dst.Height+r.Height*0.3), size, rl.NewColor(246, 216, 142, 255))
}
