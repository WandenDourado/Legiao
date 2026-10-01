package tilemap

import (
	"sort"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// spawnLayerName is the object layer that holds every spawn marker.
const spawnLayerName = "spawn"

// enemySpawnPrefix marks an object in the spawn layer as an enemy origin.
// Anything named enemy_spawn_* is picked up, so new points can be added in
// Tiled without touching code.
const enemySpawnPrefix = "enemy_spawn"

// enemyPostPrefix marks a GARRISON position: a monster that is already on the
// field when the party arrives and belongs to that spot, as opposed to
// enemy_spawn_*, which is where a wave arrives from off screen.
//
// The two prefixes are different on purpose and must stay different. A map
// with posts and no spawns is not a quiet map — it is a map whose monsters are
// standing still waiting, which is exactly what world_03 is.
const enemyPostPrefix = "enemy_post"

// climaxSpawnPrefix marks where the ambush comes from when the party reaches
// the objective. Prefixo proprio, e nao `enemy_spawn_*`, porque estes pontos
// NAO devem alimentar a corrida normal do mapa: eles so existem depois que a
// porta do climax abre, e um mapa de guarnicao nao tem corrida nenhuma antes
// disso. Misturados, a emboscada nasceria no primeiro quadro da fase.
const climaxSpawnPrefix = "climax_spawn"

// enemySentryPrefix marca um POSTO DE SENTINELA: o lugar exato onde uma
// criatura estatica de longo alcance e ancorada.
//
// Prefixo proprio, e nao `enemy_post_*`, porque a sentinela nao e guarnicao no
// sentido do sistema: ela nao patrulha, nao volta ao posto e nao tem setor —
// tem alcance 1900 e Speed 0. Passa-la pelo caminho da guarnicao lhe daria um
// `Guard` com trecho de patrulha e territorio, que sao perguntas que ela nao
// responde, e o setor poderia recusar o unico alvo que ela tem.
//
// E nao e `climax_spawn_*` porque o posto e uma POSICAO FIXA, nao uma origem de
// onde alguem vem: dois monstros nunca ocupam o mesmo posto de sentinela.
const enemySentryPrefix = "enemy_sentry"

// enemyCannonPrefix marca um POSTO DE CANHAO: a posicao exata onde um dos
// canhoes do corredor final (mapa 6) fica ancorado.
//
// Prefixo proprio, e nao `enemy_sentry_*`, apesar de os dois serem estaticos e
// de longo alcance: um canhao nao e um `entity.Enemy` — nao tem sprite, nao
// entra no EntityManager, nao pode ser atacado pela espada nem pelo espectro.
// Ele so pode ser destruido pelo julgamento roteirizado do ultimo suspiro
// (ver internal/network/host_cannon.go). Reusar o prefixo da sentinela
// misturaria os dois sistemas de armamento por acidente de nome.
const enemyCannonPrefix = "enemy_cannon"

// SpawnPoint is a named position in the world.
type SpawnPoint struct {
	Name     string
	Position rl.Vector2
}

// playerSpawnName is the default arrival point of a map.
const playerSpawnName = "player_spawn"

// SpawnPosition returns the center of the object named player_spawn.
func SpawnPosition(m *TiledMap) (rl.Vector2, bool) {
	return NamedSpawnPosition(m, playerSpawnName)
}

// NamedSpawnPosition returns the center of the named object in the spawn
// layer. A portal uses it to land the player at its own arrival marker instead
// of the map's default entry point.
func NamedSpawnPosition(m *TiledMap, name string) (rl.Vector2, bool) {
	if name == "" {
		name = playerSpawnName
	}
	for _, layer := range m.Layers {
		if layer.Type != "objectgroup" || layer.Name != spawnLayerName {
			continue
		}
		for _, object := range layer.Objects {
			if object.Name == name {
				return rl.NewVector2(object.X+object.Width/2, object.Y+object.Height/2), true
			}
		}
	}
	return rl.Vector2{}, false
}

// EnemySpawnPoints returns every enemy_spawn_* marker in the spawn layer,
// sorted by name so the order is stable across runs. Stability matters because
// the host is authoritative and picks spawn points from this slice; Go's map
// and file iteration order would otherwise make two runs of the same map differ.
func EnemySpawnPoints(m *TiledMap) []SpawnPoint {
	return markersWithPrefix(m, enemySpawnPrefix, false)
}

// EnemyPostPoints returns every enemy_post_* marker in the spawn layer, sorted
// by name. These are garrison positions, not wave origins: a monster placed
// here is on the field from the first frame and belongs to this spot.
// O nome vem SEM o prefixo: `enemy_post_vao_a_oeste` devolve `vao_a_oeste`.
//
// Isso nao e cosmetico. A composicao de cada posto mora em
// network/garrisons.go e e chaveada pelo sufixo, que e a parte que identifica
// o posto; o prefixo so diz que tipo de marcador ele e. Devolvendo o nome
// inteiro, a busca falhava nos 24 postos do mapa 3 e a fase carregava SEM UM
// MONSTRO — sem erro, so um log de "posto sem marcador" por linha que ninguem
// leu.
func EnemyPostPoints(m *TiledMap) []SpawnPoint {
	return markersWithPrefix(m, enemyPostPrefix, true)
}

// ClimaxSpawnPoints returns every climax_spawn_* marker, sorted by name.
func ClimaxSpawnPoints(m *TiledMap) []SpawnPoint {
	return markersWithPrefix(m, climaxSpawnPrefix, true)
}

// EnemySentryPoints devolve os postos `enemy_sentry_*`, ordenados por nome e
// sem o prefixo — a mesma forma de `EnemyPostPoints`, porque quem os consome
// (network/sentries.go) tambem casa pelo sufixo.
//
// A ordem alfabetica NAO e cosmetica aqui: no mapa 5 as sentinelas entram aos
// poucos, horda a horda, e a lista e percorrida do inicio. Ordem instavel faria
// a mesma fase armar postos diferentes a cada partida.
func EnemySentryPoints(m *TiledMap) []SpawnPoint {
	return markersWithPrefix(m, enemySentryPrefix, true)
}

// EnemyCannonPoints devolve os postos `enemy_cannon_*`, ordenados por nome e
// sem o prefixo — mesma forma de EnemyPostPoints/EnemySentryPoints, porque
// quem os consome (network/cannons.go) tambem casa pelo sufixo.
func EnemyCannonPoints(m *TiledMap) []SpawnPoint {
	return markersWithPrefix(m, enemyCannonPrefix, true)
}

// markersWithPrefix collects named markers from the spawn layer.
//
// The sort is not cosmetic. The host is authoritative and walks this slice to
// place monsters; Go randomises map iteration and file order is not promised,
// so without a stable order two runs of the same map would differ.
func markersWithPrefix(m *TiledMap, prefix string, strip bool) []SpawnPoint {
	var points []SpawnPoint
	for _, layer := range m.Layers {
		if layer.Type != "objectgroup" || layer.Name != spawnLayerName {
			continue
		}
		for _, object := range layer.Objects {
			if !strings.HasPrefix(object.Name, prefix) {
				continue
			}
			name := object.Name
			if strip {
				name = strings.TrimPrefix(strings.TrimPrefix(name, prefix), "_")
			}
			points = append(points, SpawnPoint{
				Name:     name,
				Position: rl.NewVector2(object.X+object.Width/2, object.Y+object.Height/2),
			})
		}
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Name < points[j].Name })
	return points
}
