package ui

import "testing"

// withHUDScale troca a escala global do pacote e devolve ao fim do teste. Ela
// e um var de pacote (hud.go) porque e uma preferencia, nao um parametro.
func withHUDScale(t *testing.T, v float32) {
	t.Helper()
	prev := hudScale
	t.Cleanup(func() { hudScale = prev })
	hudScale = v
}

// TestTextSizeAppliesScaleAndFloor e o contrato de text.go: a base e o que se
// ve na escala 1.0, a escala do jogador multiplica, e nada desce abaixo do
// piso — que e o que a queixa de 20/09/2026 realmente pedia.
func TestTextSizeAppliesScaleAndFloor(t *testing.T) {
	withHUDScale(t, 1.0)
	if got := TextSize(28); got != 28 {
		t.Errorf("TextSize(28) na escala 1.0 = %d, esperado 28", got)
	}
	if got := TextSize(10); got != MinTextSize {
		t.Errorf("TextSize(10) = %d, esperado o piso %d", got, MinTextSize)
	}

	withHUDScale(t, 1.8)
	if got := TextSize(28); got != 50 {
		t.Errorf("TextSize(28) na escala 1.8 = %d, esperado 50", got)
	}

	withHUDScale(t, 0.8)
	// 28*0.8 = 22,4: encolhe, mas continua acima do piso.
	if got := TextSize(28); got != 22 {
		t.Errorf("TextSize(28) na escala 0.8 = %d, esperado 22", got)
	}
	// Uma base pequena com escala pequena ainda para no piso.
	if got := TextSize(16); got != MinTextSize {
		t.Errorf("TextSize(16) na escala 0.8 = %d, esperado o piso %d", got, MinTextSize)
	}
}

// TestFitTextSizeIgnoresScale protege a outra metade da regra: um tamanho que
// ja veio de uma geometria escalada (o numero dentro do pip de recarga) nao
// pode levar a escala duas vezes.
func TestFitTextSizeIgnoresScale(t *testing.T) {
	withHUDScale(t, 1.8)
	if got := FitTextSize(30); got != 30 {
		t.Errorf("FitTextSize(30) = %d, esperado 30 — a escala nao se aplica aqui", got)
	}
	if got := FitTextSize(9); got != MinTextSize {
		t.Errorf("FitTextSize(9) = %d, esperado o piso %d", got, MinTextSize)
	}
}

// TestDialogueFontsGrowWithScreenAndScale cobre as duas pontas que doiam antes
// de 20/09/2026: o corpo minusculo numa janela 720p e o teto que impedia o
// texto de crescer num monitor grande.
func TestDialogueFontsGrowWithScreenAndScale(t *testing.T) {
	withHUDScale(t, 1.0)

	_, body720, _ := dialogueFonts(720)
	if body720 < 24 {
		t.Errorf("corpo do dialogo em 720p = %d; era 18 antes e foi exatamente a queixa", body720)
	}

	_, body1440, _ := dialogueFonts(1440)
	if body1440 <= 34 {
		t.Errorf("corpo do dialogo em 1440p = %d; o teto antigo (34) continua prendendo", body1440)
	}

	_, bodySmall, _ := dialogueFonts(1080)
	withHUDScale(t, 1.8)
	_, bodyBig, _ := dialogueFonts(1080)
	if bodyBig <= bodySmall {
		t.Errorf("escala 1.8 nao aumentou o corpo do dialogo (%d -> %d)", bodySmall, bodyBig)
	}

	name, body, hint := dialogueFonts(1080)
	if !(name >= body && body > hint) {
		t.Errorf("hierarquia quebrada: nome %d, corpo %d, dica %d", name, body, hint)
	}
}

// Em 540p numa tela de 1080 (renderScale 0,5) o texto tem de ocupar a MESMA
// fracao da tela que na nativa: metade dos pixels da janela, e o scaler do
// aparelho dobra. Antes, 28 px continuavam 28 e a HUD "inchava" (29/09/2026).
func TestTextSizeFollowsRenderScale(t *testing.T) {
	withHUDScale(t, 1.0)
	prev := renderScale
	t.Cleanup(func() { renderScale = prev })

	SetRenderScale(0.5)
	if got := TextSize(28); got != 14 {
		t.Errorf("TextSize(28) em 540p = %d, esperado 14", got)
	}
	// O piso segue a escala (18 -> 9), mas nao desce do tamanho de desenho
	// da fonte padrao (10).
	if got := FitTextSize(4); got != 10 {
		t.Errorf("piso em 540p = %d, esperado 10", got)
	}
	SetRenderScale(0.75)
	if got := FitTextSize(4); got != 13 {
		t.Errorf("piso em 810p = %d, esperado 13", got)
	}
	SetRenderScale(0.5)
	if got := Px(760); got != 380 {
		t.Errorf("Px(760) em 540p = %v, esperado 380", got)
	}

	SetRenderScale(0) // valor invalido: ignora
	if renderScale != 0.5 {
		t.Errorf("SetRenderScale(0) mudou a escala para %v", renderScale)
	}
}
