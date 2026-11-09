package skill

// O SELO DO MAGO: o pentagrama que arde no chao enquanto a Chuva de Meteoros
// cai.
//
// A chuva sorteia alvos no mapa INTEIRO (meteor_manager.go), entao ela e a
// unica suprema do jogo que nao aponta para quem a lancou: as pedras nascem a
// 780 unidades do proprio alvo e caem a varias telas de distancia do
// conjurador. Quem olha a tela ve o ceu desabar e nao ve DE ONDE. Este selo e
// a resposta a isso, e por isso ele nao tem regra de combate nenhuma — nao
// causa dano, nao muda alcance, nao entra em nenhum teste de colisao. Ele e
// leitura: enquanto o selo queima aos pes de alguem, aquele alguem e a origem
// da tempestade.
//
// POR QUE O SELO E UM CIRCULO E NAO UMA ELIPSE. A tentacao e achatar o Y para
// "deitar" a figura no chao. Duas regras do projeto dizem que nao:
// `doc/art_style.md` §12.2 (decalque de chao e a unica peca vista a 0 grau —
// "um circulo le como circulo e nao como elipse", justamente porque uma coisa
// sem altura nao ganha perspectiva), e o que toda marca de chao do jogo ja
// faz: o anel do meteoro, a poca do Avatar, os aneis do altar e do Santuario,
// o chevron de mira. Achatar so este selo o tornaria a unica figura de chao
// em perspectiva diferente da vizinhanca — e, em marca girante, achatamento
// tambem distorce o angulo desenhado (o erro medido no indicador de mira em
// 02/09/2026).
//
// A COR E BRASA, NAO SANGUE. `art_style.md` §12.1 fecha a questao para
// sigilos: "sigilo e giz, cinza e cera — nunca pintado em vermelho", porque
// vermelho em pentagrama le como sangue e sangue mexe na classificacao etaria
// de um app ja publicado. Aqui o selo e FOGO (a mesma familia dos meteoros,
// que sao laranja e amarelo), entao o tom mais fundo da paleta e um laranja
// queimado e nao um vermelho de sangue.

import (
	"math"
	"math/rand"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// MeteorSigilRadius e o raio do anel externo do selo.
	//
	// 170 -> 128 (10/09/2026, a pedido do Gui depois de ver em jogo). A figura
	// do mago tem ~200 px de altura (com ~103 abaixo do centro ate a sola, ver
	// entity.GroundOffset): a 170 o selo tinha 340 px de diametro, quase o
	// dobro da altura dele, e lia como uma arena desenhada no chao em vez de
	// uma marca sob os pes. A 128 sao 256 px — ainda maior que o corpo, com
	// folga suficiente para o mago andar dentro do selo sem encostar no anel.
	//
	// Continua ABAIXO do raio de impacto de um meteoro (160) de proposito: um
	// circulo maior que aquele prometeria uma area de efeito que nao existe.
	MeteorSigilRadius float32 = 128
	// meteorSigilGrow e a abertura do selo (ele se risca no chao) e
	// meteorSigilFade o apagar. Somados a MeteorRainDuration, dao a vida
	// inteira da figura.
	meteorSigilGrow float32 = 0.5
	meteorSigilFade float32 = 1.2
	// meteorSigilSpin e a rotacao do pentagrama em graus por segundo. Devagar
	// de proposito: o selo tem de parecer PESADO, nao uma roleta.
	meteorSigilSpin float32 = 11
	// meteorSigilEmberEvery e o intervalo entre lufadas de brasa (duas por
	// lufada). Com vida de ~0,7 s isto mantem ~30 particulas vivas — duas
	// ordens de grandeza abaixo das ~2.500 da propria chuva
	// (doc/skill_performance.md), que e o que permite este efeito nao entrar
	// na conta de desempenho da suprema.
	meteorSigilEmberEvery float32 = 0.045
)

// NAO HA ONDA SAINDO DO SELO. A primeira versao pulsava um anel para fora a
// cada 1,6 s, como o Avatar da Paladina faz; em jogo aquilo lia como uma area
// de efeito se abrindo — e o selo nao faz nada com quem esta em volta, ele so
// diz de quem e a tempestade. Retirado a pedido do Gui em 10/09/2026. O que
// mantem a figura viva sem prometer alcance e a tremulacao (`flick`), a
// rotacao e as brasas.

