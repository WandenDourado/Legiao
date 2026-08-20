package skill

// A bola de fogo dos canhões do corredor final (mapa 6).
//
// Ela NÃO persegue, ao contrário da esfera da Gárgula Sentinela
// (sentry_orb.go). Um canhão é uma arma física, não uma inteligência que
// escolhe corrigir o rumo — o tiro sai reto na direção que o alvo estava no
// instante do disparo, como a Bola de Fogo do Mago (fireball.go). É essa
// linha reta, e não a esquiva por movimento da esfera perseguidora, que torna
// o projétil justo apesar do dano altíssimo: quem sai do lugar em que estava
// não é atingido.
//
// O resto da forma vem de sentry_orb.go de propósito: o canhão é um monstro
// do mapa, não um personagem, então isto não entra no `ability` registry nem
// tem cooldown de jogador — quem decide cadência e alcance é
// `internal/network/cannons.go`, e quem resolve o impacto contra o jogador é
// `internal/network/host_cannon.go`.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// CannonBallSpeed é mais lenta que a Bola de Fogo do Mago (840) e um
	// pouco mais rápida que a esfera da sentinela (300): o corredor é longo
	// o bastante (até ~3200 px de alcance) para que uma esfera lenta demais
	// levasse quase dez segundos no ar, e essa demora lê como o canhão
	// quebrado, não como ameaça.
	CannonBallSpeed float32 = 460
	// CannonBallRadius é o corpo de colisão. O corredor caminhável tem 768 px;
	// o diâmetro de 700 px de cada bola é um pouco menor que a largura do
	// corredor, permitindo alguma esquiva entre as duas bolas quando lançadas
	// pelos canhões opostos.
	CannonBallRadius float32 = 350
	// CannonBallTTL é a rede de segurança contra bola órfã quando ela erra
	// todo mundo: ao alcance máximo declarado em cannons.go (3200 px) a
	// 460 px/s ela viaja ~7 s; 9 s cobre isso com folga sem deixar uma bola
	// que errou voando para sempre.
	CannonBallTTL float32 = 9.0
)

// CannonBall é um projétil reto desenhado com primitivas raylib.
//
// CannonID identifica qual posto disparou, só para depuração — a mesma
// convenção do SentryID da esfera da sentinela.
type CannonBall struct {
	ID       string
	CannonID string
	Position rl.Vector2
	Velocity rl.Vector2
	TTL      float32
	// Time acumula desde o disparo e move o núcleo e a esteira.
	Time  float32
	Trail *ParticleEmitter
}

// NewCannonBall cria a bola saindo de start em direção a dir (não precisa
// estar normalizado).
func NewCannonBall(id, cannonID string, start, dir rl.Vector2) *CannonBall {
	d := rl.Vector2Normalize(dir)
	if d.X == 0 && d.Y == 0 {
		d = rl.NewVector2(0, 1)
	}
	return &CannonBall{
		ID:       id,
		CannonID: cannonID,
		Position: start,
		Velocity: rl.Vector2Scale(d, CannonBallSpeed),
		TTL:      CannonBallTTL,
		Trail:    NewParticleEmitter(),
	}
}

// Update avança a bola em linha reta. Retorna false quando ela deve sumir
// (TTL esgotado) — a colisão com o jogador é resolvida fora daqui, em
// internal/network/host_cannon.go, pelo mesmo motivo que a esfera da
// sentinela: `skill` não mexe em vida de jogador.
func (b *CannonBall) Update(dt float32) bool {
	b.TTL -= dt
	b.Position = rl.Vector2Add(b.Position, rl.Vector2Scale(b.Velocity, dt))
	b.Time += dt
	b.emitTrail()
	b.Trail.Update(dt)
	return b.TTL > 0
}

// AdvanceVisual é a mesma coisa no cliente, sem decidir remoção — quem tira a
// bola de campo é o evento de impacto ou expiração do host.
func (b *CannonBall) AdvanceVisual(dt float32) {
	b.TTL -= dt
	b.Position = rl.Vector2Add(b.Position, rl.Vector2Scale(b.Velocity, dt))
	b.Time += dt
	b.emitTrail()
	b.Trail.Update(dt)
}

// Expired deixa o cliente podar bolas órfãs sem esperar o host.
func (b *CannonBall) Expired() bool { return b.TTL <= 0 }

// emitTrail deixa uma esteira de fogo reta atrás da bola.
func (b *CannonBall) emitTrail() {
	const n = 5
	for i := 0; i < n; i++ {
		t := float32(i) / float32(n-1) // 0 = cabeça, 1 = cauda
		off := rl.Vector2Scale(b.Velocity, -0.05*float32(i+1))
		pos := rl.Vector2Add(b.Position, off)
		radius := CannonBallRadius * (0.95 - 0.55*t)
		b.Trail.Emit(pos, rl.NewVector2(0, 0), 0.42, radius, rl.Orange)
	}
	b.Trail.Burst(b.Position, 2, 12, 45, 0.38, 16, rl.Yellow)
}
