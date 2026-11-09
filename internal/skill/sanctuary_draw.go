package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A PALETA E A DA SACERDOTISA: dourado -> amarelo divino -> branco.
//
// A versao anterior desenhava discos opacos empilhados em blending NORMAL. Um
// amarelo de alfa baixo pintado por cima da grama nao vira luz: ele vira uma
// tinta suja, e sobre o verde do mapa o miolo aparecia rosado (relato do Gui,
// 31/08/2026 — o print mostrava um circulo rosa no meio da grama). Nada aqui
// e mais desenhado em blending normal: e tudo aditivo, que e o que faz cor
// clara sobre fundo escuro SOMAR em vez de cobrir. Aditivo nao consegue puxar
// para o rosa, porque so acrescenta o que a cor tem — e dourado nao tem azul.
//
// A outra regra e a do doc: a cor mais quente ocupa a MENOR area. O branco
// vive so no nucleo e nas particulas; o dourado espalha.
var (
	sanctGold   = rl.NewColor(255, 198, 74, 255)  // dourado, a cor de fora
	sanctYellow = rl.NewColor(255, 232, 150, 255) // amarelo divino, o meio
	sanctWhite  = rl.NewColor(255, 252, 235, 255) // branco quente, o nucleo
)

// A PULSACAO E A IDENTIDADE DA MAGIA, e ela existe porque a primeira tentativa
// nao tinha nenhuma (relato do Gui, 31/08/2026).
//
// Aquela versao tinha borda firme, anel pulsante por fora, arcos
// contra-rotativos e motes em orbita achatada em volta do corpo — que e,
// item por item, o vocabulario do Escudo Sagrado e do Avatar dos Deuses. Duas
// classes com a mesma paleta dourada nao podem ter tambem a mesma GRAMATICA:
// de relance, o Santuario virava "o escudo da Paladina, so que maior".
//
// O contorno saiu inteiro. No lugar dele, ondas que NASCEM NA SACERDOTISA e
// morrem exatamente no alcance da cura. A diferenca nao e so estetica: a borda
// estatica dizia onde a magia acaba, e a onda diz de onde ela VEM — a fonte e
// ela, e o limite aparece como consequencia, uma vez por batida, no lugar de
// ficar riscado no chao o tempo todo.
//
// REGIME (Regra 3 do doc/skill_performance.md): uma onda a cada 0,9 s vivendo
// 1,4 s = no maximo 2 ondas no ar por Santuario, e um Santuario por
// Sacerdotisa. Cada onda custa 4 DrawRing.
const (
	sanctWaveEvery = 0.9
	sanctWaveLife  = 1.4
)

// SanctuaryFX holds the procedural visuals for one sanctuary: the ground
// light, the expanding pulses and the floating sacred motes. It is pure
// presentation.
type SanctuaryFX struct {
	particles []sacredParticle
	time      float32
	// waves guarda a idade de cada pulsacao viva, da mais velha para a mais
	// nova. Idade e nao raio: o raio e derivado dela no desenho, entao a onda
	// nao pode dessincronizar do proprio alfa.
	waves     []float32
	waveTimer float32
}

// NewSanctuaryFX creates the particle/pulse state for a sanctuary.
func NewSanctuaryFX() *SanctuaryFX {
	fx := &SanctuaryFX{
		// A primeira onda nasce junto com a magia: e ela que faz o
		// lancamento ter um instante proprio, sem precisar de uma animacao
		// de entrada separada.
		waves: []float32{0},
	}
	// 22 motes: teto fixo, nao emissao por quadro (Regra 5). Elas nascem uma
	// vez e se reciclam sozinhas ao sair da area.
	for i := 0; i < 22; i++ {
		fx.particles = append(fx.particles, newSacredParticle())
	}
	return fx
}

// update advances the motes and the pulse cadence.
func (fx *SanctuaryFX) update(dt float32) {
	for i := range fx.particles {
		fx.particles[i].update(dt)
	}
	fx.time += dt

	fx.waveTimer += dt
	if fx.waveTimer >= sanctWaveEvery {
		fx.waveTimer -= sanctWaveEvery
		fx.waves = append(fx.waves, 0)
	}
	// Envelhece e poda no mesmo laco, reaproveitando o array (waves[:0]) para
	// nao alocar um slice por quadro — o mesmo padrao do Avatar.
	kept := fx.waves[:0]
	for _, w := range fx.waves {
		if w += dt; w < sanctWaveLife {
			kept = append(kept, w)
		}
	}
	fx.waves = kept
}

// draw renders the full aura for s with global opacity a.
// Layers (back..front): consecrated ground, pulses, motes, body glow.
func (fx *SanctuaryFX) draw(s *Sanctuary, a float32) {
	if a <= 0 {
		return
	}
	ground := s.GroundCenter()
	// Regra 2 (doc/skill_performance.md): culling ANTES do blend mode, pelo
	// alcance visual inteiro e nao pelo raio de cura.
	if !visible(ground, SanctuaryVisualRadius) {
		return
	}

	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	fx.drawGround(ground, a)
	fx.drawPulses(ground, a)
	fx.drawParticles(ground, a)
	fx.drawBodyGlow(s.Position, a)
}

