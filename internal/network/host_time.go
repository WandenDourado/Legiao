package network

import "time"

// hostEpoch e o zero do relogio que o host carimba nos snapshots (HostT).
// Monotonico e por processo: so a DIFERENCA entre dois carimbos importa para
// o cliente (interpolation_clock.go), nunca o valor absoluto. Comeca em 1 ms
// porque 0 significa "host antigo, sem relogio".
var hostEpoch = time.Now().Add(-time.Millisecond)

func hostNowMs() int64 { return time.Since(hostEpoch).Milliseconds() }
