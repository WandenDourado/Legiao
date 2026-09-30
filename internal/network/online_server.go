package network

import (
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/WandenDourado/Legiao/internal/relay"
)

// OnlineConfigURL e o arquivo publico que diz onde esta o relay
// (online_config.go). E um endereco FIXO do Object Storage da Oracle - regiao,
// namespace e bucket nao mudam quando a VM e recriada - e o Terraform
// (infra/oci/legiao.tf) reescreve o conteudo com o IP atual da VM. Vazio
// desliga o jogo online: o menu nem mostra "CRIAR PARTIDA ONLINE".
//
// Com ele preenchido, nada extra roda em aparelho nenhum: "CRIAR PARTIDA
// ONLINE" faz o proprio app do host abrir uma conexao de SAIDA para o relay
// (OpenOnlineRoom) e o app do amigo entra pela lista de salas online, sem
// digitar IP. "CRIAR PARTIDA" continua so na rede local.
var OnlineConfigURL = "https://objectstorage.sa-saopaulo-1.oraclecloud.com/n/gr25fmxlpypi/b/legiao-public/o/online.json" // var: os testes apontam para um servidor local

// OnlineConfigured diz se este build tem jogo online.
func OnlineConfigured() bool { return OnlineConfigURL != "" }

// OnlineGameTag e a versao do protocolo DO JOGO para o relay. Suba o numero
// sempre que o que vai no fio mudar de um jeito que uma versao antiga nao
// entende (protocol.go): a lista de salas so mostra salas com a mesma tag, e
// um celular com a build velha nao entra numa partida da build nova.
const OnlineGameTag = "net2" // net2 (29/09/2026): fluxo comprimido; builds net1 nao aparecem na lista

// lastFrameNanos e o instante do ultimo quadro do laco do jogo (MarkFrame).
// E a prova de vida do host para o relay: app em segundo plano ou travado nao
// desenha quadro, deixa de responder PING, e a sala morre no servidor.
var lastFrameNanos atomic.Int64

// hostAliveWindow e quanto tempo sem quadro ainda conta como jogo rodando.
// Folgado para carga de mapa e engasgo; curto perto dos 45 s que o relay
// espera antes de derrubar a sala.
const hostAliveWindow = 20 * time.Second

// MarkFrame registra que o laco do jogo rodou mais um quadro. Chamado todo
// quadro por game/loop.go, inclusive nos quadros congelados por dialogo.
func MarkFrame() { lastFrameNanos.Store(time.Now().UnixNano()) }

func gameRunning() bool {
	return time.Since(time.Unix(0, lastFrameNanos.Load())) < hostAliveWindow
}

// onlineDialTimeout cobre o aperto de mao inteiro pelo relay (conectar, pedir
// a sala, esperar o host abrir a conexao de dados). Maior que dialTimeout,
// que foi medido para LAN; menor que connectCeiling (10 s).
const onlineDialTimeout = 6 * time.Second

// onlineAddrPrefix marca um ServerAddress que e uma sala do relay e nao um
// ip:porta. O endereco sobrevive a uma reconexao (reconnect.go) como qualquer
// outro, e o host volta ao relay com o MESMO codigo, entao a sala e achada.
const onlineAddrPrefix = "sala:"

// OnlineRoomAddr e o ServerAddress de uma sala online.
func OnlineRoomAddr(code string) string { return onlineAddrPrefix + code }

// IsOnlineRoom diz se addr aponta para uma sala do relay.
func IsOnlineRoom(addr string) bool { return strings.HasPrefix(addr, onlineAddrPrefix) }

// dialGame abre a conexao de jogo com o host, direta (LAN) ou pelo relay.
// E o unico ponto de discagem do cliente: ConnectClient e attemptReconnect
// passam por aqui.
func dialGame(addr string, timeout time.Duration) (net.Conn, error) {
	if code, ok := strings.CutPrefix(addr, onlineAddrPrefix); ok {
		server, err := relayAddress()
		if err != nil {
			return nil, err
		}
		c, err := relay.Join(server, code, onlineDialTimeout)
		if err != nil {
			forgetRelayAddress() // talvez o IP tenha mudado: reconsultar
		}
		return c, err
	}
	return net.DialTimeout("tcp", addr, timeout)
}
