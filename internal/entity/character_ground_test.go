package entity

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// O sprite e desenhado CENTRADO em Position, mas a figura dentro dele esta em
// pe. Colidir no centro deixava um vao de ~104 px ate a sola, e esse vao era
// visivel: andando acima de um tronco, a caixa ja tinha passado enquanto o pe
// ainda era desenhado sobre as raizes.

// FootLine tem que ser o mesmo numero que o metadata da folha declara em
// anchor.y. Se alguem instalar um personagem sem copiar esse campo, o fallback
// (fundo do quadro) poe a caixa 7 px abaixo da sola — pouco o bastante para
// passar despercebido e o bastante para o pe entrar em tudo.
func TestFootLineSegueOContratoDaFolha(t *testing.T) {
	for _, def := range AllCharacters() {
		if def.FrameHeight == 192 && def.FootLine != 186 {
			t.Errorf("%s: FootLine %d, esperado o anchor.y da folha (186)", def.Type, def.FootLine)
		}
	}
}

func TestGroundOffsetReachesTheSole(t *testing.T) {
	for _, def := range AllCharacters() {
		offset := GroundOffset(def)
		// A sola desenhada fica em (FootLine - FrameHeight/2) * RenderScale
		// abaixo de Position. Nada disso vale se o offset ficar perto de
		// zero: seria a caixa no centro de novo.
		if offset < 80 || offset > 130 {
			t.Errorf("%s: offset de chao %v px esta fora da faixa da sola (80-130)", def.Type, offset)
		}
	}
}

func TestGroundOffsetFallsBackWithoutAFootLine(t *testing.T) {
	// Um personagem registrado antes do campo existir nao pode dar offset
	// negativo nem colidir acima da cabeca: sem FootLine, a sola e o fundo
	// do quadro.
	def := CharacterDef{FrameHeight: 192, RenderScale: 1}
	if got, want := GroundOffset(def), float32(96); got != want {
		t.Fatalf("fallback = %v, esperado %v", got, want)
	}
}

func TestPlayerBoxSitsOnTheSoleAndKeepsItsSize(t *testing.T) {
	p := NewPlayer(rl.NewVector2(1000, 2000), CharMago)
	center, width, height := PlayerGroundBox(p)

	// O tamanho e o de sempre: todo vao por onde o jogador passava continua
	// passando. So mudou ONDE a caixa fica.
	if width != p.Radius*2 || height != p.Radius*2 {
		t.Fatalf("caixa %vx%v, esperado %v de lado", width, height, p.Radius*2)
	}
	if center.X != p.Position.X {
		t.Errorf("caixa saiu do eixo do personagem: %v", center.X)
	}
	// A borda de baixo da caixa e a linha da sola.
	sole := GroundPoint(p.Position, GetCharacter(p.CharType)).Y
	if bottom := center.Y + height/2; bottom != sole {
		t.Errorf("fundo da caixa em %v, sola em %v", bottom, sole)
	}
	if center.Y <= p.Position.Y {
		t.Errorf("caixa nao desceu: centro %v, Position %v", center.Y, p.Position.Y)
	}
}

func TestCaminhoLivreNaoMexeNoPersonagem(t *testing.T) {
	// A resolucao trabalha na caixa dos pes, que fica ~105 px abaixo de
	// Position. Se o jogador fosse reconstruido a partir do centro resolvido,
	// somar e subtrair esse offset em float32 todo quadro empurraria uma
	// fracao de pixel para sempre, mesmo sem nada no caminho. A correcao tem
	// que ser exatamente zero quando nada bloqueou.
	for _, charType := range []CharacterType{CharMago, CharPaladina, CharArqueiro} {
		p := NewPlayer(rl.NewVector2(1234, 5678), charType)
		before := p.Position
		wanted, _, _ := PlayerGroundBox(p)
		MoveByGroundCorrection(p, wanted, wanted) // nada bloqueou
		if p.Position != before {
			t.Errorf("%s: caminho livre moveu o personagem de %v para %v", charType, before, p.Position)
		}
	}
}

func TestCorrecaoDoResolvedorChegaInteiraNoPersonagem(t *testing.T) {
	p := NewPlayer(rl.NewVector2(1000, 2000), CharMago)
	wanted, _, _ := PlayerGroundBox(p)
	// O resolvedor devolveu a caixa 7 px atras no X: o personagem tem que
	// andar exatamente esses 7 px, nem mais nem menos.
	MoveByGroundCorrection(p, wanted, rl.NewVector2(wanted.X-7, wanted.Y))
	if want := (rl.NewVector2(993, 2000)); p.Position != want {
		t.Errorf("Position = %v, esperado %v", p.Position, want)
	}
}
