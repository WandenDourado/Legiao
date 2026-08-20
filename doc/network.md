# Network

The game uses TCP for gameplay state and UDP/TCP helpers for LAN discovery.

## Model

- Host is authoritative.
- Clients send input and actions.
- Host simulates enemies, projectiles, combat, deaths, respawn, and broadcasts snapshots.
- Cooldowns, the revive countdown, Game Over and the stage reset are host state too; clients only display them. See `doc/combat_rules.md`.
- Clients render snapshots from the host.

## Ports

| Port | Protocol | Use |
|---|---|---|
| 9000 | TCP | Gameplay messages |
| 9001 | UDP | Host discovery query/response and broadcast compatibility |

## Main Files

| File | Responsibility |
|---|---|
| `internal/network/protocol.go` | Message types and payload structs. |
| `internal/network/host.go` | TCP server, authoritative state, combat/projectile simulation. |
| `internal/network/host_spawn.go` | Enemy wave spawning and safe edge spawn selection. |
| `internal/network/host_player_state.go` | Host player movement/animation snapshot update. |
| `internal/network/host_attack_gate.go` | Basic-attack cadence gate from the character's attack speed. |
| `internal/network/host_skill_gate.go` | Character binding and per-skill cooldown gates. |
| `internal/network/host_cooldown_sync.go` | Snapshots the cooldown timers and broadcasts them. |
| `internal/network/host_death.go` | Death flag, revive countdown and revival. |
| `internal/network/host_reset.go` | Stage restart after a Game Over (host only). |
| `internal/network/host_test_mode.go` | Per-player no-cooldown test mode. |
| `internal/network/cooldowns.go` | Shared read-only cooldown snapshot the HUD reads on both roles. |
| `internal/network/client.go` | TCP client and received message handling. |
| `internal/network/client_projectiles.go` | Applies projectile snapshots on clients. |
| `internal/network/travel_local.go` | `StartTravel` (host announces) and the queue the game loop applies the arrival from. |
| `internal/network/client_travel.go` | Clears the mirrored world when the party changes map. |
| `internal/network/host_travel.go` | Puts every player on the arrival point after the map changed. |
| `internal/network/discovery.go` | UDP discovery, query responder, TCP subnet scan fallback. |
| `internal/network/globals.go` | Process-wide network role and snapshot maps. |
| `internal/ui/menu.go` | Host/join UI and the discovery connection flow. |

## Message Families

| Type | Direction | Purpose |
|---|---|---|
| `join` | client -> host | Register player ID/color. A PlayerID already in `h.players` is a RECONNECT, not a fresh spawn — see "Ausencia e reconexao" below. |
| `input` | client -> host | Position, sprite frame/row, sprint flag, velocity. |
| `state_update` | host -> peers | Full player snapshot list. |
| `enemy_update` | host -> peers | Full enemy snapshot list. Empty lists clear clients. |
| `projectile_update` | host -> peers | Full projectile snapshot list. Empty lists clear clients. |
| `attack` | client -> host | Request projectile fire toward a world target. |
| `combat_event` | host -> peers | Damage/death/respawn events. |
| `game_over` | host -> peers | All players are dead. |
| `respawn` | host -> peers | Respawn update. |
| `dialogue` | host -> peers | Line currently on screen; the host narrates, clients only display. See `doc/dialogue.md`. |
| `cooldown` | host -> peers | Every player's remaining skill and attack cooldowns, one snapshot per simulation tick. Display only. |
| `reset_stage` | host -> peers | The host restarted the stage: clear the mirrored world and return to the spawn. |
| `travel` | host -> peers | The party moved to another map. Carries the destination map and arrival spawn. Also sent to a single rejoining client with `reconnect: true` (see below) as a map catch-up, not a party-wide crossing. |
| `test_mode` | client -> host | Ask the host to drop that player's cooldown gates (F2). |
| `ultimate_grant` | host -> peers | Unlocks one character's ultimate for the rest of THIS RUN, on top of the campaign's set (`doc/combat_rules.md`, "O resgate"). Sent once when the last-stand rescue revives a player controlling the map's hero, and replayed to anyone who joins mid-run or reconnects. |

