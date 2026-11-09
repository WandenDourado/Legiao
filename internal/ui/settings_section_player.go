package ui

// Seção Jogador: nome de exibição (LOCAL — ver doc/settings.md, não é o
// PlayerID de rede que host_rejoin.go usa para reconexão) e Restaurar
// padrões.

import (
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/sound"
)

const maxPlayerNameRunes = 20

func (scr *SettingsScreen) updatePlayerSection(l settingsLayout) {
	s := settings.Get()

	rowLabel(l, 0, "NOME DO JOGADOR")
	rowHint(l, 0, "so local — nao viaja pela rede nesta rodada")
	fieldRect := l.controlRect(0)
	if v, changed := scr.nameField.Update(fieldRect, s.PlayerName, maxPlayerNameRunes); changed {
		settings.Update(func(s *settings.Settings) { s.PlayerName = v })
	}
	scr.nameField.Draw(fieldRect, s.PlayerName, "Sem nome")

	rowLabel(l, 2, "RESTAURAR PADROES")
	if SmallButton(l.controlRect(2), "RESTAURAR") {
		scr.applyDefaults()
	}
}

// applyDefaults volta tudo (Settings.Default()) e reaplica em cada pacote
// que guarda uma cópia própria do valor (internal/sound, internal/input,
// internal/ui) — settings.Set() sozinho só troca o modelo, não avisa quem
// leu um valor antigo e guardou.
func (scr *SettingsScreen) applyDefaults() {
	d := settings.Set(settings.Default())
	applyAllSettings(d)
	// A prévia confirma, de ouvido, que o áudio realmente voltou ao padrão —
	// sem ela "restaurar" é só um número mudando na tela.
	sound.PlayUI(sound.SFXAttackMago)
}
