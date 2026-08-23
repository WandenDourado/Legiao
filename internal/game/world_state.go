package game

import (
	"log"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/nav"
	"github.com/WandenDourado/Legiao/internal/network"
	"github.com/WandenDourado/Legiao/internal/tilemap"
	"github.com/WandenDourado/Legiao/internal/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// World is everything the loop needs from the currently loaded map. It exists
// as one struct so a portal can swap the whole map at once: renderer,
// collision, bounds and spawns always come from the same file, and nothing can
// be left pointing at the map the player just walked out of.
type World struct {
	Path        string
	Renderer    *tilemap.MapRenderer
	Collision   *tilemap.CollisionGrid
	PlayerSpawn rl.Vector2
	Bounds      world.Bounds
	EnemySpawns []tilemap.SpawnPoint
	// BossAnchor e onde o chefe da fase nasce, se a fase tiver um.
	BossAnchor rl.Vector2
	HasBoss    bool
	// EnemyPosts are garrison positions: monsters that are already on the
	// field when the party arrives. A map can have posts and no spawns
	// (world_03) or spawns and no posts (world_01, world_02); they are
	// different things and neither implies the other.
	EnemyPosts []tilemap.SpawnPoint
	// ClimaxSpawns are where the ambush comes from once the party reaches the
	// objective. Separados dos EnemySpawns de proposito: se fossem os mesmos,
	// a emboscada nasceria no carregamento do mapa.
	ClimaxSpawns []tilemap.SpawnPoint
	// EnemySentries sao os postos das gargulas: posicoes fixas onde uma
	// criatura estatica de alcance 1900 e ancorada. Nem chegada de horda nem
	// guarnicao — ver network/sentries.go para por que os tres sao coisas
	// diferentes.
	EnemySentries []tilemap.SpawnPoint
	// EnemyCannons sao os postos `enemy_cannon_*`: onde os canhoes do
	// corredor final (mapa 6) ficam ancorados. Ver network/cannons.go para
	// por que eles nao sao mais um caso de EnemySentries.
	EnemyCannons []tilemap.SpawnPoint
	// Zones are the map's named rectangles: os territorios que cada guarnicao
	// defende e a area que libera o climax. Lidas no carregamento junto do
	// resto, e nao do TiledMap sob demanda, pelo mesmo motivo que os spawns:
	// tudo que a fase precisa sai de UM arquivo e troca de uma vez.
	Zones   []tilemap.Zone
	Portals []tilemap.Portal
	// arrived is true from the frame the player lands until they step off
	// every portal, so arriving on top of one does not bounce them straight
	// back out.
	arrived bool
	// portalReveal is how far this map's portals have materialised, 0 to 1.
	// It lives here, and not in a package variable, so loading a map resets it
	// with everything else the map owns.
	portalReveal float32
	// partyTally is how many of the living party stand on each portal, in the
	// order of Portals. Counted once per frame by UpdatePortal and read by the
	// drawing, so the number over the portal is the same number the gate is
	// deciding with — recounting it in the renderer is how the two would end up
	// telling the player different things.
	partyTally []portalTally
	// arenaGate holds map-local, one-way arena progression. It must belong to
	// World so arriving on another map cannot inherit a sealed entrance.
	arenaGate arenaGate
}

// LoadWorld loads the starting map. A map that cannot be loaded at startup is
// fatal: there is no world to play in.
func LoadWorld(path string) *World {
	w := TryLoadWorld(path)
	if w == nil {
		log.Fatalf("[Tilemap] Failed to load map: %s", path)
	}
	return w
}

// TryLoadWorld loads visual, collision, spawn, portal and bounds data from a
// Tiled map, returning nil when the map cannot be read. A portal uses it so a
// broken destination leaves the player where they are instead of killing the
// process mid-session.
func TryLoadWorld(path string) *World {
	tm, err := tilemap.LoadTiledMap(path)
	if err != nil {
		log.Printf("[Tilemap] Failed to load map %s: %v", path, err)
		return nil
	}
	renderer := tilemap.NewMapRenderer(tm)
	renderer.Load()
	collisionGrid := tilemap.NewCollisionGrid(tm)
	renderer.Collision = collisionGrid // F3 debug overlay only
	bounds := world.NewBoundsFromMap(tm.Width, tm.Height, tm.TileWidth, tm.TileHeight)
	spawn, found := tilemap.SpawnPosition(tm)
	if !found {
		spawn = entity.InitialPlayerSpawn(bounds)
	}
	enemySpawns := tilemap.EnemySpawnPoints(tm)
	enemyPosts := tilemap.EnemyPostPoints(tm)
	climaxSpawns := tilemap.ClimaxSpawnPoints(tm)
	enemySentries := tilemap.EnemySentryPoints(tm)
	// `boss_anchor` e um marcador nomeado da camada spawn, entao nao precisa
	// de funcao propria em tilemap: NamedSpawnPosition ja responde.
	bossAnchor, hasBoss := tilemap.NamedSpawnPosition(tm, "boss_anchor")
	enemyCannons := tilemap.EnemyCannonPoints(tm)
	zones := tilemap.Zones(tm)
	portals := tilemap.Portals(tm)
	log.Printf("[Tilemap] %s: %d pontos de spawn de inimigo, %d postos de guarnicao, "+
		"%d postos de sentinela, %d postos de canhao, %d zonas, %d portais",
		path, len(enemySpawns), len(enemyPosts), len(enemySentries), len(enemyCannons), len(zones), len(portals))
	return &World{
		Path:          path,
		Renderer:      renderer,
		Collision:     collisionGrid,
		PlayerSpawn:   spawn,
		Bounds:        bounds,
		EnemySpawns:   enemySpawns,
		EnemyPosts:    enemyPosts,
		ClimaxSpawns:  climaxSpawns,
		EnemySentries: enemySentries,
		BossAnchor:    bossAnchor,
		HasBoss:       hasBoss,
		EnemyCannons:  enemyCannons,
		Zones:         zones,
		Portals:       portals,
	}
}

// Unload releases the map's textures.
func (w *World) Unload() {
	if w != nil && w.Renderer != nil {
		w.Renderer.Unload()
	}
}

// ApplyToHost hands the authoritative simulation the world it is simulating.
// All four values have to move together: bounds drive spawn clamping, the
// collision rects drive projectile checks, and the wave runner is rebuilt so
// the previous map's markers cannot leak into this one.
//
// A map with no enemy_spawn_* markers is a quiet map — Waves stays nil and
// nothing spawns, which is what a terrain validation map wants.
func (w *World) ApplyToHost() {
	// As ultimates sao ganhas por fase, entao quem decide se elas valem aqui e
	// o MAPA. Fica ACIMA da checagem de host: o gate e do host, mas o cliente
	// tambem le esse estado para desenhar o botao da ultimate travado, e um
	// cliente que nao souber mostraria um botao que o host vai recusar.
	network.SetUnlockedUltimates(UltimatesGrantedOn(w.Path))
	// Pelo mesmo motivo e no mesmo lugar: quem tranca o portal de um mapa de
	// emboscada enquanto ela nao acontece e uma regra que o CLIENTE tambem
	// desenha (game.PortalsUnlocked roda nas duas maquinas), entao ela e
	// declarada aqui, acima da checagem de host. Ver network/climax_pending.go.
	network.SetClimaxMap(w.Path)

	host := network.CurrentHost
	if host == nil {
		return
	}
	host.WorldBounds = w.Bounds
	host.PlayerSpawn = w.PlayerSpawn
	host.EntityManager.WorldBounds = w.Bounds
	host.EntityManager.Clear()
	// A troca de mapa e o INICIO de uma corrida nova, e o resgate do ultimo
	// suspiro e os efeitos que um heroi convocado deixou no ar pertencem a
	// corrida que fica para tras, nao a que comeca. Sem isto, o Necromante
	// resumido no climax do mapa 2 (quando ninguem jogava com ele) atravessava
	// o portal com a Legiao Espectral ainda viva em h.Skills — os espectros
	// continuavam brigando no mapa 3 — e a flag `done` do resgate continuava
	// marcada, entao o climax do mapa 3 chegava com a cena silenciosa: nao
	// aparecia a Sacerdotisa nem a ultimate dela, porque ResolveLastStand lia
	// "ja resolvido" e nao fazia nada. So o reinicio de fase (ResetStage, apos
	// Game Over) limpava os dois; a travessia normal do portal e do F8 nunca
	// limpava.
	host.Skills.Reset()
	network.ResetLastStand()
	// The grid drives enemy movement (same resolver as the player); the rects
	// derived from it drive the skill-projectile checks. Both have to come
	// from the map that was just loaded, or monsters keep colliding with the
	// previous map's obstacles.
	host.EntityManager.Solid = w.Collision
	// A MESMA grade para os dois, e nao uma lista plana derivada dela para as
	// magias: um espectro da Legiao testava obstaculo contra os ~1.400
	// retangulos do mapa 2, trinta vezes por quadro. Ver skill/legion.go.
	host.SetSolid(w.Collision)

	// The navigation mesh is derived from the same collision, once per map
	// load — not once per frame. Bots and monsters read it through
	// EntityManager.Nav; nil (the state before this line ever runs, or if
	// building somehow panicked) means "no mesh yet", the same fallback
	// nav.Follower.Desired already gives a nil Grid.
	navStart := time.Now()
	host.EntityManager.Nav = nav.Build(w.Collision, w.Bounds, nav.CellSize, nav.AgentBox)
	log.Printf("[Nav] %s: malha construida em %s", w.Path, time.Since(navStart))

	// A guarnicao entra ANTES do retorno de "mapa sem hordas", e essa ordem e a
	// regra inteira: o world_03 nao tem um unico marcador enemy_spawn_*, de
	// proposito, porque a jogabilidade dele e de guarnicao. Se isto ficasse
	// depois, o mapa 3 continuaria vazio exatamente como esta hoje.
	// A composicao vem da tabela do mapa (network/garrisons.go), nao de um
	// tipo unico: cada posto declara quantos orcs, lobos e slimes, e o orc so
	// aparece nos vaos de barricada. Um posto de um tipo so devolveria o mapa
	// a "orc em toda parte", que e o contrario do que a fase propoe.
	host.StartGarrison(w.Path, w.EnemyPosts, w.Zones)

	// As sentinelas vem LOGO DEPOIS da guarnicao e antes do retorno de "mapa sem
	// hordas", pela mesma razao: o mapa 4 nao tem marcador enemy_spawn_* nenhum,
	// e uma gargula instalada depois desse retorno nunca existiria.
	host.InstallArrivalSentries(w.Path, w.EnemySentries)
	// O chefe nasce no carregamento, como a guarnicao: ele nao e horda, ele e
	// a fase. Mapa sem entrada em bossOfMap nao faz nada.
	host.InstallBoss(w.Path, w.BossAnchor, w.HasBoss)

	// Os canhoes, pela mesma razao das sentinelas: postos fixos que ja estao
	// em campo quando o mapa carrega, nao uma corrida.
	host.InstallArrivalCannons(w.Path, w.EnemyCannons)

	// A vaga de cada classe sem humano, depois de Skills.Reset() e da
	// guarnicao/sentinelas/canhoes instalados, antes do retorno de "mapa sem
	// hordas" — um bot existe mesmo em mapas de guarnicao sem
	// enemy_spawn_*. world_travel.go chama PlaceEveryoneAtSpawn logo depois
	// de ApplyToHost, e essa ordem move os bots recem-criados junto.
	host.ReconcileBots()
	// A bot's arena-lock flag is per-agent state that ReconcileBots itself
	// never touches for a bot that already existed (only a freshly-created
	// one starts clean) — see doc/tilemap.md "Arena de mão única". Without
	// this a bot that got sealed into map N's arena would arrive at map
	// N+1 still unable to cross its own spawn corridor.
	host.ResetBotArenaLocks()

	if len(w.EnemySpawns) == 0 {
		host.Waves = nil
		// updateWaves returns early without a runner, so the HUD has to be
		// cleared here or it keeps showing the previous map's wave. The zero
		// state has Total 0, which is how DrawWaveHUD knows to draw nothing.
		network.SetWaveState(network.WaveState{})
		log.Printf("[Host] %s nao tem marcadores enemy_spawn_*; mapa sem hordas", w.Path)
		return
	}
	// A corrida de hordas vem do MAPA, entao o caminho vai junto: com as
	// definicoes numa variavel de pacote, todo mapa carregado rodava as tres
	// hordas do world_01.
	host.StartWaveRun(w.Path, w.EnemySpawns)
}
