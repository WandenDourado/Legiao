package game

// O medidor de quadro, atras do mesmo F3 que o resto da depuracao.
//
// Existe porque as otimizacoes de desenho NAO mudam um pixel na tela: sem um
// numero, "ficou mais rapido" e impressao, e um caminho de desenho novo que
// escape do culling passa despercebido.
//
// O painel e desenhado para ser LIDO NUMA CAPTURA DE TELA e diagnosticado
// depois, sem quem le ter de lembrar em que mapa estava, em que resolucao, ou
// se era host ou cliente. Cada linha aqui responde uma pergunta que uma
// captura sem ela deixaria em aberto — foi por faltar o nome do mapa numa
// captura que esta regra virou explicita.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/skill"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// perfSamples e quantos quadros a janela guarda. 120 sao dois segundos a
// 60 fps: o bastante para o PIOR quadro ser representativo e curto o bastante
// para o numero reagir enquanto se anda pelo mapa.
const perfSamples = 120

// perfMeter guarda os tempos de quadro recentes.
//
// O pior quadro da janela importa mais que a media, e essa e a razao de ele
// estar aqui: 58 fps de media com um quadro de 33 ms a cada meio segundo e
// exatamente o padrao que faz a camera trancar e cansar a vista, e uma media
// esconde isso por completo.
type perfMeter struct {
	samples [perfSamples]float32
	next    int
	filled  int
}

var meter perfMeter

func (m *perfMeter) push(dt float32) {
	m.samples[m.next] = dt
	m.next = (m.next + 1) % perfSamples
	if m.filled < perfSamples {
		m.filled++
	}
}

// window devolve o tempo medio e o pior tempo de quadro da janela, em segundos.
func (m *perfMeter) window() (avg, worst float32) {
	if m.filled == 0 {
		return 0, 0
	}
	var total float32
	for i := 0; i < m.filled; i++ {
		dt := m.samples[i]
		total += dt
		if dt > worst {
			worst = dt
		}
	}
	return total / float32(m.filled), worst
}

// DrawPerfHUD registra o quadro e, com o F3 ligado, desenha o medidor.
//
// A amostragem acontece SEMPRE, mesmo com o overlay desligado: ligar o F3 no
// meio de uma travada tem de mostrar a travada, e nao uma janela vazia que
// leva dois segundos para encher.
func DrawPerfHUD(sw, sh float32, w *World, p *entity.Player) {
	meter.push(rl.GetFrameTime())
	if !tilemap.DebugEnabled() {
		return
	}

	avg, worst := meter.window()
	stats := tilemap.FrameStats()
	enemies := entity.LastEnemyDrawCounts()

	// O resto do quadro e o que sobra depois da submissao de CPU: GPU, vsync e
	// tudo que e desenhado fora do bloco da camera (HUD, dialogo, portal).
	// Separado de proposito — se ele domina, culling nao resolve mais nada.
	sim := simMilliseconds()
	rest := avg*1000 - stats.MapMS - stats.EntityMS - sim
	if rest < 0 {
		rest = 0
	}

	drawnFX, culledFX := skill.DrawCounts()

	lines := []string{
		fmt.Sprintf("%s   %s   %.0fx%.0f   %s", mapLabel(w), roleLabel(), sw, sh, syncLabel()),
		fmt.Sprintf("fps %d   quadro %.1f ms   PIOR %.1f ms", rl.GetFPS(), avg*1000, worst*1000),
		fmt.Sprintf("cpu: mapa %.1f   entidades %.1f   simulacao %.1f   resto %.1f ms",
			stats.MapMS, stats.EntityMS, sim, rest),
		fmt.Sprintf("terreno %d quads / %d binds de shader", stats.TerrainQuads, stats.ShaderBinds),
		fmt.Sprintf("tiles %d   props %d   trilha %d quads   celulas %d",
			stats.Tiles, stats.Props, stats.TrailQuads, stats.CellsVisited),
		fmt.Sprintf("inimigos %d vivos / %d desenhados   projeteis %d",
			enemies.Alive, enemies.Drawn, enemies.Projectiles),
		fmt.Sprintf("magias: %d desenhadas / %d puladas pelo culling", drawnFX, culledFX),
		playerLabel(w, p),
	}
	// As linhas de memoria vem por ultimo, e vem de perf_mem.go: elas respondem
	// uma pergunta diferente das de cima ("esta acumulando?" contra "este
	// quadro custou quanto?") e sao lidas juntas na mesma captura.
	lines = append(lines, memLines()...)

	drawPerfPanel(sw, lines, worst)
}

