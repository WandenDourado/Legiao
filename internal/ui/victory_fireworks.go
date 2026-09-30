package ui

// Fogos de artificio da vitoria do chefe.
//
// Nao ha "instante da morte" para disparar UMA vez: o pacote de rede so
// publica um `BossState` por quadro (igual `WaveState`), entao a decisao de
// comecar e continua e do tipo "esta caido agora?", nao um evento. Isso serve
// bem a cena — o grupo ainda esta limpando o que restou em campo quando o
// primeiro fogo sobe, e ele deve subir de qualquer maneira.
//
// QUANDO comecar mora em victory.go (le o BossState); este arquivo so sabe
// simular e desenhar fogos, para nao misturar as duas responsabilidades.
//
// A FISICA E EM ALTURAS DE TELA, NAO EM PIXELS (27/09/2026,
// plans/plan_playtest_27_09.md §3). Os numeros abaixo sao os de uma tela de
// fireworkRefHeight (1080) px e sao multiplicados por screenHeight/1080. Antes
// eram px absolutos: o foguete subia no maximo 520²/(2·90) ≈ 1502 px, mas o
// ponto de estouro e sorteado entre 16% e 40% da altura da tela — numa TV 4K
// (2160 px) ~60% dos foguetes chegavam ao topo antes da linha de estouro,
// comecavam a cair e saiam por baixo da tela sem estourar. Foi assim que o Gui
// viu o bug, jogando numa TV. Alem da escala, o foguete agora estoura no topo
// da subida em qualquer caso (updateFireworks), entao nenhum desce inteiro.

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// fireworkRefHeight e a altura de tela em que os numeros abaixo valem
	// como estao escritos (ver o topo do arquivo).
	fireworkRefHeight float32 = 1080
	// fireworkMaxDT e o teto do passo da simulacao: um quadro longo (a troca
	// para a musica de vitoria carrega um stream) vira um quadro de camera
	// lenta, e nao um pulo que atravessa a linha de estouro.
	fireworkMaxDT float32 = 1.0 / 20

	fireworkGravity       float32 = 90
	fireworkRocketSpeed   float32 = 520
	fireworkLaunchEvery   float32 = 0.55
	fireworkParticleLife  float32 = 1.3
	fireworkParticleCount         = 56
	fireworkParticleSpeed float32 = 300
	// Raios de desenho. Maiores que o "fio" fisico da faisca de proposito: em
	// escala de tela, uma faisca do tamanho real e ilegivel a distancia — o
	// glow aditivo e o que vende "fogo de artificio" em vez de "confete".
	fireworkRocketRadius   float32 = 7
	fireworkRocketTrail    float32 = 6
	fireworkSparkGlowBase  float32 = 4
	fireworkSparkGlowScale float32 = 16
	fireworkSparkCore      float32 = 4.5
)

var fireworksPalette = []rl.Color{
	rl.NewColor(255, 210, 90, 255),  // dourado
	rl.NewColor(255, 120, 120, 255), // vermelho
	rl.NewColor(120, 200, 255, 255), // azul
	rl.NewColor(160, 255, 160, 255), // verde
	rl.NewColor(230, 160, 255, 255), // roxo
}

type fireworkRocket struct {
	pos, vel rl.Vector2
	color    rl.Color
	burstY   float32
}

type fireworkSpark struct {
	pos, vel      rl.Vector2
	life, maxLife float32
	color         rl.Color
}

// fireworks e o unico emissor da tela: a cena so acontece num lugar por vez
// (fim de campanha), entao um singleton evita plumbing so para isto.
var fireworks struct {
	rockets []fireworkRocket
	sparks  []fireworkSpark
	timer   float32
	running bool
	// scale e screenHeight/fireworkRefHeight do quadro corrente.
	scale float32
}

// fireworkScale converte a altura da tela no fator da fisica e do desenho.
func fireworkScale(screenHeight float32) float32 {
	if screenHeight <= 0 {
		return 1
	}
	return screenHeight / fireworkRefHeight
}

// DrawBossFireworks simula e desenha um quadro dos fogos. active liga os
// LANCAMENTOS; fogos ja no ar terminam de subir e estourar mesmo se active
// virar falso no meio do caminho (reinicio de fase), em vez de sumir cortado.
func DrawBossFireworks(active bool, screenWidth, screenHeight float32) {
	dt := rl.GetFrameTime()
	if dt > fireworkMaxDT {
		dt = fireworkMaxDT
	}
	fireworks.scale = fireworkScale(screenHeight)

	if active && !fireworks.running {
		fireworks.running = true
		fireworks.timer = 0 // primeiro fogo sobe no quadro em que a chefe cai
	} else if !active {
		fireworks.running = false
	}

	if fireworks.running {
		if fireworks.timer -= dt; fireworks.timer <= 0 {
			fireworks.timer = fireworkLaunchEvery * (0.7 + rand.Float32()*0.6)
			launchFirework(screenWidth, screenHeight)
		}
	}

	updateFireworks(dt, fireworks.scale)
	drawFireworks(fireworks.scale)
}