// Paleta do selo: brasa, nucleo claro e o laranja queimado do traco fundo.
var (
	sigilEmber = rl.NewColor(255, 120, 30, 255)
	sigilCore  = rl.NewColor(255, 244, 200, 255)
	sigilDeep  = rl.NewColor(205, 70, 18, 255)
)

// MeteorSigil e o pentagrama flamejante da Chuva de Meteoros: um selo riscado
// no chao aos pes do conjurador, que ARDE e gira pelos 15 segundos da chuva e
// depois se apaga. Position e a ancora (o centro do corpo — o pe e calculado
// no desenho), sincronizada com o dono a cada quadro como a do Avatar.
type MeteorSigil struct {
	OwnerID   string
	Position  rl.Vector2
	Age       float32
	Remaining float32
	emberAcc  float32
	Embers    *ParticleEmitter
}

// NewMeteorSigil acende um selo com a duracao inteira da chuva.
func NewMeteorSigil(ownerID string, pos rl.Vector2) *MeteorSigil {
	return &MeteorSigil{
		OwnerID:   ownerID,
		Position:  pos,
		Remaining: MeteorRainDuration,
		Embers:    NewParticleEmitter(),
	}
}

// Active reporta se o selo ainda esta na janela da chuva (o apagamento final
// ja e so desenho, como no Santuario).
func (s *MeteorSigil) Active() bool { return s.Remaining > 0 }

// Finished reporta que nem a figura nem a ultima brasa restaram.
func (s *MeteorSigil) Finished() bool {
	return s.Remaining < -meteorSigilFade && len(s.Embers.particles) == 0
}

// fade vale 1 enquanto a chuva cai e desce a 0 na janela de apagamento.
func (s *MeteorSigil) fade() float32 {
	if s.Remaining > 0 {
		return 1
	}
	return clamp01(1 + s.Remaining/meteorSigilFade)
}

// center e o ponto onde o selo toca o chao: a sola do conjurador.
//
// Vem de entity.GroundOffset e nao de um numero solto porque e a MESMA regra
// que decide onde o corpo colide com o cenario (entity/character_ground.go).
// Um selo desenhado no umbigo e um selo que anda "atras" dos pes toda vez que
// o mago sobe ou desce a tela.
func (s *MeteorSigil) center() rl.Vector2 {
	return entity.GroundPoint(s.Position, entity.GetCharacter(entity.CharMago))
}

// Update envelhece o selo, sopra as brasas e abre as ondas.
func (s *MeteorSigil) Update(dt float32) {
	s.Age += dt
	s.Remaining -= dt

	if s.Remaining > 0 {
		c := s.center()
		s.emberAcc += dt
		for s.emberAcc >= meteorSigilEmberEvery {
			s.emberAcc -= meteorSigilEmberEvery
			// Duas por lufada: uma no anel externo, uma no corpo do selo. O
			// anel e o que o olho segue; o miolo evita um vazio no meio.
			s.emitEmber(c, MeteorSigilRadius*(0.94+rand.Float32()*0.1))
			s.emitEmber(c, MeteorSigilRadius*(0.2+rand.Float32()*0.55))
		}
	}
	s.Embers.Update(dt)
}

// emitEmber solta uma brasa subindo de um ponto do selo, no raio pedido.
func (s *MeteorSigil) emitEmber(c rl.Vector2, r float32) {
	ang := rand.Float32() * 2 * math.Pi
	pos := sigilPoint(c, r, ang)
	// A brasa SOBE — o fogo tem altura, o selo nao — e se abre um pouco para
	// fora, para a figura parecer soprar em vez de vazar.
	vel := rl.NewVector2(
		float32(math.Cos(float64(ang)))*26,
		-70-rand.Float32()*80,
	)
	col := sigilEmber
	if rand.Float32() < 0.35 {
		col = sigilCore
	}
	s.Embers.Emit(pos, vel, 0.55+rand.Float32()*0.3, 5+rand.Float32()*4, col)
}

