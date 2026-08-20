package entity

// Defesa de territorio: o monstro pertence a um LUGAR, nao a uma onda.
//
// No mapa 2 todo inimigo caca o jogador mais proximo esteja ele onde estiver, e
// isso e o que faz uma horda. O mapa 3 e uma travessia: cada guarnicao guarda o
// pedaco dela, e atravessar o vao de uma barricada tira o grupo do alcance de
// quem estava atras.
//
// A PRIMEIRA VERSAO DISTO FALHOU EM TRES PONTOS, todos observados em jogo, e os
// tres estao consertados aqui:
//
//  1. OS MONSTROS TREMIAM PARADOS. Sem nada para caçar eles voltavam ao posto,
//     e "no posto" era um ponto exato: a direcao de separacao empurrava um pouco,
//     a distancia passava da tolerancia, eles davam um passo de volta, eram
//     empurrados de novo. Cinco monstros no mesmo posto viravam um enxame
//     vibrando. Agora cada um tem o PROPRIO trecho de patrulha e caminha entre
//     duas pontas, com pausa nas pontas — o vai-e-vem que o Gui pediu.
//  2. O CAMPO DE VISAO ERA CURTO. 640 px e menos de meia tela; o jogador
//     precisava quase encostar. Subiu para 1100, depois para 2600.
//  3. A PERSEGUICAO NAO TINHA COMPROMISSO. Ela terminava na borda do raio, e
//     dar dois passos para tras cancelava tudo. Passou a durar um TEMPO
//     renovado a cada golpe — e mesmo assim continuava curta demais em jogo.
//
// QUEM COMECOU A PERSEGUIR NAO DESISTE MAIS. Pedido do Gui, e a razao e de
// ritmo: o jogador anda a 200 e o orc a 130, entao qualquer teto — de tempo, de
// distancia do posto ou de setor — vira uma saida gratuita. Bastava recuar
// alguns passos e a fase inteira soltava o alvo. Agora o teto so existe para
// ADQUIRIR: o monstro so nota quem entra no campo de visao DENTRO do setor
// dele. Depois disso ele vai atras ate o fim do mapa.
//
// O que impede o exploit de puxar a fase inteira nao e mais um teto, e a
// geometria: e o setor que limita quantos acordam de uma vez, entao o grupo
// acumula perseguidores faixa por faixa em vez de acordar o mapa. E fugir para
// tras deixou de ser de graca porque o mapa passou a ter monstro atras.

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Guard is what a garrison enemy defends: a patrol and how far from it.
type Guard struct {
	// Post is the middle of this monster's beat, and the anchor every distance
	// is measured from.
	Post rl.Vector2
	// PatrolA and PatrolB are the two ends of the beat. Equal to Post when the
	// monster does not walk (a sentry).
	PatrolA, PatrolB rl.Vector2
	// Territory is the sector the post belongs to. A player outside it is not
	// this monster's problem — PARA COMECAR uma perseguicao. Uma perseguicao ja
	// em curso nao volta a fazer essa pergunta.
	Territory rl.Rectangle
	// Radius is how far the monster notices a trespasser. So vale para NOTAR:
	// depois de notado, nao ha distancia que solte o alvo.
	Radius float32
	// Slack sobreviveu como campo porque os mapas o declaram, mas nao decide
	// mais nada: ele media ate onde uma perseguicao em curso podia sair do
	// setor, e perseguicao em curso nao sai mais de nada. Fica aqui para o mapa
	// nao precisar ser reescrito, e para o dia em que algum monstro precise
	// voltar a ter coleira.
	Slack float32
}

const (
	// guardArrive e a folga para considerar uma ponta da patrulha alcancada.
	//
	// Generosa de proposito: era aqui que o tremor nascia. Com tolerancia
	// pequena a separacao entre vizinhos empurrava o monstro para fora do alvo
	// e ele passava a vida corrigindo um passo.
	guardArrive float32 = 48
	// guardPause e quanto ele espera em cada ponta antes de voltar.
	guardPause float32 = 1.4
	// patrolBeat e o comprimento do trecho de patrulha, em px.
	patrolBeat float32 = 260
)

// Active reports whether this enemy guards anything.
func (g Guard) Active() bool {
	return g.Radius > 0 && (g.Post.X != 0 || g.Post.Y != 0)
}

// covers reports whether a point is inside the guarded sector, grown by margin.
func (g Guard) covers(p rl.Vector2, margin float32) bool {
	return p.X >= g.Territory.X-margin &&
		p.X < g.Territory.X+g.Territory.Width+margin &&
		p.Y >= g.Territory.Y-margin &&
		p.Y < g.Territory.Y+g.Territory.Height+margin
}

