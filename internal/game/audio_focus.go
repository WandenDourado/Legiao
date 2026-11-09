package game

// "Silenciar em segundo plano" (Passo 5, seção Áudio): aplicado uma vez por
// quadro a partir do estado de foco da janela.
//
// MELHOR ESFORÇO DOCUMENTADO: rl.IsWindowFocused() é o único sinal
// cross-platform que o raylib-go expõe sem uma chamada específica de
// plataforma, e este ambiente de execução não tem como confirmar (sem
// SDK/NDK Android, sem `go build` disponível — ver AGENTS.md) se ele
// reflete "app minimizado" da mesma forma que reflete "janela sem foco" no
// desktop. É o candidato razoável e é o que está implementado; testar em
// campo no celular (minimizar o app com música tocando) é item da lista de
// verificação manual — ver doc/settings.md.
import (
	"github.com/WandenDourado/Legiao/internal/settings"
	"github.com/WandenDourado/Legiao/internal/sound"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// updateBackgroundAudioMute reads the current setting and window focus and
// applies (or lifts) the suppression. Independent of the player's own
// master mute — see sound.SetFocusSuppressed.
func updateBackgroundAudioMute() {
	s := settings.Get()
	if !s.MuteBackground {
		sound.SetFocusSuppressed(false)
		return
	}
	sound.SetFocusSuppressed(!rl.IsWindowFocused())
}
