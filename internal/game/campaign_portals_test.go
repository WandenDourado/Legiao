package game

// O PORTAL DE CADA FASE TEM DE LEVAR A FASE SEGUINTE.
//
// Isto parece obvio e ficou errado por meses: o portal do world_02 apontava
// para o world_01 desde quando a mata era a ultima fase da campanha, e continuou
// apontando depois de o 3, o 4 e o 5 existirem. Quem terminava o mapa 2 era
// mandado de volta para a vila.
//
// Nada pegava porque HA DUAS FONTES DE VERDADE para a ordem da campanha, e elas
// nao se conversavam: `campaignMaps`, que o F8 percorre, e a propriedade
// `target_map` de cada portal, que e o que o jogador usa. Quem testava com F8
// via a campanha inteira funcionando. O comentario em stage_skip.go chegou a
// DOCUMENTAR a divergencia em vez de corrigi-la.
//
// Este teste amarra as duas. A ultima fase e a excecao declarada: ela volta ao
// comeco de proposito.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEveryCampaignPortalLeadsToTheNextPhase(t *testing.T) {
	for i, mapPath := range campaignMaps {
		targets := portalTargetsInMap(t, mapPath)
		if len(targets) == 0 {
			t.Errorf("%s nao tem portal nenhum; a fase nao tem saida", mapPath)
			continue
		}
		want := campaignMaps[0] // a ultima fase volta ao comeco
		if i+1 < len(campaignMaps) {
			want = campaignMaps[i+1]
		}
		for _, got := range targets {
			if got != want {
				t.Errorf("%s: o portal leva a %s, e a fase seguinte e %s",
					mapPath, got, want)
			}
		}
	}
}

func TestEveryPortalTargetExists(t *testing.T) {
	// Destino escrito com erro de digitacao nao quebra nada no carregamento —
	// `TryLoadWorld` devolve nil e o jogador simplesmente nao atravessa. Um
	// portal que nao faz nada le como bug de colisao, e a causa fica a dois
	// arquivos de distancia.
	for _, mapPath := range campaignMaps {
		for _, target := range portalTargetsInMap(t, mapPath) {
			if _, err := os.Stat(filepath.Join("..", "..", target)); err != nil {
				t.Errorf("%s aponta para %s, que nao existe no disco", mapPath, target)
			}
		}
	}
}

// portalTargetsInMap le o `target_map` de cada portal direto do arquivo. Sem
// passar pelo tilemap de proposito: o arquivo e a fonte, e um bug no leitor nao
// pode esconder um bug nos dados.
func portalTargetsInMap(t *testing.T, mapPath string) []string {
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
				Name       string `json:"name"`
				Properties []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"properties"`
			} `json:"objects"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("%s nao e JSON valido: %v", mapPath, err)
	}
	var out []string
	for _, layer := range file.Layers {
		if layer.Type != "objectgroup" || layer.Name != "portals" {
			continue
		}
		for _, obj := range layer.Objects {
			for _, prop := range obj.Properties {
				if prop.Name == "target_map" {
					out = append(out, prop.Value)
				}
			}
		}
	}
	return out
}
