package skill

// A nevoa sombria da Senhora das Trevas.
//
// Ela cobre a ARENA INTEIRA, e e essa a diferenca entre ela e todo dano em area
// que o jogo ja tinha: nao se sai dela andando. As duas unicas saidas sao a
// Area Angelical da Sacerdotisa e o Avatar da Paladina — a conjuracao existe
// para transformar as duas supremas em obrigacao coordenada, e nao em opcao.
//
// Por isso a nevoa nao tem raio nem centro: tem os limites do mundo.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	// DarkFogDuration e quanto tempo ela fica no ar.
	DarkFogDuration float32 = 8.0
	// DarkFogDamagePerSec mata em ~4 s quem estiver desprotegido: alto o
	// bastante para punir posicionamento, baixo o bastante para dar ao jogador
	// tempo de correr ate o altar. Morte instantanea transformaria um erro de
	// recarga num wipe sem saida.
	DarkFogDamagePerSec float32 = 30
	// darkFogTickEvery agrupa o dano em passos legiveis, como o fogo de chao:
	// o jogador perde pedacos que da para ver, e o host manda dois eventos por
	// segundo por jogador em vez de sessenta.
	darkFogTickEvery float32 = 0.5
	// DarkFogFade e o tempo de entrada e de saida. Sem ele a tela inteira muda
	// de cor num quadro, o que le como falha de render e nao como conjuracao.
	DarkFogFade float32 = 0.8
)

// DarkFog e a nevoa cobrindo uma regiao do mundo.
type DarkFog struct {
	Bounds    rl.Rectangle
	TTL       float32
	age       float32
	tickAccum float32
}

// NewDarkFog cria a nevoa sobre uma regiao.
func NewDarkFog(bounds rl.Rectangle) *DarkFog {
	return &DarkFog{Bounds: bounds, TTL: DarkFogDuration}
}

// Update conta o tempo. Devolve false quando ela se dissipa.
func (f *DarkFog) Update(dt float32) bool {
	f.TTL -= dt
	f.age += dt
	return f.TTL > 0
}

// Tick devolve quantas vezes o dano deve ser aplicado neste quadro, e quanto
// vale cada aplicacao.
//
// Devolve CONTAGEM e nao um booleano porque um quadro longo (troca de mapa,
// pico de GC) pode atravessar mais de um passo, e engolir os passos perdidos
// faria a nevoa doer menos justamente quando o jogo esta engasgando.
func (f *DarkFog) Tick(dt float32) (times int, damage float32) {
	f.tickAccum += dt
	for f.tickAccum >= darkFogTickEvery {
		f.tickAccum -= darkFogTickEvery
		times++
	}
	return times, DarkFogDamagePerSec * darkFogTickEvery
}

// Contains reporta se um ponto esta na nevoa.
func (f *DarkFog) Contains(p rl.Vector2) bool {
	return rl.CheckCollisionPointRec(p, f.Bounds)
}

// Draw cobre a regiao. Duas camadas: um veu escuro que engole o chao e uma
// segunda camada que ONDULA, porque uma cor chapada sobre o mapa inteiro le
// como bug de render — o movimento e o que diz "isto e uma coisa, e ela esta
// acontecendo agora".
func (f *DarkFog) Draw() {
	in := clamp01(f.age / DarkFogFade)
	out := clamp01(f.TTL / DarkFogFade)
	a := in * out

	rl.DrawRectangleRec(f.Bounds, rl.NewColor(18, 10, 26, uint8(170*a)))

	rl.BeginBlendMode(rl.BlendAdditive)
	step := float32(220)
	for y := f.Bounds.Y; y < f.Bounds.Y+f.Bounds.Height; y += step {
		for x := f.Bounds.X; x < f.Bounds.X+f.Bounds.Width; x += step {
			ph := float64((x*0.9+y*1.3)/400) + float64(f.age)*1.7
			r := step * (0.55 + 0.25*float32(math.Sin(ph)))
			rl.DrawCircle(int32(x+step/2), int32(y+step/2), r,
				rl.NewColor(70, 40, 96, uint8(16*a)))
		}
	}
	rl.EndBlendMode()
}
