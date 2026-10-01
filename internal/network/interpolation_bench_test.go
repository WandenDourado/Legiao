package network

import "testing"

// sinkEnemies impede o compilador de eliminar a alocacao como codigo morto
// (ver o mesmo comentario em globals_bench_test.go).
var sinkEnemies map[string]EnemyState

// Estas medidas comparam o buffer reaproveitado (o que o pacote usa hoje)
// contra a alocacao ingenua que existia antes (um make(map[...]) por
// chamada), no mesmo processo e escala do work/perf: 83 inimigos, a
// guarnicao do world_03 (doc/performance.md).
//
// Rodar: go test ./internal/network/... -bench InterpolatedEnemies -benchmem

func seedEnemies(n int) {
	RemoteEnemiesMutex.Lock()
	RemoteEnemies = make(map[string]EnemyState, n)
	for i := 0; i < n; i++ {
		id := enemyBenchID(i)
		RemoteEnemies[id] = EnemyState{EnemyID: id, X: i, Y: i, Health: 30, MaxHealth: 30}
	}
	RemoteEnemiesMutex.Unlock()
}

func enemyBenchID(i int) string {
	const letters = "0123456789abcdef"
	b := [10]byte{'e', 'n', 'e', 'm', 'y', '_'}
	n := i
	for j := len(b) - 1; j >= 6; j-- {
		b[j] = letters[n%16]
		n /= 16
	}
	return string(b[:])
}

// BenchmarkInterpolatedEnemiesReused mede o caminho ATUAL: sem snapshot par
// (nenhuma interpolacao em curso), cai no fallback que copia RemoteEnemies
// para o buffer reaproveitado.
func BenchmarkInterpolatedEnemiesReused(b *testing.B) {
	seedEnemies(83)
	ResetInterpolation()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkEnemies = InterpolatedEnemies()
	}
}

// BenchmarkInterpolatedEnemiesNaive mede o comportamento de ANTES da
// otimizacao: um mapa novo por chamada, mesma copia de 83 entradas.
func BenchmarkInterpolatedEnemiesNaive(b *testing.B) {
	seedEnemies(83)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RemoteEnemiesMutex.Lock()
		out := make(map[string]EnemyState, len(RemoteEnemies))
		for id, e := range RemoteEnemies {
			out[id] = e
		}
		RemoteEnemiesMutex.Unlock()
		sinkEnemies = out
	}
}
