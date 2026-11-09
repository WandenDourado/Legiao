package relay

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
)

// Join conecta a sala code pelo relay e devolve a conexao ja emparelhada com
// o host: dali em diante e o fluxo do jogo, como uma conexao TCP direta.
func Join(server, code string, timeout time.Duration) (net.Conn, error) {
	c, err := net.DialTimeout("tcp", server, timeout)
	if err != nil {
		return nil, err
	}
	c.SetDeadline(time.Now().Add(timeout))
	if err := writeLine(c, "JOIN "+Version+" "+code); err != nil {
		c.Close()
		return nil, err
	}
	line, err := readLine(c)
	if err != nil {
		c.Close()
		return nil, err
	}
	if line != "OK" {
		c.Close()
		return nil, refused(line)
	}
	c.SetDeadline(time.Time{})
	return c, nil
}

// List devolve as salas abertas no relay para a versao tag do jogo.
func List(server, tag string, timeout time.Duration) ([]Room, error) {
	c, err := net.DialTimeout("tcp", server, timeout)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(timeout))
	if err := writeLine(c, "LIST "+Version+" "+tag); err != nil {
		return nil, err
	}
	var rooms []Room
	for {
		line, err := readLine(c)
		if err != nil {
			return nil, err
		}
		if line == "END" {
			return rooms, nil
		}
		f := strings.SplitN(line, " ", 5)
		if f[0] != "R" || len(f) != 5 {
			return nil, errors.New("relay: resposta invalida: " + line)
		}
		players, err1 := strconv.Atoi(f[2])
		limit, err2 := strconv.Atoi(f[3])
		if err1 != nil || err2 != nil {
			return nil, errors.New("relay: resposta invalida: " + line)
		}
		rooms = append(rooms, Room{Code: f[1], Players: players, Max: limit, Name: f[4]})
	}
}

// Stats devolve a linha de saude do servidor (Server.Stats).
func Stats(server string, timeout time.Duration) (string, error) {
	c, err := net.DialTimeout("tcp", server, timeout)
	if err != nil {
		return "", err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(timeout))
	if err := writeLine(c, "STATS "+Version); err != nil {
		return "", err
	}
	return readLine(c)
}
