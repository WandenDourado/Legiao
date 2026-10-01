package network

import "testing"

// sinkPlayers impede o compilador de eliminar a alocacao como codigo morto:
// sem um uso observavel do resultado, `out := make(...); _ = out` pode ser
// otimizado embora nunca fosse em producao (o mapa devolvido e sempre lido).
var sinkPlayers map[string]PlayerState

func seedPlayers(n int) {
	RemotePlayersMutex.Lock()
	RemotePlayers = make(map[string]PlayerState, n)
	for i := 0; i < n; i++ {
		id := enemyBenchID(i)
		RemotePlayers[id] = PlayerState{PlayerID: id, X: i, Y: i, Health: 100, MaxHealth: 100}
	}
	RemotePlayersMutex.Unlock()
}

// BenchmarkGetAllPlayersReused mede o buffer reaproveitado (o que o pacote
// usa hoje) com 4 jogadores, a contagem maxima documentada em doc/network.md.
func BenchmarkGetAllPlayersReused(b *testing.B) {
	seedPlayers(4)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkPlayers = GetAllPlayers()
	}
}

// BenchmarkGetAllPlayersNaive mede o make(map[...]) por chamada de antes da
// otimizacao.
func BenchmarkGetAllPlayersNaive(b *testing.B) {
	seedPlayers(4)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RemotePlayersMutex.Lock()
		out := make(map[string]PlayerState, len(RemotePlayers))
		for k, v := range RemotePlayers {
			out[k] = v
		}
		RemotePlayersMutex.Unlock()
		sinkPlayers = out
	}
}
