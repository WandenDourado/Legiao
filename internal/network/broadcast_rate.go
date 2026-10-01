package network

// A que taxa o host PUBLICA o estado — que nao e a taxa em que ele simula.
//
// Os dois eram a mesma coisa: UpdateSimulation transmitia tudo a cada quadro,
// entao 60 snapshots por segundo. Com 4 jogadores e 83 inimigos isso custa
// ~763 KB/s por cliente, e a maior parte e repeticao: um orc parado no proprio
// posto era retransmitido inteiro sessenta vezes por segundo.
//
// A simulacao CONTINUA a 60 Hz. So a publicacao desce. Quem recebe cobre o
// vao interpolando (interpolation.go), e os dois so funcionam juntos: baixar a
// taxa sem interpolar faria a posicao remota andar aos saltos.

// SnapshotHz e quantos snapshots de estado o host publica por segundo.
//
// 20 e o padrao da industria para jogo com host autoritativo, e o numero tem
// uma restricao de baixo: ele precisa ser alto o bastante para o atraso de
// interpolacao (interpDelay, 100 ms) segurar DOIS intervalos, senao um pacote
// perdido esvazia o buffer e a interpolacao vira extrapolacao. A 20 Hz o
// intervalo e 50 ms e cabem dois.
//
// Nao confundir com a taxa de simulacao: cooldown, dano e morte continuam
// sendo decididos a cada quadro pelo host.
const SnapshotHz = 20

const snapshotInterval float32 = 1.0 / SnapshotHz

// dueForSnapshot avanca o relogio de publicacao e diz se este quadro publica.
//
// Acumulador, e nao "a cada N quadros": o quadro nao tem duracao fixa, e
// contar quadros faria a taxa de rede subir e descer junto com o fps — o
// oposto do que se quer, porque e justamente quando o quadro engasga que a
// rede nao pode engasgar junto.
func (h *Host) dueForSnapshot(dt float32) bool {
	h.snapshotTimer += dt
	if h.snapshotTimer < snapshotInterval {
		return false
	}
	// Subtrai em vez de zerar: zerar perderia o excedente e a taxa media
	// ficaria abaixo da pedida.
	h.snapshotTimer -= snapshotInterval
	// Um quadro muito longo (carga de mapa, travada) nao deve gerar uma rajada
	// de snapshots atrasados de uma vez.
	if h.snapshotTimer > snapshotInterval {
		h.snapshotTimer = 0
	}
	return true
}
