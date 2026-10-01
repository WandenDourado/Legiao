package ui

// applyAllSettings empurra um Settings inteiro para cada pacote que guarda a
// própria cópia do valor (internal/sound, internal/input, e este pacote via
// SetHUDScale/SetActionOpacity). settings.Set/Update sozinhos só trocam o
// MODELO — quem já leu um valor antigo (o barramento de música tocando
// agora, o multiplicador de escala do botão) não é avisado sozinho.
//
// Chamada em dois lugares: uma vez no início do processo, logo depois de
// settings.Load() (game/loop.go), e de novo sempre que "Restaurar padrões"
// troca tudo de uma vez (settings_section_player.go) — um valor por um
// valor nesse caso reconstruiria exatamente esta função na mão.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/sound"
	"github.com/WandenDourado/Legiao/internal/tilemap"
)

// ApplyAllSettings is exported: game.Run (a different package) calls it once
// at startup, right after settings.Load(), to bring every bus/geometry
// value in line with whatever was on disk before the first frame draws.
func ApplyAllSettings(s settings.Settings) {
	applyAllSettings(s)
}

func applyAllSettings(s settings.Settings) {
	sound.SetMasterVolume(s.MasterVolume)
	sound.SetMuteMaster(s.MuteMaster)
	sound.SetMusicBusVolume(s.MusicVolume)
	sound.SetMuteMusic(s.MuteMusic)
	sound.SetSkillVolume(s.SkillVolume)
	sound.SetMuteSkill(s.MuteSkill)
	sound.SetAttackVolume(s.AttackVolume)
	sound.SetMuteAttack(s.MuteAttack)

	input.SetActionButtonScale(s.ActionButtonScale)
	input.SetJoystickRadiusScale(s.JoystickRadiusScale)
	input.SetLeftHanded(s.LeftHanded)
	SetActionOpacity(s.ControlsOpacity)

	SetHUDScale(s.HUDScale)
	applyTargetFPS(s.TargetFPS)
	entity.SetDeathFXEnabled(s.DeathEffect)
	tilemap.SetGroundDetailVisible(s.GroundDetail)
}
