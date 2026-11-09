package ui

import (
	"strings"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A geometria da caixa e do retrato mora em dialogue_geometry.go.

// placeholderPortraits lists characters whose reference art is still borrowed
// from somebody else. Drawing it would put another character's face on their
// line, which is worse than no portrait, so they speak unillustrated.
// Delete the entry as soon as the character has its own reference.png.
//
// Currently empty: every registered character owns its art. Installing a
// character with borrowed art means adding it here; installing its real art
// means removing it again, or the portrait stays silently blank.
var placeholderPortraits = map[entity.CharacterType]bool{}

// O cache de retratos e o ciclo de vida dele moram em portrait_cache.go.

// DrawDialogueBox draws the line the host is currently narrating. It draws
// nothing when no dialogue is running, so the caller can call it every frame.
//
// It is display only on every machine, host included: what appears here is
// whatever network.GetDialogueState() holds.
func DrawDialogueBox(screenWidth, screenHeight float32) {
	d := network.GetDialogueState()
	if !d.Active || d.Text == "" {
		return
	}

	box, portrait := dialogueLayout(screenWidth, screenHeight)

	// Dim the whole scene so the eye goes to the box and the frozen battle
	// behind it reads as paused rather than as a bug.
	rl.DrawRectangle(0, 0, int32(screenWidth), int32(screenHeight), rl.Fade(rl.Black, 0.52))

	// A ordem e o efeito: escurecimento, retrato, painel. O retrato ENTRE os
	// dois e o que produz a leitura das referencias — o busto sobe livre
	// acima da caixa e o painel cruza por cima do quadril. Desenhar o painel
	// antes devolveria a figurinha colada ao lado da caixa.
	if tex, ok := portraitTexture(d.Portrait); ok {
		drawDialoguePortrait(tex, portrait)
	}
	drawDialoguePanel(box)

	textX := box.X + Px(dialoguePadding)
	textWidth := box.Width - Px(dialoguePadding)*2

	nameSize, bodySize, hintSize := dialogueFonts(screenHeight)

	y := box.Y + Px(dialoguePadding)
	if d.Speaker != "" {
		drawDialogueNameplate(d.Speaker, box, nameSize)
		y += float32(nameSize) + Px(dialogueNameGap)
	}

	// Quanto de altura sobra para o corpo, descontando a dica no rodape.
	avail := box.Y + box.Height - Px(dialoguePadding) - float32(hintSize) - y
	lines, bodySize := fitDialogueBody(d.Text, int32(textWidth), bodySize, avail)

	for _, line := range lines {
		// Stop before spilling out of the box: a line that would be drawn
		// under the border is not readable anyway. Com o encolhimento acima
		// isto vira rede de seguranca, e nao o mecanismo — uma fala so chega
		// aqui se nem no piso ela coube.
		if y+float32(bodySize) > box.Y+box.Height-Px(dialoguePadding)-float32(hintSize) {
			break
		}
		rl.DrawText(line, int32(textX), int32(y), bodySize, rl.NewColor(42, 46, 42, 255))
		y += float32(bodySize) + Px(dialogueLineGap)
	}

	drawDialogueHint(d, box, hintSize)
}

// dialogueFonts devolve os tres tamanhos da cena: nome do orador, corpo da
// fala e dica de rodape.
//
// SUBIDOS EM 20/09/2026. As razoes eram 0.030/0.026/0.020 com piso 16 e teto
// 34 para os tres, e as duas pontas doiam: numa janela 1280x720 o corpo saia a
// 18,7 px — o texto que os testadores chamaram de ilegivel — e num monitor de
// 1440 ele PARAVA de crescer em 34, porque o teto estava abaixo do que a razao
// pedia. Agora cada papel tem a propria faixa, e a escala do jogador
// (Configuracoes > Interface) multiplica o resultado como em todo o resto do
// HUD (text.go).
func dialogueFonts(screenHeight float32) (name, body, hint int32) {
	return dialogueFontSize(screenHeight, 0.034, 24, 50),
		dialogueFontSize(screenHeight, 0.030, 24, 44),
		dialogueFontSize(screenHeight, 0.024, 18, 34)
}

// dialogueFontSize scales a font with the screen, keeps it inside the range
// authored for that role and then applies the player's HUD scale.
func dialogueFontSize(screenHeight, ratio, min, max float32) int32 {
	// Piso e teto sao pixels da tela nativa (Px, text.go).
	min, max = Px(min), Px(max)
	size := screenHeight * ratio
	if size < min {
		size = min
	}
	if size > max {
		size = max
	}
	return FitTextSize(size * hudScale)
}

// fitDialogueBody quebra a fala e, se ela nao couber na altura disponivel,
// ENCOLHE a fonte ate caber.
//
// Existe porque a caixa e um recipiente de tamanho fixo e a fala nao: a mais
// longa da campanha tem 275 caracteres (mapa 6), e com a fonte maior — ou com
// a escala do jogador em 1.8 — ela passa a pedir mais linhas do que a caixa
// tem. A alternativa era o corte silencioso do laco de desenho, que apaga o
// fim de uma fala de narrativa sem avisar ninguem. Perder tamanho e reversivel
// e visivel; perder texto nao.
//
// O piso e o de FitTextSize (MinTextSize em pixels da tela nativa): abaixo
// dele encolher nao resolve nada, so troca um problema de leitura por outro.
func fitDialogueBody(text string, maxWidth, size int32, avail float32) ([]string, int32) {
	if maxWidth <= 0 {
		return nil, size
	}
	floor := FitTextSize(0)
	for {
		lines := wrapText(text, size, maxWidth)
		if float32(len(lines))*(float32(size)+Px(dialogueLineGap)) <= avail || size <= floor {
			return lines, size
		}
		next := int32(float32(size) * 0.92)
		if next < floor {
			next = floor
		}
		size = next
	}
}

// wrapText breaks a paragraph into lines that fit maxWidth. A word longer than
// the whole line is left to overflow rather than split mid-word: that only
// happens with a malformed script, and a broken word is harder to spot than a
// line running long.
func wrapText(text string, fontSize int32, maxWidth int32) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, 4)
	current := words[0]
	for _, w := range words[1:] {
		candidate := current + " " + w
		if rl.MeasureText(candidate, fontSize) > maxWidth {
			lines = append(lines, current)
			current = w
			continue
		}
		current = candidate
	}
	return append(lines, current)
}
