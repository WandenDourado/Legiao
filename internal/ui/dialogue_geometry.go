package ui

// A geometria da cena de dialogo: onde fica a caixa e onde fica o retrato.
//
// Os dois moram na MESMA funcao de proposito. Antes a caixa saia de
// `dialogueRect` e a moldura do retrato era calculada dentro do desenho, a
// partir da caixa, com constantes soltas (`box.Height*1.47`, `frameH*0.72`,
// `box.X-frameW*0.87`). Duas fontes para uma composicao so significam que elas
// podem discordar — e o alvo de toque conhecia apenas uma delas.
//
// Aqui o retrato DERIVA da caixa, e e essa derivacao que faz o layout
// funcionar em qualquer proporcao de tela: 58% do retrato fica acima da linha
// de topo da caixa, entao a caixa cobre o quadril e nunca o peito, tanto num
// monitor 16:9 quanto num celular deitado.

import rl "github.com/gen2brain/raylib-go/raylib"

const (
	// Da caixa.
	dialogueRightMargin  = 0.055
	dialogueBottomMargin = 0.045
	dialogueHeightRatio  = 0.30
	dialoguePadding      = 22.0
	dialogueMinHeight    = 172.0
	// dialogueLineGap e dialogueNameGap sao os respiros do texto dentro da
	// caixa. Estavam cravados no desenho; moraram para ca quando a ALTURA da
	// caixa passou a ser calculada a partir deles (dialogueBoxHeight).
	dialogueLineGap = 6.0
	dialogueNameGap = 14.0
	// dialogueGuaranteedLines e quantas linhas de corpo a caixa se compromete
	// a caber no tamanho de fonte cheio. Cinco cobre a fala mais longa da
	// campanha (275 caracteres, mapa 6) nas telas que o jogo encontra; o que
	// passar disso e absorvido pelo encolhimento em fitDialogueBody, nunca
	// pelo corte.
	dialogueGuaranteedLines = 5.0
	// dialogueMaxHeightRatio impede que a escala de HUD no maximo transforme a
	// caixa em meia tela. Passado esse teto, quem cede e a fonte.
	dialogueMaxHeightRatio = 0.46

	// Do retrato.
	//
	// portraitAboveBox e a fracao do retrato que fica ACIMA do topo da caixa.
	// E o unico numero que decide se a caixa corta o personagem no quadril ou
	// no peito; mexer nele e mexer no enquadramento da cena inteira.
	portraitAboveBox = 0.58
	// portraitAspect e a proporcao da arte (1152x1280 = 0,90). Uma figura de
	// meio-corpo com braco erguido e manto aberto e mais larga que uma de
	// corpo inteiro — a arte do Mago mediu 0,86 de silhueta e estourava uma
	// tela 4:5.
	portraitAspect = 0.90
	// portraitLeftMargin afasta o retrato da borda esquerda o suficiente para
	// ele nao encostar, sem abrir um corredor vazio.
	portraitLeftMargin = 0.015
	// portraitBehindBox e quanto da LARGURA do retrato passa por tras da
	// caixa. E o que produz a sobreposicao das referencias: o busto sobe
	// livre e a caixa cruza por cima do quadril.
	portraitBehindBox = 0.62
	// Travas da coluna da caixa, para telas muito largas ou muito quadradas.
	dialogueBoxLeftMin = 0.16
	dialogueBoxLeftMax = 0.34
	// portraitMaxWidth impede que uma tela estreita receba um retrato mais
	// largo que a propria tela. E uma guarda para o caso degenerado, nao um
	// ajuste de composicao: numa tela 4:3 o retrato quer 0,554 da largura, e
	// apertar a trava abaixo disso encolheria o retrato num formato que o
	// jogo realmente encontra. Uma tela em pe (1080x2340) pediria 1,6x a
	// largura — e para essa que a trava existe.
	portraitMaxWidth = 0.58
)

