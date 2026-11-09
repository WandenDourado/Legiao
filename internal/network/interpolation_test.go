package network

import (
	"math/rand"
	"testing"
	"time"
)

// fakeClock controla o tempo da interpolacao.
type fakeClock struct{ t time.Time }

func (f *fakeClock) now() time.Time          { return f.t }
func (f *fakeClock) advance(d time.Duration) { f.t = f.t.Add(d) }

func withFakeClock(t *testing.T) *fakeClock {
	t.Helper()
	fc := &fakeClock{t: time.Unix(1000, 0)}
	prevNow, prevEpoch, prevRole := interpNow, interpEpoch, Role
	interpNow, interpEpoch, Role = fc.now, fc.t, "client"
	ResetInterpolation()
	t.Cleanup(func() {
		interpNow, interpEpoch, Role = prevNow, prevEpoch, prevRole
		ResetInterpolation()
	})
	return fc
}

// simulate: o host anda 10 px por tique (20 Hz); cada tique chega com um
// atraso de rede de base+jitter. Devolve o X desenhado em cada quadro de 16 ms.
func simulate(t *testing.T, jitterMs int, seconds int) []int {
	fc := withFakeClock(t)
	r := rand.New(rand.NewSource(7))
	type arrival struct {
		at   time.Duration
		tick int
	}
	var arrivals []arrival
	for tick := 0; tick < seconds*20; tick++ {
		delay := 40 + r.Intn(jitterMs+1)
		arrivals = append(arrivals, arrival{time.Duration(tick*50+delay) * time.Millisecond, tick})
	}
	var xs []int
	next := 0
	start := fc.t
	for frame := time.Duration(0); frame < time.Duration(seconds)*time.Second; frame += 16 * time.Millisecond {
		fc.t = start.Add(frame)
		for next < len(arrivals) && arrivals[next].at <= frame {
			tk := arrivals[next].tick
			hostT := int64(tk*50 + 1)
			recordSnapshot(map[string]PlayerState{"p": {PlayerID: "p", X: tk * 10}}, nil, hostT)
			recordSnapshot(nil, map[string]EnemyState{"e": {EnemyID: "e", X: tk * 10}}, hostT) // mesmo tique
			next++
		}
		xs = append(xs, InterpolatedPlayers()["p"].X)
	}
	return xs[len(xs)/2:] // descarta a partida: o atraso esta se ajustando
}

// Na LAN (sem oscilacao) o movimento remoto e liso: anda a cada quadro, sem
// quadros repetidos nem saltos de um tique inteiro (10 px).
func TestInterpolationIsSmoothOnLAN(t *testing.T) {
	xs := simulate(t, 0, 4)
	still, jumps := 0, 0
	for i := 1; i < len(xs); i++ {
		d := xs[i] - xs[i-1]
		if d == 0 {
			still++
		}
		if d >= 10 {
			jumps++
		}
		if d < 0 {
			t.Fatalf("voltou para tras no quadro %d: %v", i, xs[i-3:i+1])
		}
	}
	if still > len(xs)/20 || jumps > 0 {
		t.Fatalf("nao esta liso: %d quadros parados e %d saltos em %d quadros", still, jumps, len(xs))
	}
}

// Com o 4G oscilando ate 120 ms, o atraso cresce e o movimento continua liso.
func TestInterpolationAbsorbsMobileJitter(t *testing.T) {
	xs := simulate(t, 120, 6)
	still := 0
	for i := 1; i < len(xs); i++ {
		if d := xs[i] - xs[i-1]; d == 0 {
			still++
		} else if d < 0 {
			t.Fatalf("voltou para tras: %v", xs[i-3:i+1])
		}
	}
	if still > len(xs)/10 {
		t.Fatalf("%d de %d quadros parados com jitter", still, len(xs))
	}
	if d := InterpDelay(); d <= minInterpDelay {
		t.Fatalf("atraso nao cresceu com o jitter: %v", d)
	}
}

// Sem snapshot novo, segura no ultimo ponto conhecido: nunca extrapola.
func TestInterpolationNeverExtrapolates(t *testing.T) {
	fc := withFakeClock(t)
	for tk := 0; tk < 5; tk++ {
		recordSnapshot(map[string]PlayerState{"p": {X: tk * 10}}, nil, int64(tk*50+1))
		fc.advance(50 * time.Millisecond)
	}
	fc.advance(2 * time.Second) // o host sumiu
	if x := InterpolatedPlayers()["p"].X; x != 40 {
		t.Fatalf("X = %d, quer parado no ultimo conhecido (40)", x)
	}
}

// Host antigo (sem HostT): cai na hora de chegada e ainda interpola.
func TestInterpolationWorksWithOldHost(t *testing.T) {
	fc := withFakeClock(t)
	for tk := 0; tk < 10; tk++ {
		recordSnapshot(map[string]PlayerState{"p": {X: tk * 10}}, nil, 0)
		fc.advance(50 * time.Millisecond)
	}
	x1 := InterpolatedPlayers()["p"].X
	fc.advance(16 * time.Millisecond)
	x2 := InterpolatedPlayers()["p"].X
	if x2 <= x1 {
		t.Fatalf("sem movimento entre quadros com host antigo: %d -> %d", x1, x2)
	}
}
