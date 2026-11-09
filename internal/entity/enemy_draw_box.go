package entity

// Onde um inimigo aparece na tela, para o desenho poder pular quem nao aparece.
//
// Existe por causa do world_03: a guarnicao poe 83 monstros em campo desde o
// carregamento do mapa e TODOS eram desenhados, dentro ou fora da tela, porque
// nenhum caminho de desenho de entidade consultava a camera.

import rl "github.com/gen2brain/raylib-go/raylib"

// enemyBarClearance e a folga acima do sprite onde a barra de vida e desenhada
// (drawEnemyHealthBar poe a barra em y-above-8, com 3 px de altura). Sem ela a
// barra de um monstro entrando pelo topo da tela apareceria depois do corpo.
const enemyBarClearance float32 = 16

// EnemyDrawBox e o retangulo de mundo que o desenho de um inimigo ocupa: o
// quadro da animacao na escala de render, ja levantado pela linha do pe, mais
// a folga da barra de vida.
//
// Vem da geometria do QUADRO e nao do Radius de combate, e a diferenca e
// grande: o orc tem quadro de 154x134 desenhado a 1.6x (246x214 na tela) com
// Radius 45. Cullar pelo raio cortaria o monstro a ~90 px de largura e faria
// a cabeca e o espadao dele piscarem ao entrar pela borda.
func EnemyDrawBox(def EnemyDef, anim EnemyAnim, x, y float32) rl.Rectangle {
	ad := def.AnimDef(anim)
	scale := def.RenderScale
	if scale <= 0 {
		scale = 1
	}
	w := float32(ad.FrameWidth) * scale
	h := float32(ad.FrameHeight) * scale
	if w <= 0 || h <= 0 {
		// Inimigo sem folha desenha um circulo de Radius; sem geometria de
		// quadro, o raio e a unica medida que existe.
		w = def.Radius * 2
		h = def.Radius * 2
	}
	// drawEnemyDirectional levanta o quadro para a sola cair em y; o centro do
	// desenho sobe junto.
	centerY := y - enemyGroundOffset(ad, scale)
	return rl.NewRectangle(
		x-w/2,
		centerY-h/2-enemyBarClearance,
		w,
		h+enemyBarClearance,
	)
}

// RemoteEnemyDrawBox e o EnemyDrawBox de um inimigo que o cliente so conhece
// pelo snapshot.
//
// Usa a maior caixa entre as animacoes do tipo em vez de perguntar ao tracker
// qual esta tocando: o tracker e detalhe interno do desenho, e uma caixa que
// encolhe junto com a animacao faria o monstro sumir na borda da tela no
// quadro em que ele parasse de andar (o quadro de walk do orc e 100x127 contra
// 154x134 do idle).
func RemoteEnemyDrawBox(enemyType EnemyType, x, y float32) rl.Rectangle {
	def := GetEnemyDef(enemyType)
	box := EnemyDrawBox(def, AnimIdle, x, y)
	for anim := range def.Anims {
		box = union(box, EnemyDrawBox(def, anim, x, y))
	}
	return box
}

// enemyDrawBoxOf e a caixa do inimigo que o host simula, na animacao que ele
// esta tocando de verdade.
func enemyDrawBoxOf(e *Enemy) rl.Rectangle {
	return EnemyDrawBox(GetEnemyDef(e.Type), e.Anim, e.Position.X, e.Position.Y)
}

// EnemyVisible diz se a caixa aparece dentro da janela da camera. Uma janela
// de largura ou altura zero conta como "sem culling": e o que um chamador que
// nao sabe o tamanho da tela deve receber, e desenhar tudo e o comportamento
// que existia antes desta funcao.
func EnemyVisible(box, view rl.Rectangle) bool {
	if view.Width <= 0 || view.Height <= 0 {
		return true
	}
	return box.X < view.X+view.Width &&
		box.X+box.Width > view.X &&
		box.Y < view.Y+view.Height &&
		box.Y+box.Height > view.Y
}

func union(a, b rl.Rectangle) rl.Rectangle {
	x := min32(a.X, b.X)
	y := min32(a.Y, b.Y)
	right := max32(a.X+a.Width, b.X+b.Width)
	bottom := max32(a.Y+a.Height, b.Y+b.Height)
	return rl.NewRectangle(x, y, right-x, bottom-y)
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
