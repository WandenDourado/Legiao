package entity

// Quantos inimigos existiam e quantos chegaram a ser desenhados.
//
// Existe para uma pergunta que o contador do mapa nao responde: no world_03 a
// guarnicao poe 83 monstros em campo desde o carregamento, e saber quantos
// deles estao na tela e o que separa "o mapa e caro de desenhar" de "a horda e
// cara de desenhar". Se 83 estao vivos e 12 sao desenhados, o desenho nao e o
// problema — a SIMULACAO e, porque ela roda para os 83 de qualquer jeito.
//
// Sem mutex pelo mesmo motivo do tilemap: escrito e lido dentro do passe de
// desenho, na goroutine principal.

// EnemyDrawCounts e o resultado do ultimo passe de inimigos.
type EnemyDrawCounts struct {
	// Alive sao os inimigos ativos que o passe considerou.
	Alive int
	// Drawn sao os que passaram no teste de visibilidade.
	Drawn int
	// Projectiles sao os projeteis ativos desenhados no mesmo passe.
	Projectiles int
}

var enemyCounts EnemyDrawCounts

// RecordEnemyDraw publica a contagem do passe. Exportado porque ha DOIS
// caminhos de desenho de inimigo e os dois precisam reportar: o host desenha do
// EntityManager, o cliente desenha do mapa de snapshots que a rede mantem.
func RecordEnemyDraw(counts EnemyDrawCounts) { enemyCounts = counts }

// LastEnemyDrawCounts devolve a contagem do ultimo passe.
func LastEnemyDrawCounts() EnemyDrawCounts { return enemyCounts }
