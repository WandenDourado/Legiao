package collision

import rl "github.com/gen2brain/raylib-go/raylib"

// Resolve returns where an entity whose footprint is width x height ends up
// when it tries to move from `from` by `delta`.
//
// The rule is axis-separated sliding: the full move is taken when it is clear,
// otherwise whichever single axis is still free is taken alone, so an entity
// walking diagonally into a wall scrapes along it instead of stopping dead.
// This is exactly the behaviour the player already had; centralising it here
// is what makes monsters obey the same map.
//
// Returns `from` unchanged when both axes are blocked. Callers that need the
// entity to get around the obstacle instead of stalling use ResolveDetour.
func Resolve(from, delta rl.Vector2, width, height float32, s Solid) rl.Vector2 {
	full := rl.Vector2Add(from, delta)
	if s == nil || (delta.X == 0 && delta.Y == 0) {
		return full
	}

	// Already standing inside a solid — spawned on top of one, shoved in by a
	// crowd, or the map changed underneath. Refusing to move would trap the
	// entity forever, so it moves freely until it is out.
	if blocked(s, from, width, height) {
		return full
	}

	if !blocked(s, full, width, height) {
		return full
	}

	if delta.X != 0 {
		if slid := rl.NewVector2(full.X, from.Y); !blocked(s, slid, width, height) {
			return slid
		}
	}
	if delta.Y != 0 {
		if slid := rl.NewVector2(from.X, full.Y); !blocked(s, slid, width, height) {
			return slid
		}
	}
	return from
}

// ResolveDetour is Resolve plus a way around obstacles the slide cannot solve.
//
// Sliding only helps while some component of the movement still points
// somewhere useful. An entity walking straight into a fence has nothing to
// slide along and stops dead; worse, one chasing a target that sits square
// behind the fence slides a hair back toward the target's line and then blocks
// again, grinding against the same spot forever. Both cases are detected as
// "no real progress" and answered by stepping sideways along the obstacle's
// face instead.
//
// `committed` is the world direction the caller stored from the previous
// frame. Passing it back makes the entity commit to one way around the obstacle
// and keep going that way until the direct path opens, instead of re-deciding
// every frame. The commitment is what turns a detour into an actual way around:
// without it the entity gets a step sideways, immediately finds a slide that
// inches back toward its target, blocks again, and settles into a hover next to
// the obstacle. Measured against a 256 px house with the player right behind
// it: re-deciding every frame never gets around at all, committing clears the
// house in about 8 seconds.
//
// A COMMITMENT TO UMA DIRECAO DO MUNDO, E NAO A UM SENTIDO DE ROTACAO.
//
// Isto guardava um sinal (+1/-1) aplicado a `perp = (-delta.Y, delta.X)`, ou
// seja, uma rotacao de 90 graus do RUMO ATUAL. O rumo gira enquanto a criatura
// caminha, entao o mesmo sinal muda de direcao no mundo no meio do contorno — e
// era isso que travava monstro em cerca e barricada. Medido contra uma
// barricada de ponta a ponta com um vao: o monstro deslizava certinho ate 60 px
// da abertura, a componente lateral do rumo virava quase zero, o contorno
// comecava, o sinal fixo `+1` o mandava para o lado OPOSTO, e ele caminhava 27
// segundos para longe do vao que ja tinha alcancado.
//
// Agora a direcao e um vetor no mundo, escolhido por `openingSide`: a criatura
// olha ao longo da face do obstaculo para os dois lados e vai para o lado onde a
// abertura aparece primeiro. E o que uma criatura com olhos faria, e custa uma
// varredura so — no quadro em que o contorno comeca, nao a cada quadro.
//
// O vetor devolvido e o que o chamador deve guardar: o vetor nulo quando o
// caminho direto abriu.
func ResolveDetour(from, delta rl.Vector2, width, height float32, s Solid,
	committed rl.Vector2) (rl.Vector2, rl.Vector2) {
	var none rl.Vector2
	full := rl.Vector2Add(from, delta)
	next := Resolve(from, delta, width, height, s)
	if next == full {
		return next, none // path clear: whatever detour was in progress is over
	}
	if committed == none && progressed(rl.Vector2Subtract(next, from), delta) {
		return next, committed // plain slide along the obstacle, still making ground
	}

	step := rl.Vector2Length(delta)
	walk := func(dir rl.Vector2) (rl.Vector2, bool) {
		if dir == none {
			return from, false
		}
		moved := Resolve(from, rl.Vector2Scale(dir, step), width, height, s)
		return moved, moved != from
	}

	// Ja contornando: seguir. A direcao guardada so e abandonada quando ela
	// propria fica bloqueada — um canto, ou o fim da face.
	if moved, ok := walk(committed); ok {
		return moved, committed
	}

	axis := faceAxis(from, delta, width, height, s)
	side := openingSide(from, delta, axis, width, height, s)
	if side == 0 {
		// Nenhuma abertura ao alcance da vista: vai para o lado em que o alvo
		// esta. E o mesmo palpite que o deslize ja estava fazendo, e ele acerta
		// sempre que a abertura fica entre a criatura e o alvo.
		side = signOf(axis.X*delta.X + axis.Y*delta.Y)
	}
	dir := rl.Vector2Scale(axis, side)
	if moved, ok := walk(dir); ok {
		return moved, dir
	}
	// A face pode acabar antes da abertura (um canto, o fim da barricada). Nesse
	// caso vale tentar o outro lado antes de desistir.
	back := rl.Vector2Scale(dir, -1)
	if moved, ok := walk(back); ok {
		return moved, back
	}
	return next, none
}

