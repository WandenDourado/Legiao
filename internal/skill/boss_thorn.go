package skill

// O espinhão da Senhora das Trevas.
//
// A habilidade e um contrato com o jogador: "eu marco o chao onde voce esta, e
// voce tem dois segundos". Por isso o espinho nasce com a marca e NAO segue
// ninguem — se ele perseguisse, o desvio deixaria de ser uma decisao e viraria
// uma corrida que o jogador nao pode vencer.
//
// Tudo aqui e desenhado por primitivas do raylib, como manda a regra de efeitos
// do `doc/art_style.md`: a arte da chefe nao tem um pixel de espinho.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// ThornTelegraph e a janela de desvio, e ela NAO e um numero solto: sao os
	// 1,8 s do laco `attack_windup` (bracos cruzados tremendo) mais os 0,30 s
	// ate o passo do agachamento no `attack_strike` (indice 2 do PlayOrder,
	// terceiro passo a 0,10 s). Mexer aqui sem mexer no `bossDodgeWindow` ou no
	// ritmo da folha desalinha o golpe da animacao.
	ThornTelegraph float32 = 2.1
	// ThornRise e o tempo que as lascas levam para sair do chao.
	ThornRise float32 = 0.16
	// ThornLinger e quanto elas ficam de pe depois de cravar.
	ThornLinger float32 = 0.9
	// ThornRadius e o raio de acerto, e tambem o raio EXATO que a marca desenha:
	// o circulo vermelho no chao e a area de dano, sem margem escondida.
	ThornRadius float32 = 110
	// ThornDamage e o dano, aplicado UMA vez.
	ThornDamage float32 = 34

	// thornShards e quantas lascas compoem o espinho. Uma estaca sozinha lia
	// como um triangulo; um tufo le como algo que rasgou o chao.
	thornShards = 7
)

var (
	thornWarnDeep = rl.NewColor(96, 12, 24, 255)  // poca escura no chao
	thornWarnHot  = rl.NewColor(236, 74, 74, 255) // anel e rachaduras
	thornBody     = rl.NewColor(26, 22, 42, 255)  // quitina, a mesma familia dela
	thornBodyLit  = rl.NewColor(64, 56, 96, 255)
	thornEdge     = rl.NewColor(178, 150, 74, 255) // ouro envelhecido na quina
)

// Thorn e um espinho que vai irromper num ponto fixo do chao.
type Thorn struct {
	ID     string
	Center rl.Vector2
	Age    float32
	// struck marca que o dano ja saiu. As lascas ficam de pe mais um tempo
	// depois disso: quem entrou tarde na area nao leva, e e assim que tem de
	// ser — o golpe ja aconteceu.
	struck bool
}

// NewThorn marca o chao num ponto. O dano so sai ThornTelegraph segundos depois.
func NewThorn(center rl.Vector2) *Thorn {
	return &Thorn{ID: generateID(), Center: center}
}

// Update envelhece o espinho. Devolve false quando ele acaba.
func (t *Thorn) Update(dt float32) bool {
	t.Age += dt
	return t.Age <= ThornTelegraph+ThornRise+ThornLinger
}

// Erupting reporta se o espinho acabou de irromper e ainda nao causou dano.
// Chamar uma vez por quadro no host; ele se marca como resolvido.
func (t *Thorn) Erupting() bool {
	if t.struck || t.Age < ThornTelegraph {
		return false
	}
	t.struck = true
	return true
}

// Contains reporta se um ponto esta no alcance do espinho.
func (t *Thorn) Contains(p rl.Vector2) bool {
	return rl.Vector2Distance(t.Center, p) <= ThornRadius
}

// Draw desenha o aviso e depois as lascas.
func (t *Thorn) Draw() {
	if t.Age < ThornTelegraph {
		t.drawWarning()
		return
	}
	t.drawSpikes()
}

