package skill

// O PASSE DE CHAO do Cemitério: tudo que esta no nivel do solo e por isso tem
// de ser desenhado ANTES dos corpos. Ver o cabecalho de graveyard_draw.go.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DrawGround renders the cursed ground itself, in world space, back to front.
// Chamado antes dos corpos (game/renderer.go), junto com o selo do Mago.
func (g *Graveyard) DrawGround() {
	if !g.visible() {
		return
	}
	f := g.FadeAlpha()
	angle := float32(math.Atan2(float64(g.Dir.Y), float64(g.Dir.X))) * 180 / math.Pi
	center := g.Center()
	emerge := clamp01(g.Age / 0.5) // ground rots in quickly on cast

	// Rotting ground: two rotated rects (dark core + bruised rim) plus
	// irregular rot blotches so the decay looks organic.
	rimRect := rl.NewRectangle(center.X, center.Y, GraveyardLength+26, GraveyardWidth+26)
	rl.DrawRectanglePro(rimRect, rl.NewVector2(rimRect.Width/2, rimRect.Height/2), angle,
		rl.Fade(graveRotRim, 0.42*f*emerge))
	rect := rl.NewRectangle(center.X, center.Y, GraveyardLength, GraveyardWidth)
	rl.DrawRectanglePro(rect, rl.NewVector2(rect.Width/2, rect.Height/2), angle,
		rl.Fade(graveRot, 0.72*f*emerge))
	for i, h := range g.hands { // reuse hand spots as blotch anchors
		r := (18 + float32(i%3)*9) * emerge
		rl.DrawCircleV(g.point(h.u, h.v*0.9), r, rl.Fade(graveRot, 0.5*f))
	}

	// Sickly green miasma pooling over the rot (additive, breathing).
	//
	// Pousa no chao, e nao no ar: aditivo sobre o rosto de quem estivesse
	// dentro era metade do brilho esverdeado que os testadores leram como
	// poluicao. E o mesmo argumento do selo do Mago.
	t := float32(rl.GetTime())
	breath := 0.5 + 0.5*float32(math.Sin(float64(t)*1.6))
	rl.BeginBlendMode(rl.BlendAdditive)
	for i := 0; i < 3; i++ {
		u := GraveyardLength * (0.2 + 0.3*float32(i))
		p := g.point(u, float32(math.Sin(float64(t)*0.7+float64(i)*2.1))*GraveyardWidth*0.2)
		rl.DrawCircleGradient(int32(p.X), int32(p.Y), 70+14*breath,
			rl.Fade(graveGreen, (0.10+0.05*breath)*f), rl.Blank)
	}
	rl.EndBlendMode()

	// O buraco de onde cada mao sai e chao, e por isso fica aqui e nao junto
	// dos ossos: uma elipse preta a 55% por cima das botas de quem passa por
	// cima seria a mesma mancha que este passe existe para tirar.
	for i := range g.hands {
		g.drawHandHole(&g.hands[i], f)
	}
}

// drawHandHole renders the dark burst hole under one hand. Compartilha com
// drawHand (overlay) o mesmo portao de emergencia, para o buraco nao aparecer
// antes da mao que o abriu.
func (g *Graveyard) drawHandHole(h *graveHand, fade float32) {
	if g.handRise(h) <= 0 {
		return
	}
	base := g.point(h.u, h.v)
	rl.DrawEllipse(int32(base.X), int32(base.Y), 13*h.scale, 5.5*h.scale, rl.Fade(rl.Black, 0.55*fade))
}

// handRise e o quanto uma mao ja saiu da terra, 0..1. Fica aqui porque os dois
// passes dependem dele e nenhum dos dois e dono do numero.
func (g *Graveyard) handRise(h *graveHand) float32 {
	return clamp01((g.Age - 0.35 - h.delay) / 0.7)
}
