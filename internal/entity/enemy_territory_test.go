package entity

// O comportamento que o Gui descreveu, em frases, e um teste para cada:
//
//   "a ideia e que seja uma patrulhada, ficar indo de um lado para o outro"
//   "eles ficam parados e tremendo"
//   "o campo de visao de alguns monstros como o Orc esta muito baixo"
//   "quando o jogador se afasta um pouco o monstro ja para de perseguir;
//    acho interessante o monstro perseguir sempre"
//
// Nenhuma aparece em outro lugar do codigo como uma afirmacao verificavel: elas
// sao a soma de um raio, de uma tolerancia e de uma regra de desistencia. Um
// numero mexido sozinho quebra uma delas sem quebrar as outras, e o defeito so
// aparece com alguem jogando.
//
// A ULTIMA FRASE SUBSTITUIU DUAS ANTERIORES — "enquanto ele ta batendo ele
// continua na perseguicao" e "se nao conseguiu bater e se afastou muito do
// posto, deve retornar". Os testes daquelas duas foram removidos junto com o
// prazo e a coleira que elas descreviam; o teste que sobrou no lugar exige o
// contrario deles.

import (
	"math"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// guardedEnemy builds a garrison monster with a horizontal beat around origin,
// inside a territory big enough not to be the thing under test.
func guardedEnemy(origin rl.Vector2, radius float32) *Enemy {
	e := NewEnemy(EnemyTypeFast, origin.X, origin.Y)
	a, b := PatrolSegment(origin, 0)
	e.Guard = Guard{
		Post:      origin,
		PatrolA:   a,
		PatrolB:   b,
		Territory: rl.NewRectangle(origin.X-6000, origin.Y-6000, 12000, 12000),
		Radius:    radius,
	}
	return e
}

// playerAt is a living player at a world position.
func playerAt(p rl.Vector2) PlayerState {
	return PlayerState{X: int(p.X), Y: int(p.Y)}
}

// TestGuardPatrolsBackAndForth: sem ninguem por perto, o monstro anda.
//
// O defeito que este teste cobre foi visto em jogo como "parados e tremendo": o
// alvo de repouso era um ponto exato, a separacao entre vizinhos empurrava o
// monstro para fora dele e ele passava a vida corrigindo um passo. Um trecho
// com duas pontas e uma tolerancia generosa e o que troca o tremor por
// vai-e-vem.
func TestGuardPatrolsBackAndForth(t *testing.T) {
	e := guardedEnemy(rl.NewVector2(5000, 5000), 1100)
	const dt = float32(1.0 / 60.0)

	turns, moved := 0, false
	wasForward := e.patrolForward
	for i := 0; i < 60*20; i++ {
		to, walking := e.patrolStep(dt)
		if walking {
			moved = true
			// Um passo na direcao da ponta, sem fisica: o que esta em teste e a
			// decisao, nao o resolvedor de colisao.
			d := rl.Vector2Subtract(to, e.Position)
			if l := float32(math.Hypot(float64(d.X), float64(d.Y))); l > 0 {
				step := e.Speed * dt / l
				e.Position = rl.NewVector2(e.Position.X+d.X*step, e.Position.Y+d.Y*step)
			}
		}
		if e.patrolForward != wasForward {
			turns++
			wasForward = e.patrolForward
		}
	}
	if !moved {
		t.Error("o monstro nunca deu um passo: ele nao patrulha, so fica parado")
	}
	if turns < 2 {
		t.Errorf("virou %d vez(es) em 20s; uma patrulha vai e VOLTA", turns)
	}
}

// TestGuardChaseNeverGivesUp e a frase "acho interessante o monstro perseguir
// sempre", em forma verificavel.
//
// O jogador anda a 200 e o orc a 130, entao QUALQUER teto — prazo, distancia do
// posto, linha de setor — e uma saida gratuita: bastava recuar alguns passos.
// Este teste leva o alvo para longe de todas as tres coisas de uma vez.
func TestGuardChaseNeverGivesUp(t *testing.T) {
	post := rl.NewVector2(5000, 5000)
	e := guardedEnemy(post, 1100)
	players := []PlayerState{playerAt(rl.NewVector2(5400, 5000))}

	if e.guardTarget(players) == nil {
		t.Fatal("nao comecou a perseguir alguem dentro do raio de visao")
	}
	// Fora do raio de visao, fora do setor e do outro lado do mapa. Nenhuma
	// dessas coisas pode soltar o alvo.
	for _, away := range []rl.Vector2{
		rl.NewVector2(post.X+3000, post.Y),
		rl.NewVector2(post.X, post.Y+9000),
		rl.NewVector2(0, 0),
	} {
		players[0] = playerAt(away)
		if e.guardTarget(players) == nil {
			t.Fatalf("largou o alvo em (%.0f, %.0f); a perseguicao nao termina "+
				"por distancia nem por setor", away.X, away.Y)
		}
	}
	// E nem o tempo a encerra: doze segundos sem encostar no jogador.
	for i := 0; i < 60*12; i++ {
		if e.guardTarget(players) == nil {
			t.Fatalf("desistiu no segundo %.1f", float32(i)/60)
		}
	}
}

// TestGuardChaseEndsOnlyWhenNobodyIsLeftAlive: a unica saida.
//
// Importa porque a patrulha e o `Reset` da fase dependem dela — um monstro que
// nunca solta o alvo nem quando o alvo morre ficaria caminhando para um cadaver
// pelo resto da corrida.
func TestGuardChaseEndsOnlyWhenNobodyIsLeftAlive(t *testing.T) {
	post := rl.NewVector2(5000, 5000)
	e := guardedEnemy(post, 1100)
	players := []PlayerState{playerAt(rl.NewVector2(5400, 5000))}
	if e.guardTarget(players) == nil {
		t.Fatal("nao comecou a perseguir")
	}
	players[0].IsDead = true
	if e.guardTarget(players) != nil {
		t.Error("continuou perseguindo um jogador morto")
	}
	if e.chasing {
		t.Error("continuou marcado como em perseguicao sem ninguem para perseguir")
	}
}

// TestTheSlowestMonsterSeesTheFarthest.
//
// Nao e gosto: o orc anda a 130 contra os 200 do jogador. Se ele notar o
// invasor a mesma distancia que o lobo a 240, ele nunca chega a lugar nenhum e
// le como distraido em vez de pesado. A relacao — o mais lento ve mais longe —
// e o que este teste guarda; o numero exato pode mudar.
func TestTheSlowestMonsterSeesTheFarthest(t *testing.T) {
	orc := GetEnemyDef(EnemyTypeGarrison)
	wolf := GetEnemyDef(EnemyTypeFast)
	if orc.Speed >= wolf.Speed {
		t.Skip("o orc deixou de ser mais lento que o lobo; a premissa mudou")
	}
	if orc.Vision <= wolf.Vision {
		t.Errorf("orc: velocidade %.0f e visao %.0f; lobo: velocidade %.0f e "+
			"visao %.0f — o mais lento tem de ver primeiro",
			orc.Speed, orc.Vision, wolf.Speed, wolf.Vision)
	}
}

// TestGuardSeesFartherThanItsAttackRange: o campo de visao e de verdade.
//
// A primeira versao usava 640 px, e em jogo o jogador tinha de quase encostar.
// O teste nao fixa 1100: ele exige que a visao seja MUITO maior que o alcance
// do golpe, que e a propriedade que faltava.
func TestGuardSeesFartherThanItsAttackRange(t *testing.T) {
	post := rl.NewVector2(5000, 5000)
	e := guardedEnemy(post, 1100)

	far := rl.NewVector2(post.X+e.Guard.Radius*0.9, post.Y)
	if e.guardTarget([]PlayerState{playerAt(far)}) == nil {
		t.Errorf("nao viu um jogador a %.0fpx, dentro do proprio raio",
			e.Guard.Radius*0.9)
	}
	if e.Guard.Radius < e.AttackRange*8 {
		t.Errorf("raio de visao %.0f contra alcance de golpe %.0f: o monstro "+
			"so nota quem ja esta batendo nele", e.Guard.Radius, e.AttackRange)
	}
}

// TestGuardIgnoresTrespassersOutsideItsSector: o setor decide QUEM ele nota.
//
// E a unica coisa que o setor ainda faz, e e o que impede o mapa inteiro de
// acordar de uma vez agora que a perseguicao nao termina mais: o grupo acumula
// perseguidores faixa por faixa conforme avanca, em vez de puxar 136 monstros
// numa fila so.
func TestGuardIgnoresTrespassersOutsideItsSector(t *testing.T) {
	post := rl.NewVector2(5000, 5000)
	e := NewEnemy(EnemyTypeGarrison, post.X, post.Y)
	a, b := PatrolSegment(post, 0)
	// Setor estreito: uma faixa de 400px de altura, como as do mapa 3.
	e.Guard = Guard{
		Post: post, PatrolA: a, PatrolB: b, Radius: 1100,
		Territory: rl.NewRectangle(post.X-2000, post.Y-200, 4000, 400),
	}
	// Perto, mas do outro lado da fronteira do setor.
	outside := rl.NewVector2(post.X+100, post.Y+600)
	if e.guardTarget([]PlayerState{playerAt(outside)}) != nil {
		t.Error("notou alguem fora do proprio setor: quem esta do outro lado " +
			"da barricada nao e problema deste monstro")
	}
}

// TestOrcHitBoxCoversItsBody guarda a caixa de acerto do orc.
//
// O defeito que este teste cobre foi visto em jogo e fotografado: para acertar o
// orc era preciso mirar nos pes. A causa e uma confusao entre duas caixas — a
// posicao do monstro e a ANCORA (o pe, para quem tem FootLine), e todo teste de
// acerto usava um circulo de 45 px ali. Ele cobria 26% do corpo.
//
// O teste nao fixa os numeros: ele exige a PROPRIEDADE — que o circulo de
// acerto cubra a maior parte da figura desenhada.
func TestOrcHitBoxCoversItsBody(t *testing.T) {
	def := GetEnemyDef(EnemyTypeGarrison)
	idle, ok := def.Anims[AnimIdle]
	if !ok {
		t.Fatal("o orc nao tem animacao parada; nao da para medir o corpo")
	}
	// Altura desenhada da figura, do pe ao topo do quadro.
	drawn := float64(idle.FootLine) * float64(def.RenderScale)

	e := NewEnemy(EnemyTypeGarrison, 1000, 1000)
	top := float64(e.Position.Y) - float64(e.HitRadius()) +
		float64(e.HitCenter().Y-e.Position.Y)
	bottom := float64(e.Position.Y) + float64(e.HitRadius()) +
		float64(e.HitCenter().Y-e.Position.Y)
	feet := float64(e.Position.Y)

	covered := math.Min(feet, bottom) - math.Max(feet-drawn, top)
	if covered/drawn < 0.6 {
		t.Errorf("a caixa de acerto cobre %.0f%% do corpo (%.0f de %.0f px); "+
			"abaixo de 60%% o jogador tem de mirar em uma parte especifica do "+
			"monstro em vez de no monstro", 100*covered/drawn, covered, drawn)
	}
	// E ela nao pode descer abaixo do chao: acertar embaixo do bicho e o
	// defeito oposto, e e o que aconteceria engordando `Radius` sem subir o
	// centro.
	if bottom > feet+float64(e.Radius) {
		t.Errorf("a caixa de acerto desce %.0f px abaixo dos pes; o jogador "+
			"acertaria o chao vazio sob o monstro", bottom-feet)
	}
}
