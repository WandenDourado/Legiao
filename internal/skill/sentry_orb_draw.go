package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A paleta da sentinela, em tres tons, e a regra do arquivo vale aqui: o tom
// MAIS QUENTE ocupa a MENOR area. Invertido, o efeito le como uma bola branca
// com borda colorida.
//
// Os tons saem do `Color: "#6B287A"` ja registrado na EnemyDef da gargula, de
// proposito: a magia e a criatura tem que ser obviamente a mesma coisa, e o
// violeta e o que separa a sentinela da pedra cinza do castelo.
var (
	// sentryOrbDeep e o violeta escuro da corona externa.
	sentryOrbDeep = rl.NewColor(0x6B, 0x28, 0x7A, 0xFF)
	// sentryOrbMid e o violeta saturado do corpo.
	sentryOrbMid = rl.NewColor(0xA4, 0x4B, 0xD4, 0xFF)
	// sentryOrbHot e o nucleo quase branco, e so ele. Ocupa 0.3 do raio.
	sentryOrbHot = rl.NewColor(0xEB, 0xD6, 0xFF, 0xFF)
)

// Draw desenha a esfera: corona aditiva de fora para dentro, dois aneis
// contra-rotativos e a esteira.
//
// Os dois aneis giram em sentidos opostos e em velocidades diferentes (90 e
// -58 graus/s). E o truque mais barato do arquivo para uma coisa redonda nao
// parecer um circulo desenhado: sem eles a esfera nao teria como mostrar que
// esta girando, porque um circulo e igual a si mesmo em qualquer angulo.
func (o *SentryOrb) Draw() {
	x, y := o.Position.X, o.Position.Y
	// Respiro sutil. Acima de ~8% de amplitude isto vira piscada.
	pulse := 1 + 0.07*float32(math.Sin(float64(o.Time)*4.1))
	r := SentryOrbRadius * pulse

	rl.BeginBlendMode(rl.BlendAdditive)

	// 1. Corona: grande e fraca, com gradiente para o vazio. E ela que faz a
	//    esfera parecer luz em vez de tinta.
	rl.DrawCircleGradient(int32(x), int32(y), r*3.0, rl.Fade(sentryOrbDeep, 0.30), rl.Blank)
	rl.DrawCircleGradient(int32(x), int32(y), r*1.9, rl.Fade(sentryOrbDeep, 0.45), rl.Blank)

	// 2. Corpo e nucleo. O branco e o menor de todos.
	rl.DrawCircle(int32(x), int32(y), r*1.15, rl.Fade(sentryOrbMid, 0.55))
	rl.DrawCircle(int32(x), int32(y), r*0.62, rl.Fade(sentryOrbMid, 0.85))
	rl.DrawCircle(int32(x), int32(y), r*0.30, rl.Fade(sentryOrbHot, 1.0))

	// 3. Estrutura: dois arcos contra-rotativos. Sem borda, o brilho puro le
	//    como neblina.
	spin := o.Time * 90
	back := -o.Time * 58
	for i := 0; i < 3; i++ {
		a := spin + float32(i)*120
		rl.DrawRing(o.Position, r*1.35, r*1.52, a, a+62, 16, rl.Fade(sentryOrbMid, 0.70))
	}
	for i := 0; i < 2; i++ {
		a := back + float32(i)*180
		rl.DrawRing(o.Position, r*1.75, r*1.86, a, a+48, 14, rl.Fade(sentryOrbDeep, 0.55))
	}

	o.Trail.Draw()
	rl.EndBlendMode()
}

// SentryOrbBurst e o impacto: um anel que cresce e desaparece.
//
// Existe porque tirar a esfera de campo no mesmo quadro em que ela acerta le
// como bug mesmo quando nao e. O jogador tem que VER que levou o golpe.
type SentryOrbBurst struct {
	Position rl.Vector2
	TTL      float32
	Max      float32
	Sparks   *ParticleEmitter
}

// SentryOrbBurstTTL e a janela de saida do impacto.
const SentryOrbBurstTTL float32 = 0.45

// NewSentryOrbBurst cria o estouro de impacto em pos.
func NewSentryOrbBurst(pos rl.Vector2) *SentryOrbBurst {
	b := &SentryOrbBurst{Position: pos, TTL: SentryOrbBurstTTL, Max: SentryOrbBurstTTL, Sparks: NewParticleEmitter()}
	b.Sparks.Burst(pos, 26, 60, 240, 0.42, SentryOrbRadius*0.4, sentryOrbMid)
	return b
}

// Update avanca o estouro; false quando acabou.
func (b *SentryOrbBurst) Update(dt float32) bool {
	b.TTL -= dt
	b.Sparks.Update(dt)
	return b.TTL > 0
}

// Draw desenha o anel de choque: cresce rapido e some, que e o que o olho le
// como onda de impacto.
func (b *SentryOrbBurst) Draw() {
	if b.Max <= 0 {
		return
	}
	progress := 1 - b.TTL/b.Max
	if progress < 0 {
		progress = 0
	}
	alpha := 1 - progress
	r := SentryOrbRadius * (0.5 + progress*3.2)

	rl.BeginBlendMode(rl.BlendAdditive)
	rl.DrawCircleGradient(int32(b.Position.X), int32(b.Position.Y), r*0.9,
		rl.Fade(sentryOrbDeep, 0.5*alpha), rl.Blank)
	rl.DrawRing(b.Position, r*0.86, r, 0, 360, 40, rl.Fade(sentryOrbMid, 0.85*alpha))
	rl.DrawCircle(int32(b.Position.X), int32(b.Position.Y), SentryOrbRadius*0.5*alpha,
		rl.Fade(sentryOrbHot, alpha))
	b.Sparks.Draw()
	rl.EndBlendMode()
}