// mapLabel e o nome do mapa em jogo. Uma captura sem ele obriga quem analisa a
// adivinhar a fase pelos numeros, que foi exatamente o que aconteceu.
func mapLabel(w *World) string {
	if w == nil || w.Path == "" {
		return "mapa ?"
	}
	return strings.TrimSuffix(filepath.Base(w.Path), ".json")
}

// syncLabel diz se o vsync esta ATIVO e a que taxa o monitor atualiza.
//
// Existe porque a pergunta "o vsync foi habilitado?" nao tinha resposta na
// tela, e nenhuma metrica de TEMPO responde: `quadro 16,7 ms / PIOR 16,7 ms`
// e o que um jogo com a cadencia certa e o sincronismo errado mostra — o
// `SetTargetFPS` limita o intervalo, o vsync alinha a TROCA DE BUFFER ao
// retraco, e so o segundo evita a imagem rasgada. Um medidor de tempo de
// quadro nao ve tearing.
//
// Le o estado REAL da janela (`IsWindowState`) e nao a `Config`: a flag e um
// pedido ao driver, e o driver pode recusar — um perfil da placa com "vertical
// sync: off" forcado ganha do jogo, e nesse caso a `Config` diria "ligado"
// enquanto a tela rasga.
func syncLabel() string {
	hz := rl.GetMonitorRefreshRate(rl.GetCurrentMonitor())
	if rl.IsWindowState(rl.FlagVsyncHint) {
		return fmt.Sprintf("vsync ON   %d Hz", hz)
	}
	return fmt.Sprintf("vsync OFF   %d Hz", hz)
}

// roleLabel diz de que lado da rede este processo esta, porque o host e o
// cliente rodam caminhos de desenho DIFERENTES: o host desenha do
// EntityManager que ele simula, o cliente desenha do mapa de snapshots.
func roleLabel() string {
	switch network.Role {
	case "host":
		return "host"
	case "client":
		return "cliente"
	default:
		return "sozinho"
	}
}

// playerLabel situa a amostra no mapa. Duas capturas do mesmo mapa com numeros
// diferentes so sao comparaveis se der para saber de onde cada uma veio.
func playerLabel(w *World, p *entity.Player) string {
	if p == nil {
		return ""
	}
	cell := ""
	if w != nil && w.Renderer != nil && w.Renderer.Map != nil && w.Renderer.Map.TileWidth > 0 {
		cell = fmt.Sprintf("   celula %d,%d",
			int(p.Position.X)/w.Renderer.Map.TileWidth,
			int(p.Position.Y)/w.Renderer.Map.TileHeight)
	}
	return fmt.Sprintf("pos %.0f,%.0f%s", p.Position.X, p.Position.Y, cell)
}

// perfTopMargin e a distancia do painel ate o topo da tela.
//
// Nao e estetica. O jogo abre com rl.InitWindow(0, 0), que faz o raylib usar a
// resolucao do monitor para uma janela COM decoracao: a area de cliente fica
// maior que a area util do desktop e as primeiras dezenas de pixels ficam
// atras da barra de titulo. A primeira versao deste painel comecava a 8 px do
// topo e a linha de identificacao do mapa simplesmente nao aparecia nas
// capturas - o resto do painel aparecia, o que fez parecer bug de conteudo
// quando era de posicao.
//
// A margem resolve o painel; a janela em si continua cortando a borda de cima
// para todo o resto do HUD, e isso e um defeito separado (ver doc/performance.md).
const perfTopMargin = 56

func drawPerfPanel(sw float32, lines []string, worst float32) {
	const size = 18
	const pad = 8

	width := int32(0)
	for _, line := range lines {
		if v := rl.MeasureText(line, size); v > width {
			width = v
		}
	}
	x := int32(sw) - width - 2*pad
	y := int32(perfTopMargin)
	height := int32(len(lines))*(size+4) + pad

	rl.DrawRectangle(x-pad, y-pad, width+2*pad, height+pad, rl.Fade(rl.Black, 0.7))
	for i, line := range lines {
		colour := rl.RayWhite
		switch {
		case i == 0:
			// O cabecalho em amarelo para nao se perder no meio dos numeros
			// quando a captura for lida fora de contexto.
			colour = rl.Yellow
		case i == 1 && worst > 0.020:
			// Acima de 20 ms o quadro deixou de caber em 60 fps e a camera
			// comeca a andar em passos desiguais — que e o defeito que o
			// jogador sente como tontura.
			colour = rl.Orange
		}
		rl.DrawText(line, x, y+int32(i)*(size+4), size, colour)
	}
}
