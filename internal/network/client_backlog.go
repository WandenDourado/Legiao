package network

// Drenagem do backlog de snapshot no cliente.
//
// O socket e newline-delimited JSON (host.go: ClientConn.send escreve dado +
// '\n' + Flush em toda mensagem). Enquanto o app fica em background (tela
// bloqueada), o SO para de entregar leituras ao processo mas o host continua
// publicando a SnapshotHz; ao voltar, bufio.Reader ja tem uma fila inteira de
// mensagens no buffer TCP, prontas sem nova leitura de rede. Decodificar e
// aplicar cada uma delas — inclusive o estado que nunca chegou a aparecer na
// tela — e o pico de CPU que da o segundo tranco do "lag que aumenta com o
// tempo de partida" (doc/performance.md).
//
// So o ULTIMO snapshot de cada tipo importa: um estado de jogador ou inimigo
// e substituido pelo proximo inteiro, entao aplicar um que ja tem sucessor no
// buffer e trabalho jogado fora. EVENTOS (combate, dialogo, viagem, reset,
// magias) sao o oposto — cada um acontece uma vez e nunca pode ser descartado.

import (
	"encoding/json"
	"log"
)

// snapshotBacklogLimit e um teto de seguranca por chamada: nada no jogo
// produz mais que umas centenas de snapshots enfileirados (a rede publica a
// SnapshotHz), mas sem um teto um peer malformado que nunca fecha a linha
// prenderia o laco de leitura.
const snapshotBacklogLimit = 512

// isSnapshotType reporta se mensagens deste tipo sao substituidas pela
// proxima da mesma familia. Ver a nota de topo do arquivo.
func isSnapshotType(t MessageType) bool {
	switch t {
	case MsgStateUpdate, MsgEnemyUpdate, MsgProjectileUpdate, MsgCooldown:
		return true
	default:
		return false
	}
}

// parseMessage decodifica uma linha newline-delimited em uma Message. ok e
// falso numa linha malformada, que o chamador pula em vez de tratar como erro
// de conexao — uma linha corrompida nao deve derrubar o socket sozinha.
func parseMessage(line []byte) (Message, bool) {
	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		log.Printf("client failed to unmarshal message: %v", err)
		return Message{}, false
	}
	return msg, true
}

// drainStaleSnapshots devolve o snapshot mais RECENTE do mesmo tipo que ja
// esta no buffer de leitura, sem bloquear esperando rede nova. Se msg nao e
// um tipo de snapshot, devolve msg sem tocar no buffer.
//
// Qualquer mensagem de tipo DIFERENTE encontrada durante a drenagem e um
// EVENTO (ou um snapshot de outra familia) e nunca e descartada: e tratada na
// hora, aplicando a mesma regra recursivamente ao que vier depois dela — o
// que tambem drena o backlog dela, se houver.
func (c *Client) drainStaleSnapshots(msg Message) Message {
	if !isSnapshotType(msg.Type) {
		return msg
	}
	for i := 0; i < snapshotBacklogLimit && c.reader.Buffered() > 0; i++ {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			// O proximo ReadBytes do readLoop vai bater no mesmo erro e
			// encerrar a conexao; devolver o que ja temos deixa este quadro
			// desenhar com o ultimo estado bom.
			return msg
		}
		next, ok := parseMessage(line)
		if !ok {
			continue
		}
		if next.Type != msg.Type {
			c.handleMessage(c.drainStaleSnapshots(next))
			continue
		}
		msg = next
	}
	return msg
}
