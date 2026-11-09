package ui

// O fim do teste automatico (perf_bench.go): escolher o nivel pela medida,
// aplicar, gravar e escrever a frase que fica na tela.

import (
	"fmt"
	"log"
	"sort"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// finish escolhe, aplica, grava e devolve a frase de resultado.
func (b *perfBench) finish() string {
	b.release()

	pick, fps := len(b.cands)-1, settings.FrameCap30
	for i, c := range b.cands {
		if c.cost > 0 && c.cost <= benchGPU60 {
			pick, fps = i, settings.FrameCap60
			break
		}
	}
	if fps != settings.FrameCap60 {
		for i, c := range b.cands {
			if c.cost > 0 && c.cost <= benchGPU30 {
				pick = i
				break
			}
		}
	}
	tier := settings.PresetLow
	if fps == settings.FrameCap60 {
		tier = settings.PresetMedium
		if pick == 0 {
			tier = settings.PresetHigh
		}
	}
	chosen := b.cands[pick]
	s := settings.Update(func(s *settings.Settings) {
		settings.ApplyPreset(s, tier)
		s.RenderHeight = chosen.height
		s.TargetFPS = fps
	})
	applyTargetFPS(s.TargetFPS)
	entity.SetDeathFXEnabled(s.DeathEffect)
	tilemap.SetGroundDetailVisible(s.GroundDetail)
	// Grava ja: se o resultado trocou a resolucao, o proximo gesto e
	// REINICIAR AGORA (ou fechar o app), e nenhum dos dois passa pelo VOLTAR.
	if err := settings.Save(); err != nil {
		log.Printf("[Graficos] falha ao gravar o resultado do teste: %v", err)
	}

	// O numero vai junto: e o que permite comparar aparelhos por relato.
	ms := fmt.Sprintf("%.1f ms", chosen.cost*1000)
	if !mobileBuild {
		return fmt.Sprintf("%s a %d fps (%s)", qualityLabels[tier], fps, ms)
	}
	return fmt.Sprintf("%s: %s a %d fps (%s)", qualityLabels[tier], renderHeightLabel(chosen.height), fps, ms)
}

// cancel desfaz o teste sem aplicar nada.
func (b *perfBench) cancel() {
	b.release()
	applyTargetFPS(settings.Get().TargetFPS)
}

func (b *perfBench) release() {
	rl.UnloadRenderTexture(b.rt)
	b.fence.unload()
	b.scene.unload()
	perfBenchRunning = false
}

func median(xs []float32) float32 {
	if len(xs) == 0 {
		return 1
	}
	c := append([]float32(nil), xs...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c[len(c)/2]
}
