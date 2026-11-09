package network

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/WandenDourado/Legiao/internal/relay"
)

func TestOnlineErrorTextNeverShowsRawErrors(t *testing.T) {
	cases := map[error]string{
		&relay.RefusedError{Reason: relay.ReasonRoomFull}:     "cheia",
		&relay.RefusedError{Reason: relay.ReasonRoomNotFound}: "encerrada",
		&relay.RefusedError{Reason: relay.ReasonServerFull}:   "lotado",
		&relay.RefusedError{Reason: relay.ReasonMonthlyLimit}: "limite do mes",
		&relay.RefusedError{Reason: relay.ReasonBadCommand}:   "Atualize o jogo",
		errOnlineDisabled:                          "desativado",
		errors.New("dial tcp: connection refused"): "Verifique sua internet",
	}
	for err, want := range cases {
		if got := OnlineErrorText(err); !strings.Contains(got, want) {
			t.Errorf("%v -> %q, quer algo com %q", err, got, want)
		}
	}
}

func TestTimeoutReadsAsSlowServer(t *testing.T) {
	_, err := net.DialTimeout("tcp", "192.0.2.1:9000", 50*time.Millisecond) // TEST-NET: nunca responde
	if err == nil {
		t.Skip("rede de teste respondeu")
	}
	if got := OnlineErrorText(err); !strings.Contains(got, "internet") {
		t.Fatalf("timeout -> %q", got)
	}
}

func TestReconnectHintExplainsHostLeft(t *testing.T) {
	prev := ServerAddress
	defer func() { ServerAddress = prev; setReconnectErr(nil) }()

	ServerAddress = "192.168.0.10:9000" // LAN: sem dica
	setReconnectErr(&relay.RefusedError{Reason: relay.ReasonRoomNotFound})
	if h := ReconnectHint(); h != "" {
		t.Fatalf("LAN mostrou dica %q", h)
	}
	ServerAddress = OnlineRoomAddr("K7M2Q")
	if h := ReconnectHint(); !strings.Contains(h, "anfitriao saiu") {
		t.Fatalf("dica = %q", h)
	}
}
