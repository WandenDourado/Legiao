package network

// A gargula liga tres coisas que nao se conhecem: a tabela de postos
// (`sentryPosts`), o arquivo do mapa (`enemy_sentry_*`) e quem pede sentinela
// (`sentriesOnArrival` e `WaveDef.Sentries`). Como no caso da guarnicao, o nome
// e a UNICA amarra entre elas, e a divergencia falha em silencio: a fase
// simplesmente fica mais facil do que foi escrita, sem erro nenhum.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSentryPostsExistInMap: todo posto declarado tem marcador, e todo
// marcador tem posto.
//
// A segunda metade importa tanto quanto a primeira. Um `enemy_sentry_*` no mapa
// que ninguem declara e uma gargula que o level designer desenhou e que nunca
// vai existir em jogo — o oposto exato do defeito que um teste normalmente
// procura, e igualmente invisivel.
func TestSentryPostsExistInMap(t *testing.T) {
	for mapPath, posts := range sentryPosts {
		names := sentryNamesInMap(t, mapPath)
		if len(names) == 0 {
			t.Fatalf("%s nao tem nenhum marcador enemy_sentry_*", mapPath)
		}
		seen := map[string]bool{}
		for _, post := range posts {
			if !names[post] {
				t.Errorf("%s: a tabela pede o posto %q, que o mapa nao tem", mapPath, post)
			}
			if seen[post] {
				t.Errorf("%s: o posto %q aparece duas vezes na tabela", mapPath, post)
			}
			seen[post] = true
		}
		for name := range names {
			if !seen[name] {
				t.Errorf("%s: o marcador enemy_sentry_%s nao esta na tabela; "+
					"ele nunca vai ser ocupado", mapPath, name)
			}
		}
	}
}

// TestNobodyAsksForMoreSentriesThanTheMapDeclares.
//
// `armSentries` corta o excesso e registra um log, mas um log e um relatorio de
// bug tardio: a horda final do mapa 5 pedindo mais gargulas do que existem sai
// mais facil do que foi escrita e ninguem percebe.
func TestNobodyAsksForMoreSentriesThanTheMapDeclares(t *testing.T) {
	want := func(mapPath string, n int, who string) {
		if have := len(SentryPostsFor(mapPath)); n > have {
			t.Errorf("%s: %s pede %d sentinelas e o mapa declara %d postos",
				mapPath, who, n, have)
		}
	}
	for mapPath, n := range sentriesOnArrival {
		want(mapPath, n, "a chegada")
	}
	for _, runs := range []map[string][]WaveDef{waveRuns, climaxRuns} {
		for mapPath, defs := range runs {
			for _, def := range defs {
				if def.Sentries > 0 {
					want(mapPath, def.Sentries, def.Name)
				}
			}
		}
	}
}

// TestSentryOrdersOnlyGrowWithinARun.
//
// `WaveDef.Sentries` e um TOTAL ACUMULADO, e nao um acrescimo — quem arma
// preenche a diferenca ate o numero pedido. Uma horda que pedisse menos que a
// anterior nao tiraria gargula nenhuma de campo; ela simplesmente nao faria
// nada, e quem escreveu a tabela achando que estava aliviando a fase nao teria
// como saber.
func TestSentryOrdersOnlyGrowWithinARun(t *testing.T) {
	for mapPath, defs := range waveRuns {
		previous := 0
		for _, def := range defs {
			if def.Sentries == 0 {
				continue
			}
			if def.Sentries < previous {
				t.Errorf("%s (%s): pede %d sentinelas depois de uma horda que "+
					"ja pedia %d; o campo nunca encolhe, entao isto nao faz nada",
					mapPath, def.Name, def.Sentries, previous)
			}
			previous = def.Sentries
		}
	}
}

// TestSentriesArriveAfterThePhaseHasIntroducedItself.
//
// Pedido do Gui, e vale escrever: no mapa de hordas a gargula nao entra na
// primeira onda. Uma torre de alcance 1900 antes de a fase ter dito o que ela e
// nao le como virada, le como armadilha.
func TestSentriesArriveAfterThePhaseHasIntroducedItself(t *testing.T) {
	for mapPath, defs := range waveRuns {
		if len(defs) > 0 && defs[0].Sentries > 0 {
			t.Errorf("%s: a primeira horda ja traz %d gargula(s)",
				mapPath, defs[0].Sentries)
		}
	}
}

// sentryNamesInMap devolve os sufixos dos marcadores enemy_sentry_* do arquivo.
// Le o JSON direto, sem passar pelo tilemap, pelo mesmo motivo que
// `postNamesInMap`: o arquivo e a fonte, e um bug no leitor nao pode esconder
// um bug nos dados.
func sentryNamesInMap(t *testing.T, mapPath string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", mapPath))
	if err != nil {
		t.Fatalf("nao consegui ler %s: %v", mapPath, err)
	}
	var file struct {
		Layers []struct {
			Name    string `json:"name"`
			Type    string `json:"type"`
			Objects []struct {
				Name string `json:"name"`
			} `json:"objects"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("%s nao e JSON valido: %v", mapPath, err)
	}
	names := map[string]bool{}
	for _, layer := range file.Layers {
		if layer.Type != "objectgroup" || layer.Name != "spawn" {
			continue
		}
		for _, obj := range layer.Objects {
			if !strings.HasPrefix(obj.Name, "enemy_sentry_") {
				continue
			}
			names[strings.TrimPrefix(obj.Name, "enemy_sentry_")] = true
		}
	}
	return names
}
