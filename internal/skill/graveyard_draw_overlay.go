package skill

// O PASSE DE SOBREPOSICAO do Cemitério: o que esta acima do chao e continua
// passando na frente dos corpos. Ver o cabecalho de graveyard_draw.go.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawOverlay renders the skeletal hands and the rising souls, in world space.
// Chamado no bloco de magias, depois dos corpos — a mao agarrando a perna de
// quem esta dentro do cemiterio e o efeito, nao um defeito.
func (g *Graveyard) DrawOverlay() {
	if !g.visible() {
		return
	}
	f := g.FadeAlpha()

	// Skeleton hands clawing out of the dirt.
	for i := range g.hands {
		g.drawHand(&g.hands[i], f)
	}

	// Souls drifting upward.
	rl.BeginBlendMode(rl.BlendAdditive)
	for _, pt := range g.Souls.particles {
		drawParticle(pt)
	}
	rl.EndBlendMode()
}

// drawHand renders one skeletal hand: palm, four clawing fingers with joints
// and a thumb, with a slow desperate sway. O buraco de onde ela sai e desenhado
// no passe de chao (drawHandHole).
func (g *Graveyard) drawHand(h *graveHand, fade float32) {
	rise := g.handRise(h)
	if rise <= 0 {
		return
	}
	base := g.point(h.u, h.v)
	t := float32(rl.GetTime())
	sway := float32(math.Sin(float64(t)*1.9+float64(h.phase))) * 2.2 * h.scale
	s := h.scale * rise

	bone := rl.Fade(graveBone, fade)
	shade := rl.Fade(rl.NewColor(150, 140, 118, 255), fade)
	wrist := rl.NewVector2(base.X+sway*0.4, base.Y-7*s)
	rl.DrawLineEx(base, wrist, 5*s, shade) // forearm bone
	palm := rl.NewVector2(wrist.X+sway*0.6, wrist.Y-8*s)
	rl.DrawCircleV(palm, 5.2*s, bone) // palm

	// Four fingers fanning upward, two segments each, curling like claws.
	for i := 0; i < 4; i++ {
		fx := (float32(i) - 1.5) * 4.6 * s
		knee := rl.NewVector2(palm.X+fx+sway, palm.Y-9*s)
		tip := rl.NewVector2(knee.X+fx*0.35+sway*0.6, knee.Y-7*s+float32(i%2)*2*s)
		rl.DrawLineEx(palm, knee, 2.4*s, bone)
		rl.DrawLineEx(knee, tip, 1.8*s, bone)
		rl.DrawCircleV(knee, 1.6*s, shade) // knuckle
	}
	// Thumb to the side.
	thumb := rl.NewVector2(palm.X+8*s*h.flip, palm.Y-3*s)
	rl.DrawLineEx(palm, thumb, 2.2*s, bone)

	// Faint green grave-light licking the bones.
	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(palm.X), int32(palm.Y-6*s), 16*s,
		rl.Fade(graveGreen, 0.16*fade), rl.Blank)
	rl.EndBlendMode()
}
