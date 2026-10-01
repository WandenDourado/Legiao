package game

// QUEM ja tem a suprema, em cada fase.
//
// A regra e uma frase: **o personagem ganha a dele na fase seguinte a cena em
// que ele se ergue**. O Necromante se levanta no mapa 2, entao a Legiao
// Espectral esta na mao a partir do mapa 3; a Sacerdotisa se levanta no 3 e o
// altar vale a partir do 4; o Arqueiro se levanta no 4 e as Flechas Celestiais
// valem no 5.
//
// Isto NAO e uma tabela nova. Ele e derivado das duas listas que ja existem —
// `campaignMaps`, que e a ordem das fases, e `lastStandHeroes`, onde cada fase
// declara o heroi dela. Uma terceira lista dizendo a mesma coisa seria uma
// lista para divergir no primeiro conserto, e este repositorio ja aprendeu isso
// com as hordas: `waveDefs` era global e toda fase nova herdava as hordas do
// mapa 1 ate alguem lembrar de editar a variavel.
//
// O efeito pratico e que uma FASE NOVA HERDA A REGRA SOZINHA: basta entrar em
// `campaignMaps` e declarar (ou nao) o heroi dela.
//
// Isto era um booleano — `UltimatesUnlockedOn`, "a partir do mapa 2 todo mundo
// tem a sua". Com ele, o grupo chegava ao mapa 3 com as tres supremas, e a
// dificuldade de cada fase, que e calibrada CONTRA as supremas que existem ali
// (orc no mapa 3 porque a Legiao apaga massa fraca; gargula no mapa 4 porque
// ela bate de fora do altar), media a coisa errada.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
)

// UltimatesGrantedOn devolve o conjunto de personagens que ja tem a suprema
// nesta fase.
//
// Mapa fora da campanha (sandbox, teste de terreno) libera TODO MUNDO, que e o
// comportamento que a versao booleana ja tinha: travar um mapa por onde ninguem
// progride so atrapalharia quem esta experimentando.
func UltimatesGrantedOn(path string) map[entity.CharacterType]bool {
	idx := campaignIndexOf(path)
	if idx < 0 {
		return allUltimates()
	}
	granted := make(map[entity.CharacterType]bool, idx)
	// So as fases ANTERIORES a esta contam: a cena da fase atual ainda nao
	// aconteceu, e quem se ergue nela nao pode chegar com a suprema na mao.
	for i := 0; i < idx; i++ {
		if char, ok := network.LastStandCharacterFor(campaignMaps[i]); ok {
			granted[char] = true
		}
	}
	return granted
}

// campaignIndexOf devolve a posicao da fase na campanha, ou -1.
func campaignIndexOf(path string) int {
	for i, m := range campaignMaps {
		if m == path {
			return i
		}
	}
	return -1
}

// allUltimates e o elenco inteiro liberado.
func allUltimates() map[entity.CharacterType]bool {
	defs := entity.AllCharacters()
	out := make(map[entity.CharacterType]bool, len(defs))
	for _, d := range defs {
		out[d.Type] = true
	}
	return out
}
