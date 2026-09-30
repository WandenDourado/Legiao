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
		t.Errorf("%s esta em climaxRuns; o climax daqui e a travessia do corredor, nao um cerco", mapPath)
	}
	if _, ok := garrisons[mapPath]; ok {
		t.Errorf("%s esta em garrisons; o mapa nao tem enemy_post_*", mapPath)
	}
	if _, ok := sentryPosts[mapPath]; ok {
		t.Errorf("%s esta em sentryPosts; a gargula daqui e o canhao, nao a sentinela ranged normal", mapPath)
	}
}

// TestMap6DeclaresPaladinaAsLastStandHero guarda a ligacao entre o mapa e quem
// se ergue. Desde 27/08/2026 ela vale por um motivo mais forte do que antes: a
// cena nao destroi mais os canhoes: quem os derruba e a espada da Paladina,
// atravessando sob o Avatar dos Deuses. Se o heroi do ultimo suspiro do mapa 6
// deixar de ser ela, o resgate devolve o grupo num corredor que ninguem em
// campo tem a ferramenta de calar.
func TestMap6DeclaresPaladinaAsLastStandHero(t *testing.T) {
	hero, ok := LastStandCharacterFor("assets/maps/world_06.json")
	if !ok {
		t.Fatal("world_06.json nao declara heroi do ultimo suspiro")
	}
	if hero != entity.CharPaladina {
		t.Errorf("o heroi do mapa 6 e %s, esperava %s", hero, entity.CharPaladina)
	}
}

// TestTheCorridorCannotBeTankedWithoutTheAvatar guarda a regra que faz o mapa
// 6 ser o mapa 6: NAO DA PARA ATRAVESSAR NA FORCA BRUTA.
//
// A conta e a travessia inteira, e nao um ciclo de escudo solto — o teste
// anterior comparava um ciclo de recarga contra a vida cheia, e essa
// aproximacao deixou de dizer alguma coisa quando a Paladina passou a ter 300
// de vida e a curar 20% a cada lancamento. O que importa e se o orcamento
// dela cobre o caminho de CannonRange ate os postos.
//
// Se um dia cobrir, a fase muda de natureza sem ninguem editar uma linha de
// roteiro: o Avatar dos Deuses deixa de ser o pedagio da travessia e a cena do
// ultimo suspiro deixa de ter para que servir.
func TestTheCorridorCannotBeTankedWithoutTheAvatar(t *testing.T) {
	// Velocidade do jogador: literal em entity/player.go (PlayerSpeed).
	const playerSpeed float32 = 200

	// Quanto tempo ela leva para cruzar o alcance dos canhoes andando.
	crossing := CannonRange / playerSpeed
	// Cada salva traz uma bola por canhao, e sao dois postos.
	salvos := crossing / CannonCooldown
	incoming := salvos * 2 * CannonDamage

	// Tudo o que ela tem para gastar nesse tempo: a vida cheia mais um escudo
	// e uma cura por recarga.
	paladina := entity.MaxHealthFor(entity.CharPaladina)
	casts := crossing / skill.ShieldCooldown
	budget := paladina + casts*(skill.ShieldMaxHP+paladina*skill.ShieldCastHeal)

	if incoming <= budget {
		t.Errorf("atravessar o alcance dos canhoes leva %.0fs e custa %.0f de dano; "+
			"a Paladina tem %.0f para gastar (%.0f de vida + %.1f escudos de %.0f "+
			"com %.0f de cura). Ela atravessa o corredor no peito, e o Avatar dos "+
			"Deuses deixou de ser necessario",
			crossing, incoming, budget, paladina, casts, skill.ShieldMaxHP,
			paladina*skill.ShieldCastHeal)
	}
}

