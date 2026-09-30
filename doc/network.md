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
| `internal/network/online_server.go`, `online_config.go`, `host_online.go`, `online_rooms.go` | Jogo online pelo relay: URL da configuracao, endereco do relay com cache, sala do host, lista de salas. |
| `internal/relay/`, `cmd/relay/` | Protocolo, servidor e cliente do relay (so biblioteca padrao). |
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
| `compress` | host -> peer | Ultima linha em texto puro: dali em diante o fluxo desta conexao e deflate. So vai para quem pediu `compress: "deflate"` no `join`. |
| `ping` / `pong` | client -> host -> client | Eco para medir a latencia que o HUD mostra ("PING 45 ms"). A cada 2 s. |
| `join_rejected` | host -> peer | Entrada recusada: `full` (partida com `MaxPartySize` humanos) ou `version` (`proto` diferente). O cliente volta ao menu com a frase certa. |

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

**Aparecer na lista nao e a mesma coisa que ser alcancavel.** O caminho 1 acima e
de SAIDA do host (ele transmite), e nenhum firewall de entrada o barra; entrar na
partida depende da porta de jogo (TCP 9000) estar aberta NA MAQUINA DO HOST. Uma
sala visivel e inalcancavel e, por isso, um estado normal e esperado — e o que
"Abrir a conexao do cliente" abaixo existe para tratar.

## Abrir a conexao do cliente

`network.StartClientSession` (`internal/network/client_connect.go`) e a porta
unica: o menu escolhe a sala e a cor, e dali em diante socket, prazo e join sao
da camada de rede. Tres regras, e as tres nasceram do mesmo bug:

1. **O dial tem prazo** (`dialTimeout`, 3 s, o mesmo da reconexao). Uma porta
   FECHADA responde na hora (RST); uma porta bloqueada por firewall de entrada
   DESCARTA o SYN, e um `net.Dial` sem prazo ficava pendurado o tempo limite do
   sistema — uns dois minutos no Android.
2. **Enquanto o dial esta pendente, a sessao esta em espera e diz isso.**
   `AwaitingHost()` (conectando **ou** reconectando) congela o quadro em
   `game/loop.go` e o renderer desenha a sobreposicao — "Conectando ao
   anfitriao..." ou "Reconectando...". `TickReconnect` vigia esse estado ANTES
   de testar `CurrentClient`, com um teto folgado (`connectCeiling`, 10 s) para
   o caso de o dial nao respeitar o proprio prazo.
3. **Falhar volta ao menu com o motivo.** `network.SessionFailure` diz QUAL
   falha (`FailureConnect` = nunca conectou; `FailureReconnect` = conectou, caiu
   e esgotou `ReconnectWindow`); `ui/menu.go` escolhe a frase. Nao ha
   `log.Fatalf` no caminho de conexao.

O sintoma que isso corrige, para reconhecer se ele voltar: o cliente entra num
mapa **sem inimigos e sem outros jogadores**, andando normalmente, sem
sobreposicao nenhuma, e o host nao o registra. Papel de cliente nao simula nada
(so o host chama `UpdateSimulation`), entao "sem conexao" e "mundo vazio" tem a
mesma aparencia. O log do HOST separa as duas causas: com
`[Host] New client connected` a conexao existe e o problema e no join; sem ele, o
SYN nao chegou e o problema e alcancabilidade.

## Jogo online (relay na Oracle)

Fora da rede local, a partida passa por um **relay** proprio (`cmd/relay`,
pacote `internal/relay`) numa VM Always Free da Oracle (`infra/oci/legiao.tf`).
Ninguem instala nada alem do jogo e ninguem abre porta em roteador: todo mundo
conecta de SAIDA no relay, que so emparelha as pontas e copia bytes. A partida
continua simulada no aparelho de quem hospeda.

1. **"CRIAR PARTIDA"** continua so na rede local. **"CRIAR PARTIDA ONLINE"**
   (so aparece com `network.OnlineConfigURL` preenchido): personagem, depois nome
   da sala num teclado desenhado pelo jogo (`ui/onscreen_keyboard.go`, porque
   o raylib nao abre o teclado do Android), depois `network.OpenOnlineRoom`
   registra a sala e so entao a partida comeca (`Host.ServeOnline`). Se o
   relay nao responde, o jogador le isso na mesma tela.
