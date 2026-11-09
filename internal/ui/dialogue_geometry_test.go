package ui

import "testing"

// As telas que o jogo realmente encontra. A ultima e retrato em pe: o jogo e
// paisagem, mas uma tela estreita nao pode PRODUZIR geometria invalida, e e
// justamente ali que a trava de largura entra.
var dialogueScreens = []struct {
	name          string
	width, height float32
}{
	{"1920x1080 desktop", 1920, 1080},
	{"2340x1080 celular", 2340, 1080},
	{"1440x1080 tablet", 1440, 1080},
	{"1280x720 HD", 1280, 720},
	{"3840x2160 4K", 3840, 2160},
	{"1080x2340 em pe", 1080, 2340},
}

// clamped diz se a trava de largura entrou. Ela muda o que se pode exigir do
// layout: com a trava ativa o retrato encolhe, entao a borda de baixo dele
// deixa de cair na borda da tela. Tudo o mais continua valendo.
func clamped(width, portraitWidth float32) bool {
	return portraitWidth >= width*portraitMaxWidth-0.01
}

// TestDialoguePortraitRunsOffTheBottom protege o corte na coxa.
//
// Se o retrato terminar acima da borda de baixo, a parte dele a ESQUERDA da
// caixa fica com uma barra reta de manto cortado boiando no ar — o defeito
// oposto ao que esta composicao busca. A caixa esconde o corte no trecho que
// cobre, mas nao no trecho que sobra ao lado dela.
func TestDialoguePortraitRunsOffTheBottom(t *testing.T) {
	for _, s := range dialogueScreens {
		_, portrait := dialogueLayout(s.width, s.height)
		base := portrait.Y + portrait.Height
		if base > s.height+0.01 {
			t.Errorf("%s: base do retrato em %.1f, abaixo da tela (%.1f)", s.name, base, s.height)
		}
		if portrait.Y < 0 {
			t.Errorf("%s: topo do retrato em %.1f, fora da tela", s.name, portrait.Y)
		}
		if clamped(s.width, portrait.Width) {
			// Tela degenerada: o retrato encolheu para caber na largura, entao
			// a borda de baixo dele sobe. Ver o comentario da ancora.
			continue
		}
		// Tolerancia e nao igualdade: em float32 a derivacao nao devolve o
		// mesmo bit que a subtracao direta.
		if base < s.height-0.01 {
			t.Errorf("%s: base do retrato em %.1f, esperado %.1f (borda de baixo)",
				s.name, base, s.height)
		}
	}
}

// TestDialogueBoxCrossesThePortrait protege a sobreposicao.
//
// Sem ela o retrato vira uma figura isolada ao lado de um painel, que e
// exatamente a leitura que esta mudanca saiu de.
func TestDialogueBoxCrossesThePortrait(t *testing.T) {
	for _, s := range dialogueScreens {
		box, portrait := dialogueLayout(s.width, s.height)
		if overlap := (portrait.X + portrait.Width) - box.X; overlap <= 0 {
			t.Errorf("%s: caixa comeca em %.1f, depois do retrato (fim em %.1f)",
				s.name, box.X, portrait.X+portrait.Width)
		}
		if box.Y <= portrait.Y {
			t.Errorf("%s: topo da caixa (%.1f) nao esta abaixo do topo do retrato (%.1f) — "+
				"o busto precisa subir livre acima do painel", s.name, box.Y, portrait.Y)
		}
	}
}

// TestDialogueBoxCutsTheHipNotTheChest e a invariante central do layout.
//
// portraitAboveBox e o unico numero que decide onde a caixa corta o
// personagem, e desde que a ancora passou a ser a linha de topo da caixa ele
// vale em TODA tela, trava de largura ativa ou nao.
func TestDialogueBoxCutsTheHipNotTheChest(t *testing.T) {
	for _, s := range dialogueScreens {
		box, portrait := dialogueLayout(s.width, s.height)
		above := (box.Y - portrait.Y) / portrait.Height
		if diff := above - portraitAboveBox; diff > 0.01 || diff < -0.01 {
			t.Errorf("%s: %.3f do retrato acima da caixa, esperado %.2f",
				s.name, above, portraitAboveBox)
		}
	}
}

