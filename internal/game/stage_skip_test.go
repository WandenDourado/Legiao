package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNextCampaignMap(t *testing.T) {
	cases := []struct {
		name   string
		from   string
		want   string
		wantOK bool
	}{
		{"a primeira fase leva a segunda", "assets/maps/world_01.json", "assets/maps/world_02.json", true},
		{"a segunda fase leva a terceira", "assets/maps/world_02.json", "assets/maps/world_03.json", true},
		{"a terceira fase leva a quarta", "assets/maps/world_03.json", "assets/maps/world_04.json", true},
		// A ultima fase nao tem para onde pular, e isso NAO e erro: F8 so
		// registra e deixa o jogador onde esta. O caso e escrito pelo FIM da
		// lista, e nao pelo nome do mapa, para nao voltar a mentir quando a
		// campanha ganhar a fase seguinte.
		{"a ultima fase nao pula", campaignMaps[len(campaignMaps)-1], "", false},
		// Um mapa fora da campanha (validacao de terreno, teste solto) nao tem
		// proxima fase definida. Adivinhar uma levaria o testador para um lugar
		// que ninguem pediu.
		{"mapa fora da campanha nao pula", "assets/maps/sandbox.json", "", false},
		{"caminho vazio nao pula", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := nextCampaignMap(tc.from)
			if ok != tc.wantOK || got != tc.want {
				t.Errorf("nextCampaignMap(%q) = (%q, %v), esperado (%q, %v)",
					tc.from, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestCampaignMapsAllExist(t *testing.T) {
	// Um caminho errado em campaignMaps so apareceria quando alguem apertasse
	// F8 e nada acontecesse — travelTo recusa carregar e devolve o mundo atual,
	// silenciosamente. Barato conferir aqui.
	for _, m := range campaignMaps {
		path := filepath.Join("..", "..", m)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("campaignMaps aponta para %s, que nao existe: %v", m, err)
		}
	}
}

func TestLastCampaignMapIsTheCampaignsFinalEntry(t *testing.T) {
	// Shift+F8 (jumpToLastCampaignMap) confia neste valor para levar o grupo
	// direto a fase mais recente, de qualquer mapa — inclusive um fora de
	// campaignMaps, onde nextCampaignMap nao teria de onde partir. Um valor
	// errado aqui levaria o testador para o lugar errado sem log nenhum
	// avisando, porque "" so acontece com a lista vazia.
	want := campaignMaps[len(campaignMaps)-1]
	if got := lastCampaignMap(); got != want {
		t.Errorf("lastCampaignMap() = %q, esperava %q (a ultima entrada de campaignMaps)", got, want)
	}
	if want != "assets/maps/world_06.json" {
		t.Errorf("a ultima fase da campanha e %q, nao world_06.json — Shift+F8 "+
			"ainda aponta para o mapa certo, mas este teste precisa saber quem "+
			"e o alvo esperado hoje", want)
	}
}

func TestCampaignMapsHasNoDuplicates(t *testing.T) {
	// Mapa repetido faria a busca linear parar na primeira ocorrencia, entao a
	// segunda metade da lista viraria inalcancavel por F8.
	seen := map[string]int{}
	for i, m := range campaignMaps {
		if first, dup := seen[m]; dup {
			t.Errorf("%s aparece nas posicoes %d e %d", m, first, i)
			continue
		}
		seen[m] = i
	}
}