// dialogueLayout devolve a caixa e o retrato da cena, nessa ordem.
//
// A caixa e calculada primeiro porque ela e quem tem restricao dura: precisa
// caber texto legivel e ficar acima do rodape. O retrato se acomoda a ela.
func dialogueLayout(screenWidth, screenHeight float32) (box, portrait rl.Rectangle) {
	boxHeight := dialogueBoxHeight(screenHeight)
	boxY := screenHeight - boxHeight - screenHeight*dialogueBottomMargin

	// O retrato sai pela borda de baixo da tela de proposito: e o corte na
	// coxa que faz o personagem parecer presente na cena em vez de colado
	// nela. A altura vem de quanto dele tem de sobrar acima da caixa — e e
	// essa derivacao que faz a borda de baixo cair exatamente no lugar.
	portraitHeight := (screenHeight - boxY) / (1 - portraitAboveBox)
	portraitWidth := portraitHeight * portraitAspect
	if widest := screenWidth * portraitMaxWidth; portraitWidth > widest {
		portraitWidth = widest
		portraitHeight = portraitWidth / portraitAspect
	}
	portraitX := screenWidth * portraitLeftMargin
	// A ancora e a LINHA DE TOPO DA CAIXA, nao a borda de baixo da tela.
	//
	// Sem a trava de largura as duas dao o mesmo numero — a altura do retrato
	// foi derivada exatamente para isso. Com a trava ativa elas divergem, e a
	// diferenca decide se ha cena: ancorado na borda de baixo, um retrato
	// encolhido numa tela em pe cabia INTEIRO abaixo do topo da caixa e o
	// orador sumia atras do painel. Ancorado aqui, o rosto sobe acima da
	// caixa em qualquer tela; o preco, so no caso degenerado, e o corte da
	// arte aparecer na faixa a esquerda do painel.
	portraitY := boxY - portraitHeight*portraitAboveBox

	boxX := portraitX + portraitWidth*portraitBehindBox
	if leftmost := screenWidth * dialogueBoxLeftMin; boxX < leftmost {
		boxX = leftmost
	}
	if rightmost := screenWidth * dialogueBoxLeftMax; boxX > rightmost {
		boxX = rightmost
	}

	box = rl.NewRectangle(
		boxX,
		boxY,
		screenWidth-boxX-screenWidth*dialogueRightMargin,
		boxHeight,
	)
	portrait = rl.NewRectangle(
		portraitX,
		portraitY,
		portraitWidth,
		portraitHeight,
	)
	return box, portrait
}

// dialogueBoxHeight e a altura da caixa: a razao da tela, ou o que o TEXTO
// exige, o que for maior.
//
// A razao sozinha bastava enquanto a fonte era fixa. Desde que ela passou a
// crescer com a tela e com a escala do jogador (dialogueFonts), uma caixa de
// 30% da altura deixou de ser garantia de nada: numa janela 1280x720 ela tem
// 216 px e o texto pede 250. Derivar do conteudo e o que mantem a promessa de
// que o que foi escrito cabe.
//
// Nao depende do TEXTO na tela, so dos tamanhos de fonte — de proposito: esta
// funcao tambem produz o alvo de toque (DialogueBoxRect), e um retangulo que
// mudasse a cada fala seria um alvo que se mexe embaixo do dedo.
func dialogueBoxHeight(screenHeight float32) float32 {
	// Os pixels fixos sao da tela nativa (Px, text.go).
	minH := Px(dialogueMinHeight)
	h := screenHeight * dialogueHeightRatio
	if h < minH {
		h = minH
	}
	name, body, hint := dialogueFonts(screenHeight)
	need := Px(dialoguePadding)*2 + float32(name) + Px(dialogueNameGap) +
		dialogueGuaranteedLines*(float32(body)+Px(dialogueLineGap)) + float32(hint)
	if need > h {
		h = need
	}
	max := screenHeight * dialogueMaxHeightRatio
	if max < minH {
		max = minH
	}
	if h > max {
		h = max
	}
	return h
}

// DialogueBoxRect expoe a area da caixa para a camada de input tratar um toque
// dentro dela como "proxima linha". Le a mesma funcao que o desenho, entao o
// alvo de toque e sempre o retangulo que esta na tela.
func DialogueBoxRect(screenWidth, screenHeight float32) rl.Rectangle {
	box, _ := dialogueLayout(screenWidth, screenHeight)
	return box
}
