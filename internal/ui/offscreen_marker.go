package ui

// A seta de borda de tela para a gargula que a camera nao mostra (playtest de
// 27/09/2026). A gargula tem alcance global (entity.SentryGlobalRange): ela
// atira de fora da tela, e o Arqueiro — quem tem de derruba-la — precisava
// andar ate ela so para descobrir onde estava. A seta diz ONDE, e o jogador
// decide o caminho.
//
// Este arquivo so sabe geometria de tela e desenho; QUEM recebe seta mora em
// game/sentry_markers.go, que le as gargulas vivas do lado certo da rede.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// markerInset e a folga entre a seta e a borda, em pixels base (vezes
	// UIScale): encostada na borda, metade dela sairia da tela.
	markerInset float32 = 46
	// markerRadius e o disco da seta; markerTip, quanto a ponta sai dele.
	markerRadius float32 = 21
	markerTip    float32 = 17
)

// As cores da propria gargula (Color "#6B287A" da EnemyDef, e o violeta da
// esfera em skill/sentry_orb_draw.go): a seta e a criatura tem de ser
// obviamente a mesma coisa.
var (
	markerDeep = rl.NewColor(0x6B, 0x28, 0x7A, 0xFF)
	markerMid  = rl.NewColor(0xA4, 0x4B, 0xD4, 0xFF)
	markerHot  = rl.NewColor(0xEB, 0xD6, 0xFF, 0xFF)
)

// EdgeMarkerPoint projeta um alvo de tela na borda da tela.
//
// Devolve onScreen verdadeiro quando o alvo ja esta visivel (nada a indicar).
// Senao, `at` e onde a reta do centro da tela ate o alvo cruza o retangulo da
// tela recuado de `inset`, e `dir` e a direcao normalizada dessa reta — para
// onde a seta aponta. E a interseccao, e nao um clamp por eixo, para a seta
// ficar NA DIRECAO do alvo: um clamp jogaria toda gargula em diagonal para o
// canto da tela.
func EdgeMarkerPoint(target rl.Vector2, sw, sh, inset float32) (at, dir rl.Vector2, onScreen bool) {
	if target.X >= 0 && target.X <= sw && target.Y >= 0 && target.Y <= sh {
		return target, rl.Vector2{}, true
	}
	center := rl.NewVector2(sw/2, sh/2)
	d := rl.Vector2Subtract(target, center)
	length := rl.Vector2Length(d)
	if length <= 0 {
		return center, rl.Vector2{}, true
	}
	dir = rl.Vector2Scale(d, 1/length)

	halfW, halfH := sw/2-inset, sh/2-inset
	if halfW < 1 {
		halfW = 1
	}
	if halfH < 1 {
		halfH = 1
	}
	// Menor escala que leva o vetor ate uma das quatro bordas recuadas.
	t := float32(math.MaxFloat32)
	if dir.X != 0 {
		t = halfW / float32(math.Abs(float64(dir.X)))
	}
	if dir.Y != 0 {
		if ty := halfH / float32(math.Abs(float64(dir.Y))); ty < t {
			t = ty
		}
	}
	return rl.Vector2Add(center, rl.Vector2Scale(dir, t)), dir, false
}

// SentryMarkerInset e a folga da borda ja escalada pela Interface.
func SentryMarkerInset() float32 { return markerInset * PixelScale() }

// DrawSentryMarker desenha a seta de uma gargula fora da tela em `at`,
// apontando para `dir`. `strength` (0..1) vem da distancia: perto e forte,
// longe e discreta — dez gargulas no mapa 7 nao podem virar dez alarmes iguais.
func DrawSentryMarker(at, dir rl.Vector2, strength float32) {
	strength = clamp01(strength)
	// PixelScale, e nao UIScale: raio, ponta e folga sao pixels fixos.
	s := PixelScale() * (0.8 + 0.2*strength)
	r := markerRadius * s
	alpha := 0.45 + 0.55*strength
	// Respiro lento: le como "viva", sem virar pisca-pisca.
	pulse := 0.5 + 0.5*float32(math.Sin(rl.GetTime()*3.0))

	// A ponta, primeiro, para o disco cobrir a base dela.
	tip := rl.Vector2Add(at, rl.Vector2Scale(dir, r+markerTip*s))
	side := rl.NewVector2(-dir.Y, dir.X)
	base := rl.Vector2Add(at, rl.Vector2Scale(dir, r*0.35))
	left := rl.Vector2Add(base, rl.Vector2Scale(side, r*0.85))
	right := rl.Vector2Subtract(base, rl.Vector2Scale(side, r*0.85))
	// raylib quer os vertices em sentido anti-horario na tela; com o eixo Y
	// para baixo, (tip, right, left) e o anti-horario para qualquer dir.
	rl.DrawTriangle(tip, right, left, rl.Fade(markerMid, alpha))
	rl.DrawTriangle(tip, left, right, rl.Fade(markerMid, alpha))

	// Disco escuro de fundo: a seta tem de ler sobre grama, pedra e nevoa.
	rl.DrawCircleV(at, r+2*s, rl.Fade(rl.Black, 0.55*alpha))
	rl.DrawCircleV(at, r, rl.Fade(markerDeep, alpha))
	// Aro claro: e o que separa a seta do fundo verde ou cinza a distancia.
	rl.DrawRing(at, r-2*s, r, 0, 360, 32, rl.Fade(markerMid, alpha))
	// O "olho" da gargula: o mesmo nucleo claro da esfera dela.
	rl.DrawCircleV(at, r*(0.34+0.08*pulse), rl.Fade(markerHot, alpha))
}