// Draw desenha o selo em espaco de mundo.
func (s *MeteorSigil) Draw() {
	f := s.fade()
	c := s.center()
	// Culling primeiro: um circulo so, com folga para as brasas que sobem
	// acima do anel. Ver skill/view.go.
	if !visible(c, MeteorSigilRadius*1.6) {
		return
	}
	if f <= 0 && len(s.Embers.particles) == 0 {
		return
	}

	grow := easeOutCubic(clamp01(s.Age / meteorSigilGrow))
	r := MeteorSigilRadius * grow
	// Tremulacao: duas senoides de periodos que nao batem, para o brilho nunca
	// repetir exatamente — uma so daria um piscar de metronomo.
	flick := 0.82 +
		0.11*float32(math.Sin(float64(s.Age)*9.3)) +
		0.07*float32(math.Sin(float64(s.Age)*4.1))
	spin := s.Age * meteorSigilSpin * float32(math.Pi) / 180

	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	// Chao chamuscado sob o conjurador.
	rl.DrawCircleGradient(int32(c.X), int32(c.Y), r*0.95,
		rl.Fade(sigilDeep, 0.16*f), rl.Blank)
	rl.DrawCircleGradient(int32(c.X), int32(c.Y), r*0.6,
		rl.Fade(sigilEmber, 0.16*f*flick), rl.Blank)

	// Anel externo, o anel interno classico do selo, e as marcas de runa entre
	// os dois girando ao contrario (contrarrotacao = figura viva).
	rl.DrawRing(c, r-4, r+4, 0, 360, 64, rl.Fade(sigilDeep, 0.55*f))
	rl.DrawRing(c, r-1.5, r+1.5, 0, 360, 64, rl.Fade(sigilEmber, 0.85*f*flick))
	rl.DrawRing(c, r*0.63-1, r*0.63+1, 0, 360, 56, rl.Fade(sigilEmber, 0.55*f*flick))
	for i := 0; i < 12; i++ {
		a := -spin*1.4 + float32(i)*float32(math.Pi)/6
		rl.DrawLineEx(sigilPoint(c, r*0.86, a), sigilPoint(c, r*0.96, a), 3,
			rl.Fade(sigilEmber, 0.5*f*flick))
	}

	// A ESTRELA. Cinco pontas no anel, ligadas de duas em duas — e cada traco
	// vai em tres passadas (corpo fundo, brasa, nucleo claro), que e o que faz
	// uma linha reta ler como chama e nao como risco.
	var pts [5]rl.Vector2
	for i := 0; i < 5; i++ {
		// -pi/2 poe uma ponta para o norte no instante zero.
		a := spin - float32(math.Pi)/2 + float32(i)*2*float32(math.Pi)/5
		pts[i] = sigilPoint(c, r, a)
	}
	for i := 0; i < 5; i++ {
		a, b := pts[i], pts[(i+2)%5]
		rl.DrawLineEx(a, b, 9, rl.Fade(sigilDeep, 0.5*f))
		rl.DrawLineEx(a, b, 5, rl.Fade(sigilEmber, 0.8*f*flick))
		rl.DrawLineEx(a, b, 2, rl.Fade(sigilCore, 0.9*f*flick))
	}
	// Brasa acesa em cada vertice: sao as cinco "velas" do selo, e o que mata
	// a aparencia de desenho geometrico.
	for _, p := range pts {
		rl.DrawCircleGradient(int32(p.X), int32(p.Y), 21,
			rl.Fade(sigilEmber, 0.55*f*flick), rl.Blank)
		rl.DrawCircle(int32(p.X), int32(p.Y), 4.5, rl.Fade(sigilCore, 0.9*f))
	}

	for _, pt := range s.Embers.particles {
		drawParticle(pt)
	}
}

// sigilPoint devolve o ponto do selo no angulo ang (radianos) e raio r.
func sigilPoint(center rl.Vector2, r, ang float32) rl.Vector2 {
	return rl.NewVector2(
		center.X+float32(math.Cos(float64(ang)))*r,
		center.Y+float32(math.Sin(float64(ang)))*r,
	)
}
