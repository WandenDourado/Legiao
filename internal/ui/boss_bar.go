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
	bossBarWidthFrac = 0.50 // fracao da largura da tela
	// bossBarLeftFrac reserva a faixa esquerda da tela para o HUD do jogador
	// (ui.DrawHealthBar, y=60, mesma faixa vertical da barra de chefe) nao
	// ficar por baixo dela. Sem isto a barra de chefe comecava exatamente onde
	// a de vida termina (as duas centradas em 20% da largura) e o texto
	// "HP/HP" do jogador aparecia dentro da moldura do chefe.
	//
	// 0.32 → 0.24 (31/08/2026, a pedido do Gui: a barra estava afastada demais
	// da borda esquerda). `DrawHealthBar` termina em ~20% da largura da tela
	// (barra de 0.20*sw a partir de x=10px); 0.24 deixa 4 pontos percentuais
	// de folga — o suficiente para o texto do chefe nao encostar no HUD do
	// jogador sem devolver a barra para perto do centro da tela.
	bossBarLeftFrac   = 0.24
	bossBarHeight     = 26
	bossBarTop        = 54 // abaixo do contador de horda, que fica no topo
	bossBarNameBase   = 26 // BASE: TextSize aplica a escala do jogador (text.go)
	bossBarSegments   = 4  // marcas a cada 25%
	bossBarFrameThick = 3

	// Aviso da conjuracao.
	fogWarnFontBase = 52
	fogWarnSubBase  = 26
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
	// Centrada na faixa que sobra a DIREITA da reserva do HUD do jogador, nao
	// na tela inteira — e o que a afasta da barra de vida e do contador de
	// jogadores sem grudar na borda direita.
	region := screenWidth * (1 - bossBarLeftFrac)
	x := screenWidth*bossBarLeftFrac + (region-w)/2
	y := Px(bossBarTop)
	// Altura e moldura em pixels da tela nativa (Px, text.go).
	barH := Px(bossBarHeight)

	// Nome centrado ACIMA da barra. Acima e nao dentro: dentro, o texto some
	// contra o preenchimento cheio e reaparece quando ele esvazia, o que faz o
	// nome piscar ao longo da luta.
	name := boss.Name
	if name == "" {
		name = "Chefe"
	}
	nameSize := TextSize(bossBarNameBase)
	tw := rl.MeasureText(name, nameSize)
	rl.DrawText(name, int32(x+(w-float32(tw))/2), int32(y-Px(4))-nameSize,
		nameSize, bossBarNameFg)

	rl.DrawRectangleRec(rl.NewRectangle(x, y, w, barH), bossBarBack)

	frac := boss.Health / boss.MaxHealth
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	if frac > 0 {
		fw := w * frac
		rl.DrawRectangleRec(rl.NewRectangle(x, y, fw, barH), bossBarFill)
		rl.DrawRectangleRec(rl.NewRectangle(x, y, fw, barH*0.35), bossBarShine)
	}

	// Marcas de 25%. Uma barra de 400 de vida sem subdivisao nao da sensacao de
	// progresso: o jogador bate por um minuto e ve o mesmo vermelho. Com as
	// marcas, cada quarto vencido e um marco visivel.
	for i := 1; i < bossBarSegments; i++ {
		mx := x + w*float32(i)/float32(bossBarSegments)
		rl.DrawRectangleRec(rl.NewRectangle(mx-Px(1), y, max(Px(2), 1), barH), bossBarMark)
	}

	rl.DrawRectangleLinesEx(rl.NewRectangle(x, y, w, barH),
		max(Px(bossBarFrameThick), 1), bossBarFrame)

	if boss.Casting {
		drawFogWarning(screenWidth, screenHeight, boss.CastLeft)
	}
}

// drawFogWarning e o aviso da conjuracao.
//
// Ele existe porque a danca — que e o telegrafo desenhado na propria chefe —
// acontece do outro lado da arena. Um grupo segurando um portao
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
		Px(fogWarnBorder), rl.Fade(fogWarnRed, 0.75))

	warnSize := TextSize(fogWarnFontBase)
	warnSubSize := TextSize(fogWarnSubBase)
	txt := "PERIGO - NEVOA SOMBRIA"
	tw := rl.MeasureText(txt, warnSize)
	tx := int32((screenWidth - float32(tw)) / 2)
	ty := int32(screenHeight*0.30)
	// Sombra atras: o texto tem de ser legivel sobre o chao claro do salao e
	// sobre a propria nevoa.
	off := int32(max(Px(3), 1))
	rl.DrawText(txt, tx+off, ty+off, warnSize, rl.Fade(rl.Black, 0.75))
	rl.DrawText(txt, tx, ty, warnSize, fogWarnRed)

	sub := "ENTRE NA AREA ANGELICAL"
	sw2 := rl.MeasureText(sub, warnSubSize)
	sy := ty + warnSize + int32(Px(8))
	rl.DrawText(sub, int32((screenWidth-float32(sw2))/2)+2, sy+2, warnSubSize,
		rl.Fade(rl.Black, 0.7))
	rl.DrawText(sub, int32((screenWidth-float32(sw2))/2), sy, warnSubSize,
		fogWarnSubFg)
}

