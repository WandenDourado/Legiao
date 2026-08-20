package tilemap

import "time"

// Contagem do que o quadro anterior realmente desenhou.
//
// Existe para que "ficou mais rapido" deixe de ser impressao. O culling e uma
// mudanca que NAO altera um pixel na tela, entao a unica forma de saber se ele
// esta funcionando - ou se algum caminho novo escapou dele - e ver o numero
// cair.
//
// Sem mutex de proposito: tudo isto e incrementado dentro do passe de desenho,
// que roda na goroutine principal entre BeginDrawing e EndDrawing. Um mutex
// aqui custaria mais que o proprio contador.

// Stats e o que um quadro desenhou.
type Stats struct {
	// TerrainQuads sao os quads de terreno: a base mais cada camada.
	TerrainQuads int
	// ShaderBinds e quantos Begin/EndShaderMode o terreno fez. Cada um esvazia
	// o batch do rlgl, entao este numero E o numero de draw calls do terreno -
	// e era 18.844 por quadro no world_03.
	ShaderBinds int
	// Tiles sao os tiles comuns desenhados (ground_detail, foreground...).
	Tiles int
	// Props sao as pecas de manifesto desenhadas (arvore, casa, cerca, muralha).
	Props int
	// TrailQuads sao os quads de fita de trilha emitidos. Cada quad custa 12
	// chamadas de cgo, entao ele conta separado dos quads de terreno.
	TrailQuads int
	// CellsVisited sao as celulas que os lacos por celula chegaram a visitar.
	// Comparado com TerrainQuads, mostra quanto do laco e trabalho e quanto e
	// varredura.
	CellsVisited int

	// MapMS e EntityMS sao o tempo de CPU gasto empilhando as chamadas de
	// desenho do mapa e das entidades, em milissegundos.
	//
	// Medem SUBMISSAO, nao a GPU: o raylib acumula em batch e o trabalho real
	// acontece no EndDrawing. E justamente por isso que os dois numeros valem
	// tanto - se a soma deles for pequena e o quadro mesmo assim for longo, o
	// gargalo esta na GPU ou no vsync, e nao em nada que culling resolva. Se
	// MapMS sozinho for metade do quadro, esta.
	MapMS, EntityMS float32
}

var (
	frameStats Stats
	lastStats  Stats
)

// beginFrameStats zera a contagem do quadro que comeca.
func beginFrameStats() { frameStats = Stats{} }

// endFrameStats publica a contagem do quadro que terminou. Publicar no fim, e
// nao ler o contador vivo, evita que o HUD mostre um quadro pela metade -
// ele e desenhado depois do mapa, entao leria a contagem incompleta.
func endFrameStats() { lastStats = frameStats }

// FrameStats devolve o que o ultimo quadro completo desenhou.
func FrameStats() Stats { return lastStats }

func milliseconds(d time.Duration) float32 {
	return float32(d.Seconds() * 1000)
}