// drawWarning desenha a marca no chao.
//
// Sao QUATRO informacoes empilhadas, e cada uma responde uma pergunta que as
// outras nao respondem:
//
//   - a poca escura diz ONDE (e o raio exato do dano, sem margem escondida);
//   - o anel que FECHA diz QUANTO FALTA — e o relogio, e e a unica coisa aqui
//     que da ao jogador o instante de sair;
//   - as rachaduras que CRESCEM do centro dizem que algo vem DE BAIXO, e nao
//     que ali e so um chao pintado;
//   - as pontas espiando no fim dizem O QUE vem, meio segundo antes de vir.
//
// A primeira versao tinha so um pulso e um anel. Pulso sozinho diz "perigo" e
// nao diz quando; e a marca ficava identica do primeiro ao ultimo instante, o
// que ensina o jogador a ignora-la.
func (t *Thorn) drawWarning() {
	p := t.Age / ThornTelegraph // 0 -> 1
	cx, cy := int32(t.Center.X), int32(t.Center.Y)

	// A urgencia SOBE: quase invisivel no comeco, gritante no fim. Uma marca de
	// intensidade constante e ruido de fundo depois do terceiro espinhao.
	urg := p * p

	rl.DrawCircle(cx, cy, ThornRadius, rl.NewColor(
		thornWarnDeep.R, thornWarnDeep.G, thornWarnDeep.B, uint8(40+90*urg)))
	rl.DrawCircleLines(cx, cy, ThornRadius, rl.Fade(thornWarnHot, 0.5+0.5*urg))
	rl.DrawCircleLines(cx, cy, ThornRadius-2, rl.Fade(thornWarnHot, 0.35+0.4*urg))

	// O relogio: o anel interno fecha ate o centro.
	rl.DrawCircleLines(cx, cy, ThornRadius*(1-p), rl.Fade(thornWarnHot, 0.95))
	rl.DrawCircleLines(cx, cy, ThornRadius*(1-p)-1, rl.Fade(thornWarnHot, 0.55))

	// Rachaduras: crescem do centro para a borda ao longo da espera.
	for i := 0; i < thornShards; i++ {
		ang := float64(i)*(2*math.Pi/thornShards) + float64(t.Age)*0.35
		l := ThornRadius * (0.25 + 0.72*p)
		ex := t.Center.X + float32(math.Cos(ang))*l
		ey := t.Center.Y + float32(math.Sin(ang))*l
		rl.DrawLineEx(t.Center, rl.NewVector2(ex, ey), 2+2*urg,
			rl.Fade(thornWarnHot, 0.25+0.55*urg))
	}

	// Ultimos 25%: as pontas comecam a espiar. E o aviso final, e ele e da
	// forma do que vem — nao de uma cor.
	if p > 0.75 {
		peek := (p - 0.75) / 0.25
		for i := 0; i < thornShards; i++ {
			b, h, w := t.shardGeom(i, peek*0.28)
			rl.DrawTriangle(
				rl.NewVector2(b.X-w, b.Y), rl.NewVector2(b.X+w, b.Y),
				rl.NewVector2(b.X, b.Y-h), rl.Fade(thornBody, 0.9))
		}
	}
}

// drawSpikes desenha o tufo de lascas cravado.
func (t *Thorn) drawSpikes() {
	grow := clamp01((t.Age - ThornTelegraph) / ThornRise)
	life := clamp01((ThornTelegraph + ThornRise + ThornLinger - t.Age) / 0.35)

	// A cratera fica: o chao rachou e nao desrracha enquanto as lascas estao la.
	rl.DrawCircle(int32(t.Center.X), int32(t.Center.Y), ThornRadius*0.92,
		rl.Fade(thornWarnDeep, 0.55*life))

	// Onda de impacto, so no instante em que sai.
	if grow < 1 {
		rl.DrawCircleLines(int32(t.Center.X), int32(t.Center.Y),
			ThornRadius*(0.6+1.1*grow), rl.Fade(thornWarnHot, (1-grow)*0.9))
	}

	// As lascas de tras primeiro, as da frente por cima: e o que da profundidade
	// ao tufo em vez de sete triangulos empilhados na mesma altura.
	for i := 0; i < thornShards; i++ {
		b, h, w := t.shardGeom(i, grow)
		tip := rl.NewVector2(b.X, b.Y-h)
		rl.DrawTriangle(
			rl.NewVector2(b.X-w, b.Y), rl.NewVector2(b.X+w, b.Y), tip,
			rl.Fade(thornBody, life))
		// Quina iluminada num lado so — a luz do jogo vem de cima a esquerda.
		rl.DrawTriangle(
			rl.NewVector2(b.X-w, b.Y), rl.NewVector2(b.X-w*0.25, b.Y), tip,
			rl.Fade(thornBodyLit, 0.75*life))
		rl.DrawLineEx(rl.NewVector2(b.X-w, b.Y), tip, 1.6, rl.Fade(thornEdge, 0.85*life))
	}
}

// shardGeom devolve a base, a altura e a meia-largura da lasca `i`.
//
// As lascas sao deterministicas a partir do indice e nao sorteadas: um espinho
// que muda de forma entre quadros pisca. Alturas diferentes por indice sao o
// que faz o tufo ler como pedra quebrada em vez de uma cerca.
func (t *Thorn) shardGeom(i int, grow float32) (base rl.Vector2, h, w float32) {
	ang := float64(i) * (2 * math.Pi / thornShards)
	// Raio menor que o de dano: as lascas ficam DENTRO do circulo que o jogador
	// viu marcado, senao a arte promete uma area maior do que a que machuca.
	r := ThornRadius * 0.52
	base = rl.NewVector2(
		t.Center.X+float32(math.Cos(ang))*r,
		t.Center.Y+float32(math.Sin(ang))*r*0.55, // achatado: o chao e visto de cima
	)
	alt := []float32{1.0, 0.62, 0.85, 0.48, 0.94, 0.55, 0.74}[i%7]
	h = ThornRadius * 2.1 * alt * grow
	w = ThornRadius * 0.17 * (0.7 + 0.5*alt)
	return base, h, w
}
