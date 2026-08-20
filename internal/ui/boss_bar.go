package ui

// A barra de vida do chefe.
//
// Ela existe separada da barra flutuante que todo inimigo tem porque a pergunta
// e outra. A barra flutuante responde "quanta vida tem AQUELE bicho ali"; com
// trinta inimigos em campo, ela e uma entre trinta. A barra de chefe responde
// "quanto falta para a fase acabar", e por isso mora no HUD, em espaco de tela,
// como manda `doc/camera.md` para o que nao e mundo.

import (
	"math"

	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	bossBarWidthFrac  = 0.60 // fracao da largura da tela
	bossBarHeight     = 26
	bossBarTop        = 54 // abaixo do contador de horda, que fica no topo
	bossBarNameSize   = 24
	bossBarSegments   = 4 // marcas a cada 25%
	bossBarFrameThick = 3

	// Aviso da conjuracao.
	fogWarnFontSize = 46
	fogWarnSubSize  = 22
	fogWarnBorder   = 10
	// fogWarnLead tem de bater com `bossCastLead` do host: e o denominador da
	// aceleracao do piscar. Fora de sincronia, o aviso pisca depressa cedo
	// demais ou nunca chega ao ritmo urgente.
	fogWarnLead float32 = 2.4
)

var (
	bossBarBack   = rl.NewColor(18, 16, 22, 220)
	bossBarFill   = rl.NewColor(150, 26, 40, 255)   // carmim ferido, o acento dela
	bossBarShine  = rl.NewColor(196, 58, 74, 255)   // faixa clara no topo do preenchimento
	bossBarFrame  = rl.NewColor(150, 126, 62, 255)  // ouro envelhecido, o mesmo do decote
	bossBarMark   = rl.NewColor(24, 20, 16, 190)
	bossBarNameFg = rl.NewColor(232, 224, 210, 255)
	fogWarnRed    = rl.NewColor(226, 58, 58, 255)
	fogWarnSubFg  = rl.NewColor(236, 220, 200, 255)
)

// DrawBossBar desenha a barra do chefe da fase, se houver um.
//
// Le o estado que a camada de rede publica, entao funciona igual no host e no
// cliente — mesma regra do DrawWaveHUD.
func DrawBossBar(screenWidth, screenHeight float32) {
	boss := network.GetBossState()
	if !boss.Present || boss.MaxHealth <= 0 {
		return
	}

	w := screenWidth * bossBarWidthFrac
	x := (screenWidth - w) / 2
	y := float32(bossBarTop)

	// Nome centrado ACIMA da barra. Acima e nao dentro: dentro, o texto some
	// contra o preenchimento cheio e reaparece quando ele esvazia, o que faz o
	// nome piscar ao longo da luta.
	name := boss.Name
	if name == "" {
		name = "Chefe"
	}
	tw := rl.MeasureText(name, bossBarNameSize)
	rl.DrawText(name, int32(x+(w-float32(tw))/2), int32(y-bossBarNameSize-4),
		bossBarNameSize, bossBarNameFg)

	rl.DrawRectangleRec(rl.NewRectangle(x, y, w, bossBarHeight), bossBarBack)

	frac := boss.Health / boss.MaxHealth
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	if frac > 0 {
		fw := w * frac
		rl.DrawRectangleRec(rl.NewRectangle(x, y, fw, bossBarHeight), bossBarFill)
		rl.DrawRectangleRec(rl.NewRectangle(x, y, fw, bossBarHeight*0.35), bossBarShine)
	}

	// Marcas de 25%. Uma barra de 400 de vida sem subdivisao nao da sensacao de
	// progresso: o jogador bate por um minuto e ve o mesmo vermelho. Com as
	// marcas, cada quarto vencido e um marco visivel.
	for i := 1; i < bossBarSegments; i++ {
		mx := x + w*float32(i)/float32(bossBarSegments)
		rl.DrawRectangleRec(rl.NewRectangle(mx-1, y, 2, bossBarHeight), bossBarMark)
	}

	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, w, bossBarHeight),
		bossBarFrameThick, bossBarFrame)

	if boss.Casting {
		drawFogWarning(screenWidth, screenHeight, boss.CastLeft)
	}
}

// drawFogWarning e o aviso da conjuracao.
//
// Ele existe porque a danca — que e o telegrafo desenhado na propria chefe —
// acontece do outro lado de uma arena de 5120 px. Um grupo segurando um portao
// nao ve a chefe, e sem aviso a nevoa deixa de ser uma coisa que se responde e
// vira uma coisa que acontece com voce.
//
// Pisca, e o piscar ACELERA conforme o tempo acaba: a frequencia e a segunda
// informacao, e ela chega pela visao periferica de quem esta olhando para os
// proprios pes.
func drawFogWarning(screenWidth, screenHeight, left float32) {
	if left < 0 {
		left = 0
	}
	rate := 4 + 8*(1-clamp01(left/fogWarnLead))
	on := math.Mod(float64(rl.GetTime())*float64(rate), 2) < 1.15
	if !on {
		return
	}

	// Moldura vermelha na borda da tela. Ela nao tapa nada do campo de jogo, e
	// e o que a visao periferica pega sem o jogador desviar o olhar.
	rl.DrawRectangleLinesEx(rl.NewRectangle(0, 0, screenWidth, screenHeight),
		fogWarnBorder, rl.Fade(fogWarnRed, 0.75))

	txt := "PERIGO - NEVOA SOMBRIA"
	tw := rl.MeasureText(txt, fogWarnFontSize)
	tx := int32((screenWidth - float32(tw)) / 2)
	ty := int32(screenHeight*0.30)
	// Sombra atras: o texto tem de ser legivel sobre o chao claro do salao e
	// sobre a propria nevoa.
	rl.DrawText(txt, tx+3, ty+3, fogWarnFontSize, rl.Fade(rl.Black, 0.75))
	rl.DrawText(txt, tx, ty, fogWarnFontSize, fogWarnRed)

	sub := "ENTRE NA AREA ANGELICAL"
	sw2 := rl.MeasureText(sub, fogWarnSubSize)
	sy := ty + fogWarnFontSize + 8
	rl.DrawText(sub, int32((screenWidth-float32(sw2))/2)+2, sy+2, fogWarnSubSize,
		rl.Fade(rl.Black, 0.7))
	rl.DrawText(sub, int32((screenWidth-float32(sw2))/2), sy, fogWarnSubSize,
		fogWarnSubFg)
}

