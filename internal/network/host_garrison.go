package network

// Instala a guarnicao de um mapa de defesa de territorio.
//
// A diferenca para `StartWaveRun` e o relogio: uma corrida de hordas nasce ao
// longo do tempo e o mapa fica vazio ate a primeira onda; uma guarnicao ja
// esta em campo quando o grupo entra. Nao ha marcha de aquecimento — foi a
// primeira coisa que o Gui pediu para este mapa.

import (
	"log"
	"math"
	"math/rand"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// guardRadiusFor is the individual reach of a post, in world units.
//
// 2600 px. Foram 640, depois 1100, depois 1700, e as tres vezes o raio ficou
// curto pelo mesmo erro de raciocinio: eu tratava a distancia entre duas
// barricadas (1280 px) como teto, com medo de dois postos vizinhos acordarem
// juntos.
//
// Esse teto nao existe. Quem impede um guarda de reagir a quem esta do outro
// lado da barricada e o SETOR — `Guard.covers` recusa alvo fora do retangulo,
// por perto que ele esteja. O raio nunca atravessa uma linha de barricada, e
// portanto pode ser generoso: o limite util e o tamanho do setor, nao a
// distancia entre eles. As faixas do mapa 3 tem 1280 px de altura e 7680 de
// largura, entao 2600 acorda o posto quando o grupo entra na faixa, e nao
// quando ele ja passou.
//
// Este numero e SO A AQUISICAO — depois de notado, nao ha distancia que solte o
// alvo (ver entity/enemy_territory.go). E ele e um PISO por tipo: o orc traz o
// proprio `Vision`, porque o mais lento do elenco precisa ver primeiro.
const guardRadiusFor float32 = 2600

// garrisonSpread is how far from the post a squad's members are scattered.
//
// Nascer todo mundo no mesmo pixel poe cinco orcs dentro uns dos outros, e a
// direcao em que a separacao os empurra no primeiro quadro e aleatoria — eles
// explodem para os lados antes de o jogador chegar perto. Espalhar na criacao
// e mais barato do que deixar o steering resolver.
//
// 150 e nao 90: cada monstro agora patrulha um trecho de 260 px centrado onde
// nasceu, e com 90 de espalhamento os trechos de um esquadrao de cinco se
// cruzavam o tempo todo — a separacao voltava a empurrar e o vai-e-vem virava
// empurra-empurra.
const garrisonSpread float32 = 150

// StartGarrison puts the map's garrison on the field, at the posts the map
// declares and inside the territories the map draws.
//
// Called once after the map loads, like StartWaveRun. A map that declares
// neither runs empty, which is what a terrain-validation map wants.
func (h *Host) StartGarrison(mapPath string, points []tilemap.SpawnPoint,
	zones []tilemap.Zone) {
	h.stageMap = mapPath
	// Guardados para o RESET. `ResetStage` limpa o EntityManager inteiro, e
	// sem repor a guarnicao o mapa 3 recomecava vazio: o grupo perdia, dava
	// F5 e atravessava as quatro barricadas sem encontrar ninguem. E o mesmo
	// tipo de estado-de-corrida que ja deixou a cena do climax marcada como
	// tocada depois de um reset.
	h.stagePosts, h.stageZones = points, zones

	squads := GarrisonFor(mapPath)
	if len(squads) == 0 {
		return
	}

	// Os postos vem do MAPA e a composicao da tabela; o nome e a unica coisa
	// que liga os dois. Um posto sem esquadrao e um esquadrao sem posto sao os
	// dois erros que so aparecem em jogo como "faltou monstro ali", entao os
	// dois viram log.
	posts := make(map[string]rl.Vector2, len(points))
	for _, p := range points {
		posts[p.Name] = p.Position
	}

	rng := rand.New(rand.NewSource(int64(len(squads))))
	placed, missing := 0, 0
	for _, squad := range squads {
		pos, ok := posts[squad.Post]
		if !ok {
			log.Printf("[Guarnicao] %s: o mapa nao tem enemy_post_%s", mapPath, squad.Post)
			missing++
			continue
		}
		territory, hasZone := tilemap.ZoneAt(zones, pos)
		if !hasZone {
			// Sem setor o guarda persegue sem limite, que e o oposto do que a
			// fase quer. Vale um aviso alto: e um retangulo faltando em
			// `zones`, nao um detalhe de afinacao.
			log.Printf("[Guarnicao] posto %s esta fora de todo territorio; "+
				"ele vai perseguir sem limite", squad.Post)
		}
		for i, kind := range squad.Types() {
			at := scatterAround(pos, i, rng)
			// CADA UM TEM O PROPRIO TRECHO. Compartilhar o posto punha cinco
			// monstros mirando o mesmo pixel: a separacao entre vizinhos
			// empurrava, a distancia passava da tolerancia, eles corrigiam, e o
			// resultado em jogo foi um enxame vibrando parado. O trecho e
			// centrado onde ele nasceu, nao no posto do esquadrao.
			a, b := entity.PatrolSegment(at, i)
			e := entity.NewEnemy(kind, at.X, at.Y)
			// Raio e folga vem do TERRITORIO quando ele os declara, e caem no
			// padrao quando nao. A escala e do mapa: a faixa do mapa 3 tem
			// 7680 px de largura e o grupo a atravessa pelo lado longo; a do
			// mapa 4 tem 3328 e e atravessada pelo lado curto.
			radius := guardRadiusFor
			if territory.GuardRadius > 0 {
				radius = float32(territory.GuardRadius)
			}
			// E o TIPO pode ver mais longe que o territorio manda. Piso, nunca
			// teto: um mapa que declare um raio generoso continua valendo para
			// todo mundo, mas o orc nunca enxerga menos do que a criatura dele
			// precisa para chegar a tempo.
			if vision := entity.GetEnemyDef(kind).Vision; vision > radius {
				radius = vision
			}
			e.Guard = entity.Guard{
				Post:      at,
				PatrolA:   a,
				PatrolB:   b,
				Territory: territory.Rect,
				Radius:    radius,
				Slack:     float32(territory.ChaseSlack),
			}
			h.EntityManager.AddEnemy(e)
			placed++
		}
	}
	log.Printf("[Guarnicao] %s: %d monstros em %d postos (%d posto(s) sem marcador)",
		mapPath, placed, len(squads)-missing, missing)
}

// RestoreGarrison poe a guarnicao de volta depois de um reinicio de fase.
//
// Separado de StartGarrison so pelo log: repor nao e instalar, e um log de
// instalacao a cada F5 esconderia o de verdade. Mapa sem guarnicao nao faz
// nada, que e o caso dos mapas 1 e 2.
func (h *Host) RestoreGarrison() {
	if len(GarrisonFor(h.stageMap)) == 0 {
		return
	}
	h.StartGarrison(h.stageMap, h.stagePosts, h.stageZones)
}

// scatterAround spreads a squad in a ring around its post.
//
// O indice entra no angulo em vez de ser sorteado inteiro: assim os membros de
// um esquadrao ficam repartidos em volta do posto em vez de poderem cair todos
// do mesmo lado, e o sorteio so quebra a simetria.
func scatterAround(post rl.Vector2, index int, rng *rand.Rand) rl.Vector2 {
	if index == 0 {
		return post
	}
	ang := float64(index)*2.399 + rng.Float64()*0.6 // 2.399 rad: angulo aureo
	r := garrisonSpread * float32(math.Sqrt(float64(index)))
	return rl.NewVector2(post.X+r*float32(math.Cos(ang)),
		post.Y+r*float32(math.Sin(ang)))
}

// climaxRuns e a emboscada de cada mapa, separada de `waveRuns` porque ela NAO
// comeca com o mapa: quem a dispara e a chegada do grupo ao objetivo.
var climaxRuns = map[string][]WaveDef{
	"assets/maps/world_03.json": world03Climax,
	"assets/maps/world_04.json": world04Climax,
}

// StartClimax poe a emboscada em campo, a partir dos marcadores climax_spawn_*.
//
// Ela e instalada como uma corrida de hordas de UMA horda, e nao como um
// sistema proprio. Isso devolve de graca o HUD, o teto de simultaneos, o
// anuncio e — o que mais importa — `partyIsFalling`, que so avalia o gatilho
// `on_last_stand` quando existe corrida em fase de luta. Sem isso a cena da
// Sacerdotisa nunca abriria, e o silencio nao daria erro nenhum.
//
// Chamar duas vezes e recusado: a emboscada e um set piece, e um grupo que sai
// da esplanada e volta nao merece uma segunda.
func (h *Host) StartClimax(mapPath string, points []tilemap.SpawnPoint) bool {
	defs := climaxRuns[mapPath]
	if len(defs) == 0 {
		return false
	}
	if len(points) == 0 {
		log.Printf("[Climax] %s declara emboscada mas nao tem marcador "+
			"climax_spawn_*; ela nasceria nas bordas do mundo", mapPath)
		return false
	}
	if h.Waves != nil {
		return false
	}
	// As gargulas do mapa 4 NASCIAM AQUI, filtradas dos marcadores
	// `climax_spawn_stream_*`. Elas sairam para `sentries.go` e entram com o
	// mapa: enquanto o fogo de longa distancia so comecava no saguao, todo o
	// corredor ate ele era corpo a corpo, e corpo a corpo e exatamente o que a
	// Area Angelical — que o grupo ja tem nesta fase — responde inteiro.
	h.Waves = NewWaveRunner(points, defs)
	log.Printf("[Climax] %s: emboscada iniciada de %d pontos", mapPath, len(points))
	return true
}
