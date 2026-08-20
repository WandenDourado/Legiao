package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A esfera sombria da Gargula Sentinela do mapa 4.
//
// Ela e a primeira magia do jogo que pertence a um MONSTRO e nao a um
// personagem, entao nao entra no `ability` registry (que existe para mapear
// personagem -> magia) nem tem cooldown de jogador. O que ela herda inteiro do
// resto do arquivo e a regra dura: visual 100% procedural, nenhuma imagem.
//
// A esfera trava o alvo NO LANCAMENTO e persegue esse jogador mesmo que ele
// corra. Travar, em vez de re-mirar a cada quadro, e decisao de leitura: uma
// esfera que troca de dono no meio do voo e indesviavel, e o time nao consegue
// reagir a quem foi marcado. Perseguir mas nao re-mirar da os dois: a ameaca e
// clara e a fuga e possivel.
const (
	// SentryOrbSpeed e mais lenta que a bola de fogo do Mago (840) de
	// proposito. A esfera nao e para ser desviada por reflexo - e para ser
	// desviada por MOVIMENTO, e para isso ela precisa levar um tempo visivel
	// atravessando o saguao.
	SentryOrbSpeed float32 = 300

	// SentryOrbTurnRate limita quanto a esfera corrige o rumo por segundo, em
	// graus. Perseguicao perfeita e imbloqueavel: com 150 graus/s a esfera faz
	// a curva, mas quem corre em diagonal e muda de direcao consegue abrir
	// distancia. E este numero, e nao a velocidade, que decide se a magia e
	// justa.
	SentryOrbTurnRate float32 = 150

	// SentryOrbRadius e o corpo de colisao. O visual desenha bem maior que
	// isso: a corona nao machuca, so o nucleo.
	SentryOrbRadius float32 = 26

	// SentryOrbTTL e o padrao usado quando o chamador nao calcula um TTL
	// proprio (ttl <= 0 em NewSentryOrb) — evita esfera orfa quando o alvo
	// morre ou sai do mapa. Com alcance global (doc/tilemap.md,
	// SentryGlobalRange) o chamador real (host_sentry_orb.go) sempre calcula
	// o TTL pela distancia: uma esfera lancada do outro lado do mapa nao
	// caberia nestes 9s.
	SentryOrbTTL float32 = 9.0

	// SentryOrbMaxTTL e o teto do TTL calculado pela distancia — um disparo
	// absurdamente longe nao pode viver para sempre (doc/plan_avanco_bots_e_gargula.md
	// §B2).
	SentryOrbMaxTTL float32 = 40.0
)

// SentryOrb e um projetil perseguidor desenhado com primitivas raylib.
//
// TargetID e o jogador travado no lancamento; o host reescreve TargetPos a
// cada tick com a posicao real dele, e o cliente faz o mesmo com a posicao
// replicada. Nenhum dos dois decide dano: isso e do host, em host_sentry_orb.go.
type SentryOrb struct {
	ID       string
	SentryID string // qual sentinela lancou, para depuracao
	TargetID string
	Position rl.Vector2
	Velocity rl.Vector2
	TTL      float32
	// Time acumula desde o lancamento e move os aneis e o pulso. Sem ele a
	// esfera seria uma bola parada que translada - o olho le isso como sprite.
	Time  float32
	Trail *ParticleEmitter
}

// NewSentryOrb cria a esfera saindo de start em direcao ao alvo.
//
// ttl <= 0 cai para SentryOrbTTL. O chamador real (host_sentry_orb.go)
// sempre passa um TTL calculado pela distancia do disparo — alcance global
// (SentryGlobalRange) significa que a esfera pode nascer a milhares de
// pixels do alvo, e os 9s fixos de antes nunca chegariam lá
// (doc/plan_avanco_bots_e_gargula.md §B2).
func NewSentryOrb(id, sentryID, targetID string, start, target rl.Vector2, ttl float32) *SentryOrb {
	dir := rl.Vector2Subtract(target, start)
	if dir.X == 0 && dir.Y == 0 {
		dir = rl.NewVector2(0, 1)
	}
	if ttl <= 0 {
		ttl = SentryOrbTTL
	}
	return &SentryOrb{
		ID:       id,
		SentryID: sentryID,
		TargetID: targetID,
		Position: start,
		Velocity: rl.Vector2Scale(rl.Vector2Normalize(dir), SentryOrbSpeed),
		TTL:      ttl,
		Trail:    NewParticleEmitter(),
	}
}