2. O relay gera um **codigo unico** (`#K7M2Q`); nomes podem repetir.
3. "ENTRAR EM PARTIDA" consulta `LIST` a cada 3 s (`online_rooms.go`) e mostra
   as salas como `ONLINE <nome> #<codigo> <jogadores>` antes das da LAN, com o
   estado do servidor no subtitulo (`ui/discovery_entries.go`).
4. Escolher uma sala vira o `ServerAddress` `sala:<codigo>`. `dialGame`
   (`online_server.go`) e o unico ponto de discagem do cliente, no connect e na
   reconexao.
5. A conexao chega ao MESMO `acceptLoop` da LAN, com `RemoteAddr` proprio
   (`relay/<id>`), porque `h.peers` e indexado por ele.

**A sala morre com o host.** O relay manda `PING` a cada 15 s e derruba a
sala (e as conexoes dos jogadores dela) sem `PONG` por 45 s. O app so responde
enquanto o laco do jogo roda (`network.MarkFrame`, chamado todo quadro em
`game/loop.go`): app fechado, em segundo plano ou sem rede = sala some. No
fim do jogo `network.ShutdownOnline` fecha a sala na hora. Quando o jogo volta
a rodar, o host se registra de novo com o MESMO codigo (codigo + segredo), e
um cliente dentro da janela de reconexao a encontra.

**Onde o app acha o relay** (`online_config.go`): o IP NAO fica no app. O app
le `network.OnlineConfigURL`, um objeto publico (`online.json`, bucket
`legiao-public`, leitura sem listagem) que o Terraform reescreve com o IP
atual da VM: `{"relay":"ip:9000","enabled":true}`. A URL nao muda se a VM for
recriada. Consulta no maximo a cada 10 min; sem rede, vale a ultima copia boa
(memoria, depois `online.json` na pasta do `settings.json`). Um relay que nao
responde forca reconsultar. `"enabled": false` (variavel `online_enabled` do
Terraform) desliga o online em todos os apps sem atualizar nenhum. Nao usa o
Vault de proposito: o endereco nao e segredo, e um cofre exigiria credencial
dentro do app publicado.

### Capacidade do relay

Com o fluxo comprimido, o gargalo da VM Always Free
(`VM.Standard.E2.1.Micro`: 1/8 de OCPU garantido, 1 GB, ate 50 Mbps) passou a
ser a **CPU**, nao a banda. Medido em 29/09/2026 com um gerador de carga contra
o proprio relay (cada amigo recebendo 1650 bytes a 20 Hz = ~33 KB/s, o pior
caso comprimido, e mandando 100 bytes a 20 Hz de input):

| Amigos jogando | CPU (de um nucleo) | Memoria | Saida |
|---|---|---|---|
| 50 | 6,3% | 6 MB | ~14 Mbps |
| 70 | 7,9% | 7 MB | ~20 Mbps |
| 100 | 13,5% | 8 MB | ~28 Mbps |

~0,12% de nucleo por amigo. Limites em `relay.NewServer`: **64 amigos no
servidor** (~8% de um nucleo, 65% do 1/8 garantido, com folga para a CPU da
Oracle ser mais lenta que a da medicao; ~17 Mbps), **10 por sala** (11 com o
host), 50 salas (uma sala vazia custa quase nada), 512 conexoes (2 por amigo +
controle + `LIST`), 40 por IP, 90 s de partida sem trafego derruba a conexao,
teto de 8 TiB enviados por mes (em memoria; reiniciar zera). A 33 KB/s por
amigo, 8 TiB sao ~70 mil horas-amigo por mes.

Saude: `STATS LEGIAO2` (output `health_check` do Terraform); `taxa_kbs` e o
envio medio dos ultimos 10 s. Builds anteriores a compressao usam a tag
`net1` e nao aparecem na lista (`OnlineGameTag` = `net2`), entao todo amigo
contado no teto esta comprimido.

A dificuldade do jogo foi afinada para ate 5 jogadores;
salas de 11 funcionam, mas a fase
fica mais facil.

Regras que valem nao reinventar:

- **O aperto de mao le um byte por vez** (`relay.readLine`): ler com buffer
  engoliria o inicio do fluxo JSON que vem logo depois.
- **`network.OnlineGameTag`** separa versoes do jogo na lista. Suba quando o
  que vai no fio mudar de forma incompativel.
