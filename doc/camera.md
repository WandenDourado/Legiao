# Camera And World Bounds

Use this doc for camera behavior, world-space rendering, bounds, spawn distance, and coordinate conversion.

## World Bounds

World size comes from the loaded Tiled map:

```go
bounds := world.NewBoundsFromMap(tm.Width, tm.Height, tm.TileWidth, tm.TileHeight)
```

For the current village map (60x45 tiles at 128px), bounds are `7680x5760`.

Do not use screen constants for world logic.

## Camera

`internal/game/camera.go` owns `Camera2DState`.

- Offset: screen center.
- Target: player position clamped to world bounds.
- Zoom: `Config.CameraZoom` — **1,0 no PC, 0,8 no celular** (27/09/2026).
- Rotation: `0`.
- No smoothing/lerp; camera follows directly.

Clamp rule, in WORLD units (the screen covers `sw/zoom × sh/zoom` of the map):

```go
halfW, halfH := sw/zoom/2, sh/zoom/2
camera.Target.X = clamp(target.X, halfW, bounds.Width-halfW)
camera.Target.Y = clamp(target.Y, halfH, bounds.Height-halfH)
```

### Zoom por plataforma

O playtest de 27/09/2026 relatou pouca visão do mapa, sobretudo no celular:
não se via o lobo chegando nem a gárgula a derrubar. O celular passou a 0,8
(25% a mais por eixo, ~56% mais área). O PC ficou em 1,0 de propósito: a tela
já é grande, e escala não inteira muda a nitidez do terreno, que usa filtro
POINT. O valor mora em `Config` (`DefaultConfig`/`AndroidConfig`), sem
`if isAndroid`.

- **Nunca além do mapa.** `effectiveZoom` só afasta a câmera até o mapa encher
  a tela no eixo mais justo: no corredor do mapa 6 (2560 px) um celular de
  2400 px fica em 0,94 em vez de mostrar faixas pretas. Nunca aproxima além
  de 1, então o PC não muda.
- **Ancoragem em pixel de TELA.** O arredondamento que evita o terreno
  "saltar" (`doc/performance.md`, C2) passou a ser `round(Target·zoom)/zoom`:
  é `Target·zoom` que tem de ser inteiro. A zoom 1 é a conta de antes.
- **Tudo que corta pelo visível já sabia de zoom** (`tilemap.NewViewport`,
  `skill.SetDrawView`, culling de inimigos): o celular desenha ~56% mais
  área, e esse é o custo a olhar no F3 nos mapas 2 e 3, que já foram o
  gargalo no celular (`doc/performance.md`). Se pesar, o botão é o número em
  `AndroidConfig` — 0,85 já devolve metade da área extra.
- Raio de audição (`sound.SFXHearRadius`, 1600) continua além da borda da tela
  a 0,8 (1500 de mundo para cada lado num aparelho de 2400 px).

## Setas das gárgulas fora da tela

A gárgula tem alcance global e atira de fora da tela; para achá-la, o Arqueiro
tinha de andar até ela. Desde 27/09/2026 cada gárgula viva que a câmera não
mostra ganha uma seta violeta na borda da tela, na reta do centro da tela até
ela (`ui.EdgeMarkerPoint` — interseção com a borda recuada, não clamp por
eixo, senão tudo iria para os cantos). Perto é forte, longe é discreta
(`markerStrength`: cheia até 1800 de mundo, mínima a partir de 5000), para as
dez do mapa 7 não virarem dez alarmes iguais. Desenhada em espaço de tela,
antes dos controles do celular; fonte: `network.LivingSentryCenters` (host:
`EntityManager`; cliente: o espelho interpolado). Só gárgula — decisão do Gui.

**Só gárgula acordada** (28/09/2026). No mapa 4 as torres ficam em campo desde
a chegada, caladas até a maioria do grupo alcançar o saguão ("As sentinelas
despertaram", `network/sentry_wake.go`). A seta espera o mesmo momento: o host
publica o veredicto da porta (`network.SentriesAwake`) a cada quadro em
`handleSentryOrbTick`, e ele viaja para o cliente em
`EnemyUpdatePayload.SentriesAwake` — a porta é decidida só no host (maioria no
degrau, com trava), então não dá para derivar do outro lado. Nos mapas sem
porta (5 e 7) a gárgula nasce acordada e a seta aparece desde que ela exista.
`TestSentryMarkersWaitForTheWake`.

## Render Space Rule

World-space drawing must happen inside the map renderer camera callback in `DrawFrame`.

World-space examples:

- map layers
- map boundary
- players
- enemies and health bars
- projectiles

Screen-space examples:

- health bar
- player count
- server address
- respawn/game-over text
- Android joystick/action controls

Screen-space UI must stay outside camera transforms.

## Mouse Coordinate Conversion

Raylib mouse position is screen-space. Convert to world-space before attack targeting:

```go
screenPos := rl.GetMousePosition()
worldPos := rl.GetScreenToWorld2D(screenPos, cam.Camera)
```

Android aim uses a direction vector relative to the player and does not need this conversion.

## Bounds Initialization Order

Set host entity bounds after the host exists:

1. Initialize window.
2. Load map and compute bounds.
3. Show menu; host may be created here.
4. If host exists, assign `network.CurrentHost.EntityManager.WorldBounds = bounds`.

Assigning bounds before menu creation leaves host bounds at zero and projectiles expire immediately.

## Player Spawn

`player_spawn` in the loaded Tiled map is the shared spawn for local and
authoritative network player state.

- Current spawn: the map object, with the center of map-derived
  `world.Bounds` retained only as a malformed-map fallback.
- Host registration, joining clients, and respawn use this same position.

## Enemy Spawn Distance

Authoritative enemy spawn is in `internal/network/host_spawn.go`.

- Spawn candidates are on map edges, `180px` outside the playable rectangle.
- The host avoids candidates within `520px` of alive players.
- If no candidate passes after the retry budget, the farthest sampled candidate is used.

