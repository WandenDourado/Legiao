package ui

// A cerca de GPU do teste automatico (perf_bench.go), 29/09/2026.
//
// O OpenGL ES nao tem glFinish no raylib, mas LER um pixel de volta obriga o
// driver a terminar tudo o que foi pedido antes dele: a fila da GPU e em
// ordem. A cerca e uma RenderTexture de 1x1: desenha nela (uma amostra da
// textura que se quer esperar, ou so um clear) e le de volta. Custa uma
// FBO e um pixel — nada perto da cena.
//
// Por que isto existe: ate esta data o teste media o TEMPO DO QUADRO, que
// e quantizado pelo vsync (8,3 ms num painel de 120 Hz como o do S25 Ultra,
// 16,7 a 60 Hz), inclui a simulacao da partida rodando atras e a propria
// tela de Configuracoes. No aparelho do Gui nenhuma resolucao, nem 540p,
// "segurava 60" — o limite nao era pixel — e o teste escolhia Baixa, com
// 30 fps e sem as cinzas. Cercando antes e depois, o relogio mede so a
// cena: envio pela CPU + execucao na GPU, sem vsync no meio.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type gpuFence struct {
	probe rl.RenderTexture2D
}

func newGPUFence() *gpuFence {
	return &gpuFence{probe: rl.LoadRenderTexture(1, 1)}
}

// wait so volta quando a GPU terminou tudo o que foi pedido ate aqui. Com
// src valida, a cerca depende de src — o que deixa explicito para o driver
// que a textura tem de estar pronta antes da leitura.
func (f *gpuFence) wait(src rl.Texture2D) {
	rl.BeginTextureMode(f.probe)
	if src.ID != 0 {
		rl.DrawTexturePro(src, rl.NewRectangle(0, 0, 1, 1), rl.NewRectangle(0, 0, 1, 1), rl.Vector2{}, 0, rl.White)
	} else {
		rl.ClearBackground(rl.Black)
	}
	rl.EndTextureMode()
	rl.UnloadImage(rl.LoadImageFromTexture(f.probe.Texture))
}

func (f *gpuFence) unload() {
	rl.UnloadRenderTexture(f.probe)
}
