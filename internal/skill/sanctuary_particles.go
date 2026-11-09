package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// sacredParticle is a single floating mote inside the sanctuary. It drifts
// upward slowly with an individual fade in/out, giving the "sacred" feel.
//
// O offset e RELATIVO ao centro, e e por isso que a aura movel nao precisou de
// nada aqui: as motes acompanham a Sacerdotisa de graca, porque nunca souberam
// onde ficavam no mundo.
type sacredParticle struct {
	offset rl.Vector2 // position relative to sanctuary center
	phase  float32    // per-particle phase for fade in/out
	speed  float32    // upward drift speed
	radius float32
	seed   float32
}

// newSacredParticle spawns one mote at a random offset inside the radius.
// The angle is turned into a full 360° unit vector (cos/sin) so motes are
// spread evenly across BOTH the left and right halves of the area.
func newSacredParticle() sacredParticle {
	ang := float32(rl.GetRandomValue(0, 359)) * math.Pi / 180
	dist := float32(rl.GetRandomValue(0, 100)) / 100 * SanctuaryRadius * 0.85
	dir := rl.NewVector2(float32(math.Cos(float64(ang))), float32(math.Sin(float64(ang))))
	return sacredParticle{
		offset: rl.Vector2Scale(dir, dist),
		phase:  float32(rl.GetRandomValue(0, 100)) / 100 * math.Pi * 2,
		speed:  18 + float32(rl.GetRandomValue(0, 26)),
		radius: 2 + float32(rl.GetRandomValue(0, 3)),
		seed:   float32(rl.GetRandomValue(0, 1000)) / 1000,
	}
}

// update advances the mote (upward drift + respawn when it leaves the area).
func (p *sacredParticle) update(dt float32) {
	p.phase += dt * 1.5
	p.offset.Y -= p.speed * dt
	if p.offset.Y < -SanctuaryRadius*0.8 {
		*p = newSacredParticle()
		p.offset.Y = SanctuaryRadius * 0.8
	}
}

// alpha returns the individual fade in/out factor (0..1..0 over its cycle).
func (p *sacredParticle) alpha() float32 {
	a := (float32(math.Sin(float64(p.phase))) + 1) / 2
	if a < 0 {
		return 0
	}
	if a > 1 {
		return 1
	}
	return a
}

// drawParticles renders the rising motes: um nucleo branco com brilho, uma
// cruz de faisca curta, e uma brasa dourada por baixo em parte delas.
//
// Ja esta dentro do blend aditivo aberto por SanctuaryFX.draw. As motes se
// cullam UMA A UMA (Regra 2): o centro da aura pode estar fora da tela com
// motes subindo dentro dela.
//
// O brilho de cada uma leva o waveBoost: quando uma pulsacao passa por cima
// dela, a mote acende. E o que faz a onda parecer atravessar o santuario em
// vez de passar por cima dele.
func (fx *SanctuaryFX) drawParticles(center rl.Vector2, a float32) {
	for i := range fx.particles {
		p := &fx.particles[i]
		pos := rl.Vector2Add(center, p.offset)
		dist := rl.Vector2Length(p.offset)
		pa := (p.alpha()*0.75 + fx.waveBoost(dist)) * a
		if pa > a {
			pa = a
		}
		if pa <= 0 || !visible(pos, p.radius*4) {
			continue
		}
		// brasa dourada por baixo em um terco delas: da temperatura sem
		// tirar o branco do lugar de cor mais quente.
		if int(p.seed*10)%3 == 0 {
			rl.DrawCircleGradient(int32(pos.X), int32(pos.Y), p.radius*3.4,
				rl.Fade(sanctGold, pa*0.5), rl.Blank)
		}
		rl.DrawCircleGradient(int32(pos.X), int32(pos.Y), p.radius*2.2,
			rl.Fade(sanctYellow, pa*0.55), rl.Blank)
		rl.DrawCircleV(pos, p.radius*0.8, rl.Fade(sanctWhite, pa*0.9))
		drawSparkle(pos, p.radius*2.6, pa)
	}
}

// drawSparkle draws a small 4-point star (plus) centered at pos, used for the
// white sacred sparkles. l is the half-length of each arm.
func drawSparkle(pos rl.Vector2, l float32, a float32) {
	if a <= 0 {
		return
	}
	rl.DrawLineV(
		rl.NewVector2(pos.X-l, pos.Y),
		rl.NewVector2(pos.X+l, pos.Y),
		rl.Fade(sanctWhite, a*0.65),
	)
	rl.DrawLineV(
		rl.NewVector2(pos.X, pos.Y-l),
		rl.NewVector2(pos.X, pos.Y+l),
		rl.Fade(sanctWhite, a*0.65),
	)
}