// Update persegue target e avanca a esfera. Retorna false quando ela deve
// sumir (TTL). O host chama; a colisao com o jogador e resolvida fora daqui,
// porque `skill` nao mexe em vida de jogador.
//
// hasTarget falso quer dizer "o alvo morreu ou sumiu": a esfera segue reto, e
// nao para no ar nem persegue um fantasma.
func (o *SentryOrb) Update(dt float32, target rl.Vector2, hasTarget bool) bool {
	o.TTL -= dt
	o.steer(dt, target, hasTarget)
	o.Position = rl.Vector2Add(o.Position, rl.Vector2Scale(o.Velocity, dt))
	o.Time += dt
	o.emitTrail()
	o.Trail.Update(dt)
	return o.TTL > 0
}

// AdvanceVisual e a mesma coisa no cliente, sem decidir remocao - quem tira a
// esfera de campo e o evento de impacto do host.
func (o *SentryOrb) AdvanceVisual(dt float32, target rl.Vector2, hasTarget bool) {
	o.TTL -= dt
	o.steer(dt, target, hasTarget)
	o.Position = rl.Vector2Add(o.Position, rl.Vector2Scale(o.Velocity, dt))
	o.Time += dt
	o.emitTrail()
	o.Trail.Update(dt)
}

// Expired deixa o cliente podar esferas orfas sem esperar o host.
func (o *SentryOrb) Expired() bool { return o.TTL <= 0 }

// steer curva a velocidade em direcao ao alvo, no maximo SentryOrbTurnRate
// graus por segundo. A velocidade ESCALAR nunca muda: a esfera nao acelera
// para alcancar, ela so vira - que e o que torna a fuga possivel.
func (o *SentryOrb) steer(dt float32, target rl.Vector2, hasTarget bool) {
	if !hasTarget {
		return
	}
	want := rl.Vector2Subtract(target, o.Position)
	if want.X == 0 && want.Y == 0 {
		return
	}
	cur := math.Atan2(float64(o.Velocity.Y), float64(o.Velocity.X))
	des := math.Atan2(float64(want.Y), float64(want.X))
	// Diferenca menor no circulo, para a esfera virar pelo lado curto.
	diff := math.Mod(des-cur+3*math.Pi, 2*math.Pi) - math.Pi
	maxTurn := float64(SentryOrbTurnRate) * math.Pi / 180 * float64(dt)
	if diff > maxTurn {
		diff = maxTurn
	} else if diff < -maxTurn {
		diff = -maxTurn
	}
	a := cur + diff
	o.Velocity = rl.NewVector2(
		float32(math.Cos(a))*SentryOrbSpeed,
		float32(math.Sin(a))*SentryOrbSpeed,
	)
}

// emitTrail deixa uma esteira que CURVA junto com a trajetoria - e ela, e nao
// a esfera, que conta ao jogador que a coisa esta perseguindo. Uma esteira
// reta faria a curva parecer teleporte.
func (o *SentryOrb) emitTrail() {
	const n = 4
	for i := 0; i < n; i++ {
		t := float32(i) / float32(n-1) // 0 = cabeca, 1 = cauda
		off := rl.Vector2Scale(o.Velocity, -0.05*float32(i+1))
		pos := rl.Vector2Add(o.Position, off)
		o.Trail.Emit(pos, rl.NewVector2(0, 0), 0.38, SentryOrbRadius*(0.9-0.6*t), sentryOrbMid)
	}
	o.Trail.Burst(o.Position, 1, 8, 28, 0.32, SentryOrbRadius*0.35, sentryOrbDeep)
}
