package ui

// Seção Interface (Passo 5): escala do HUD e o indicador de mira aplicam na
// hora. "Mostrar nomes dos jogadores" ficou de fora — depende de um campo de
// nome replicado (JoinPayload/PlayerState, internal/network) que este round
// não abre, porque a regra não-negociável do pedido é que Configurações
// NUNCA entra na rede (ver doc/settings.md).

import (
	"github.com/WandenDourado/Legiao/internal/settings"
)

func (scr *SettingsScreen) updateInterfaceSection(l settingsLayout) {
	s := settings.Get()

	// "E DO TEXTO" nao e enfeite: desde 20/09/2026 este slider rege todo o
	// texto do jogo, dialogo incluido, e nao so o tamanho da barra de vida.
	rowLabel(l, 0, "ESCALA DO HUD E DO TEXTO")
	res := scr.hudScaleSlider.Update(l.controlRect(0), scaleToUnit(s.HUDScale, 0.8, settings.HUDScaleMax))
	DrawSlider(l.controlRect(0), res.Value)
	if res.Changed {
		v := unitToScale(res.Value, 0.8, settings.HUDScaleMax)
		SetHUDScale(v)
		settings.Update(func(s *settings.Settings) { s.HUDScale = v })
	}

	rowLabel(l, 1, "MOSTRAR INDICADOR DE MIRA")
	if v, changed := ToggleRow(l.toggleRect(1), s.ShowAimHint); changed {
		settings.Update(func(s *settings.Settings) { s.ShowAimHint = v })
	}

	rowLabel(l, 2, "MOSTRAR NOMES DOS JOGADORES")
	rowHint(l, 2, "fora desta rodada — ver doc/settings.md")
}