// TestCannonFallsToTwoEmpoweredSwings guarda a unica rota que existe para
// calar o corredor: atravessar sob o Avatar dos Deuses e bater.
//
// Os tres numeros so valem juntos. Se alguem subir CannonMaxHP, ou baixar
// SwordAvatarDamage, ou encurtar AvatarDuration, a travessia deixa de caber na
// suprema — e como o Avatar recarrega em 60 s DENTRO do alcance dos canhoes,
// nao caber nao e "ficou dificil", e "a fase travou".
func TestCannonFallsToTwoEmpoweredSwings(t *testing.T) {
	swingsPerCannon := CannonMaxHP / skill.SwordAvatarDamage
	if swingsPerCannon > 3 {
		t.Errorf("cada canhao pede %.1f golpes supremos (%.0f de vida / %.0f de dano); "+
			"mais de 3 nao cabe na janela do Avatar junto com a caminhada",
			swingsPerCannon, CannonMaxHP, skill.SwordAvatarDamage)
	}
	// E a espada NORMAL nao pode dar conta: se desse, a suprema deixaria de ser
	// o pedagio da travessia e o corredor viraria paciencia.
	if CannonMaxHP/skill.SwordDamage < 4 {
		t.Errorf("a espada sem suprema derruba um canhao em %.1f golpes; "+
			"barato demais para um corredor que cobra %.0f de dano a cada %.1fs",
			CannonMaxHP/skill.SwordDamage, CannonDamage, CannonCooldown)
	}
}

// Aqui morava TestCannonBallsSealTheCorridor, que exigia que o CORPO da bola
// cobrisse os 768 px da faixa caminhavel. Ele ja estava falhando: o raio da
// bola foi baixado para 350 (diametro 700) justamente para "permitir alguma
// esquiva entre as duas bolas", como o comentario de CannonBallRadius diz — e
// o teste continuou cobrando o contrario.
//
// Quem sela o corredor agora e o ESTOURO, e nao o corpo da bola: a bola pode
// ser esquivada, a explosao dela nao. E isso que o teste abaixo guarda.

// O estouro e um circulo centrado em QUEM FOI ATINGIDO, e as duas medidas
// abaixo so significam alguma coisa junto com isso
// (network.cannonBallImpactPoint, que poe a explosao sobre o corpo em vez de
// onde a bola estava):
//
//  1. O raio precisa pegar de PAREDE A PAREDE, senao andar colado na parede
//     vira a resposta e a bola deixa de ser parede de fogo.
//  2. E nao pode pegar o COMPRIMENTO do corredor, que e a direcao em que o
//     grupo de fato se espalha: uma explosao que alcanca todo mundo
//     independente de onde estao apaga a unica coisa que o grupo pode fazer a
//     respeito.
//
// Quem esta ATRAS de quem foi atingido sangra por simetria — o circulo se abre
// metade para a frente e metade para tras a partir do corpo acertado. Isso e
// consequencia de ONDE a explosao nasce, e nao de uma medida propria: tentei
// resolver com uma capsula esticada para a frente em 28/08/2026 e era remendo
// para uma explosao que nascia no lugar errado.
func TestCannonBlastSpansTheCorridorButNotItsLength(t *testing.T) {
	const corridorWidth float32 = 768
	if skill.CannonBlastRadius*2 < corridorWidth {
		t.Errorf("o estouro tem %.0f px de largura contra %.0f px de faixa "+
			"caminhavel: andar colado na parede sai ileso",
			skill.CannonBlastRadius*2, corridorWidth)
	}
	if skill.CannonBlastRadius*2 > CannonRange/2 {
		t.Errorf("o estouro tem %.0f px de diametro contra %.0f px de alcance do "+
			"canhao: espalhar ao longo do corredor deixou de ser resposta",
			skill.CannonBlastRadius*2, CannonRange)
	}
}

// O ESTOURO ALCANCA MUITO MAIS DO QUE A BOLA ENCOSTA, e essa folga e o desenho
// inteiro: ela e o quanto a explosao sobra para os lados de quem foi atingido.
//
// A bola encosta a CannonBallRadius+34 do corpo e a explosao nasce SOBRE ele,
// abrindo CannonBlastRadius. A folga (630 - 384 = 246 px por lado, alem do
// corpo acertado) e o que faz quem esta atras sangrar. Se o estouro encolher
// ate o tamanho do toque, volta o defeito de 28/08/2026: so quem a bola
// encostou se machuca.
func TestTheBlastReachesPastTheBodyItHit(t *testing.T) {
	touch := skill.CannonBallRadius + cannonBallPlayerRadius
	if skill.CannonBlastRadius <= touch {
		t.Errorf("a bola encosta a %.0f px e o estouro alcanca %.0f px: a explosao "+
			"nao passa de quem foi atingido, e quem esta atras nao sangra",
			touch, skill.CannonBlastRadius)
	}
}

