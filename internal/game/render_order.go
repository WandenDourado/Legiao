package game

// A ORDEM EM QUE OS CORPOS SAO DESENHADOS TEM DE SER A MESMA TODO QUADRO.
//
// O renderer percorria `network.InterpolatedPlayers()` — um mapa — direto no
// laco de desenho. Um `for range` sobre mapa em Go da uma ordem ALEATORIA a
// cada passagem, e para desenhar isso e fatal: dois jogadores em cima um do
// outro trocavam de lugar na pilha a 60 vezes por segundo, e o que se ve na
// tela e um piscar entre os dois. Foi o relato do Gui em 27/08/2026, junto com
// o mesmo defeito nas magias sobrepostas (skill/draw_order.go e
// ability/registry.go).
//
// A ordem escolhida nao e so estavel, e a CERTA: por Y crescente, ou seja quem
// esta mais ao sul — mais perto da camera, nesta vista de cima — e desenhado
// por ultimo e aparece na frente. E a mesma leitura de profundidade que o
// `FootLine` dos monstros ja usa. O ID desempata, porque dois corpos podem
// dividir o mesmo Y e um empate tem de ser resolvido sempre do mesmo jeito.

import (
	"sort"

	"github.com/WandenDourado/Legiao/internal/network"
)

// drawablePlayer e um jogador remoto pronto para desenhar, com a chave que o
// desempata.
//
// O ID vem da CHAVE do mapa e nao do campo de dentro do estado: a chave e o
// que o resto do renderer ja usa para reconhecer o jogador local, e um campo
// que chegasse vazio pelo fio faria dois corpos empatarem para sempre.
type drawablePlayer struct {
	ID    string
	State network.PlayerState
}

// drawablePlayers devolve os jogadores REMOTOS visiveis, em ordem de
// profundidade estavel.
//
// Fora ficam dois casos, os mesmos de antes: o jogador local, que e desenhado
// pelo proprio `p.Draw()` no lugar que a profundidade dele mandar, e quem
// espera dentro de um portal — esse corpo sumiu de proposito, e o contador
// desenhado sobre o portal ja explica a ausencia.
func drawablePlayers(players map[string]network.PlayerState) []drawablePlayer {
	out := make([]drawablePlayer, 0, len(players))
	for id, state := range players {
		if id == network.LocalPlayerID || state.InPortal {
			continue
		}
		out = append(out, drawablePlayer{ID: id, State: state})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].State.Y != out[j].State.Y {
			return out[i].State.Y < out[j].State.Y
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// drawableEnemies devolve os monstros espelhados no CLIENTE em ordem estavel.
//
// Por tipo e depois por ID, que e EXATAMENTE a ordem que o host ja usa em
// `entity.visibleEnemies`. As duas telas mostram a mesma cena; elas nao podem
// empilhar os corpos de jeitos diferentes.
func drawableEnemies(enemies map[string]network.EnemyState) []network.EnemyState {
	out := make([]network.EnemyState, 0, len(enemies))
	for _, e := range enemies {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].EnemyID < out[j].EnemyID
	})
	return out
}

// drawableProjectileIDs devolve as chaves dos projeteis espelhados em ordem
// estavel, pelo mesmo motivo. Devolve as chaves e nao os valores porque quem
// desenha ja segura o mutex do mapa e le direto dele.
func drawableProjectileIDs(projectiles map[string]network.ProjectileState) []string {
	ids := make([]string, 0, len(projectiles))
	for id := range projectiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
