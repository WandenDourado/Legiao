package ui

// Seção Áudio: quatro barramentos (Passo 2) mais o mudo em segundo plano.
// Cada slider aplica na hora (settings.Update + o setter de internal/sound),
// e uma prévia sonora toca ao SOLTAR — nunca durante o arraste, que viraria
// metralhadora (Passo 4, item 4).

import (
	"fmt"

	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/sound"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// previewSFX é o efeito representativo de cada barramento, tocado ao soltar
// o slider (sound.PlayUI, sem espacialização — não existe jogador local na
// tela de Configurações). Master usa o mesmo som do ataque: não tem timbre
// próprio, e o ataque é o efeito mais curto e menos intrusivo do catálogo.
const (
	previewSkillSFX  = sound.SFXFireLaunch
	previewAttackSFX = sound.SFXAttackMago
)

func (scr *SettingsScreen) updateAudioSection(l settingsLayout) {
	s := settings.Get()

	scr.audioBusRow(l, 0, "VOLUME GERAL", &scr.masterSlider, s.MasterVolume, s.MuteMaster,
		func(v float32) { sound.SetMasterVolume(v); settings.Update(func(s *settings.Settings) { s.MasterVolume = v }) },
		func(m bool) { sound.SetMuteMaster(m); settings.Update(func(s *settings.Settings) { s.MuteMaster = m }) },
		"") // master has no single instrument of its own to preview

	scr.audioBusRow(l, 1, "MUSICA", &scr.musicSlider, s.MusicVolume, s.MuteMusic,
		func(v float32) { sound.SetMusicBusVolume(v); settings.Update(func(s *settings.Settings) { s.MusicVolume = v }) },
		func(m bool) { sound.SetMuteMusic(m); settings.Update(func(s *settings.Settings) { s.MuteMusic = m }) },
		"") // a trilha já está tocando: a resposta em tempo real É a prévia (Passo 4, item 4)

	scr.audioBusRow(l, 2, "HABILIDADES", &scr.skillSlider, s.SkillVolume, s.MuteSkill,
		func(v float32) { sound.SetSkillVolume(v); settings.Update(func(s *settings.Settings) { s.SkillVolume = v }) },
		func(m bool) { sound.SetMuteSkill(m); settings.Update(func(s *settings.Settings) { s.MuteSkill = m }) },
		previewSkillSFX)

	scr.audioBusRow(l, 3, "ATAQUE BASICO", &scr.attackSlider, s.AttackVolume, s.MuteAttack,
		func(v float32) { sound.SetAttackVolume(v); settings.Update(func(s *settings.Settings) { s.AttackVolume = v }) },
		func(m bool) { sound.SetMuteAttack(m); settings.Update(func(s *settings.Settings) { s.MuteAttack = m }) },
		previewAttackSFX)

	rowLabel(l, 4, "SILENCIAR EM SEGUNDO PLANO")
	rowHint(l, 4, "desktop: janela sem foco   mobile: app minimizado")
	if v, changed := ToggleRow(l.toggleRect(4), s.MuteBackground); changed {
		settings.Update(func(s *settings.Settings) { s.MuteBackground = v })
	}
}

// audioBusRow draws one "LABEL  ====slider====  [mute]  NN%" row and wires
// its three interactions: drag (setVol on every change), release (preview
// sound), and the mute button (setMute, value preserved).
func (scr *SettingsScreen) audioBusRow(l settingsLayout, i int, label string, sl *Slider, value float32, muted bool,
	setVol func(float32), setMute func(bool), preview sound.SFX) {

	rowLabel(l, i, label)
	ctrl := l.controlRect(i)
	muteRect := rl.NewRectangle(ctrl.X+ctrl.Width+ctrl.Height*0.15, ctrl.Y-ctrl.Height*0.1, ctrl.Height*1.2, ctrl.Height*1.2)
	sliderRect := rl.NewRectangle(ctrl.X, ctrl.Y, ctrl.Width-ctrl.Height*1.5, ctrl.Height*0.42)
	sliderRect.Y = ctrl.Y + ctrl.Height*0.29

	res := sl.Update(sliderRect, value)
	DrawSlider(sliderRect, res.Value)
	if res.Changed {
		setVol(res.Value)
	}
	if res.Released && preview != "" {
		sound.PlayUI(preview)
	}

	pct := fmt.Sprintf("%3.0f%%", res.Value*100)
	size := int32(max(ctrl.Height*0.42, Px(13)))
	rl.DrawText(pct, int32(sliderRect.X+sliderRect.Width+10), int32(ctrl.Y+ctrl.Height*0.1), size, rl.Fade(rl.White, 0.85))

	if newMuted := MuteButton(muteRect, muted); newMuted {
		setMute(!muted)
	}
}
