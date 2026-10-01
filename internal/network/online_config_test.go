package network

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// useTestRelay aponta o jogo para um arquivo de configuracao local que manda
// para relayAddr, com o cache em disco numa pasta temporaria.
func useTestRelay(t *testing.T, relayAddr string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"relay":"` + relayAddr + `","enabled":true}`))
	}))
	pointConfigAt(t, srv.URL)
	t.Cleanup(srv.Close)
	return srv
}

func pointConfigAt(t *testing.T, url string) {
	t.Helper()
	prevURL, prevCache := OnlineConfigURL, onlineCachePath
	dir := t.TempDir()
	OnlineConfigURL = url
	onlineCachePath = func() string { return filepath.Join(dir, "online.json") }
	resetOnlineConfig()
	t.Cleanup(func() {
		OnlineConfigURL, onlineCachePath = prevURL, prevCache
		resetOnlineConfig()
	})
}

func resetOnlineConfig() {
	cfgMu.Lock()
	cfgAddr, cfgAt, cfgOff = "", time.Time{}, false
	cfgMu.Unlock()
}

func TestRelayAddressComesFromConfigAndIsCached(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte(`{"relay":"203.0.113.7:9000","enabled":true}`))
	}))
	defer srv.Close()
	pointConfigAt(t, srv.URL)

	for i := 0; i < 3; i++ {
		addr, err := relayAddress()
		if err != nil || addr != "203.0.113.7:9000" {
			t.Fatalf("relayAddress = %q, %v", addr, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("consultou a rede %d vezes em vez de 1", hits.Load())
	}
}

// Sem rede (servidor de configuracao fora), vale a ultima copia gravada em
// disco - inclusive num processo novo, que ainda nao tem nada em memoria.
func TestRelayAddressFallsBackToDiskCache(t *testing.T) {
	var up atomic.Bool
	up.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up.Load() {
			http.Error(w, "fora", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"relay":"203.0.113.8:9000","enabled":true}`))
	}))
	defer srv.Close()
	pointConfigAt(t, srv.URL)
	if _, err := relayAddress(); err != nil {
		t.Fatal(err)
	}
	up.Store(false)
	resetOnlineConfig() // "abre o app de novo"
	addr, err := relayAddress()
	if err != nil || addr != "203.0.113.8:9000" {
		t.Fatalf("sem rede: relayAddress = %q, %v", addr, err)
	}
}

func TestOnlineKillSwitch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"relay":"203.0.113.9:9000","enabled":false}`))
	}))
	defer srv.Close()
	pointConfigAt(t, srv.URL)
	if _, err := relayAddress(); !errors.Is(err, errOnlineDisabled) {
		t.Fatalf("online desligado no arquivo, mas relayAddress = %v", err)
	}
	if _, state := listOnlineRooms(); state != OnlineDisabled {
		t.Fatalf("estado = %v, quer OnlineDisabled", state)
	}
}

func TestBadConfigIsRejected(t *testing.T) {
	for _, body := range []string{`nao e json`, `{"relay":"sem-porta","enabled":true}`} {
		if _, err := parseOnlineConfig([]byte(body)); err == nil {
			t.Fatalf("aceitou configuracao invalida: %s", body)
		}
	}
}
