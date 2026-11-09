//go:build !android

package ui

// Seção Controles no desktop (Passo 5): mira contínua vs só-no-clique é a
// única opção que entra nesta rodada. Remapear teclas exige uma tabela de
// binds inteira — fica marcado como fase 2 (ver doc/settings.md) em vez de
// meio-feito.

import (
	"github.com/WandenDourado/Legiao/internal/settings"
)

func (scr *SettingsScreen) updateControlsSection(l settingsLayout) {
	s := settings.Get()

	rowLabel(l, 0, "MIRA SEGUE O MOUSE")
	rowHint(l, 0, "desligado: so aponta no instante do clique")
	if v, changed := ToggleRow(l.toggleRect(0), s.ContinuousMouseAim); changed {
		settings.Update(func(s *settings.Settings) { s.ContinuousMouseAim = v })
	}

	rowLabel(l, 1, "REMAPEAR TECLAS")
	rowHint(l, 1, "fora desta rodada — exige tabela de binds (fase 2)")
}
