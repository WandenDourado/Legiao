package settings

// Default são os valores de fábrica. MusicVolume e SkillVolume/AttackVolume
// preservam os antigos `sound.Volume` (0.5) e `sound.SFXVolume` (0.85), que
// eram constantes antes deste pacote existir — ver doc/audio.md.
func Default() Settings {
	return Settings{
		MasterVolume: 1.0,
		MusicVolume:  0.5,
		SkillVolume:  0.85,
		AttackVolume: 0.85,
		// Ligado por padrão: "nada pior que música tocando no bolso" é o
		// argumento do próprio pedido, e é uma mudança de comportamento
		// deliberada em relação ao jogo antes desta tela existir (ele nunca
		// mutava sozinho). Documentado em doc/settings.md.
		MuteBackground: true,

		TargetFPS:    FrameCap60,
		DeathEffect:  true,
		GroundDetail: true,
		VSync:        true,
		Fullscreen:   false,

		ActionButtonScale:   1.0,
		ControlsOpacity:     1.0,
		LeftHanded:          false,
		JoystickRadiusScale: 1.0,

		ContinuousMouseAim: false,

		HUDScale:    1.0,
		ShowAimHint: true,

		PlayerName: "",
	}
}

// clamped devolve s com cada campo dentro da própria faixa, campo a campo —
// nunca o struct inteiro descartado por um valor ruim. É o que deixa um JSON
// parcialmente editado à mão (ou um campo futuro cujo default mudou) cair em
// algo jogável em vez de travar Load().
func (s Settings) clamped() Settings {
	s.MasterVolume = clamp01(s.MasterVolume)
	s.MusicVolume = clamp01(s.MusicVolume)
	s.SkillVolume = clamp01(s.SkillVolume)
	s.AttackVolume = clamp01(s.AttackVolume)

	s.RenderHeight = clampRenderHeight(s.RenderHeight)
	if s.DisplayWidth < 0 || s.DisplayHeight < 0 {
		s.DisplayWidth, s.DisplayHeight = 0, 0
	}

	switch s.TargetFPS {
	case FrameCapUncapped, FrameCap30, FrameCap60:
	default:
		s.TargetFPS = FrameCap60
	}

	s.ActionButtonScale = clampF(s.ActionButtonScale, 0.8, 1.3)
	s.ControlsOpacity = clamp01(s.ControlsOpacity)
	s.JoystickRadiusScale = clampF(s.JoystickRadiusScale, 0.7, 1.5)

	s.HUDScale = clampF(s.HUDScale, 0.8, HUDScaleMax)

	if len([]rune(s.PlayerName)) > maxPlayerNameLen {
		r := []rune(s.PlayerName)
		s.PlayerName = string(r[:maxPlayerNameLen])
	}
	return s
}

func clamp01(v float32) float32 { return clampF(v, 0, 1) }

func clampF(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