// A explosao tem de sair da tela antes da proxima salva. Se durar mais que a
// recarga, os estouros se empilham e o corredor vira uma mancha de fogo
// continua, em que nao da para ler onde caiu cada tiro — e o jogador perde a
// unica pista de para onde correr.
func TestBurstFadesBeforeTheNextSalvo(t *testing.T) {
	if skill.CannonBallBurstTTL >= CannonCooldown {
		t.Errorf("a explosao dura %.2fs e a salva vem a cada %.2fs: o fogo de uma "+
			"nao sai da tela antes da proxima",
			skill.CannonBallBurstTTL, CannonCooldown)
	}
}

// O CANHAO ATIRA DESDE O PRIMEIRO QUADRO DA FASE, mesmo com o grupo longe
// (network.cannonAim) — o corredor tem de LER como corredor de canhao antes de
// alguem entrar no alcance. O que faz esse tiro de aviso nao virar dano de
// graca e o tempo de voo da bola: ela morre pouco depois do fim do alcance.
//
// Se a bola passar a viajar bem mais do que o alcance, o "aviso" comeca a
// matar gente que o canhao nem deveria estar mirando, e a regra "o alcance e a
// linha em que se comeca a sangrar" cai sem ninguem editar CannonRange.
func TestBlindShotDiesAroundTheEdgeOfTheRange(t *testing.T) {
	flight := skill.CannonBallTTL * skill.CannonBallSpeed
	if flight < CannonRange {
		t.Errorf("a bola viaja %.0f px e o canhao alcanca %.0f: ela morre antes "+
			"de cruzar o proprio alcance", flight, CannonRange)
	}
	// A folga aceitavel e um estouro: mais do que isso e o tiro as cegas
	// alcanca quem esta fora do alcance declarado.
	if slack := flight - CannonRange; slack > skill.CannonBlastRadius {
		t.Errorf("a bola viaja %.0f px alem do alcance de %.0f (folga %.0f, "+
			"estouro %.0f): o tiro de aviso passou a machucar fora do alcance",
			flight, CannonRange, slack, skill.CannonBlastRadius)
	}
}

// A barra de vida do canhao tem de ficar ACIMA da peca, e nao dentro dela. A
// altura sai do manifesto do tileset, que e o unico lugar onde o tamanho da
// arte existe de verdade — se alguem redesenhar o canhao mais alto e so
// remedir o manifesto, a barra some dentro da cabeca sem ninguem notar.
func TestCannonBarClearsTheArt(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "assets", "castle_cannons_manifest.json"))
	if err != nil {
		t.Skipf("sem castle_cannons_manifest.json (%v)", err)
	}
	var m struct {
		Pieces map[string]struct {
			Source struct{ Width, Height float32 } `json:"source"`
		} `json:"pieces"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("castle_cannons_manifest.json ilegivel: %v", err)
	}
	piece, ok := m.Pieces["cannon_west"]
	if !ok {
		t.Fatal("o manifesto nao tem cannon_west; a barra nao tem contra o que se medir")
	}
	if CannonBarAbove <= piece.Source.Height {
		t.Errorf("a barra fica %.0f px acima do chao e a peca tem %.0f px de altura: "+
			"a barra desenha dentro do canhao", CannonBarAbove, piece.Source.Height)
	}
	if CannonBarHalfWidth*2 > piece.Source.Width*1.2 {
		t.Errorf("a barra tem %.0f px de largura contra %.0f px de peca: ela "+
			"transborda o canhao e passa a apontar para o chao ao lado",
			CannonBarHalfWidth*2, piece.Source.Width)
	}
}
