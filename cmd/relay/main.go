// Comando relay e o servidor publico do jogo online (internal/relay). Roda na
// VM Always Free da Oracle (infra/oci/legiao.tf) como servico systemd.
//
//	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o legiao-relay ./cmd/relay
//
// Saude, de qualquer maquina com bash:
//
//	exec 3<>/dev/tcp/<ip>/9000; echo "STATS LEGIAO2" >&3; cat <&3
package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/WandenDourado/Legiao/internal/relay"
)

func main() {
	addr := flag.String("listen", ":9000", "endereco TCP de escuta")
	flag.Parse()
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("[relay] %v", err)
	}
	s := relay.NewServer()
	log.Printf("[relay] protocolo %s escutando em %s", relay.Version, *addr)
	go func() {
		for range time.Tick(10 * time.Minute) {
			log.Printf("[relay] %s", s.Stats())
		}
	}()
	log.Fatal(s.Serve(ln))
}
