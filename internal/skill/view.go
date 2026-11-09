package skill

// O CULLING DAS MAGIAS: o que a camera nao mostra nao e desenhado.
//
// Esta e a regra que a Chuva de Meteoros do Mago quebrou, e a razao de este
// arquivo existir em vez de cada magia resolver por conta propria.
//
// A chuva sorteia alvos no MAPA INTEIRO durante 15 segundos, um a cada 25 ms
// (~40 por segundo). Cada meteoro vive 1,4 s, entao em regime ha ~56 no ar, e
// cada um alimenta um rastro de particulas. `DrawMeteors` desenhava TODOS —
// inclusive os que caiam do outro lado do mapa, longe de qualquer jogador.
// Numa tela de 3840x2160 sobre um mapa de 7680x6400, isso e desenhar seis
// vezes mais do que existe para ver.
//
// O terreno e os inimigos ja aprenderam essa licao (doc/performance.md, C3 e
// C4: o `world_03` caiu de 23.044 para 1.071 quads por quadro). As magias
// ficaram de fora porque, uma a uma, elas sao pequenas: uma bola de fogo por
// vez nao paga culling. A ultimate e onde a conta vira — e e sempre a ultimate
// que o jogador lembra de ter derrubado o fps.
//
// POR QUE ESTADO DE PACOTE E NAO UM PARAMETRO. `Manager.Draw*` tem dezesseis
// pontos de entrada (meteoro, legiao, cemiterio, santuario, flecha, bola de
// fogo, escudo, espada, avatar, area angelical, flecha celestial, espinho,
// nevoa, esfera, bola de canhao, explosao) e todos sao chamados de dentro de
// UM bloco de camera, no mesmo quadro, na mesma goroutine — a que detem o
// contexto OpenGL. Um parametro em dezesseis assinaturas so para carregar o
// mesmo valor ate o fim seria mais codigo dizendo menos. O `tilemap` ja usa
// esta forma para as estatisticas de quadro (`beginFrameStats`).

import rl "github.com/gen2brain/raylib-go/raylib"

var (
	// drawView e a janela visivel em unidades de mundo, publicada uma vez por
	// quadro pelo renderer. Largura ou altura zero DESLIGA o culling: e o que
	// um teste sem camera recebe, e desenhar demais e a falha segura.
	drawView rl.Rectangle
	// drawn/culled contam o quadro corrente, para o painel do F3. Uma magia
	// nova que esqueca de chamar `visible` aparece como um numero de
	// "desenhadas" que nao cai quando a camera se afasta.
	drawnCount  int
	culledCount int
)

// SetDrawView publica a janela visivel e zera os contadores do quadro.
// Chamada uma vez por quadro, antes de qualquer desenho de magia.
func SetDrawView(view rl.Rectangle) {
	drawView = view
	drawnCount, culledCount = 0, 0
}

// DrawCounts devolve quantas coisas foram desenhadas e quantas foram puladas
// neste quadro.
func DrawCounts() (drawn, culled int) { return drawnCount, culledCount }

// visible reporta se um circulo de raio radius centrado em center aparece na
// janela, e ja contabiliza a decisao.
//
// Circulo e nao retangulo porque toda magia deste jogo e desenhada a partir de
// um centro e um alcance — anel, gradiente, rajada de particulas. Quem tem
// geometria comprida (o rastro do meteoro) chama duas vezes, uma por ponta.
func visible(center rl.Vector2, radius float32) bool {
	if drawView.Width <= 0 || drawView.Height <= 0 {
		drawnCount++
		return true
	}
	if center.X+radius < drawView.X || center.X-radius > drawView.X+drawView.Width ||
		center.Y+radius < drawView.Y || center.Y-radius > drawView.Y+drawView.Height {
		culledCount++
		return false
	}
	drawnCount++
	return true
}

// visibleAny e para quem tem duas pontas distantes: o meteoro desenha a marca
// no chao E a rocha caindo 780 unidades acima dela, e sumir com a marca porque
// a rocha saiu da tela seria pior que nao cullar.
func visibleAny(a rl.Vector2, ra float32, b rl.Vector2, rb float32) bool {
	if drawView.Width <= 0 || drawView.Height <= 0 {
		drawnCount++
		return true
	}
	inside := func(c rl.Vector2, r float32) bool {
		return !(c.X+r < drawView.X || c.X-r > drawView.X+drawView.Width ||
			c.Y+r < drawView.Y || c.Y-r > drawView.Y+drawView.Height)
	}
	if inside(a, ra) || inside(b, rb) {
		drawnCount++
		return true
	}
	culledCount++
	return false
}
