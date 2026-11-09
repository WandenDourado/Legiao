package ui

// Aba Graficos (28/09/2026): qualidade (Baixa/Media/Alta), teste automatico,
// resolucao de desenho do celular e as duas chaves de efeito. Estudo e
// numeros em plans/plan_qualidade_grafica_mobile.md. Desde 29/09/2026 a
// linha de resolucao mostra o tamanho valendo e oferece REINICIAR AGORA
// (render_restart.go).
//
// A qualidade e um NOME para uma combinacao de campos (settings/quality.go):
// escolher um nivel liga os campos dele, e mexer num campo a mao deixa o
// nivel "personalizado". Nada disto entra na rede.

import (
	"fmt"
	"log"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/tilemap"
)

var qualityLabels = []string{"BAIXA", "MEDIA", "ALTA"}

var qualityPresets = []settings.Preset{settings.PresetLow, settings.PresetMedium, settings.PresetHigh}

func renderHeightLabel(h int32) string {
	if h == 0 {
		return "NATIVA"
	}
	return fmt.Sprintf("%dp", h)
}

func (scr *SettingsScreen) updateGraphicsSection(l settingsLayout) {
	if scr.bench != nil {
		scr.updateBench(l)
		return
	}
	s := settings.Get()

	rowLabel(l, 0, "QUALIDADE")
	active := -1
	preset := settings.PresetOf(s)
	for i, p := range qualityPresets {
		if p == preset {
			active = i
		}
	}
	if active < 0 {
		rowHint(l, 0, "personalizada")
	}
	if idx := ChoiceRow(l.controlRect(0), qualityLabels, active); idx >= 0 {
		applyGraphics(func(s *settings.Settings) { settings.ApplyPreset(s, qualityPresets[idx]) })
	}

	rowLabel(l, 1, "CONFIGURAR SOZINHO")
	if scr.benchResult != "" {
		rowHint(l, 1, scr.benchResult)
	} else {
		rowHint(l, 1, "testa este aparelho por uns segundos e escolhe")
	}
	if SmallButton(l.buttonRect(1), "TESTAR") {
		scr.bench = newPerfBench(s)
		scr.benchResult = ""
	}

	rowLabel(l, 2, "RESOLUCAO")
	if mobileBuild {
		// O tamanho VALENDO, nao so a regra: sem isto nao havia como saber
		// se a troca pegou (relato de 29/09/2026, render_restart.go).
		rowHint(l, 2, renderNowHint(s))
		labels := make([]string, len(settings.RenderHeights))
		cur := -1
		for i, h := range settings.RenderHeights {
			labels[i] = renderHeightLabel(h)
			if h == s.RenderHeight {
				cur = i
			}
		}
		if idx := ChoiceRow(l.controlRect(2), labels, cur); idx >= 0 {
			h := settings.RenderHeights[idx]
			applyGraphics(func(s *settings.Settings) { s.RenderHeight = h })
		}
	} else {
		rowHint(l, 2, "no PC e sempre a do monitor (a maior possivel)")
	}

	rowLabel(l, 3, "MONSTROS VIRAM CINZAS")
	rowHint(l, 3, "efeito de morte; desligar alivia o celular")
	if v, changed := ToggleRow(l.toggleRect(3), s.DeathEffect); changed {
		applyGraphics(func(s *settings.Settings) { s.DeathEffect = v })
	}

	rowLabel(l, 4, "FLORES E GALHOS NO CHAO")
	rowHint(l, 4, "enfeite do mapa; pesa pouco")
	if v, changed := ToggleRow(l.toggleRect(4), s.GroundDetail); changed {
		applyGraphics(func(s *settings.Settings) { s.GroundDetail = v })
	}

	scr.updateRestartButton(l)
}

// applyGraphics muda o modelo e empurra para quem guarda a propria copia: o
// teto de quadro (raylib), a desintegracao (entity) e o detalhe do chao
// (tilemap). A resolucao so vale quando a janela nasce e nao tem o que
// empurrar.
//
// E GRAVA NA HORA, ao contrario das outras abas (que gravam so ao SAIR da
// tela): a resolucao so tem efeito depois de o jogo fechar, e fechar o app
// direto da tela de Configuracoes — o gesto natural para "reabrir" — perdia
// a escolha. Aqui so ha cliques, nunca arraste, entao gravar custa um
// arquivo pequeno por toque.
func applyGraphics(fn func(*settings.Settings)) {
	s := settings.Update(fn)
	applyTargetFPS(s.TargetFPS)
	entity.SetDeathFXEnabled(s.DeathEffect)
	tilemap.SetGroundDetailVisible(s.GroundDetail)
	if err := settings.Save(); err != nil {
		log.Printf("[Graficos] falha ao gravar settings.json: %v", err)
	}
}
