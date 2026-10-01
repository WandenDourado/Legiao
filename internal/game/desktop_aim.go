package game

// "Mira segue o mouse" (Configuracoes > Controles > Desktop) e o pequeno
// helper que os dois modos (continuo e so-no-clique) compartilham: converter
// a posicao do mouse num vetor de mira valido para o AimJoystick, que e o
// mesmo estado que o chevron de mira (ui/aim_indicator.go) desenha tanto no
// desktop quanto no celular.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/input"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// aimDirTo e o vetor normalizado de from para to, ou o ultimo valor de
// referencia (zero) quando os dois pontos coincidem — Vector2Normalize de um
// vetor zero e NaN, e nunca deve chegar ao desenho.
func aimDirTo(from, to rl.Vector2) rl.Vector2 {
	delta := rl.Vector2Subtract(to, from)
	if rl.Vector2Length(delta) < 0.001 {
		return rl.NewVector2(0, 1)
	}
	return rl.Vector2Normalize(delta)
}

// updateDesktopAimFromMouse aponta o AimIndicator para o mouse TODO quadro —
// o modo "continuo" da opção de Controles. cam converte a posição de tela
// para o mundo, exatamente como o clique de ataque já faz.
func updateDesktopAimFromMouse(p *entity.Player, aj *input.AimJoystick, cam Camera2DState) {
	screenPos := rl.GetMousePosition()
	mousePos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
	aj.AimDir = aimDirTo(p.Position, mousePos)
}
