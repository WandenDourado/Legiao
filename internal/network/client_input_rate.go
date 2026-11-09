package network

// Taxa de publicacao do INPUT do cliente.
//
// O laco do jogo chamava SendMessage(MsgInput) a cada quadro: 60 marshals
// JSON, 60 escritas com Flush (uma syscall cada) e 60 despertares do radio
// por segundo, contra os 20 Hz que o HOST ja publica (broadcast_rate.go). O
// jogador LOCAL nunca depende do proprio snapshot — ele e simulado no proprio
// quadro — entao baixar esta taxa so muda como os OUTROS o veem, e a
// interpolacao de 100 ms (interpolation.go) ja cobre o vao.
//
// Mesmo desenho do host: acumulador de tempo, nao "a cada N quadros" (um
// quadro nao tem duracao fixa). A unica diferenca e a transicao de
// movimento: parar de andar ou mudar de direcao bruscamente publica na hora,
// porque a interpolacao so cobre uma trajetoria CONTINUA — segurar esses dois
// casos ate o proximo tique faria o boneco remoto "grudar" no ultimo ponto
// por ate um intervalo inteiro.

// InputHz e quantos snapshots de input o cliente publica por segundo fora de
// uma transicao de movimento. Mesma ordem de grandeza do SnapshotHz do host:
// nao ha ganho em publicar input mais rapido do que o host redistribui.
const InputHz = 20

const inputInterval float32 = 1.0 / InputHz

// movementBucket classifica uma componente de velocidade em parado/positivo/
// negativo. Comparar buckets em vez do float bruto evita disparar uma
// publicacao extra a cada oscilacao de ponto flutuante enquanto a velocidade
// so muda de magnitude, nao de sentido.
func movementBucket(v float32) int {
	switch {
	case v > 0.5:
		return 1
	case v < -0.5:
		return -1
	default:
		return 0
	}
}

// clientInputState guarda o ritmo de publicacao do input do cliente. E
// variavel de pacote, e nao campo do Client, pelo mesmo motivo de
// clientIdentity: so existe um cliente por processo e quem publica e sempre o
// laco do jogo, na mesma goroutine.
//
// due() e um metodo em vez de uma funcao solta sobre a variavel de pacote
// para que o teste possa exercitar a cadencia com o proprio estado, sem
// depender da variavel global nem de uma ordem de execucao entre testes.
type clientInputState struct {
	timer   float32
	hasLast bool
	lastVX  int
	lastVY  int
}

var inputState clientInputState

// due avanca o relogio de publicacao e diz se ESTE quadro deve publicar: por
// cadencia normal, ou porque o movimento acabou de mudar de jeito que a
// interpolacao nao absorve sozinha.
func (s *clientInputState) due(dt, velX, velY float32) bool {
	s.timer += dt
	due := false
	if s.timer >= inputInterval {
		s.timer -= inputInterval
		// Um quadro muito longo (carga de mapa, travada) nao deve gerar uma
		// rajada de publicacoes atrasadas de uma vez.
		if s.timer > inputInterval {
			s.timer = 0
		}
		due = true
	}

	vx, vy := movementBucket(velX), movementBucket(velY)
	transitioned := s.hasLast && (vx != s.lastVX || vy != s.lastVY)
	s.hasLast = true
	s.lastVX, s.lastVY = vx, vy

	return due || transitioned
}

// SendPlayerInput publica o input do jogador local se este quadro for devido
// (ver clientInputState.due). Chamado uma vez por quadro pelo laco do jogo; a
// decisao de publicar ou nao mora inteiramente aqui, nao no chamador.
func SendPlayerInput(payload InputPayload, dt float32) {
	if !inputState.due(dt, payload.VelX, payload.VelY) {
		return
	}
	SendMessage(Message{
		Type:    MsgInput,
		Payload: MustMarshal(payload),
	})
}
