package network

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/WandenDourado/Legiao/internal/settings"
)

// O endereco do relay NAO fica gravado no app. O app le um arquivo publico de
// configuracao (OnlineConfigURL, um objeto no Object Storage da Oracle que o
// Terraform escreve a partir do IP da VM) e guarda a ultima copia boa em disco.
// Se a VM for recriada e o IP mudar, basta o `terraform apply` reescrever o
// arquivo: nenhum app publicado quebra. O mesmo arquivo e o interruptor de
// emergencia ("enabled": false desliga o online sem atualizar o app).
//
// Nao e segredo, por isso nao vai num cofre: um cofre pediria credencial
// dentro do app, e credencial num app publicado deixa de ser segredo. O que
// protege o servidor sao os limites do relay (relay.NewServer).

// onlineConfig e o formato de infra/oci/legiao.tf (objeto online.json).
type onlineConfig struct {
	Relay   string `json:"relay"`
	Enabled bool   `json:"enabled"`
}

const (
	onlineConfigTTL     = 10 * time.Minute
	onlineConfigTimeout = 5 * time.Second
	onlineConfigMax     = 4 << 10 // o arquivo tem ~60 bytes; 4 KiB e folga
)

var errOnlineDisabled = errors.New("jogo online desativado no servidor")

var (
	cfgMu   sync.Mutex
	cfgAddr string    // ultimo endereco valido, mesmo vencido
	cfgAt   time.Time // quando cfgAddr foi confirmado pela rede
	cfgOff  bool      // o arquivo disse "enabled": false

	// onlineCachePath e variavel para os testes nao tocarem a pasta real.
	onlineCachePath = func() string {
		if dir := settings.DataDir(); dir != "" {
			return filepath.Join(dir, "online.json")
		}
		return ""
	}
)

// relayAddress devolve o ip:porta do relay. Consulta a rede no maximo a cada
// onlineConfigTTL; sem rede, vale a ultima copia (em memoria ou em disco).
func relayAddress() (string, error) {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	if cfgAddr != "" && time.Since(cfgAt) < onlineConfigTTL {
		return liveAddrLocked()
	}
	cfg, err := fetchOnlineConfig(OnlineConfigURL)
	if err != nil {
		log.Printf("[Online] configuracao indisponivel (%v); usando a ultima conhecida", err)
		if cfgAddr == "" {
			cached, cerr := readOnlineCache()
			if cerr != nil {
				return "", err
			}
			cfgAddr, cfgOff = cached.Relay, !cached.Enabled
		}
		return liveAddrLocked()
	}
	cfgAddr, cfgOff, cfgAt = cfg.Relay, !cfg.Enabled, time.Now()
	writeOnlineCache(cfg)
	return liveAddrLocked()
}

func liveAddrLocked() (string, error) {
	if cfgOff {
		return "", errOnlineDisabled
	}
	return cfgAddr, nil
}

// forgetRelayAddress forca a proxima chamada a consultar a rede. Chamado
// quando o relay nao responde: talvez o IP tenha mudado. O endereco antigo
// fica como reserva se a consulta falhar.
func forgetRelayAddress() {
	cfgMu.Lock()
	cfgAt = time.Time{}
	cfgMu.Unlock()
}

func fetchOnlineConfig(url string) (onlineConfig, error) {
	var cfg onlineConfig
	resp, err := (&http.Client{Timeout: onlineConfigTimeout}).Get(url)
	if err != nil {
		return cfg, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return cfg, fmt.Errorf("configuracao online: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, onlineConfigMax))
	if err != nil {
		return cfg, err
	}
	return parseOnlineConfig(body)
}

func parseOnlineConfig(b []byte) (onlineConfig, error) {
	var cfg onlineConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("configuracao online invalida: %w", err)
	}
	if _, _, err := net.SplitHostPort(cfg.Relay); err != nil {
		return cfg, fmt.Errorf("configuracao online: relay %q invalido", cfg.Relay)
	}
	return cfg, nil
}

func readOnlineCache() (onlineConfig, error) {
	p := onlineCachePath()
	if p == "" {
		return onlineConfig{}, errors.New("sem pasta de cache")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return onlineConfig{}, err
	}
	return parseOnlineConfig(b)
}

func writeOnlineCache(cfg onlineConfig) {
	p := onlineCachePath()
	if p == "" {
		return
	}
	b, _ := json.Marshal(cfg)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
		if err := os.WriteFile(p, b, 0o644); err != nil {
			log.Printf("[Online] nao consegui gravar o cache: %v", err)
		}
	}
}
