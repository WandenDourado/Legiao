package ui

// UM PONTO DE VERDADE PARA O TAMANHO DO TEXTO.
//
// Existe por causa do relato dos testadores em 20/09/2026: "o texto da HUD e
// dos dialogos e pequeno demais". A tela de Configuracoes ja tinha uma
// "ESCALA DO HUD" desde a rodada anterior, mas ela era lida por UM lugar so —
// DrawHealthBar. As outras cinquenta e tantas chamadas de rl.DrawText do jogo
// ignoravam a preferencia do jogador, entao aumentar a escala mexia na barra
// de vida e em nada mais.
//
// A regra agora e: todo texto de HUD passa por aqui, e o numero escrito no
// codigo e o tamanho BASE — o que se ve na escala 1.0. A escala do jogador
// multiplica; o piso protege de uma base pequena demais para um celular.
//
// O QUE NAO PASSA POR AQUI, e de proposito:
//
//   - Texto em espaco de MUNDO (game/portal_counter.go). Aquele e desenhado
//     dentro do bloco de camera e ja cresce e encolhe com o zoom; multiplicar
//     pela escala da HUD o faria variar duas vezes.
//   - Tamanhos derivados de uma geometria que JA foi escalada (o numero dentro
//     do pip de recarga, o rotulo dentro do botao de acao). Esses usam
//     FitTextSize, que so aplica o piso — aplicar a escala de novo dobraria o
//     fator.

import rl "github.com/gen2brain/raylib-go/raylib"

// MinTextSize e o menor tamanho que ainda se le num celular com o aparelho na
// mao. Era 14 espalhado por varios arquivos, e 14 e o tamanho em que a fonte
// padrao do raylib comeca a fechar os contornos.
const MinTextSize = 18

// PIXEL AUTORADO E PIXEL DA TELA NATIVA (29/09/2026).
//
// Todo numero fixo da interface — o 22 de uma fonte base, os 20 px da barra
// de vida, o teto de 760 px de um painel, o piso de 18 px — foi escrito
// olhando a tela na resolucao nativa. Com Configuracoes > Graficos >
// Resolucao abaixo da nativa a janela tem MENOS pixels (540 de altura numa
// tela de 1080) e o scaler do aparelho amplia: um numero fixo passava a
// ocupar o dobro da tela, enquanto o que e proporcional (sh*0.08) ficava
// igual. Relato do Gui: "em 540p a HUD fica bem grande".
//
// renderScale e quantos pixels DESTA janela valem um pixel da tela nativa:
// 1 na nativa (e sempre no PC), 0,5 em 540p numa tela de 1080. game.Run
// anota uma vez, o mesmo fator que ja encolhe o zoom da camera
// (game/render_scale.go). Px converte; TextSize, o piso e PixelScale ja
// passam por ele. Trocar a resolucao muda a nitidez, nunca o tamanho.
var renderScale float32 = 1

// SetRenderScale anota o fator da resolucao de desenho (ver acima).
func SetRenderScale(k float32) {
	if k > 0 {
		renderScale = k
	}
}

// Px converte um tamanho autorado em pixels da tela nativa para pixels desta
// janela. Todo numero fixo de interface em espaco de tela passa por aqui.
func Px(v float32) float32 { return v * renderScale }

// TextSize converte um tamanho BASE (o numero autorado, em pixels na escala
// 1.0) no tamanho efetivo desta maquina.
func TextSize(base float32) int32 { return FitTextSize(base * hudScale * renderScale) }

// FitTextSize aplica so o piso, para um tamanho que ja veio de uma geometria
// escalada. Ver o cabecalho. O piso e em pixels da tela nativa (Px), mas
// nunca abaixo de 10: e o tamanho de desenho da fonte padrao do raylib.
func FitTextSize(size float32) int32 {
	floor := max(Px(MinTextSize), 10)
	if size < floor {
		return int32(floor)
	}
	return int32(size)
}

// UIScale expoe a escala escolhida para quem precisa esticar GEOMETRIA que
// ja e proporcional a tela (sh*0.035). Geometria em pixels fixos usa
// PixelScale.
func UIScale() float32 { return hudScale }

// PixelScale e UIScale para geometria autorada em pixels fixos (o raio de 21
// px da seta da gargula): a escala do jogador vezes a da resolucao.
func PixelScale() float32 { return hudScale * renderScale }

// DrawUIText desenha com o tamanho base ja convertido.
func DrawUIText(text string, x, y, base float32, c rl.Color) {
	rl.DrawText(text, int32(x), int32(y), TextSize(base), c)
}

// MeasureUIText mede com o mesmo tamanho que DrawUIText desenharia — usar
// rl.MeasureText com a base crua e o jeito de centralizar errado.
func MeasureUIText(text string, base float32) float32 {
	return float32(rl.MeasureText(text, TextSize(base)))
}
