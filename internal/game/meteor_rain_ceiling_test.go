package game

// NENHUM MAPA DA CAMPANHA ESTOURA O TETO DE DESEMPENHO DA CHUVA DE METEOROS.
//
// MeteorRainIntervalFor (internal/skill/meteor.go) escala o intervalo de
// nascimento pela area do mapa para manter a densidade constante — ver
// doc/combat_rules.md ("A chuva mede-se por densidade, nao por contagem") e
// internal/skill/meteor_scale_test.go, que testa a funcao isoladamente com
// areas sinteticas. Este teste e o complemento: ele le os arquivos de mapa
// REAIS da campanha (a mesma lista que TestEveryCampaignPortalLeadsToTheNextPhase
// usa) e confirma que nenhum, hoje, produz um intervalo abaixo do teto de
// desempenho medido (doc/performance.md secao 8).
//
// O teto e aplicado DENTRO de MeteorRainIntervalFor (MeteorRainMinInterval),
// entao este teste nao pode falhar por um mapa maior que o world_05 — a
// funcao ja trava nesse caso. O que ele pega e diferente: se algum dia o
// clamp for removido ou contornado (por exemplo, um chamador que multiplica o
// intervalo por conta propria em vez de usar MeteorRainIntervalFor), a queda
// de fps aparece aqui como teste vermelho, e nao como reclamacao de fps num
// mapa novo.
import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/WandenDourado/Legiao/internal/skill"
)

func TestNoCampaignMapBreaksTheMeteorRainPerformanceCeiling(t *testing.T) {
	for _, mapPath := range campaignMaps {
		w, h := mapWorldSize(t, mapPath)
		iv := skill.MeteorRainIntervalFor(w, h)
		if iv < skill.MeteorRainMinInterval {
			t.Errorf("%s (%.0fx%.0f px): MeteorRainIntervalFor devolveu %.6f, "+
				"abaixo do teto de desempenho %.6f (doc/performance.md secao 8)",
				mapPath, w, h, iv, skill.MeteorRainMinInterval)
		}
	}
}

// mapWorldSize le width/height/tilewidth/tileheight direto do JSON do mapa e
// devolve o tamanho do mundo em pixels — o mesmo calculo que
// world.NewBoundsFromMap faz a partir do tilemap.TiledMap carregado.
func mapWorldSize(t *testing.T, mapPath string) (float32, float32) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", mapPath))
	if err != nil {
		t.Fatalf("nao consegui ler %s: %v", mapPath, err)
	}
	var file struct {
		Width      int `json:"width"`
		Height     int `json:"height"`
		TileWidth  int `json:"tilewidth"`
		TileHeight int `json:"tileheight"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("%s nao e JSON valido: %v", mapPath, err)
	}
	if file.Width <= 0 || file.Height <= 0 || file.TileWidth <= 0 || file.TileHeight <= 0 {
		t.Fatalf("%s: grade ou tile invalidos (width=%d height=%d tilewidth=%d tileheight=%d)",
			mapPath, file.Width, file.Height, file.TileWidth, file.TileHeight)
	}
	return float32(file.Width * file.TileWidth), float32(file.Height * file.TileHeight)
}
