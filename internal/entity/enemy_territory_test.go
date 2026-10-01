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

// AS DUAS PORTAS DE AQUISICAO (ver `Guard.wants`).
//
// O SETOR e o que impede o mapa inteiro de acordar de uma vez agora que a
// perseguicao nao termina mais: o grupo acumula perseguidores faixa por faixa
// conforme avanca, em vez de puxar 136 monstros numa fila so. Raio generoso,
// limitado pelo retangulo.
//
// A AMEACA e a porta curta que ignora o retangulo, e ela existe porque o setor
// tinha virado escudo do jogador — ver o teste logo abaixo dela.
//
// Os quatro testes seguintes cobrem as duas e a fronteira entre elas.

// sectorGuard e um posto num setor ESTREITO, como as faixas do mapa 3: quem
// esta acima ou abaixo da linha nao entrou no pedaco deste monstro.
func sectorGuard(post rl.Vector2) *Enemy {
	e := NewEnemy(EnemyTypeGarrison, post.X, post.Y)
	a, b := PatrolSegment(post, 0)
	e.Guard = Guard{
		Post: post, PatrolA: a, PatrolB: b, Radius: 1100,
		Territory: rl.NewRectangle(post.X-2000, post.Y-200, 4000, 400),
	}
	return e
}

func TestGuardIgnoresTrespassersOutsideItsSector(t *testing.T) {
	post := rl.NewVector2(5000, 5000)
	e := sectorGuard(post)
	// Fora do setor E fora do alcance de qualquer arma do elenco: este
	// realmente nao e problema dele.
	far := rl.NewVector2(post.X+100, post.Y+GuardThreatRadius()+400)
	if e.guardTarget([]PlayerState{playerAt(far)}) != nil {
		t.Error("notou alguem fora do proprio setor e fora de alcance: quem " +
			"esta do outro lado da barricada, longe, nao e problema deste monstro")
	}
}

// A LINHA DO SETOR NAO E ESCUDO (23/08/2026).
//
// Relato do Gui: "os jogadores estao conseguindo matar os monstros de longe,
// sem precisar entrar no posto". A faixa do mapa 3 tem 1280 px de altura e a
// flecha alcanca 1120: dava para ficar na faixa de tras e limpar o posto
// seguinte um a um, porque `covers` exigia o jogador DENTRO do retangulo para o
// guarda sequer olhar.
func TestGuardDefendsItsPostAgainstSomeoneShootingFromOutsideTheSector(t *testing.T) {
	post := rl.NewVector2(5000, 5000)
	e := sectorGuard(post)
	// Fora do setor (a faixa acaba 200 px abaixo do posto), mas perto o
	// bastante para acertar o posto daqui.
	shooter := rl.NewVector2(post.X, post.Y+GuardThreatRadius()-100)
	if e.guardTarget([]PlayerState{playerAt(shooter)}) == nil {
		t.Error("um jogador ao alcance de acertar o posto foi ignorado por " +
			"estar do outro lado da linha do setor")
	}
}

// O raio de AMEACA e mais curto que o de setor, e isso e o ponto: a segunda
// porta nao pode virar "todo monstro do mapa acorda". Ela troca alcance por
// geometria — perto, em qualquer retangulo; longe, so no meu.
func TestTheThreatDoorIsShorterThanTheSectorDoor(t *testing.T) {
	const sectorRadius = float32(2600) // guardRadiusFor, network/host_garrison.go
	if GuardThreatRadius() >= sectorRadius {
		t.Errorf("raio de ameaca (%.0f) nao e mais curto que o de setor (%.0f); "+
			"a porta curta viraria a porta larga e o mapa acordaria inteiro",
			GuardThreatRadius(), sectorRadius)
	}
}

// E ele tem de cobrir a arma de MAIOR alcance do elenco, ou a classe que
// alcanca mais longe continua com o exploit para ela sozinha.
func TestTheThreatRadiusCoversTheLongestWeaponInTheCast(t *testing.T) {
	for _, def := range AllCharacters() {
		if reach := AttackReach(def.Type); reach > GuardThreatRadius() {
			t.Errorf("%s alcanca %.0f px e o posto so acorda a %.0f: ela mata "+
				"de fora sem ser notada", def.Type, reach, GuardThreatRadius())
		}
	}
}

// LEVAR DANO E NOTAR. A porta de ameaca cobre a geometria previsivel; esta e a
// rede embaixo dela — magia de area, flecha celestial, um angulo que ninguem
// previu. Se o golpe chegou, o guarda foi encontrado.
func TestAGuardThatIsHitComesAfterWhoeverIsThere(t *testing.T) {
	post := rl.NewVector2(5000, 5000)
	e := sectorGuard(post)

	// Longe demais para as duas portas de aquisicao.
	sniper := rl.NewVector2(post.X, post.Y+GuardThreatRadius()+3000)
	players := []PlayerState{playerAt(sniper)}
	if e.guardTarget(players) != nil {
		t.Fatal("o alvo estava fora das duas portas e mesmo assim foi notado")
	}

	e.TakeDamage(1)
	if e.guardTarget(players) == nil {
		t.Error("o guarda levou dano de longe e continuou patrulhando como se " +
			"nada tivesse acontecido")
	}
}

// Nao vale para quem nao guarda nada: um monstro de horda ja caca o mais
// proximo, e marcar `chasing` nele so criaria estado que ninguem le.
func TestTakingDamageDoesNotInventAChaseForANonGuard(t *testing.T) {
	e := NewEnemy(EnemyTypeFast, 0, 0)
	e.TakeDamage(1)
	if e.chasing {
		t.Error("um monstro sem posto ficou marcado como perseguindo")
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
