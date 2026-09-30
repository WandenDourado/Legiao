package ui

// A cena do teste automatico de desempenho (perf_bench.go).
//
// E proposital que ela seja mais pesada que o jogo: o teste dura segundos com
// o aparelho frio, e a partida dura dezenas de minutos com ele quente. As
// quantidades, em "telas de pixel" (1,0 = a tela inteira pintada uma vez):
//
//	terreno: 2 camadas opacas de tile .............. 2,0
//	monstros: 40 sprites com alfa .................. ~0,7
//	brilho aditivo: 6 circulos em degrade .......... ~0,8
//	poeira: 1500 quadradinhos atualizados em Go .... CPU + chamadas de desenho
//
// ~3,5 telas contra os ~2,5–3 de um momento cheio de luta (terreno 1,5–2,
// cenario e monstros ~0,5, magias; plans/plan_qualidade_grafica_mobile.md).
// Tudo e proporcional a ALTURA da RenderTexture, entao cada resolucao desenha
// a mesma cena com menos ou mais pixel — que e o que o scaler vai fazer.

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/assets"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const benchParticles = 1500

type benchScene struct {
	ground, slime, orc rl.Texture2D
}

func newBenchScene() *benchScene {
	return &benchScene{
		ground: rl.LoadTexture(assets.Path("assets/tilesets/terrain_grass.png")),
		slime:  rl.LoadTexture(assets.Path("assets/sprites/enemies/slime/slime.png")),
		orc:    rl.LoadTexture(assets.Path("assets/sprites/enemies/orc/idle.png")),
	}
}

func (s *benchScene) unload() {
	for _, t := range []rl.Texture2D{s.ground, s.slime, s.orc} {
		if t.ID != 0 {
			rl.UnloadTexture(t)
		}
	}
}

func (s *benchScene) draw(w, h, t float32) {
	rl.ClearBackground(rl.Black)
	tile := h * 0.095 // uma celula de 128 px de mundo, com o zoom 0,8 do celular
	for layer := 0; layer < 2; layer++ {
		off := float32(layer) * tile * 0.5
		tint := rl.White
		if layer == 1 {
			tint = rl.NewColor(200, 220, 190, 255)
		}
		for y := -off; y < h; y += tile {
			for x := -off; x < w; x += tile {
				src := rl.NewRectangle(0, 0, float32(s.ground.Width), float32(s.ground.Height))
				rl.DrawTexturePro(s.ground, src, rl.NewRectangle(x, y, tile+1, tile+1), rl.Vector2{}, 0, tint)
			}
		}
	}

	size := h * 0.22
	for i := 0; i < 40; i++ {
		fi := float64(i)
		x := w * float32(0.5+0.45*math.Sin(fi*1.7+float64(t)*0.6))
		y := h * float32(0.5+0.42*math.Cos(fi*2.3+float64(t)*0.5))
		if i%2 == 0 && s.slime.ID != 0 {
			frame := float32((i + int(t*9)) % 6)
			rl.DrawTexturePro(s.slime, rl.NewRectangle(frame*256, 0, 256, 256),
				rl.NewRectangle(x, y, size, size), rl.NewVector2(size/2, size/2), float32(fi*37), rl.White)
		} else if s.orc.ID != 0 {
			frame := float32((i + int(t*6)) % 8)
			rl.DrawTexturePro(s.orc, rl.NewRectangle(frame*154, 6*134, 154, 134),
				rl.NewRectangle(x, y, size*1.15, size), rl.NewVector2(size*0.57, size/2), 0, rl.White)
		}
	}

	rl.BeginBlendMode(rl.BlendAdditive)
	for i := 0; i < 6; i++ {
		fi := float64(i)
		x := w * float32(0.5+0.35*math.Sin(fi*2.1+float64(t)*0.9))
		y := h * float32(0.5+0.3*math.Cos(fi*1.3+float64(t)*0.7))
		rl.DrawCircleGradient(int32(x), int32(y), h*0.3, rl.Fade(rl.NewColor(255, 150, 60, 255), 0.35), rl.Fade(rl.NewColor(255, 150, 60, 255), 0))
	}
	rl.EndBlendMode()

	p := h * 0.006
	for i := 0; i < benchParticles; i++ {
		fi := float64(i)
		a := fi*0.618 + float64(t)*(0.3+math.Mod(fi, 7)*0.05)
		x := w*0.5 + float32(math.Cos(a)*float64(w)*0.45*math.Mod(fi*0.37, 1))
		y := h*0.5 + float32(math.Sin(a*1.3)*float64(h)*0.45*math.Mod(fi*0.53, 1))
		rl.DrawRectanglePro(rl.NewRectangle(x, y, p, p), rl.NewVector2(p/2, p/2), float32(fi)+t*90, rl.Fade(rl.LightGray, 0.8))
	}
}