// PatrolSegment builds a beat around a post, fanned out by index.
//
// Cada monstro do esquadrao recebe um angulo diferente, entao eles patrulham em
// direcoes diferentes em vez de andarem em coluna. E o mesmo angulo aureo que
// espalha o nascimento, pela mesma razao: indices vizinhos nao podem cair na
// mesma direcao.
func PatrolSegment(post rl.Vector2, index int) (a, b rl.Vector2) {
	ang := float64(index) * 2.399
	dx := patrolBeat / 2 * float32(math.Cos(ang))
	dy := patrolBeat / 2 * float32(math.Sin(ang))
	return rl.NewVector2(post.X-dx, post.Y-dy), rl.NewVector2(post.X+dx, post.Y+dy)
}

// wants reports whether the monster should be chasing this point.
//
// UMA PERGUNTA SO, E ELA E DE AQUISICAO: o alvo esta no meu setor e ao alcance
// da minha vista? Quem ja esta perseguindo nao pergunta nada — `chasing` sai
// verdadeiro direto.
//
// Eram duas perguntas com dois regimes (setor + folga, raio x coleira), e o
// regime de MANUTENCAO era a saida gratuita que o jogador tinha: a 200 contra os
// 130 do orc, recuar alguns passos passava de qualquer teto e a guarnicao
// inteira soltava o alvo no mesmo quadro.
func (g Guard) wants(target rl.Vector2, chasing bool) bool {
	if chasing {
		return true
	}
	if !g.covers(target, 0) {
		return false
	}
	return rl.Vector2Distance(g.Post, target) <= g.Radius
}

// guardTarget decides who a garrison enemy is after this frame.
//
// Returns nil when it should go back to patrolling. Note what it does NOT do:
// it does not heal on the way back. Recuar e voltar e desgaste legitimo —
// decisao do Gui — e curar no retorno so existiria para punir uma tatica que o
// mapa nao proibe.
func (e *Enemy) guardTarget(players []PlayerState) *PlayerState {
	if !e.Guard.Active() {
		return e.FindNearestPlayer(players)
	}

	var best *PlayerState
	bestDist := float32(math.MaxFloat32)
	for i := range players {
		if players[i].IsDead {
			continue
		}
		p := rl.NewVector2(float32(players[i].X), float32(players[i].Y))
		if !e.Guard.wants(p, e.chasing) {
			continue
		}
		if d := rl.Vector2Distance(e.Position, p); d < bestDist {
			bestDist, best = d, &players[i]
		}
	}
	if best == nil {
		// A UNICA forma de uma perseguicao acabar: nao sobrou ninguem vivo para
		// perseguir. Nao ha prazo, nao ha coleira e nao ha linha de setor que a
		// encerre — quem levantou os olhos vai atras ate o fim do mapa.
		e.chasing = false
		return nil
	}
	e.chasing = true
	return best
}

// NoteAttack existe so para o chamador nao precisar saber que a perseguicao
// deixou de ter prazo.
//
// Ela renovava o tempo de uma perseguicao que se gastava sozinha. Sem prazo nao
// ha o que renovar, mas a chamada continua marcando o monstro como engajado —
// que e o que importa se algum dia um tipo novo voltar a ter coleira.
func (e *Enemy) NoteAttack() {
	if e.Guard.Active() {
		e.chasing = true
	}
}

// patrolStep walks the beat, and is what the monster does when nobody is
// trespassing.
//
// Devolve o ponto a perseguir. A pausa nas pontas nao e enfeite: sem ela o
// monstro inverte o sentido no mesmo quadro em que chega, e a virada fica
// identica ao tremor que este arquivo veio consertar.
func (e *Enemy) patrolStep(dt float32) (rl.Vector2, bool) {
	if !e.Guard.Active() {
		return rl.Vector2{}, false
	}
	// A patrulha e o que sobra para quem NAO tem alvo, e com a perseguicao
	// permanente isso passou a querer dizer uma coisa so: ninguem vivo por
	// perto que ele tenha notado. O monstro que perseguiu meio mapa e viu o
	// grupo inteiro cair patrulha ali mesmo, longe do posto — o `Guard.Post`
	// continua sendo o centro do trecho, entao ele volta caminhando.
	target := e.Guard.PatrolB
	if !e.patrolForward {
		target = e.Guard.PatrolA
	}
	if e.patrolWait > 0 {
		e.patrolWait -= dt
		return rl.Vector2{}, false
	}
	if rl.Vector2Distance(e.Position, target) <= guardArrive {
		e.patrolForward = !e.patrolForward
		e.patrolWait = guardPause
		return rl.Vector2{}, false
	}
	return target, true
}
