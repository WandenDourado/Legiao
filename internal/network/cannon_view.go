package network

// A BARRA DE VIDA DO CANHAO, e o pouco de estado que ela precisa nas DUAS
// maquinas (pedido do Gui, 27/08/2026).
//
// O canhao passou a cair por espada, e sem barra o jogador nao tem como saber
// se esta machucando pedra ou perdendo tempo: com 150 de vida e 75 por golpe
// supremo, a arte so troca no SEGUNDO acerto. Ate la, nada na tela responde.
//
// Por que um estado proprio e nao `RemoteEnemies`: o canhao nao e um
// `entity.Enemy` e nao vai virar um so para ganhar uma barra — ele nao anda,
// nao caca ninguem, nao conta em `EnemiesLeft` e nao pode ser mirado por
// projetil. Ele e decoracao do mapa com uma arma atras.
//
// Por que a POSICAO vem do mapa e nao da rede: as duas maquinas carregam o
// mesmo mapa e leem os mesmos marcadores `enemy_cannon_*`, entao a posicao e a
// vida cheia se instalam localmente (SetCannonPosts, chamado de
// game.World.ApplyToHost ACIMA da checagem de host). So o que muda com o jogo
// — a vida que sobrou e a queda — viaja pela rede. Isso resolve de graca o
// problema de um cliente que entra no meio: ele ja nasce com os postos certos.

import (
	"sync"

	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// CannonBar e um posto como o desenho precisa dele.
type CannonBar struct {
	Position  rl.Vector2
	HP        float32
	Max       float32
	Destroyed bool
}

var (
	cannonViewMu sync.RWMutex
	cannonView   []*CannonBar
)

// SetCannonPosts declara os postos do mapa que acabou de carregar, todos de pe
// e com vida cheia. Chamado nas duas maquinas.
func SetCannonPosts(posts []tilemap.SpawnPoint) {
	cannonViewMu.Lock()
	defer cannonViewMu.Unlock()
	cannonView = make([]*CannonBar, 0, len(posts))
	for _, p := range posts {
		cannonView = append(cannonView, &CannonBar{Position: p.Position, HP: CannonMaxHP, Max: CannonMaxHP})
	}
}

// SetCannonHealth atualiza a vida do posto naquela posicao. hp <= 0 marca a
// queda, e a barra some junto com o canhao de pe.
func SetCannonHealth(pos rl.Vector2, hp float32) {
	cannonViewMu.Lock()
	defer cannonViewMu.Unlock()
	for _, c := range cannonView {
		if !sameCannonSpot(c.Position, pos) {
			continue
		}
		c.HP = hp
		c.Destroyed = hp <= 0
		return
	}
}

// ResetCannonBars devolve todo posto a vida cheia e de pe, sem precisar da
// lista do mapa. Chamado no reinicio de fase nas duas maquinas — o host de
// RestoreCannons, o cliente de applyStageReset — pela mesma razao que
// ClearWreckedPieces: um canhao derrubado numa tentativa perdida nao pode
// aparecer ferido na tentativa seguinte.
func ResetCannonBars() {
	cannonViewMu.Lock()
	defer cannonViewMu.Unlock()
	for _, c := range cannonView {
		c.HP = c.Max
		c.Destroyed = false
	}
}

// CannonsStanding reporta se ainda ha canhao de pe no mapa em jogo.
//
// E a porta da fase 6: o portal do fim do corredor so aparece quando os DOIS
// caem (game.PortalsUnlocked). Mora aqui, e nao no Host, porque o portal e
// desenhado e atravessado nas duas maquinas — o cliente precisa da mesma
// resposta, e a tem pelo mesmo caminho da barra (posicao do mapa, vida por
// MsgCannonState).
//
// Mapa sem canhao devolve falso, que e o que faz esta regra nao existir em
// nenhum outro mapa sem ninguem precisar perguntar qual mapa e este.
func CannonsStanding() bool {
	cannonViewMu.RLock()
	defer cannonViewMu.RUnlock()
	for _, c := range cannonView {
		if !c.Destroyed {
			return true
		}
	}
	return false
}

// CannonsSilenced reporta se o mapa em jogo DECLARA canhoes e todos ja
// cairam.
//
// E o "fim da luta" do corredor da fase 6, que nao tem horda nenhuma: o
// gatilho on_waves_cleared do dialogo (game.stageFightOver) le isto para que
// a cena de depois dos canhoes toque. Nao e o mesmo que !CannonsStanding():
// mapa sem canhao devolve FALSO aqui, porque um mapa quieto nao terminou luta
// nenhuma e nao pode disparar cena de fim de fase.
func CannonsSilenced() bool {
	cannonViewMu.RLock()
	defer cannonViewMu.RUnlock()
	if len(cannonView) == 0 {
		return false
	}
	for _, c := range cannonView {
		if !c.Destroyed {
			return false
		}
	}
	return true
}

// CannonBars devolve uma COPIA dos postos ainda de pe, para o desenho. Copia
// porque quem chama e a thread de desenho e quem escreve e a de rede.
func CannonBars() []CannonBar {
	cannonViewMu.RLock()
	defer cannonViewMu.RUnlock()
	out := make([]CannonBar, 0, len(cannonView))
	for _, c := range cannonView {
		if c.Destroyed {
			continue
		}
		out = append(out, *c)
	}
	return out
}

// sameCannonSpot casa um posto por posicao, com a mesma tolerancia de meia
// celula que a troca de arte usa (tilemap.WreckPieceAt) e pela mesma razao: o
// marcador da camada `spawn` e o objeto da camada `buildings` sao pontos
// distintos do mesmo mapa que coincidem por construcao.
func sameCannonSpot(a, b rl.Vector2) bool {
	const tol float32 = 64
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= tol && dy <= tol
}
