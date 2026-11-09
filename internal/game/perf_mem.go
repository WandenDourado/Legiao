package game

// O medidor de MEMORIA, atras do mesmo F3 do medidor de quadro.
//
// Existe porque a hipotese "o jogo acumula recurso conforme avanca nas fases"
// nao podia ser confirmada nem descartada: o projeto media tempo de quadro,
// draw call e quad, e nao media um unico byte. Sem numero, "acho que esta
// vazando" e "acho que o mapa 6 e mais pesado que o mapa 1" sao a mesma frase,
// e as duas pedem correcoes opostas.
//
// COMO LER O PAINEL. Tire uma captura na PRIMEIRA fase, logo depois do
// carregamento, e outra na fase em que o jogo ja engasga, tambem logo depois do
// carregamento (parado, sem horda em campo — senao o que se compara e a briga,
// nao a fase). Depois:
//
//   - `heap` parecido nas duas e `pico` muito maior: nao ha vazamento; o que
//     ha e um pico de alocacao dentro do quadro, e o suspeito e o caminho que
//     aloca por quadro, nao o que retem.
//   - `heap` subindo fase a fase e nunca voltando: vazamento do lado do Go.
//     Os contadores da terceira linha dizem QUAL colecao.
//   - `heap` estavel e o jogo lento assim mesmo: o custo esta na GPU ou na
//     VRAM, e a linha `VRAM` e a que importa — ela conta TEXTURAS, nao bytes,
//     e cada retrato de dialogo e ~6 MB, cada atlas de manifesto ate 16 MB.
//   - `goroutines` subindo: conexao que nao fecha (rede), nao memoria de jogo.
//
// A amostragem so acontece com o F3 ligado, e a cada `memSampleFrames` quadros:
// `runtime.ReadMemStats` PARA O MUNDO enquanto le, e um medidor que trava o
// quadro que ele veio medir mente sobre o proprio quadro.

import (
	"fmt"
	"runtime"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/ui"
)

// memSampleFrames e o intervalo entre leituras. 30 quadros e meio segundo a
// 60 fps: rapido o bastante para o numero reagir a uma troca de fase enquanto
// se olha para ele, raro o bastante para a parada do mundo nao aparecer no
// medidor de quadro logo acima.
const memSampleFrames = 30

// memMeter guarda a ultima leitura e o pico da sessao.
//
// O PICO e o campo que responde a pergunta. Um heap que sobe e volta e o GC
// trabalhando, que e o comportamento certo; um pico que sobe a cada fase e
// nunca desce e memoria que ninguem devolveu.
type memMeter struct {
	countdown int
	heapMB    float32
	sysMB     float32
	peakMB    float32
	objects   uint64
	numGC     uint32
	// A TAXA DE ALOCACAO e o numero que separa "esta retendo" de "esta
	// gerando lixo". Um heap de 2,5 MB com nove mil ciclos de GC nao esta
	// vazando nada: esta alocando e descartando centenas de MB por segundo, e
	// o custo aparece como pausa de GC e trabalho de marcacao espalhado pelos
	// outros nucleos. Sem esta linha, o `GC 9003` da captura era um numero
	// grande sem denominador.
	lastTotal uint64
	lastAt    time.Time
	allocMBs  float32
}

var memory memMeter

// sample le as estatisticas do runtime, no maximo uma vez a cada
// memSampleFrames chamadas.
func (m *memMeter) sample() {
	if m.countdown > 0 {
		m.countdown--
		return
	}
	m.countdown = memSampleFrames

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	const mb = 1024 * 1024
	m.heapMB = float32(ms.HeapAlloc) / mb
	m.sysMB = float32(ms.Sys) / mb
	m.objects = ms.HeapObjects
	m.numGC = ms.NumGC
	if m.heapMB > m.peakMB {
		m.peakMB = m.heapMB
	}

	now := time.Now()
	if !m.lastAt.IsZero() && ms.TotalAlloc >= m.lastTotal {
		if elapsed := now.Sub(m.lastAt).Seconds(); elapsed > 0 {
			m.allocMBs = float32(float64(ms.TotalAlloc-m.lastTotal) / mb / elapsed)
		}
	}
	m.lastTotal, m.lastAt = ms.TotalAlloc, now
}

// memLines sao as tres linhas de memoria do painel do F3.
//
// Devolve nil quando o overlay esta desligado, para o chamador poder somar o
// resultado a lista sem checar nada.
func memLines() []string {
	if !tilemap.DebugEnabled() {
		return nil
	}
	memory.sample()
	mirror := network.Mirror()

	return []string{
		fmt.Sprintf("heap %.1f MB (pico %.1f / sys %.1f)   objetos %s   GC %d   lixo %.0f MB/s",
			memory.heapMB, memory.peakMB, memory.sysMB,
			thousands(memory.objects), memory.numGC, memory.allocMBs),
		fmt.Sprintf("goroutines %d   VRAM: mapa %d  inimigos %d  retratos %d",
			runtime.NumGoroutine(),
			tilemap.TextureCacheSize(),
			entity.EnemyTextureCacheSize(),
			ui.PortraitCacheSize()),
		fmt.Sprintf("espelho: jogadores %d  inimigos %d  projeteis %d   anims %d   espectros %d",
			mirror.Players, mirror.Enemies, mirror.Projectiles,
			entity.RemoteAnimCount(), network.SpecterCount()),
	}
}

// thousands separa milhares com ponto. Numero de objeto de heap passa de um
// milhao com facilidade, e "1238471" numa captura de tela nao se le.
func thousands(n uint64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	head := len(s) % 3
	if head > 0 {
		out = append(out, s[:head]...)
	}
	for i := head; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, '.')
		}
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}
