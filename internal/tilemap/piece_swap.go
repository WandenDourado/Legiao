package tilemap

// Troca de arte de um objeto do mapa em runtime.
//
// Existe por causa de UM caso, e o contrato e estreito de proposito: os
// canhoes do corredor do mapa 6 tem duas versoes no manifesto — de pe e em
// ruinas — e a Paladina derruba um de cada vez, a espada, no meio da fase
// (internal/network/cannons.go). Marcar o posto como silenciado nao basta: a
// arma pararia de atirar e a arte continuaria intacta, e o jogador nao teria
// como saber, olhando, qual canhao ainda mata — que e a unica informacao que
// importa enquanto ele atravessa o corredor.
//
// Por que a chave e a POSICAO e nao o nome do objeto: o posto que o host
// conhece vem do marcador `enemy_cannon_*` da camada `spawn`, e a decoracao
// vem de um objeto da camada `buildings`. Os dois sao pontos do mesmo mapa e
// coincidem por construcao, mas nao ha — nem deve haver — tabela ligando um
// nome ao outro. Casar por posicao dispensa essa tabela e continua valendo
// para qualquer mapa futuro que declare os proprios postos.
//
// Por que o sufixo e implicito: a peca em ruinas se chama `<peca>_wrecked` no
// mesmo manifesto, com a MESMA ancora (centro do plinto na linha do chao) e o
// MESMO footprint de colisao. Assim a troca nao mexe em colisao nem desloca a
// arte um pixel — o plinto fica onde estava e so o cano se despedaca. Se a
// peca `_wrecked` nao existir no manifesto, nada acontece: um mapa sem a
// variante desenha a peca de pe, e nao um buraco.

import (
	"math"
	"sync"
)

// wreckedSuffix e o nome da variante em ruinas de uma peca do manifesto.
const wreckedSuffix = "_wrecked"

// wreckMatchRadius e quanto o objeto de decoracao pode estar longe do marcador
// e ainda ser o mesmo posto. Meia celula: o mapa poe os dois no mesmo ponto,
// e uma tolerancia maior arriscaria pegar a peca vizinha.
const wreckMatchRadius float32 = 64

// wreckedSpots sao os pontos do mapa cuja peca deve desenhar em ruinas. Fica
// aqui, e nao no Host, porque o CLIENTE tambem precisa dela: quem so assiste a
// cena tem de ver os canhoes caidos igual a quem hospeda.
//
// O mutex nao e zelo: quem escreve e a thread de rede (o handler de
// MsgCannonState no cliente, o golpe de espada no host) e quem le e a thread que
// desenha, todo quadro. Sem ele um append no meio de um Draw e corrida de
// dados sobre uma fatia que esta sendo percorrida.
var (
	wreckedMutex sync.RWMutex
	wreckedSpots []struct{ X, Y float32 }
)

// WreckPieceAt marca o posto naquela posicao como destruido.
func WreckPieceAt(x, y float32) {
	wreckedMutex.Lock()
	defer wreckedMutex.Unlock()
	for _, s := range wreckedSpots {
		if sameSpot(s.X, s.Y, x, y) {
			return
		}
	}
	wreckedSpots = append(wreckedSpots, struct{ X, Y float32 }{x, y})
}

// ClearWreckedPieces desfaz todas as trocas. Chamado no carregamento de mapa e
// no reinicio de fase: um canhao destruido numa tentativa perdida nao pode
// aparecer em ruinas na tentativa seguinte, que e a mesma razao pela qual
// `RestoreCannons` repoe o posto (internal/network/host_reset.go).
func ClearWreckedPieces() {
	wreckedMutex.Lock()
	defer wreckedMutex.Unlock()
	wreckedSpots = nil
}

func sameSpot(ax, ay, bx, by float32) bool {
	return float32(math.Abs(float64(ax-bx))) <= wreckMatchRadius &&
		float32(math.Abs(float64(ay-by))) <= wreckMatchRadius
}

// pieceNameAt devolve o nome que o objeto deve desenhar: o dele, ou a variante
// em ruinas quando o posto caiu e a variante existe no manifesto.
func (r *ManifestRenderer) pieceNameAt(name string, x, y float32) string {
	wreckedMutex.RLock()
	defer wreckedMutex.RUnlock()
	if len(wreckedSpots) == 0 {
		return name
	}
	for _, s := range wreckedSpots {
		if !sameSpot(s.X, s.Y, x, y) {
			continue
		}
		if _, ok := r.manifest.Pieces[name+wreckedSuffix]; ok {
			return name + wreckedSuffix
		}
		return name
	}
	return name
}
