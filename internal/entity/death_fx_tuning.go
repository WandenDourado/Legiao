package entity

// Os numeros da morte por desintegracao (death_fx.go).
//
// Vieram VERBATIM do preview aprovado pelo Gui em 28/09/2026
// (desintegracao_preview.html, FX_MOB e FX_BOSS). Mexer aqui e mexer no que
// ele viu: se um numero mudar, mude la tambem, ou o preview passa a mentir.
//
// Tempos em segundos, distancias em px de mundo.

import "image/color"

type deathTuning struct {
	Flash      float32 // clarao do golpe final
	Pop, PopT  float32 // o corpo incha Pop (fracao) e volta em PopT
	Char       float32 // o corpo escurece para carvao antes de ceder
	Delay      float32 // quando o primeiro grao solta
	Sweep      float32 // quanto a frente leva para atravessar o corpo
	Noise      float32 // quanto o ruido desalinha a frente (0 = linha reta)
	NoiseScale float32 // tamanho das manchas de ruido, em celulas
	Edge       float32 // a brasa acende este tanto antes do grao soltar
	EdgeA      float32 // alfa maximo da brasa (aditiva)
	LifeMin    float32
	LifeMax    float32
	Kick       float32 // empurrao inicial aleatorio
	WindAcc    float32 // aceleracao do vento
	Drag       float32 // velocidade terminal = WindAcc / Drag
	Lift       float32
	Turb       float32
	EmberFrac  float32 // fracao dos graos que vira faisca
	Hot        float32 // quanto tempo o grao recem-solto fica em brasa
	PuffP      float32 // chance de um grao soltar um tufo de fumaca
	PuffLife   float32
	PuffR0     float32
	PuffR1     float32
	PuffA      float32
	CellScale  float32 // multiplica deathFXCellWorld (o chefe usa grao maior)
}

var deathTuneMob = deathTuning{
	Flash: 0.09, Pop: 0.06, PopT: 0.18, Char: 0.22, Delay: 0.10,
	Sweep: 0.55, Noise: 0.32, NoiseScale: 5, Edge: 0.06, EdgeA: 0.45,
	LifeMin: 0.55, LifeMax: 1.0, Kick: 45, WindAcc: 520, Drag: 2.0,
	Lift: 60, Turb: 140, EmberFrac: 0.05, Hot: 0.07,
	PuffP: 0.035, PuffLife: 0.9, PuffR0: 6, PuffR1: 34, PuffA: 0.22,
	CellScale: 1,
}

// deathTuneBoss e a versao lenta da Imperatriz: a queda dela fecha a campanha
// e merece durar. O dialogo final espera o corpo sumir (BossDeathFXPlaying).
var deathTuneBoss = deathTuning{
	Flash: 0.16, Pop: 0.03, PopT: 0.3, Char: 0.9, Delay: 0.45,
	Sweep: 2.4, Noise: 0.28, NoiseScale: 6, Edge: 0.16, EdgeA: 0.45,
	LifeMin: 0.9, LifeMax: 1.8, Kick: 30, WindAcc: 300, Drag: 1.4,
	Lift: 45, Turb: 120, EmberFrac: 0.08, Hot: 0.12,
	PuffP: 0.03, PuffLife: 1.4, PuffR0: 10, PuffR1: 60, PuffA: 0.2,
	CellScale: 1.5,
}

// Paleta. CHAR_TINT multiplica o corpo "queimado"; a brasa e aditiva.
var (
	deathCharTint = [3]float32{0.50, 0.46, 0.43}
	deathEmber    = color.RGBA{255, 125, 40, 255}
	deathSpark    = color.RGBA{255, 190, 90, 255}
)

const (
	// deathWindBaseDeg e para onde o vento leva as cinzas: para cima e para a
	// direita. Cada morte sorteia +-deathWindJitterDeg em volta, para uma
	// horda inteira nao se desfazer no mesmo angulo.
	deathWindBaseDeg   = -35
	deathWindJitterDeg = 20
	// deathTintLevels quantiza o escurecimento do corpo. No jogo o tint e
	// continuo; a quantizacao so existe para o grao vizinho nao piscar entre
	// dois tons quase iguais.
	deathTintLevels = 16
	// deathFXWorldGrace: depois de trocar de mapa ou reiniciar a fase, nada
	// dispara por este tempo. O cliente ainda recebe um snapshot do mapa
	// velho depois da troca, e esses monstros "sumindo" nao morreram.
	deathFXWorldGrace = 2.0
	// deathCellCoverage: celula com menos que isso de pixel opaco nao vira
	// grao. Mesmo corte do preview.
	deathCellCoverage = 0.30
)
