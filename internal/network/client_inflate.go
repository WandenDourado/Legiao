package network

import (
	"bufio"
	"compress/flate"
)

// startInflate troca o leitor desta conexao para deflate, a partir do byte
// seguinte ao marcador MsgCompress (host_peer_writer.go).
//
// flate.NewReader sobre o bufio.Reader antigo nao le alem do que precisa (o
// bufio e um io.ByteReader), entao nenhum byte comprimido ja bufferizado se
// perde na troca. O leitor novo tambem vira c.reader, que e o que a drenagem
// de snapshots (client_backlog.go) consulta.
func (c *Client) startInflate(r *bufio.Reader) *bufio.Reader {
	nr := bufio.NewReader(flate.NewReader(r))
	c.mu.Lock()
	c.reader = nr
	c.mu.Unlock()
	return nr
}
