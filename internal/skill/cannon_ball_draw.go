package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Draw desenha o núcleo em chamas + esteira, aditivo. Mesma ideia da Bola de
// Fogo do Mago (fireball.go) — vermelho fora, laranja no meio, amarelo no
// centro — mas maior e mais lento de pulsar: isto sai de uma arma de cerco,
// não de um feitiço rápido, e o tamanho é o que avisa de longe que este
// impacto dói mais que o normal.
func (b *CannonBall) Draw() {
	pulse := 1 + 0.08*float32(math.Sin(float64(b.Time)*5.5))
	r := CannonBallRadius * pulse

	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(b.Position.X), int32(b.Position.Y), r*2.1,
		rl.Fade(rl.Red, 0.22), rl.Blank)
	rl.DrawCircle(int32(b.Position.X), int32(b.Position.Y), r, rl.Fade(rl.Red, 0.60))
	rl.DrawCircle(int32(b.Position.X), int32(b.Position.Y), r*0.72, rl.Fade(rl.Orange, 0.88))
	rl.DrawCircle(int32(b.Position.X), int32(b.Position.Y), r*0.32, rl.Fade(rl.Yellow, 1.0))
	b.Trail.Draw()
	rl.EndBlendMode()
}

// CannonBallBurst é o impacto: uma explosão de fogo que cresce e some, como o
// SentryOrbBurst da esfera da sentinela mas em paleta de fogo. Existe pelo
// mesmo motivo: tirar a bola de campo no quadro em que ela acerta sem mostrar
// nada leria como bug.
type CannonBallBurst struct {
	Position rl.Vector2
	TTL      float32
	Max      float32
	Sparks   *ParticleEmitter
}

// CannonBallBurstTTL é a janela de saída da explosão de impacto. Um pouco
// mais longa que a da sentinela (0.45): o golpe é mais forte, e o estouro
// precisa ler como maior, não só mais colorido.
const CannonBallBurstTTL float32 = 0.6

// NewCannonBallBurst cria a explosão de impacto em pos.
func NewCannonBallBurst(pos rl.Vector2) *CannonBallBurst {
	b := &CannonBallBurst{Position: pos, TTL: CannonBallBurstTTL, Max: CannonBallBurstTTL, Sparks: NewParticleEmitter()}
	b.Sparks.Burst(pos, 34, 90, 320, 0.5, CannonBallRadius*0.45, rl.Orange)
	b.Sparks.Burst(pos, 16, 60, 180, 0.55, CannonBallRadius*0.3, rl.Fade(rl.Red, 0.9))
	return b
}

// Update avança a explosão; false quando acabou.
func (b *CannonBallBurst) Update(dt float32) bool {
	b.TTL -= dt
	b.Sparks.Update(dt)
	return b.TTL > 0
}

// Draw desenha o anel de choque em chamas.
func (b *CannonBallBurst) Draw() {
	if b.Max <= 0 {
		return
	}
	progress := 1 - b.TTL/b.Max
	if progress < 0 {
		progress = 0
	}
	alpha := 1 - progress
	r := CannonBallRadius * (0.6 + progress*4.0)

	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(b.Position.X), int32(b.Position.Y), r*0.95,
		rl.Fade(rl.Red, 0.45*alpha), rl.Blank)
	rl.DrawRing(b.Position, r*0.84, r, 0, 360, 40, rl.Fade(rl.Orange, 0.85*alpha))
	rl.DrawCircle(int32(b.Position.X), int32(b.Position.Y), CannonBallRadius*0.55*alpha,
		rl.Fade(rl.Yellow, alpha))
	b.Sparks.Draw()
	rl.EndBlendMode()
}