func launchFirework(screenWidth, screenHeight float32) {
	x := screenWidth*0.15 + rand.Float32()*screenWidth*0.7
	burstY := screenHeight*0.16 + rand.Float32()*screenHeight*0.24
	fireworks.rockets = append(fireworks.rockets, fireworkRocket{
		pos: rl.NewVector2(x, screenHeight+10),
		vel: rl.NewVector2((rand.Float32()*2-1)*24*fireworks.scale,
			-fireworkRocketSpeed*fireworkScale(screenHeight)),
		color:  fireworksPalette[rand.Intn(len(fireworksPalette))],
		burstY: burstY,
	})
}

// updateFireworks avanca um passo. scale e fireworkScale da tela (ver o topo
// do arquivo): a gravidade e escalada junto da velocidade de lancamento, para
// o topo da subida ser sempre a mesma FRACAO da tela.
func updateFireworks(dt, scale float32) {
	gravity := fireworkGravity * scale
	aliveRockets := fireworks.rockets[:0]
	for _, r := range fireworks.rockets {
		r.vel.Y += gravity * dt
		r.pos.X += r.vel.X * dt
		r.pos.Y += r.vel.Y * dt
		// Chegou a linha OU parou de subir: o segundo caso e a rede de
		// seguranca que garante que nenhum foguete desce sem estourar,
		// qualquer que seja a tela ou o engasgo de quadro.
		if r.pos.Y <= r.burstY || r.vel.Y >= 0 {
			burst(r.pos, r.color, scale)
			continue
		}
		aliveRockets = append(aliveRockets, r)
	}
	fireworks.rockets = aliveRockets

	aliveSparks := fireworks.sparks[:0]
	for _, s := range fireworks.sparks {
		if s.life -= dt; s.life <= 0 {
			continue
		}
		s.vel.Y += gravity * dt
		s.vel.X *= 0.98
		s.vel.Y *= 0.98
		s.pos.X += s.vel.X * dt
		s.pos.Y += s.vel.Y * dt
		aliveSparks = append(aliveSparks, s)
	}
	fireworks.sparks = aliveSparks
}

// burst estoura um foguete em fireworkParticleCount faiscas, radiais e com
// velocidade variada — igual em espirito ao `ParticleEmitter.Burst` que as
// magias usam em mundo, so que em espaco de tela.
func burst(pos rl.Vector2, c rl.Color, scale float32) {
	for i := 0; i < fireworkParticleCount; i++ {
		a := rand.Float64() * 2 * math.Pi
		s := fireworkParticleSpeed * scale * (0.35 + rand.Float32()*0.75)
		fireworks.sparks = append(fireworks.sparks, fireworkSpark{
			pos:     pos,
			vel:     rl.NewVector2(float32(math.Cos(a))*s, float32(math.Sin(a))*s),
			life:    fireworkParticleLife * (0.7 + rand.Float32()*0.5),
			maxLife: fireworkParticleLife,
			color:   c,
		})
	}
}

func drawFireworks(scale float32) {
	rl.BeginBlendMode(rl.BlendAdditive)
	for _, r := range fireworks.rockets {
		rl.DrawCircleGradient(int32(r.pos.X), int32(r.pos.Y), fireworkRocketRadius*1.8*scale, r.color, rl.Blank)
		rl.DrawCircle(int32(r.pos.X), int32(r.pos.Y), fireworkRocketRadius*scale, r.color)
		rl.DrawLineEx(r.pos, rl.NewVector2(r.pos.X-r.vel.X*0.05, r.pos.Y-r.vel.Y*0.05),
			fireworkRocketTrail*scale, rl.Fade(r.color, 0.7))
	}
	for _, s := range fireworks.sparks {
		alpha := clamp01(s.life / s.maxLife)
		c := rl.Fade(s.color, alpha)
		rl.DrawCircleGradient(int32(s.pos.X), int32(s.pos.Y),
			(fireworkSparkGlowBase+fireworkSparkGlowScale*alpha)*scale, c, rl.Blank)
		rl.DrawCircle(int32(s.pos.X), int32(s.pos.Y), (fireworkSparkCore*alpha+1)*scale, c)
	}
	rl.EndBlendMode()
}