- **`relay.Version`** (`LEGIAO2`) e a versao do aperto de mao; mudar exige
  publicar o binario novo do relay junto.
- **O IP muda se a VM for recriada**, e isso so quebra alguem se o
  `online.json` nao for reescrito: rode `terraform apply` depois. O Terraform
  ainda ignora mudancas de imagem e de metadados da instancia para nao recriar
  a VM sem querer.
- Salas sao publicas (quem tem o app ve e entra). Sem criptografia.

## Envio do host: fila por jogador e compressao

**O laco do jogo nunca espera a rede** (`host_peer_writer.go`). Cada conexao
tem uma fila e uma goroutine que escreve; `broadcast`/`sendTo` so empilham.
Antes o host dava `Flush` em cada jogador dentro do quadro, segurando
`peersMutex`: um celular com 4G ruim enchia o buffer TCP e travava a partida
de todo mundo. Regras da fila:

- **Snapshot** (`isSnapshotType`: jogadores, inimigos, projeteis, cooldowns)
  nao acumula: o novo substitui o anterior do mesmo tipo que ainda nao saiu.
  Quem esta atrasado recebe o estado mais recente.
- **Evento** nunca e descartado nem reordenado; snapshot nao pula evento.
- Tudo o que esta na fila sai num lote com UM `Flush` (as tres mensagens do
  tique viram um pacote).
- Lote que nao sai em 5 s (`peerWriteTimeout`) ou fila com 1024 itens fecha a
  conexao; o cliente reconecta e o resto da partida segue.
- `ClientConn` montado a mao (testes) nao tem fila e continua sincrono.

**Compressao** (deflate, `flate.BestSpeed`): o cliente pede no `join`
(`Compress: "deflate"`), o host responde com a linha `compress` e comprime
tudo dali em diante nesta conexao (`client_inflate.go` troca o leitor, e a
drenagem de snapshots para no marcador). Cliente ou host antigos simplesmente
nao negociam e seguem em texto puro. Medido com o tique do pior mapa (11
jogadores, 83 inimigos, 20 projeteis) numa CPU de servidor de 2,1 GHz:

| Etapa | Custo por tique | Trafego por amigo |
|---|---|---|
| JSON (ja existia) | 103 us | ~224 KB/s |
| Comprimir, BestSpeed, por amigo | 57 us, na goroutine de envio | ~33 KB/s (6,7x menos) |
| Comprimir, nivel padrao | 203 us | ~24 KB/s (9,4x) - nao compensa |
| Descomprimir no cliente | 33 us, na goroutine de leitura | - |

Num celular ~4x mais lento isso da ~0,5% de um nucleo no cliente e ~5% no host
com 10 amigos, fora da thread do jogo. Ganho real e maior que o medido: no
teste as posicoes mudavam inteiras a cada tique. O cliente->host (input) segue
em texto puro: e pequeno. `TCP_NODELAY` ja e o padrao do Go nas duas pontas e
no relay.

## Protecoes do host (`host_guard.go`)

Qualquer pessoa com o app pode entrar numa sala online, entao o host se
protege (a LAN ganha junto):

- **Linha maxima de 64 KiB** por mensagem do cliente (`maxClientLine`, tamanho
  do `bufio.Reader`); maior que isso derruba a conexao. Antes o host decodificava
  JSON sem limite.
- **Taxa maxima de 120 mensagens/s** por jogador (rajada de 240); o excesso e
  descartado. O jogo legitimo manda ~20-60/s.
- **`MaxPartySize` = 11** humanos com o host (casa com os 10 amigos por sala do
  relay); bots nao contam, e quem ja tem vaga (reconexao) sempre volta.
- **Versao**: o cliente manda `proto` (`OnlineGameTag`) no `join`; diferente
  da do host e recusada. Vazio (cliente anterior ao campo) e aceito.
- Recusas vao como `join_rejected` e fecham a conexao DEPOIS de entregue
  (`closeAfter` na fila de envio).

## Mensagens ao jogador (online)

Nenhum erro cru chega a tela: `network.OnlineErrorText` (`online_errors.go`)
separa "o servidor recusou" (`relay.RefusedError`: sala cheia, sala encerrada,
servidor lotado, limite do mes, versao incompativel) de "o servidor nem
respondeu" (internet do jogador, servidor fora). Onde aparece:

