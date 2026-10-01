package skill

// A ORDEM DE DESENHO DAS MAGIAS TEM DE SER A MESMA TODO QUADRO.
//
// Quase toda colecao de efeito deste pacote e um `map[string]*Coisa`, e um
// `for range` sobre mapa em Go percorre em ordem ALEATORIA — de proposito, e
// diferente a cada passagem. Para simular tanto faz: a soma nao depende da
// ordem. Para DESENHAR, nao: duas magias sobrepostas trocavam de lugar na
// pilha a 60 vezes por segundo, e o que se ve na tela e um piscar entre as
// duas. Foi o relato do Gui em 27/08/2026, junto com o mesmo defeito nos
// jogadores empilhados (game/renderer.go).
//
// A correcao e ordenar por ID antes de desenhar. O ID de um efeito nunca muda
// depois de criado, entao a ordem RELATIVA de dois efeitos vivos e a mesma em
// todos os quadros — que e a unica propriedade que apaga o piscar. Nao e ordem
// de criacao (o ID e texto, entao "f10" vem antes de "f9") e nao precisa ser:
// o olho nao le a sequencia, so a estabilidade.
//
// Colecao que ja e SLICE (FireGrounds, Explosions, Thorns, Bursts) nao passa
// por aqui: ela ja tem uma ordem, e e a de insercao.

import "sort"

// orderedIDs devolve as chaves da colecao em ordem estavel.
//
// Aloca uma fatia de strings por chamada, uma vez por quadro por colecao. A
// maior delas e a Chuva de Meteoros, com regime de ~87 efeitos no ar
// (doc/skill_performance.md) — ao lado das ~2.500 particulas que a mesma
// magia desenha, isto e ruido.
func orderedIDs[T any](m map[string]T) []string {
	if len(m) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