// TestDialogueBoxStaysOnScreen cobre o resto: coluna dentro das travas,
// largura util positiva e nada saindo pela direita.
func TestDialogueBoxStaysOnScreen(t *testing.T) {
	for _, s := range dialogueScreens {
		box, portrait := dialogueLayout(s.width, s.height)
		if box.X < s.width*dialogueBoxLeftMin-0.01 || box.X > s.width*dialogueBoxLeftMax+0.01 {
			t.Errorf("%s: coluna da caixa em %.3f da largura, fora de [%.2f, %.2f]",
				s.name, box.X/s.width, dialogueBoxLeftMin, dialogueBoxLeftMax)
		}
		if box.Width <= dialoguePadding*2 {
			t.Errorf("%s: largura de %.1f nao cabe texto", s.name, box.Width)
		}
		if right := box.X + box.Width; right > s.width {
			t.Errorf("%s: caixa termina em %.1f, fora da tela", s.name, right)
		}
		if bottom := box.Y + box.Height; bottom > s.height {
			t.Errorf("%s: caixa termina em %.1f, abaixo da tela", s.name, bottom)
		}
		if portrait.Width > s.width*portraitMaxWidth+0.01 {
			t.Errorf("%s: retrato com %.3f da largura da tela, acima da trava %.2f",
				s.name, portrait.Width/s.width, portraitMaxWidth)
		}
	}
}

// TestDialogueBoxRectMatchesLayout e o teste que existe por causa de um risco
// concreto: o alvo de TOQUE e este retangulo, e o desenho e o outro. Se os
// dois se separarem, o jogador toca a caixa e nada avanca — no celular, sem
// teclado, isso trava a cena.
func TestDialogueBoxRectMatchesLayout(t *testing.T) {
	for _, s := range dialogueScreens {
		box, _ := dialogueLayout(s.width, s.height)
		if got := DialogueBoxRect(s.width, s.height); got != box {
			t.Errorf("%s: DialogueBoxRect %v != caixa desenhada %v", s.name, got, box)
		}
	}
}

// TestDialogueBoxHoldsGuaranteedLines e a invariante que nasceu com a fonte
// maior de 20/09/2026: a caixa tem de caber o nome do orador, as linhas que
// ela promete e a dica do rodape, sem depender do encolhimento de emergencia
// (fitDialogueBody). Antes disso a altura era 30% da tela e pronto — numa
// janela 1280x720 isso deixou de bastar no instante em que o corpo passou de
// 18 para 24 px.
func TestDialogueBoxHoldsGuaranteedLines(t *testing.T) {
	for _, s := range dialogueScreens {
		box, _ := dialogueLayout(s.width, s.height)
		name, body, hint := dialogueFonts(s.height)
		need := dialoguePadding*2 + float32(name) + dialogueNameGap +
			dialogueGuaranteedLines*(float32(body)+dialogueLineGap) + float32(hint)
		if need > s.height*dialogueMaxHeightRatio {
			// Tela em que o teto entra: quem cede e a fonte, nao a caixa.
			continue
		}
		if box.Height < need-0.01 {
			t.Errorf("%s: caixa com %.1f de altura, precisa de %.1f para %d linhas",
				s.name, box.Height, need, int(dialogueGuaranteedLines))
		}
	}
}

// TestDialogueBoxNeverEatsTheScreen protege o teto: com a escala de HUD no
// maximo a caixa cresce, mas nao vira meia tela — passado o teto, quem encolhe
// e a fonte (fitDialogueBody).
func TestDialogueBoxNeverEatsTheScreen(t *testing.T) {
	withHUDScale(t, 1.8)
	for _, s := range dialogueScreens {
		box, _ := dialogueLayout(s.width, s.height)
		if box.Height > s.height*dialogueMaxHeightRatio+0.01 {
			t.Errorf("%s: caixa com %.1f de altura, acima do teto %.1f",
				s.name, box.Height, s.height*dialogueMaxHeightRatio)
		}
		if bottom := box.Y + box.Height; bottom > s.height+0.01 {
			t.Errorf("%s: caixa termina em %.1f, abaixo da tela", s.name, bottom)
		}
	}
}
