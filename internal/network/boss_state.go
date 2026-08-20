package network

import "sync"

// O estado do chefe, publicado pelo host e lido pelo HUD.
//
// Mesma forma de `WaveState` e pelo mesmo motivo: a barra de chefe e desenhada
// identica no host e no cliente, e nenhum dos dois deve ir procurar a criatura
// no EntityManager para saber quanta vida ela tem. O host publica; quem desenha
// so le.
//
// `Present` false e o estado normal da campanha inteira — so o mapa 7 tem
// chefe. E isso e o que faz `DrawBossBar` nao desenhar nada nos outros seis
// sem precisar saber qual mapa esta carregado.
type BossState struct {
	Present   bool
	Name      string
	Health    float32
	MaxHealth float32
	// Casting e verdadeiro enquanto a chefe DANCA e a nevoa ainda nao entrou.
	// E o unico aviso que o jogador tem, e ele precisa chegar ao HUD: a danca
	// acontece do outro lado de uma arena de 5120 px e um grupo que esta
	// segurando um portao simplesmente nao ve a chefe.
	Casting bool
	// CastLeft e quantos segundos faltam para a nevoa. Alimenta a contagem no
	// aviso — "PERIGO" sem relogio diz que algo vem, nao diz quando.
	CastLeft float32
}

var (
	currentBoss      BossState
	currentBossMutex sync.Mutex
)

// SetBossState publica o estado do chefe. Chamado uma vez por quadro pelo host.
func SetBossState(b BossState) {
	currentBossMutex.Lock()
	defer currentBossMutex.Unlock()
	currentBoss = b
}

// GetBossState devolve uma copia do ultimo estado publicado.
func GetBossState() BossState {
	currentBossMutex.Lock()
	defer currentBossMutex.Unlock()
	return currentBoss
}