## Discovery Flow

Current join flow starts three discovery paths (client side):

1. `StartDiscoveryListener()` passively receives the host's `LEGION_HOST:0.0.0.0:9000` UDP broadcasts on port 9001. This is the primary path and does NOT depend on the Windows firewall allowing inbound UDP 9001 for the query round-trip, so it is what lets an Android client find a Windows host.
2. `StartQuerySender(9000)` sends `LEGION_QUERY:<reply_port>` to UDP broadcast port 9001 and receives direct `LEGION_RESPONSE:host:9000` (answer from the host's `StartQueryResponder`).
3. `StartTCPScan(9000)` scans the local subnet as fallback. If the outbound-IP probe fails it now falls back to the first non-loopback interface IP via `firstLANIP()` instead of skipping the scan.

Host starts:

1. TCP server on port 9000.
2. UDP broadcast announcements for desktop compatibility.
3. UDP query responder on port 9001 for Android-friendly direct responses.

The join screen has no manual-IP field and no "Scan TCP" button: the three paths
above already run on entry and refresh the list on their own, so the only control
is Back. Back also calls `StopDiscovery()` + `ClearDiscoveredHosts()`, otherwise
the scanners keep running behind the main menu and a second Join starts a
duplicate set of them.

## Taxa de publicacao e interpolacao

**A simulacao roda a 60 Hz; a PUBLICACAO dela roda a 20 Hz** (`SnapshotHz`, em
`internal/network/broadcast_rate.go`). Cooldown, dano e morte continuam sendo
decididos a cada quadro pelo host — o que desceu foi so a frequencia com que o
estado vai para o fio.

O cliente cobre o vao **interpolando** (`internal/network/interpolation.go`):
guarda os dois ultimos snapshots e desenha `interpDelay` (100 ms) atras do mais
recente, entre os dois. Os dois so funcionam juntos — baixar a taxa sem
interpolar faria toda posicao remota andar tres quadros parada e um salto.

Regras que valem nao reinventar:

- **O atraso e o preco, e ele e o certo aqui.** O que se ve de um jogador
  remoto esta ~100 ms velho. O host e autoritativo e ja decide todo o combate;
  o cliente nunca precisou que a posicao remota fosse instantanea, precisou que
  fosse CONTINUA. Suavizar sem atraso exigiria extrapolar, e extrapolacao erra
  a direcao toda vez que alguem muda de rumo.
- **Interpola so o que e continuo**: posicao e velocidade. Vida, morte, quadro
  de animacao e identidade vem do snapshot mais recente sem misturar — meia
  morte nao existe, e um quadro de sprite interpolado seria um numero entre
  dois desenhos.
- **`interpDelay` tem de segurar DOIS intervalos de snapshot.** A 20 Hz o
  intervalo e 50 ms e 100 ms cabem dois; com menos, um pacote atrasado esvazia
  o buffer e a interpolacao vira extrapolacao.
- **`ResetInterpolation()` no reinicio da fase.** Mora em `RequestLocalReset`,
  que e o unico ponto por onde os dois papeis passam. Sem ele, interpolar do
  campo de batalha ate o ponto de spawn faria todo mundo DESLIZAR pelo mapa.
- **Nada publica fora do tique.** Ja aconteceu tres vezes de um caminho
  paralelo transmitir a 60 Hz e anular a taxa: o handler de `MsgInput`,
  `UpdatePlayerState` (chamado a cada quadro pelo laco do jogo, para o proprio
  host) e `UpdatePlayerPosition`. Os tres so atualizam estado agora.

`game.DrawFrame` desenha de `InterpolatedPlayers()` e `InterpolatedEnemies()`;
no host as duas devolvem o estado direto, porque ele o tem a cada quadro. A
LOGICA (chegada do grupo, gatilho de dialogo) continua lendo `GetAllPlayers()`,
que e o snapshot autoritativo: decidir com o desenho atrasado seria decidir com
100 ms de erro.

## Identidade vai uma vez, estado vai sempre

Cor, personagem e vida maxima do jogador, e tipo, cor e vida maxima do inimigo,
nao mudam depois que a entidade aparece — e iam no fio 60 vezes por segundo.
Agora sao **anunciados uma vez** e omitidos dos snapshots seguintes
(`internal/network/wire.go`); quem recebe recompoe pelo cache de identidade,
entao `RemotePlayers` e `RemoteEnemies` continuam com todos os campos.

| Momento | O que vai |
|---|---|
| Join, disconnect, reinicio de fase | `BroadcastRoster()`: jogadores COM identidade |
| Cliente entrando | `sendStateToClient`: jogadores E inimigos completos, so para ele |
| Primeiro snapshot de um inimigo | o `EnemyState` inteiro |
| Todo tique depois disso | so o que muda |

Duas armadilhas que isso cria e que ja estao cobertas:

- **O anunciante e um so para todos os peers**, entao um cliente que entra no
  meio da partida receberia snapshots magros de inimigos cuja identidade ele
  nunca viu. Por isso `sendStateToClient` manda a lista completa de inimigos so
  para ele — sem isso, um mapa de guarnicao (o `world_03` nasce com 83) faria o
  cliente novo desenhar 83 monstros sem tipo.
- **Campo vazio significa "igual ao ultimo", nunca "apagou".** Lista vazia
  continua significando "limpe tudo", que e outra coisa.

Ganho medido nos tamanhos: `PlayerState` de 231 para 165 bytes, `EnemyState` de
125 para 71. Com a taxa de 20 Hz, o trafego por cliente com 4 jogadores e 83
inimigos cai de **763 KB/s para 162 KB/s** — 4,7x menos.

## Ausencia e reconexao

Uma tela bloqueada no Android nao necessariamente derruba o TCP: o processo e
pausado e o socket pode ficar de pe por minutos, sem nenhum decode error do
lado do host. Por isso o host nao apaga o jogador na primeira falha de leitura
— ele MARCA o slot ausente e da uma janela para o jogador voltar.

- **Identidade estavel.** `ui/menu.go` gera o `PlayerID` UMA VEZ por processo
  (`ensureLocalPlayerID`) e reusa em toda tentativa de conexao; cor e
  personagem ficam em `network.LocalPlayerColor`/`LocalPlayerCharacter` para o
  rejoin reenviar os mesmos valores. Limite aceito: se o SO mata o app, e uma
  sessao nova e o jogador perde o slot — nao ha persistencia em disco.
- **Host: `handleClient`'s defer nao deleta mais** — chama `markAbsent`
  (`internal/network/host_absence.go`), que guarda `absentSince` e preserva
  posicao, vida, `IsDead`, `RespawnIn` e todos os cooldowns (ja sao mapas por
  `playerID`, intocados). `tickAbsence`, chamado uma vez por tique de
  `UpdateSimulation`, remove o slot de vez apos `ReconnectGrace` (5 minutos) —
  so entao um `PlayerID` reaproveitado volta a ser um join novo.
- **`MsgJoin` com um `PlayerID` conhecido e reconexao**, tratada por
  `Host.handleJoin` (`internal/network/host_rejoin.go`): preserva o
  `PlayerState` existente, so atualiza cor/personagem e limpa a ausencia.
  Manda `BroadcastRoster` + `sendStateToClient` (identidade completa) +
  `sendCurrentMapTo` (um `travel` so para ele, com `reconnect: true`, para o
  mapa que o grupo esta agora — sem isso quem reconecta depois de um portal
  fica sozinho no mapa antigo) + `sendDialogueTo`.
- **Duas conexoes com o mesmo `PlayerID`** (a antiga ainda nao caiu):
  `supersedeConnection` fecha a antiga e marca `ClientConn.superseded`
  (`atomic.Bool`) antes de trocar `c.playerID` na nova. O defer da conexao
  velha confere essa flag e NAO marca nem apaga o slot que ja e da nova.
- **Ausente nao segura o jogo.** `network.PresentPlayers()` e
  `GetAllPlayers()` filtrado por `!Absent` — todo "o grupo inteiro..." tem de
  ler dali: o Game Over do host (`checkGameOver`), `partyIsFalling`
  (`game/dialogue.go`), `partyArrived` (`game/climax_gate.go`) e a porta do
  portal (`game/portal_party.go`). `GetAllPlayers()` continua sendo o
  snapshot completo — o que o desenho e o HUD usam, ausente incluido, com o
  marcador "reconectando...". Um ausente continua VULNERAVEL: se morrer
  parado, morreu.
- **Cliente: silencio, nao so erro de decode.** O host publica a 20 Hz
  (`SnapshotHz`), entao `client_liveness.go` mede o tempo desde a ULTIMA
  mensagem recebida (qualquer tipo) e trata 5 s de silencio
  (`clientSilenceTimeout`) como conexao perdida — sem depender do timeout de
  TCP do Android, que e longo demais. `keepalive.go` liga `SetKeepAlive` dos
  dois lados como reforco, nao como deteccao primaria.
- **Reconector** (`internal/network/reconnect.go`): ao perder a conexao,
  `TickReconnect` (chamado do laco do jogo a cada quadro, papel cliente) entra
  em estado "reconectando" — input local congelado, mundo espelhado continua
  desenhado, `ui.DrawReconnectOverlay` mostra a contagem regressiva dos 5
  minutos (`ReconnectWindow`). Tenta `net.Dial` com backoff (1 s -> 5 s), e ao
  conectar reenvia `MsgJoin` com o MESMO id/cor/personagem
  (`rejoin`), que tambem chama `ResetInterpolation()` e limpa inimigos,
  projeteis e efeitos espelhados ANTES do primeiro snapshot novo — os dois
  snapshots guardados sao de minutos atras, e sem isso todo mundo desliza pelo
  mapa. Estourada a janela sem sucesso, `giveUp` encerra a sessao
  (`network.SessionEnded`) e o menu mostra uma mensagem clara em vez de deixar
  o app num limbo.
- **`Client.Send` tem mutex e falha em silencio** sem conexao (entre
  `dropConn` e `rebind`) — nada de `log.Printf` por quadro, o laco do jogo
  chama `Send` a cada quadro de `MsgInput`.

## O mapa e do grupo, nao da maquina

Trocar de mapa era **local**: cada um atravessava o portal quando queria, e a
partida ficava em dois mundos — o host simulando o antigo, o cliente sozinho no
novo, sem nada em campo. Agora o mapa e parte do protocolo.

| Quem | O que faz |
|---|---|
| Host (ou sessao solo, que e o proprio host) | Avalia o portal, chama `StartTravel`: transmite `travel` e enfileira a chegada para si. |
| Cliente | Nunca decide. Ao receber `travel`, limpa inimigos/projeteis/visuais espelhados e enfileira a chegada. |
| O laco do jogo, nos dois papeis | `ApplyPendingTravel` consome a fila e troca o `World` inteiro de uma vez. |

- **A rede nao carrega o mapa, ela nomeia o destino.** Quem recebe le spawn,
  bounds, colisao e portais do proprio arquivo. Por isso a mensagem tem dois
  campos e nao um estado de mundo.
- **A rede tambem nao troca o `World`** — ela nao o possui, o pacote `game`
  possui. Por isso a chegada e ENFILEIRADA, o mesmo desenho de
  `reset_local.go`.
- **`ResetInterpolation()` na chegada.** Sem ele, os dois snapshots guardados
  pertencem ao mapa que acabou de ser deixado e todo jogador remoto DESLIZA
  pelo mapa novo a partir de onde estava no antigo. Mesmo cuidado do reinicio
  de fase.
- **`PlaceEveryoneAtSpawn` no host.** Um jogador vivo se corrige sozinho no
  quadro seguinte, ao publicar a posicao; um MORTO nao publica nada, e o host
  continuaria anunciando um cadaver nas coordenadas do mapa anterior — e o
  ressuscitaria la.
- **Chegar nao e reviver.** `applyTravel` nao limpa `LocalPlayerDead` nem
  `GameOver`, ao contrario de `applyStageReset`: quem morreu a caminho do portal
  atravessa morto e espera a contagem do host do outro lado.

A regra de quando isso dispara — todos os jogadores vivos no mesmo portal — e do
mapa, e esta em `doc/tilemap.md`.

## Espera no portal (`PlayerState.InPortal`)

Com bots no grupo (`doc/plan_bots_de_classe.md`), a porta quase nunca ficava
com todo mundo dentro do MESMO retangulo ao MESMO tempo — bastava um bot
entrar e sair de novo para a condicao nunca fechar. A saida: quem entra na
zona do portal SOME da tela e CONGELA, liberando o retangulo para o resto do
grupo. Vale para humano e para bot; a porta continua contando o grupo
inteiro (`countParty`, `doc/tilemap.md`).

`InPortal bool` (`json:"in_portal,omitempty"`) e estado de tique, como
`IsDead` e `Absent` — nao e identidade, entao **nao** entra em `slimPlayer`
(`wire.go`) e nao precisa de mensagem nova de protocolo.

- **Quem decide e o host.** `Host.tickPortalPresence()`
  (`host_portal_presence.go`), chamado de `UpdateSimulation` antes de
  `tickBots`, testa `entity.GroundBoxAt` de cada jogador VIVO contra os
  retangulos que `game.UpdatePortal` publicou via `SetPartyPortals` — a
  MESMA caixa que `countParty` usa, para a espera e a contagem nunca
  discordarem. Morto nunca entra em espera: viaja junto, continua caido.
- **Quem congela e o dono do movimento.** Um bot: `tickBots` pula quem esta
  `InPortal` por inteiro — sem `View`, sem `Intent`, sem ataque. Um humano: a
  posicao vem do CLIENTE (`MsgInput`), entao o host so pode marcar a flag —
  `game.ProcessInput` le o espelho local (`network.LocalPlayerInPortal`,
  atualizado por `SyncLocalPlayer` do mesmo jeito que `LocalPlayerDead`) e
  devolve direcao zero, ignorando ataque e habilidade, o mesmo padrao que
  `network.GameOver` ja usa ali.
- **Cancelar e local.** ESC (desktop) ou o botao SAIR (Android) empurram o
  jogador para fora do retangulo (`game.UpdatePortalCancel`,
  `portal_cancel.go`) e zeram o espelho local na hora; o host confirma
  sozinho no proximo tique de `tickPortalPresence`, sem mensagem de rede
  nova.
- **Limpeza obrigatoria.** `InPortal` de TODOS e apagado em
  `PlaceEveryoneAtSpawn` (travessia) e em `ResetStage` (F5). Sem isso o grupo
  chegaria ao mapa seguinte — ou a fase reiniciada — invisivel e sem
  controle.

## Snapshot Rules

- `RemotePlayers`, `RemoteEnemies`, and `RemoteProjectiles` are shared render snapshots.
- Snapshot getters return copies.
- Clients replace snapshot maps from host messages, they do not merge partial
  deltas. A UNICA excecao e a identidade omitida (ver acima): o cliente
  substitui a lista e COMPLETA os campos que vieram vazios.
- The host also updates its own `RemotePlayers` for rendering.
- Empty enemy/projectile snapshots are meaningful and must be sent to clear stale remote entities.

## Animation And Actions

`MsgInput` includes:

```json
{
  "player_id": "player_...",
  "x": 100,
  "y": 100,
  "current_frame": 2,
  "current_row": 1,
  "is_sprinting": false,
  "vel_x": 0,
  "vel_y": 200
}
```

Remote players render with their own character sprite sheet via `entity.DrawWizardStateAt()`. The default (fallback) character is the Mago. Projectiles are created only by the host and broadcast through `projectile_update`.

The ally health bar (`entity.DrawAllyHealthBar`, called from `game.DrawFrame`)
reads straight off this same snapshot: `Health` from every tick, `MaxHealth`
recomposed by the identity cache from the join-time announce (see "Identidade
vai uma vez, estado vai sempre" above). No new field went on the wire for it.

## Operational Notes

- LAN play requires host and clients on the same network.
- Windows firewall must allow inbound TCP 9000 for the host.
- Discovery is LAN-only, not internet matchmaking.
- Do not reintroduce Java/MulticastLock code unless there is a verified Android requirement; the current query-response path avoids passive broadcast receive dependence.