// drawGround lays the consecrated light on the floor: gradientes concentricos
// do dourado largo ao nucleo branco pequeno. Gradiente e nao disco solido —
// o disco de borda dura era metade do problema do visual antigo, e agora e
// tambem o que deixa a onda ser a unica linha da magia.
func (fx *SanctuaryFX) drawGround(c rl.Vector2, a float32) {
	breath := 1 + 0.04*float32(math.Sin(float64(fx.time)*1.7))
	x, y := int32(c.X), int32(c.Y)
	rl.DrawCircleGradient(x, y, SanctuaryRadius*breath,
		rl.Fade(sanctGold, 0.15*a), rl.Blank)
	rl.DrawCircleGradient(x, y, SanctuaryRadius*0.6*breath,
		rl.Fade(sanctYellow, 0.13*a), rl.Blank)
	rl.DrawCircleGradient(x, y, SanctuaryRadius*0.22*breath,
		rl.Fade(sanctWhite, 0.22*a), rl.Blank)
}

// waveRadius converte a idade de uma onda no raio dela.
//
// A desaceleracao (ease-out) e o que faz a pulsacao parecer emitida e nao
// disparada: ela sai rapido de perto da Sacerdotisa e vai perdendo forca ate
// morrer exatamente em SanctuaryRadius, que e onde a cura acaba. Crescimento
// linear leria como um anel mecanico passando por cima do chao.
func waveRadius(age float32) float32 {
	p := clamp01(age / sanctWaveLife)
	return SanctuaryRadius * p * (2 - p)
}

// waveAlpha e a forca da onda: nasce forte, some no fim, e ainda sobe do zero
// nos primeiros 12% para a onda brotar do corpo dela em vez de piscar pronta.
func waveAlpha(age float32) float32 {
	p := clamp01(age / sanctWaveLife)
	born := clamp01(p / 0.12)
	return born * (1 - p) * (1 - p)
}

// drawPulses desenha as ondas de luz consagrada que saem da Sacerdotisa.
//
// Cada onda e QUATRO aneis concentricos e nao um so: um anel fino tem borda
// dura de linha desenhada, e quatro com espessura crescente e alfa decrescente
// leem como luz difusa. A crista clareia para o branco no comeco (perto dela,
// onde a energia e nova) e cai para o dourado conforme abre — o mesmo
// principio de "mais quente = menor" aplicado ao tempo em vez do raio.
func (fx *SanctuaryFX) drawPulses(c rl.Vector2, a float32) {
	for _, age := range fx.waves {
		w := waveAlpha(age) * a
		if w <= 0 {
			continue
		}
		r := waveRadius(age)
		if r < 6 {
			continue
		}
		p := clamp01(age / sanctWaveLife)
		crest := lerpSanctColor(sanctWhite, sanctGold, p)
		// halo largo, quase transparente -> crista fina e clara
		rl.DrawRing(c, r-16, r+16, 0, 360, 48, rl.Fade(sanctGold, 0.10*w))
		rl.DrawRing(c, r-9, r+9, 0, 360, 48, rl.Fade(crest, 0.16*w))
		rl.DrawRing(c, r-4, r+4, 0, 360, 56, rl.Fade(crest, 0.30*w))
		rl.DrawRing(c, r-1.5, r+1.5, 0, 360, 56, rl.Fade(sanctWhite, 0.45*w))
	}
}

// waveBoost devolve o quanto uma mote a dist do centro deve clarear porque uma
// onda esta passando por ela.
//
// E o detalhe que amarra as duas metades do efeito: sem ele as motes flutuam
// no proprio ritmo e a pulsacao passa por cima delas como se fossem camadas
// separadas; com ele a onda ACENDE o que atravessa, e o chao inteiro responde
// a batida. So a onda mais forte sobre cada mote conta — somar duas cristas
// estouraria o alfa.
func (fx *SanctuaryFX) waveBoost(dist float32) float32 {
	const band = 34
	best := float32(0)
	for _, age := range fx.waves {
		d := dist - waveRadius(age)
		if d < 0 {
			d = -d
		}
		if d >= band {
			continue
		}
		if b := (1 - d/band) * waveAlpha(age); b > best {
			best = b
		}
	}
	return best
}

// drawBodyGlow hugs the priestess herself: so um brilho quente no corpo, sem
// aneis nem motes em orbita — orbita e o vocabulario do Escudo Sagrado. Aqui a
// unica coisa que diz "a magia e dela" e o fato de a luz ser mais forte onde
// ela esta, e de cada onda nascer desse ponto.
func (fx *SanctuaryFX) drawBodyGlow(body rl.Vector2, a float32) {
	pulse := 1 + 0.07*float32(math.Sin(float64(fx.time)*3.1))
	rl.DrawCircleGradient(int32(body.X), int32(body.Y), 88*pulse,
		rl.Fade(sanctYellow, 0.12*a), rl.Blank)
	rl.DrawCircleGradient(int32(body.X), int32(body.Y), 34*pulse,
		rl.Fade(sanctWhite, 0.16*a), rl.Blank)
}

// lerpSanctColor mistura duas cores da paleta por t (0..1), canal a canal.
func lerpSanctColor(c0, c1 rl.Color, t float32) rl.Color {
	return rl.NewColor(
		uint8(float32(c0.R)+(float32(c1.R)-float32(c0.R))*t),
		uint8(float32(c0.G)+(float32(c1.G)-float32(c0.G))*t),
		uint8(float32(c0.B)+(float32(c1.B)-float32(c0.B))*t),
		255,
	)
}
