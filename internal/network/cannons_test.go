package network

// O canhao do mapa 6 nao tem tabela de postos, ao contrario da gargula
// sentinela e da guarnicao: `InstallArrivalCannons` recebe os marcadores
// `enemy_cannon_*` direto do mapa (ver internal/game/world_state.go), porque
// so existe UM mapa com canhao ate agora. O que este arquivo guarda em vez
// disso e o que quebraria em silencio se isso mudasse: um canhao aparecendo
// (ou desaparecendo) num mapa sem ninguem perceber, o mapa 6 sem heroi do
// ultimo suspiro, e o numero que faz o briefing do Gui ser verdade — "causam
// muito dano" — deixando de ser verdade porque alguem aliviou uma constante.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"
)

// TestOnlyMap6HasCannonMarkers guarda que o canhao e uma coisa do corredor
// final. Um `enemy_cannon_*` desenhado por engano em outro mapa da campanha
// ficaria mudo hoje (nada le esse marcador fora do carregamento do mapa 6),
// o que e exatamente o tipo de defeito silencioso que os testes irmaos
// (TestSentryPostsExistInMap, TestGarrisonPostsExistInMap) existem para
// pegar do outro lado.
//
// `campaignMaps` mora no pacote game (stage_skip.go), que importa network — e
// nao o contrario — entao este teste nao pode le-la, e varre todos os
// arquivos de assets/maps direto do disco.
func TestOnlyMap6HasCannonMarkers(t *testing.T) {
	maps, err := filepath.Glob(filepath.Join("..", "..", "assets", "maps", "*.json"))
	if err != nil || len(maps) == 0 {
		t.Fatalf("nao consegui listar assets/maps: %v", err)
	}
	for _, abs := range maps {
		mapPath := "assets/maps/" + filepath.Base(abs)
		names := cannonNamesInMap(t, mapPath)
		if mapPath == "assets/maps/world_06.json" {
			if len(names) != 2 {
				t.Errorf("%s: esperava 2 marcadores enemy_cannon_*, achei %d (%v)",
					mapPath, len(names), names)
			}
			continue
		}
		if len(names) != 0 {
			t.Errorf("%s: tem marcador(es) enemy_cannon_* (%v), mas so o mapa 6 "+
				"instala canhoes; eles ficariam parados e mudos em campo", mapPath, names)
		}
	}
}

// cannonNamesInMap devolve os sufixos dos marcadores enemy_cannon_* do
// arquivo. Le o JSON direto, sem passar pelo tilemap, pelo mesmo motivo que
// postNamesInMap e sentryNamesInMap: o arquivo e a fonte.
func cannonNamesInMap(t *testing.T, mapPath string) map[string]bool {
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
			if !strings.HasPrefix(obj.Name, "enemy_cannon_") {
				continue
			}
			names[strings.TrimPrefix(obj.Name, "enemy_cannon_")] = true
		}
	}
	return names
}

// TestMap6HasNoHordeOrGarrison guarda a diferenca do mapa 6 declarada no
// cabecalho de build_world_06.py e no comentario de cannons.go: a pressao
// inteira vem dos dois canhoes, que atiram desde a chegada — nao ha corrida
// de hordas nem guarnicao aqui. Uma entrada esquecida numa dessas tabelas
// tentaria armar uma horda ou uma guarnicao num mapa sem os marcadores para
// isso (TestWaveTypeOrderCoversEveryComposition e TestGarrisonPostsExistInMap
// ja pegam o caso oposto), entao aqui o teste e so: nenhuma delas conhece o
// mapa 6.
func TestMap6HasNoHordeOrGarrison(t *testing.T) {
	const mapPath = "assets/maps/world_06.json"
	if _, ok := waveRuns[mapPath]; ok {
		t.Errorf("%s esta em waveRuns; o mapa nao tem enemy_spawn_* para armar", mapPath)
	}
	if _, ok := climaxRuns[mapPath]; ok {
		t.Errorf("%s esta em climaxRuns; o climax daqui e o julgamento da Paladina, nao um cerco", mapPath)
	}
	if _, ok := garrisons[mapPath]; ok {
		t.Errorf("%s esta em garrisons; o mapa nao tem enemy_post_*", mapPath)
	}
	if _, ok := sentryPosts[mapPath]; ok {
		t.Errorf("%s esta em sentryPosts; a gargula daqui e o canhao, nao a sentinela ranged normal", mapPath)
	}
}

// TestMap6DeclaresPaladinaAsLastStandHero guarda a ligacao entre o mapa e
// quem se ergue: o resgate roteirizado (castCannonJudgment) so faz sentido
// se o heroi do ultimo suspiro do mapa 6 for a Paladina.
func TestMap6DeclaresPaladinaAsLastStandHero(t *testing.T) {
	hero, ok := LastStandCharacterFor("assets/maps/world_06.json")
	if !ok {
		t.Fatal("world_06.json nao declara heroi do ultimo suspiro")
	}
	if hero != entity.CharPaladina {
		t.Errorf("o heroi do mapa 6 e %s, esperava %s", hero, entity.CharPaladina)
	}
}

// TestCannonsOutrunEscudoSagrado guarda o numero que faz o briefing do Gui
// verdade: "os jogadores provavelmente vao tentar usar o escudo... mas mesmo
// com uma boa estrategia eles so vao conseguir chegar ate proximo da metade".
//
// Isso so e verdade se um UNICO jogador parado sob fogo, reforcando o Escudo
// Sagrado toda vez que ele recarrega, ainda assim perder mais vida do que o
// escudo consegue repor num ciclo de recarga. Cada salva simultânea traz dois
// impactos a cada CannonCooldown; o teste soma o dano desse intervalo contra
// ShieldMaxHP e falha se algum dos dois lados mudar sem o outro perceber.
func TestCannonsOutrunEscudoSagrado(t *testing.T) {
	hitsPerShieldCycle := skill.ShieldCooldown / CannonCooldown * 2
	incoming := hitsPerShieldCycle * CannonDamage
	// PlayerMaxHealth e um literal espalhado por entity/player.go e host.go
	// (nao ha constante exportada); 100 e o valor que os dois usam hoje.
	const playerMaxHealth float32 = 100
	if incoming <= skill.ShieldMaxHP {
		t.Fatalf("um ciclo de recarga do Escudo Sagrado (%.1fs) so leva %.0f de "+
			"dano dos dois canhoes, e o escudo absorve %.0f: o escudo sozinho "+
			"aguentaria o corredor inteiro", skill.ShieldCooldown, incoming, skill.ShieldMaxHP)
	}
	if incoming <= skill.ShieldMaxHP+playerMaxHealth {
		t.Errorf("um ciclo de recarga do escudo (%.1fs) leva %.0f de dano; isso "+
			"mal passa do escudo (%.0f) mais a vida cheia (%.0f) — o briefing pede "+
			"'muito dano', nao um aperto", skill.ShieldCooldown, incoming, skill.ShieldMaxHP, playerMaxHealth)
	}
}

// TestCannonBallsSealTheCorridor keeps the wall-of-fire geometry tied to the
// actual walkable width between the two cannon posts.
func TestCannonBallsSealTheCorridor(t *testing.T) {
	const corridorWidth float32 = 768
	if skill.CannonBallRadius*2 < corridorWidth {
		t.Fatalf("diametro da bola = %.0f, menor que a largura do corredor = %.0f; sobra uma rota de esquiva",
			skill.CannonBallRadius*2, corridorWidth)
	}
}
