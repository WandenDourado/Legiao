# Project Structure

## Entry Points

| Path | Responsibility |
|---|---|
| `AGENTS.md` | Single agent entrypoint, hard rules, doc routing, skill discovery. |
| `cmd/desktop/main.go` | Desktop launcher, calls `game.Run(game.DefaultConfig())`. |
| `cmd/android/build/main.go` | Android launcher used by the Gradle/raylib build. |

## Main Packages

| Path | Responsibility |
|---|---|
| `internal/assets/` | `assets.Path()` abstraction for desktop vs Android asset paths. |
| `internal/collision/` | Shared movement resolution (footprint vs solid space) used by players and monsters. |
| `internal/dialogue/` | Narrative scripts loaded from `assets/dialogues/` and the cursor over a running scene. |
| `internal/entity/` | Player, enemy, projectile data and drawing helpers. |
| `internal/game/` | Main run loop orchestration, input processing, camera, rendering, collision. |
| `internal/input/` | Touch state, aim joystick, platform-specific control drawing. |
| `internal/network/` | TCP host/client, UDP/TCP discovery, protocol, shared snapshots. |
| `internal/system/` | Reusable game systems: combat, movement, spawn, upgrade. |
| `internal/tilemap/` | Tiled JSON/TMJ/TSX loading, tileset rendering, collision extraction (células pintadas + retângulos de manifesto). |
| `internal/ui/` | Menu and HUD. |
| `internal/world/` | Map-derived bounds. |

## Assets

| Path | Responsibility |
|---|---|
| `assets/dialogues/` | One narrative script file per map, named after the map. |
| `assets/maps/` | Runtime maps bundled into Android APK. |
| `assets/tilesets/` | Tileset image and TSX files. |
| `assets/sprites/` | Sprite sheets and per-character `reference.png` files used by character selection. |
| `assets/shaders/` | GLSL pairs per platform (`*_330` desktop, `*_100` Android ES 2.0): terrain blending and the grayscale pass used for dead players. |
| `maps/` | Source copies of maps; runtime should prefer `assets/maps/`. |

## Android Build Tree

| Path | Responsibility |
|---|---|
| `cmd/android/build/androidcompile.bat` | Compiles Go native libraries and copies assets. |
| `cmd/android/build/android/AndroidManifest.xml` | Android app manifest and permissions. |
| `cmd/android/build/android/build.gradle` | Android module build config. |
| `cmd/android/build/android/assets/` | Generated/copied APK asset tree. |
| `cmd/android/build/android/libs/` | Generated native libraries. |

## Agent Skills

| Path | Responsibility |
|---|---|
| `skills/legiao-android-build/` | Android APK/AAB build workflow for agents. |
| `skills/create-character-sprites/` | Directional character sprite generation, validation, sheet assembly, and metadata workflow for agents. |
| `skills/create-tiled-map/` | Zoneamento, bioma, densidade e auditoria de mapa. Decide *onde as coisas ficam*. |
| `skills/create-tiled-assets/` | Atlas, manifesto medido, keying e footprint. Decide *como as coisas existem*. |
| `skills/install-character-sprites/` | Registers validated character sprite output as a selectable Legiao character. |
| `mcp/legiao-android-build/` | MCP compatibility notes for the Android build skill. |

## Ferramentas offline (`work/`)

Scripts que rodam fora do jogo. Não fazem parte do binário; existem para
decidir por medição em vez de por impressão.

| Path | Responsibility |
|---|---|
| `work/tiled-map-world0N/build_*.sh` + `place_objects.py` | Reconstrói um mapa inteiro do zero, de forma reproduzível. Um mapa é o script que o gera, não o JSON. |
| `work/tiled-map-world02/preview_terrain.py` | Espelha `terrain_renderer.go` **e o shader**: a única forma de julgar uma transição de bioma sem compilar. |
| `work/tiled-assets/measure_terrain.py` | Régua de terreno (luminância, saturação, contraste local, matiz). |
| `work/tiled-assets/dark_biome_fit.py` | Encaixa arte gerada no bioma: curva de ombro (clara demais), gamma (escura demais), des-franja do matte. |
| `work/tiled-assets/build_*.py` | Um script por asset integrado, guardando como aquela folha específica foi tratada. |
| `work/combat-verify/`, `work/dialogue-verify/` | Cópias dos arquivos reais compiladas contra stubs de raylib/host, para testar regra sem abrir o jogo. |
| `work/orc-guarnicao/build_orc.py` | Monta as folhas do orc a partir do pacote do fornecedor e emite o `orc_manifest.json`. A folha é o script que a gera; escala e animações por parâmetro. |
| `work/orc-guarnicao/verify_facing.py` | Espelha `enemyRowForHeading` e desenha a bússola das 16 direções. Prova que os números da tabela em Go são os **certos**, o que um teste escrito junto com a função não prova. |

## Where To Add Code

| Need | Location |
|---|---|
| New entity or draw helper | `internal/entity/` |
| New game rule | `internal/system/` |
| New network message | `internal/network/protocol.go` plus host/client handlers |
| New multiplayer host behavior | Small focused file in `internal/network/` |
| New menu/HUD behavior | `internal/ui/` |
| New input behavior | `internal/input/` and `internal/game/input_handler.go` |
| New map/tileset behavior | `internal/tilemap/` |
| New dialogue trigger or scene rule | `internal/game/dialogue.go` plus `internal/dialogue/` (e decida se a cena é da corrida ou do mapa: `Trigger.PerRun`) |
| Nova fase na campanha | `game.campaignMaps` (F8 e a regra da ultimate leem dela) e `network.waveRuns` |
| Estado que dura uma corrida da fase | Limpe no `Host.ResetStage`; se for do lado do jogo, observe `network.StageGeneration()` |
| Change to how anything walks into obstacles | `internal/collision/` (never a second copy of the rule) |
| O que um objeto de mapa bloqueia (casa, cerca, árvore) | O footprint no manifesto em `assets/*_manifest.json`, medido contra a arte — nunca código |
| Como um footprint de manifesto vira espaço sólido | `internal/tilemap/collision_footprints.go` |
| Onde fica a caixa de colisão de um personagem | `internal/entity/character_ground.go` (nos pés, derivada de `CharacterDef.FootLine`) |
| Como um inimigo encara a direção em que anda | Radial gira a arte (`radialAngleFor`); direcional escolhe linha (`internal/entity/enemy_sprite_direction.go`). Nunca os dois |
| Geometria de uma folha de inimigo direcional | `assets/sprites/enemies/<bicho>/*_manifest.json`, copiada na `EnemyDef` com teste cobrando que as duas concordem |
| Monstro que já está no mapa quando o grupo chega | Marcador `enemy_post_*` no Tiled + `Host.PlaceGarrison`. É outra coisa que `enemy_spawn_*`, que é de onde uma horda chega |
| Platform-specific behavior | Build-tagged files |
