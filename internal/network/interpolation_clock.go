package network

import (
	"sort"
	"time"
)

// Relogio da interpolacao: em que instante DO HOST o cliente deve desenhar.
//
// Cada chegada da uma amostra "hora local - HostT" = diferenca entre os
// relogios + o atraso daquela mensagem. A MENOR amostra recente e a melhor
// estimativa da diferenca pura (a mensagem que menos esperou); o quanto as
// outras passam dela e a variacao da rede (jitter). O atraso de desenho e
// minInterpDelay mais essa variacao (percentil 90), dentro de [min, max], e
// anda devagar (interpSlew) para a mudanca nao aparecer como tranco.

const (
	minInterpDelay = 100 * time.Millisecond // dois intervalos de snapshot a 20 Hz
	maxInterpDelay = 300 * time.Millisecond
	interpWindow   = 40   // amostras (~2 s a 20 Hz)
	interpSlew     = 0.05 // ms de ajuste por ms de relogio
)

var (
	// interpNow e variavel para os testes controlarem o tempo.
	interpNow   = time.Now
	interpEpoch = time.Now()

	offsets   []int64 // amostras recentes de local - HostT, em ms
	delayMs   = float64(minInterpDelay.Milliseconds())
	lastSlewM int64
)

func localMs() int64 { return interpNow().Sub(interpEpoch).Milliseconds() }

// observeArrival registra a chegada de um snapshot e devolve o t dele. Com
// host antigo (hostT 0) o t e a propria hora de chegada.
func observeArrival(hostT int64) int64 {
	now := localMs()
	if hostT == 0 {
		hostT = now
	}
	offsets = append(offsets, now-hostT)
	if len(offsets) > interpWindow {
		offsets = offsets[len(offsets)-interpWindow:]
	}
	return hostT
}

// targetDelayMs e o atraso que a rede de agora pede.
func targetDelayMs() float64 {
	if len(offsets) == 0 {
		return float64(minInterpDelay.Milliseconds())
	}
	sorted := append([]int64(nil), offsets...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	jitter := sorted[len(sorted)*9/10] - sorted[0]
	d := float64(minInterpDelay.Milliseconds() + jitter)
	return min(d, float64(maxInterpDelay.Milliseconds()))
}

// renderTime e o instante, no relogio do host, que deve aparecer agora.
func renderTime() int64 {
	now := localMs()
	if lastSlewM != 0 {
		step := float64(now-lastSlewM) * interpSlew
		if t := targetDelayMs(); t > delayMs {
			delayMs = min(t, delayMs+step)
		} else {
			delayMs = max(t, delayMs-step)
		}
	}
	lastSlewM = now
	base := offsets[0]
	for _, o := range offsets {
		base = min(base, o)
	}
	return now - base - int64(delayMs)
}

// InterpDelay e o atraso de desenho em uso (para o HUD de rede).
func InterpDelay() time.Duration {
	interpMutex.Lock()
	defer interpMutex.Unlock()
	return time.Duration(delayMs) * time.Millisecond
}

func resetInterpClock() {
	offsets = offsets[:0]
	delayMs = float64(minInterpDelay.Milliseconds())
	lastSlewM = 0
}
