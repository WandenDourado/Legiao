//go:build android

package ui

// Seção Controles no Android (Passo 5): tamanho e opacidade dos botões,
// modo canhoto e raio do joystick aplicam na hora via internal/input
// (control_prefs.go) e internal/ui (action_button.go). Joystick fixo ou
// dinâmico ficou de fora — ver doc/settings.md.

import (
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/settings"
)

func (scr *SettingsScreen) updateControlsSection(l settingsLayout) {
	s := settings.Get()

	rowLabel(l, 0, "TAMANHO DOS BOTOES")
	res := scr.actionScaleSlider.Update(l.controlRect(0), scaleToUnit(s.ActionButtonScale, 0.8, 1.3))
	DrawSlider(l.controlRect(0), res.Value)
	if res.Changed {
		v := unitToScale(res.Value, 0.8, 1.3)
		input.SetActionButtonScale(v)
		settings.Update(func(s *settings.Settings) { s.ActionButtonScale = v })
	}

	rowLabel(l, 1, "OPACIDADE DOS CONTROLES")
	res = scr.opacitySlider.Update(l.controlRect(1), s.ControlsOpacity)
	DrawSlider(l.controlRect(1), res.Value)
	if res.Changed {
		SetActionOpacity(res.Value)
		settings.Update(func(s *settings.Settings) { s.ControlsOpacity = res.Value })
	}

	rowLabel(l, 2, "MODO CANHOTO")
	rowHint(l, 2, "espelha o cluster: joystick a direita, acoes a esquerda")
	if v, changed := ToggleRow(l.toggleRect(2), s.LeftHanded); changed {
		input.SetLeftHanded(v)
		settings.Update(func(s *settings.Settings) { s.LeftHanded = v })
	}

	rowLabel(l, 3, "RAIO DO JOYSTICK")
	res = scr.radiusSlider.Update(l.controlRect(3), scaleToUnit(s.JoystickRadiusScale, 0.7, 1.5))
	DrawSlider(l.controlRect(3), res.Value)
	if res.Changed {
		v := unitToScale(res.Value, 0.7, 1.5)
		input.SetJoystickRadiusScale(v)
		settings.Update(func(s *settings.Settings) { s.JoystickRadiusScale = v })
	}

	rowLabel(l, 4, "JOYSTICK FIXO OU DINAMICO")
	rowHint(l, 4, "fora desta rodada — ver doc/settings.md")
}