- **Criar sala online**: a frase exata na propria tela de nome.
- **Entrar / reconexao esgotada**: `network.LastFailureDetail()` no aviso do menu.
- **Reconectando...**: `network.ReconnectHint()` abaixo da contagem ("O
  anfitriao saiu ou perdeu a conexao. Aguardando ele voltar...").
- **Host em jogo**: `ui.DrawNetworkBadge` sob a barra de vida - "SALA ONLINE
  #K7M2Q", ou "SERVIDOR ONLINE FORA - RECONECTANDO..." quando a ligacao com o
  relay cai (`relay.Listener.Connected`).
- **Lista de salas**: subtitulo diz se o servidor esta fora ou desativado;
  salas com vaga primeiro, as mais cheias antes.
- **Cliente em jogo**: `ui.DrawNetworkBadge` mostra "PING 45 ms" (verde abaixo
  de 80, amarelo ate 150, vermelho acima) - `client_ping.go`.
- **Entrada recusada pelo host**: "A partida esta cheia." / "Sua versao do jogo
  e diferente da do anfitriao."

## Taxa de publicacao e interpolacao

**A simulacao roda a 60 Hz; a PUBLICACAO dela roda a 20 Hz** (`SnapshotHz`, em
`internal/network/broadcast_rate.go`). Cooldown, dano e morte continuam sendo
decididos a cada quadro pelo host — o que desceu foi so a frequencia com que o
estado vai para o fio.

O cliente cobre o vao **interpolando** (`interpolation.go`,
`interpolation_clock.go`): guarda os ultimos snapshots num buffer curto,
carimbados com o **relogio do host** (`HostT`, campo `t` de `state_update` e
`enemy_update`), e desenha um instante um pouco atras do mais novo, entre os
dois que o cercam.

**Ate 28/09/2026 a interpolacao nao funcionava**: guardava so os dois ultimos
snapshots (50 ms de historia), carimbados pela hora de CHEGADA, e desenhava
100 ms atras - fora do intervalo. O fator de mistura saia sempre 0 e jogadores
remotos e inimigos andavam aos saltos de 20 Hz. `state_update` e
`enemy_update` do mesmo tique ainda viravam dois snapshots a 0 ms um do outro.

Regras que valem nao reinventar:

- **Relogio do host, nao hora de chegada.** O 4G entrega em rajadas; carimbar
  pela chegada transformaria cada rajada num tranco. Mensagens com `HostT` a
  ate 5 ms uma da outra sao o mesmo tique e viram um snapshot so. Host antigo
  (sem `t`) cai na hora de chegada e ainda interpola.
- **O atraso se adapta a rede.** Cada chegada da uma amostra `hora local -
  HostT`; a menor das ultimas ~40 e a diferenca de relogios, e o quanto as
  outras passam dela (percentil 90) e a oscilacao. Atraso = 100 ms + oscilacao,
  entre 100 e 300 ms, mudando devagar (5%) para nao dar solavanco. Medido em
  simulacao: LAN ~114 ms e 0 quadros parados; 4G oscilando ate 120 ms ~194 ms
  e 1 quadro parado em 188.
- **O atraso e o preco, e ele e o certo aqui.** O host e autoritativo e ja
  decide todo o combate; o cliente precisa de posicao remota CONTINUA, nao
  instantanea.
- **Nunca extrapola.** Sem snapshot novo, segura no ultimo ponto conhecido.
- **Interpola so o que e continuo**: posicao e velocidade. Vida, morte, quadro
  de animacao e identidade vem do snapshot mais recente sem misturar.
- **`ResetInterpolation()` no reinicio da fase e na reconexao.** Esquece o
  buffer e o relogio.
- **Nada publica fora do tique.** Ja aconteceu tres vezes de um caminho
  paralelo transmitir a 60 Hz e anular a taxa: o handler de `MsgInput`,
  `UpdatePlayerState` e `UpdatePlayerPosition`. Os tres so atualizam estado.

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
  sessao nova e o jogador perde o slot — nao ha persistencia em disco. A
  EXCECAO e o REINICIAR E VOLTAR de Configuracoes > Graficos (29/09/2026):
  antes de matar o processo ele grava `resume.json` (`internal/resume`, uso
  unico, vale 4 min) com id, cor, personagem e endereco, e o processo novo
  (`ui/resume_session.go`) reconecta com o MESMO id — o host ve uma
  reconexao comum. Do lado do host a partida morre com o processo; o
  bilhete guarda `StageMap()` e a sala online (nome, CODIGO e segredo), e ele
  volta no comeco da fase (`StartTravel` aplicado antes do primeiro quadro,
  `game/loop.go`). A sala volta com o MESMO codigo (`OpenOnlineRoomAs` ->
  `relay.HostAs`: o servidor devolve o codigo a quem tem o segredo, ou a
  qualquer um se ele estiver livre), entao os amigos, cuja reconexao disca o
  codigo antigo, voltam sozinhos — como na rede local. Voltam como join novo
  (a partida e nova). Testes: `network/session_resume_test.go`.
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

## Janela de imunidade (`PlayerState.Invuln`)

`Invuln float32` (`json:"invuln,omitempty"`) são os segundos que restam da
janela de imunidade que o host concedeu a um jogador (`GrantInvulnerability`:
renascer, resgate, reerguimento da vitória). Estado de tique, como `InPortal`:
veredicto do host, zero fora da janela — o `omitempty` não custa um byte no
tique comum. `Host.publishInvulnerability` (`host_invulnerability.go`) copia as
janelas para `h.players` a cada quadro, e `UpdateLocalPlayerState` preserva o
campo como preserva `InPortal`/`Absent`.

Só o desenho lê: é o que faz o corpo piscar em toda máquina
(`entity.InvulnBlinkGray`). A fase do piscar vem do relógio local, não do
número — ele chega a ~20 Hz e serrilharia o ritmo. E ele lê o **mapa** de
janelas, não `IsInvulnerable`: a paz de vitória responde verdadeiro para todos
até o fim da fase, e piscar por ela faria a cena de encerramento inteira
piscar.

A velocidade extra do renascer **não** viaja: a borda morto → vivo já chega a
toda máquina, e cada uma arma o próprio bônus (`network.ArmLocalRespawnBoost`).

## Espera no portal (`PlayerState.InPortal`)

Com bots no grupo, a porta quase nunca ficava
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

## Todo mundo na mesma build

O jogo é host-autoritativo, mas o jogador **local** é simulado no próprio quadro (`interpolation.go`: o local nunca passa pela interpolação). Isso significa que qualquer constante de simulação compartilhada tem de ser a mesma nas duas pontas — com `entity.GameSpeed` diferente entre host e cliente, o cliente prevê 270 px/s, o host corrige para 200 e o resultado é rubber-banding contínuo.

**Não há verificação de versão de protocolo no `MsgJoin` hoje.** Um cliente de build antiga entra normalmente e o sintoma aparece só como "está borrachudo". Um campo de assinatura de simulação no `JoinPayload`, com o host recusando (ou avisando) quando diferente, é o próximo passo natural — e paga a si mesmo em todo ajuste futuro de ritmo ou velocidade.

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
- Windows firewall must allow inbound TCP 9000 for the host. A regra do firewall
  e por CAMINHO do executavel: renomear ou mover o binario do jogo faz o Windows
  perguntar de novo, e um "Cancelar" nessa pergunta fica gravado como bloqueio.
  O sintoma no cliente e "Nao foi possivel conectar ao anfitriao (porta 9000
  bloqueada?)" mesmo com a sala aparecendo na lista — ver "Abrir a conexao do
  cliente" acima.
- Discovery is LAN-only, not internet matchmaking.
- Do not reintroduce Java/MulticastLock code unless there is a verified Android requirement; the current query-response path avoids passive broadcast receive dependence.

## Porta de despertar das gárgulas (`EnemyUpdatePayload.SentriesAwake`)

`SentriesAwake bool` (`json:"sentries_awake,omitempty"`) é o veredicto de
`Host.sentriesMayFire` (`sentry_wake.go`), publicado a cada quadro em
`handleSentryOrbTick` e enviado no mesmo pacote que as próprias gárgulas. Só a
seta de borda de tela lê (`network.SentriesAwake`, `LivingSentryCenters`): no
mapa 4 ela espera "As sentinelas despertaram". Não é derivado no cliente porque
a porta é "maioria dos vivos no degrau, com trava" — duas máquinas contando o
grupo em quadros diferentes podiam discordar.
