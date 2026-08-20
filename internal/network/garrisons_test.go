package network

// Os dois testes deste arquivo vieram de defeitos que chegaram ate o jogo, e
// os dois falhavam EM SILENCIO: nenhum crash, nenhum erro de compilacao, so a
// fase se comportando de um jeito que ninguem pediu.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
)

// TestWaveTypeOrderCoversEveryComposition guarda o sorteio das hordas.
//
// `takePending` percorre `waveTypeOrder` e nao o mapa `pending`, para o sorteio
// ser reproduzivel. O preco disso e que um tipo fora da lista fica invisivel: a
// funcao nunca o devolve, e `pendingTotal()` continua contando — entao a horda
// nao so nasce incompleta como NUNCA TERMINA, porque o total pendente jamais
// chega a zero.
//
// Foi o que aconteceu com o orc na emboscada do mapa 3: oito orcs no papel,
// nenhum em campo, e o grupo matando slime e lobo sem que a fase avancasse.
func TestWaveTypeOrderCoversEveryComposition(t *testing.T) {
	known := map[entity.EnemyType]bool{}
	for _, kind := range waveTypeOrder {
		known[kind] = true
	}
	check := func(label string, runs map[string][]WaveDef) {
		for mapPath, defs := range runs {
			for _, def := range defs {
				for kind, n := range def.Composition {
					if n > 0 && !known[kind] {
						t.Errorf("%s %s (%s) pede %d de %q, que nao esta em "+
							"waveTypeOrder: eles nunca nascem e a horda nunca "+
							"termina", label, mapPath, def.Name, n, kind)
					}
				}
			}
		}
	}
	check("waveRuns", waveRuns)
	check("climaxRuns", climaxRuns)
}

// TestGarrisonPostsExistInMap guarda a ligacao entre a tabela e o mapa.
//
// O nome do posto e a UNICA coisa que liga `garrisons.go` ao arquivo do mapa, e
// os dois lados ja divergiram de duas maneiras: uma linha de barricada com dois
// vaos gerou dois marcadores com o mesmo nome, e depois `EnemyPostPoints`
// passou a devolver o nome com o prefixo `enemy_post_` enquanto a tabela usava
// so o sufixo. Nos dois casos a fase carregou sem um monstro.
//
// O teste le o JSON do mapa direto, sem passar pelo tilemap, de proposito: e o
// arquivo que e a fonte, e um bug no leitor nao pode esconder um bug nos dados.
func TestGarrisonPostsExistInMap(t *testing.T) {
	for mapPath, squads := range garrisons {
		names := postNamesInMap(t, mapPath)
		if len(names) == 0 {
			t.Fatalf("%s nao tem nenhum marcador enemy_post_*", mapPath)
		}
		seen := map[string]bool{}
		for _, squad := range squads {
			if !names[squad.Post] {
				t.Errorf("%s: a guarnicao pede o posto %q, que o mapa nao tem",
					mapPath, squad.Post)
			}
			if seen[squad.Post] {
				t.Errorf("%s: o posto %q aparece duas vezes na guarnicao",
					mapPath, squad.Post)
			}
			seen[squad.Post] = true
			if squad.Total() == 0 {
				t.Errorf("%s: o posto %q nao poe ninguem em campo",
					mapPath, squad.Post)
			}
		}
		for name := range names {
			if !seen[name] {
				t.Errorf("%s: o marcador %q nao tem guarnicao; ele fica vazio "+
					"em jogo", mapPath, name)
			}
		}
	}
}

// postNamesInMap devolve os sufixos dos marcadores enemy_post_* do arquivo.
func postNamesInMap(t *testing.T, mapPath string) map[string]bool {
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
			if !strings.HasPrefix(obj.Name, "enemy_post_") {
				continue
			}
			names[strings.TrimPrefix(obj.Name, "enemy_post_")] = true
		}
	}
	return names
}

// TestClimaxIsUnwinnableWithoutTheRescue guarda a forma do climax do mapa 3.
//
// O pedido do Gui e uma frase: "o climax deve ser impossivel de passar sem a
// ultimate da Sacerdotisa". Em codigo isso e uma propriedade so — a horda tem
// de se REPOR — e ela e facil de perder sem querer: basta alguem trocar
// `Endless` por um total maior achando que esta subindo a dificuldade. Uma
// horda finita, por maior que seja, e uma questao de aguentar; a diferenca
// entre as duas coisas nao aparece em nenhum outro lugar do codigo.
func TestClimaxIsUnwinnableWithoutTheRescue(t *testing.T) {
	for mapPath, defs := range climaxRuns {
		if len(defs) == 0 {
			t.Errorf("%s tem climaxRuns vazio", mapPath)
			continue
		}
		endless := false
		for _, def := range defs {
			if def.Endless {
				endless = true
			}
		}
		if !endless {
			t.Errorf("%s: nenhuma horda do climax se repoe, entao o grupo "+
				"pode vencer matando o ultimo — e a ultimate da Sacerdotisa "+
				"deixa de ser a unica saida", mapPath)
		}
	}
}

// TestClimaxUsesTheStrongestEnemy: o cerco e feito de orc.
//
// Nao e gosto: o orc e a unica criatura cujos numeros foram calibrados contra
// uma ultimate (ver entity/orc_legion_test.go). Um cerco de slime e lobo seria
// atravessavel na forca bruta, e o climax voltaria a ser uma horda comum.
func TestClimaxUsesTheStrongestEnemy(t *testing.T) {
	for mapPath, defs := range climaxRuns {
		for _, def := range defs {
			if !def.Endless {
				continue
			}
			if def.Composition[entity.EnemyTypeGarrison] == 0 {
				t.Errorf("%s (%s): o cerco nao tem orc", mapPath, def.Name)
			}
		}
	}
}
