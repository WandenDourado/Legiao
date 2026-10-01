// Package relay liga jogadores do Legiao pela internet sem porta aberta em
// roteador nenhum.
//
// Todo mundo conecta de SAIDA num servidor publico (cmd/relay, na VM Always
// Free da Oracle): quem hospeda mantem uma conexao de controle aberta, e cada
// amigo que entra faz o servidor pedir ao host uma conexao de dados nova. O
// servidor so emparelha as duas pontas e copia bytes; a partida continua
// sendo simulada no aparelho de quem hospeda (network.Host).
//
// O protocolo e texto, uma linha por comando, e so existe no aperto de mao.
// Depois dele a conexao e o fluxo do jogo, byte a byte:
//
//	host -> relay  HOST LEGIAO2 <codigo|-> <segredo> <tag> <nome>  (controle)
//	relay -> host  ROOM <codigo> | ERR <motivo>
//	relay -> host  OPEN <id>  (alguem quer entrar)  |  PING
//	host -> relay  PONG  (so enquanto o jogo do host esta rodando de verdade)
//	host -> relay  DATA <id>                         (conexao de dados nova)
//	amigo -> relay JOIN LEGIAO2 <codigo>
//	relay -> amigo OK | ERR <motivo>
//	amigo -> relay LIST LEGIAO2 <tag>
//	relay -> amigo R <codigo> <jogadores> <max> <nome> ... END
//	qualquer -> relay STATS LEGIAO2  ->  uma linha de contadores
//
// <tag> e a versao do protocolo DO JOGO (network.OnlineGameTag): a lista so
// mostra salas com a mesma tag, para builds diferentes nao se misturarem.
package relay

import (
	"crypto/rand"
	"errors"
	"io"
	"strings"
	"unicode"
)

// Version muda quando o aperto de mao muda. Cliente e servidor de versoes
// diferentes se recusam na hora, em vez de corromper o fluxo do jogo.
const Version = "LEGIAO2"

const (
	maxLine    = 256
	maxNameLen = 24
	codeLen    = 5
	// Sem 0/O, 1/I/L: o codigo aparece na tela e pode ser ditado.
	codeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

// Room e uma sala anunciada pelo servidor. Players conta so quem entrou pelo
// relay (o host nao entra na conta); Max e o teto do servidor para isso.
type Room struct {
	Code    string
	Name    string
	Players int
	Max     int
}

var errLineTooLong = errors.New("relay: linha longa demais")

// Motivos de recusa que o servidor manda em "ERR <motivo>". O jogo os
// reconhece (network.OnlineErrorText) para dizer ao jogador o que houve.
const (
	ReasonRoomNotFound = "sala nao encontrada"
	ReasonRoomFull     = "sala cheia"
	ReasonServerFull   = "servidor cheio, tente mais tarde"
	ReasonMonthlyLimit = "limite mensal do servidor atingido"
	ReasonHostTimeout  = "anfitriao nao respondeu"
	ReasonBadRegister  = "registro invalido"
	ReasonBadCommand   = "versao ou comando invalido"
)

// RefusedError e uma recusa explicita do servidor (ele respondeu, e disse
// nao), diferente de uma falha de rede (ele nem respondeu).
type RefusedError struct{ Reason string }

func (e *RefusedError) Error() string { return "relay: " + e.Reason }

func refused(line string) error {
	return &RefusedError{Reason: strings.TrimPrefix(line, "ERR ")}
}

// readLine le UMA linha, um byte por vez. Ler com buffer poderia engolir os
// primeiros bytes do fluxo do jogo que vem logo depois do aperto de mao.
func readLine(r io.Reader) (string, error) {
	var b [1]byte
	buf := make([]byte, 0, 64)
	for {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return "", err
		}
		if b[0] == '\n' {
			return strings.TrimRight(string(buf), "\r"), nil
		}
		if len(buf) >= maxLine {
			return "", errLineTooLong
		}
		buf = append(buf, b[0])
	}
}

func writeLine(w io.Writer, s string) error {
	_, err := io.WriteString(w, s+"\n")
	return err
}

// randomString sorteia n caracteres de alphabet.
func randomString(n int, alphabet string) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func newCode() string   { return randomString(codeLen, codeAlphabet) }
func newID() string     { return randomString(16, "abcdefghijklmnopqrstuvwxyz0123456789") }
func NewSecret() string { return randomString(24, "abcdefghijklmnopqrstuvwxyz0123456789") }

// CleanName deixa o nome da sala seguro para uma linha do protocolo e para a
// fonte do jogo: sem controle, sem quebra, no maximo maxNameLen runas.
func CleanName(name string) string {
	out := make([]rune, 0, maxNameLen)
	for _, r := range strings.TrimSpace(name) {
		if len(out) == maxNameLen {
			break
		}
		if unicode.IsControl(r) {
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return "Sala"
	}
	return string(out)
}

// validTag aceita so letras, digitos, '.', '-' e '_', ate 16 caracteres.
func validTag(t string) bool {
	if t == "" || len(t) > 16 {
		return false
	}
	for _, r := range t {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_'
		if !ok {
			return false
		}
	}
	return true
}

func validCode(c string) bool {
	if len(c) != codeLen {
		return false
	}
	for _, r := range c {
		if !strings.ContainsRune(codeAlphabet, r) {
			return false
		}
	}
	return true
}