// faceAxis is the world axis the obstacle's face runs along, as seen from an
// entity pressed against it.
//
// Derivado de QUAL EIXO FICOU BLOQUEADO, e nao do rumo: numa barricada
// horizontal a face e sempre o eixo X, esteja a criatura vindo de frente ou de
// esguelha. Era essa estabilidade que faltava — tirar a tangente do rumo faz a
// face "girar" junto com quem anda por ela.
//
// Com os dois eixos bloqueados (um canto), a face e o eixo MENOS dominante do
// rumo: quem vai para o norte contorna pelos lados.
func faceAxis(from, delta rl.Vector2, width, height float32, s Solid) rl.Vector2 {
	xFree := !blocked(s, rl.NewVector2(from.X+delta.X, from.Y), width, height)
	yFree := !blocked(s, rl.NewVector2(from.X, from.Y+delta.Y), width, height)
	switch {
	case xFree && !yFree:
		return rl.NewVector2(1, 0)
	case yFree && !xFree:
		return rl.NewVector2(0, 1)
	}
	if absOf(delta.X) < absOf(delta.Y) {
		return rl.NewVector2(1, 0)
	}
	return rl.NewVector2(0, 1)
}

// detourProbeSteps is how far along the obstacle's face an entity looks for a
// way through before committing, in footprint-sized steps.
//
// 96 passos de ~40 px sao ~3800 px, que e mais da metade da largura do mapa 3 —
// o suficiente para achar o vao de qualquer barricada a partir de qualquer
// posto. O custo e ate 192 consultas ao mapa, e ele so acontece no quadro em que
// um contorno COMECA: enquanto a criatura esta contornando, a direcao guardada
// e seguida sem varredura nenhuma.
const detourProbeSteps = 96

// openingSide looks along the obstacle's face in both directions at once and
// answers which way the nearest way through is: +1, -1, or 0 for "no opening in
// sight".
//
// A varredura alterna os dois lados (+1, -1, +2, -2, ...) de proposito: ela
// devolve a abertura MAIS PROXIMA, e nao a primeira de um lado escolhido a dedo.
//
// "Abertura" e uma pergunta de duas partes: cabe estar ali, E dali o passo na
// direcao que eu queria ir esta livre. A segunda metade tem de ser UM PASSO, e
// nao um salto de vários — sondando fundo demais a consulta pula por cima do
// obstaculo e responde "livre" em cima da propria parede, que foi como uma
// versao anterior disto reprovou em todos os cenarios de uma vez.
func openingSide(from, delta, axis rl.Vector2, width, height float32, s Solid) float32 {
	length := rl.Vector2Length(delta)
	if length <= 0 {
		return 0
	}
	stride := width
	if height > stride {
		stride = height
	}
	ahead := rl.Vector2Scale(delta, stride/length)
	for i := 1; i <= detourProbeSteps; i++ {
		for _, side := range [2]float32{1, -1} {
			at := rl.Vector2Add(from, rl.Vector2Scale(axis, side*float32(i)*stride))
			if blocked(s, at, width, height) {
				continue
			}
			if !blocked(s, rl.Vector2Add(at, ahead), width, height) {
				return side
			}
		}
	}
	return 0
}

func absOf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// signOf never returns 0: um contorno precisa de um lado, e "nenhum" faria a
// criatura parar de novo. Empate vai para o positivo.
func signOf(v float32) float32 {
	if v < 0 {
		return -1
	}
	return 1
}

// detourProgressShare is how much of the intended step a deflected move must
// still carry in the intended direction to count as progress, and so to be
// taken instead of starting a detour. A diagonal slide along a wall keeps
// about half of it; an entity pressed flat against a wall keeps almost none —
// that near-zero slide is the one this threshold exists to catch. Measured on
// a monster chasing a player straight through a fence: it scores under 0.01.
const detourProgressShare = 0.15

// progressed reports whether the movement actually applied still advances the
// entity along the direction it wanted to go.
func progressed(moved, delta rl.Vector2) bool {
	lengthSq := delta.X*delta.X + delta.Y*delta.Y
	if lengthSq <= 0 {
		return false
	}
	return moved.X*delta.X+moved.Y*delta.Y >= detourProgressShare*lengthSq
}

// Progressed is progressed, exported for callers that need the same "did
// this step actually get anywhere" judgment outside a single ResolveDetour
// call — the monster non-progress gate that decides when to consult a
// navigation mesh instead of just investing straight ahead
// (doc/plan_navegacao_bots_monstros.md §4) is the reason this exists.
func Progressed(moved, delta rl.Vector2) bool {
	return progressed(moved, delta)
}

// `detourSides` vivia aqui e ordenava os dois lados a partir de um sinal, com
// `+1` como preferencia cega. Ela saiu junto com o sinal: quem escolhe o lado
// agora e `openingSide`, olhando o mapa em vez de chutar.
