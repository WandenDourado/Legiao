# Changelog

## 2026-09-28 — Seta das gárgulas só depois de elas despertarem

No mapa 4 as setas de borda de tela já apareciam na chegada, apontando para
torres ainda caladas. Agora elas esperam o mesmo momento em que as gárgulas
podem atirar ("As sentinelas despertaram"): o host publica o veredicto da porta
de despertar (`network.SentriesAwake`) e o envia ao cliente em
`EnemyUpdatePayload.SentriesAwake`. Mapas sem porta (5 e 7) não mudam.
`TestSentryMarkersWaitForTheWake`.

## 2026-09-28 — Setas das gárgulas de volta à tela; o ápice da fase mede o grupo somado

**Setas das gárgulas não apareciam (bug).** A chamada `drawSentryMarkers` em
`game.DrawFrame` sumiu numa edição paralela de `renderer.go` (a do teste de
desempenho e da desintegração); sem ela o linker descartou o indicador inteiro
e o executável compilou sem aviso. Chamada restaurada, e
`TestRendererDrawsSentryMarkers` lê o `renderer.go` e exige a chamada.

**Ápice da fase (`on_last_stand`).** A condição deixou de ser "todo mundo de
pé abaixo de 30%" e passou a ser a vida do grupo somada em até 35% da máxima
somada (`dialogue.LastStandPartyHealth`, `game.partyHealthShare`). Um
personagem quase cheio com o resto do time caído já dispara; o grupo inteiro
de vida cheia, não. Ver `doc/combat_rules.md`, "O último suspiro".

## 2026-09-28 — Mais visão: zoom 0,8 no celular e setas das gárgulas fora da tela

Jogadores relataram pouca visão do mapa; o Arqueiro precisava andar até a
gárgula para saber onde ela estava. Duas mudanças (`doc/camera.md`):

- **Zoom por plataforma** (`Config.CameraZoom`): celular 0,8, PC 1,0. O clamp
  da câmera passou a medir em unidades de mundo, a ancoragem de pixel passou a
  ser em pixel de tela, e a câmera nunca se afasta além do mapa (corredor do
  mapa 6 fica em ~0,94 num celular de 2400 px, sem faixas pretas).
- **Setas das gárgulas** (`ui/offscreen_marker.go`, `game/sentry_markers.go`,
  `network/sentry_markers.go`): uma seta violeta na borda da tela para cada
  gárgula viva fora da visão, mais forte perto e discreta longe.

Não medido: fps no celular com ~56% mais área desenhada (olhar o F3 nos mapas
2 e 3). Testes novos: `game/camera_zoom_test.go`,
`ui/offscreen_marker_test.go`. `go build`/`go vet`/`go test ./internal/...`:
as mesmas falhas pré-existentes, nenhuma nova.

## 2026-09-28 — Aba Gráficos: qualidade, teste automático, resolução do celular, configuração salva no Android

Depois do estudo `plans/plan_qualidade_grafica_mobile.md` (celulares antigos
travando). Implementado o que dá mais ganho:

- **Configuração salva no Android** (`settings/store_android.go`, via
  `rl.HomeDir()` = armazenamento privado do app). Pré-requisito de tudo: até
  hoje qualquer escolha voltava ao padrão ao fechar o app.
- **Resolução de desenho no celular** (Nativa/900p/720p/540p): a janela nasce
  menor e o scaler de hardware amplia de graça. 720p numa tela 2400×1080 são
  56% menos pixels em tudo. Vale na próxima abertura; o zoom da câmera é
  reescalado para mostrar o mesmo mapa.
- **Qualidade Baixa/Média/Alta** (30/60 fps, desintegração, flores e galhos,
  resolução sugerida) e **Configurar sozinho**: teste de alguns segundos que
  mede o aparelho em cada resolução e escolhe.
- **Flores e galhos no chão** desligáveis (ganho pequeno, medido).
- Aba nova **Gráficos**; a chave da desintegração saiu de Vídeo para lá.

Verificado no container: `go build ./...`, `go vet`, testes novos
(`settings/quality_test.go`) e as mesmas falhas pré-existentes nos outros
pacotes; a aba e o teste automático rodaram de verdade num display virtual
(inclusive com o modo celular forçado). **Não compilado para Android** (sem
NDK aqui): conferir `store_android.go` e a abertura em 720p no primeiro build.

## 2026-09-28 — Monstros viram cinzas, chave para desligar, engrenagem ao lado da horda

Pedido do Gui depois de aprovar o preview (`plans/plan_morte_desintegracao.md`).

**Desintegração.** Todo monstro que morre se desfaz em cinzas levadas pelo
vento, com frente em brasa, a partir do quadro exato que estava na tela.
Sem rede: cada máquina percebe a morte comparando o que desenhou com a lista
autoritativa. Teto de fragmentos por plataforma, grão maior no Android, e a
versão lenta da Imperatriz segura o diálogo final até o corpo dela sumir.
Detalhes em `doc/combat_rules.md`, "Morte dos monstros".

**Chave nas Configurações.** Vídeo e desempenho → "Monstros viram cinzas"
(`settings.DeathEffect`, padrão ligado), por causa dos relatos de lentidão e
aquecimento no celular. `settings.Load` passou a partir de `Default()` para
um settings.json antigo não desligar a opção nova.

**Engrenagem.** Saiu do canto e foi para a esquerda do contador "Horda
N/M", com mais que o dobro do tamanho e mais opaca.

Verificado no container: `go build ./...`, `go vet` e `go test` de
`entity`, `settings`, `ui` e `game` (as falhas que aparecem já existiam
antes: `enemy_boss_anim`, `enemy_chase_slack`, `slider` e as de mapa que
dependem de `assets/`); o efeito e a engrenagem foram renderizados de verdade
pelo raylib num display virtual. Falta: jogar (host e cliente) e medir no F3
do celular com a chave ligada e desligada.

## 2026-09-27 — Retorno do playtest: dano invisível da Imperatriz, Game Over na vitória, fôlego do renascer, fogos na TV, chuva +2%

Cinco pontos levantados pelo Gui depois de jogar a campanha
(`plans/plan_playtest_27_09.md`, executado e removido).

**Game Over depois da vitória (bug).** Névoa e espinhões da Imperatriz
continuavam no ar depois da queda; o diálogo final congelava a simulação e,
quando fechava, a névoa (30/s no mapa inteiro, até 8 s) matava o grupo de 200
de vida. A queda agora apaga névoa, espinhões e esferas de gárgula
(`clearThreatsOnVictory`; o cliente deriva o mesmo do `BossState`) e liga a
paz de vitória: `IsInvulnerable` verdadeiro para todos e nenhum Game Over até a
fase virar.

**Dano "de lugar nenhum" na Imperatriz (bug).** O `EnemyDef` dela
(`AttackDamage` 22, `AttackRange` 1400) passava pelo caminho genérico de
ataque, que cobrava 22 de alguém a até 1530 px a cada 4 s, sem nada na tela.
`checkEnemyPlayerCollisions` agora pula chefe. De quebra: o anel de contato só
fere espectro (pedido do Gui) e só aparece com Legião em campo; o espinhão
fotografa e mede pelo pé, não pelo centro do sprite.

**Fogos que caíam sem estourar (bug).** A física do foguete era em px
absolutos (topo da subida ≈ 1502 px) e a linha de estouro, proporcional à
tela — numa TV 4K a maioria caía. Agora tudo escala por
`screenHeight/1080`, o foguete estoura no topo da subida em qualquer caso e o
passo tem teto de 1/20 s.

**Fôlego do renascer.** 30% de vida (sem mudança), 2,5 s de imunidade, passo
1,4× e o corpo piscando colorido ↔ cinza. Campo novo `PlayerState.Invuln`
(piscar em toda máquina, para toda janela concedida). A gárgula não mira quem
está imune. A animação de caminhada passou a acompanhar os multiplicadores de
passo (conserta também o deslize do Santuário).

**Chuva de Meteoros +2%.** `MeteorRainDensityBoost` = 1,02 na densidade de
referência; world_05 intocado (teto de desempenho).

Testes novos: `network/host_playtest_2709_test.go`,
`ui/victory_fireworks_test.go`, `entity/invuln_blink_test.go`,
`TestMeteorRainBoostReachesTheMagoMaps`. `go build`/`go vet`/`go test
./internal/...` no container de nuvem: as mesmas falhas pré-existentes de
antes, nenhuma nova.

## 2026-09-21 — Ritmo do jogo reafinado: 1.35 → 1.20

O Gui jogou com o `GameSpeed` de 1.35 (270 px/s) e achou rápido demais.
Baixado para **1.20** (240 px/s) — troca de um número só, em
`internal/entity/gamespeed.go`. As duas invariantes que o fator não pode
quebrar continuam de pé no valor novo: o pé do orc não desliza
(`TestWalkCycleKeepsUpWithTheFeet`) e o alcance de ataque não mudou
(`TestAttackReachSurvivesGameSpeed`, `TestBlindShotDiesAroundTheEdgeOfTheRange`).
`go build`/`go vet`/`go test ./internal/...` rodados de novo no container de
nuvem: as mesmas 16 falhas pré-existentes de sempre, nenhuma nova. `doc/game_speed.md`
atualizado com o número e o histórico do ajuste.

## 2026-09-20 — Retorno do playtest: texto maior, jogo mais rápido, cemitério no chão e o diálogo que não destravava

Quatro pontos levantados por jogadores numa sessão de teste com grupo.

**Diálogo travado na reconexão (bug).** Uma testadora caiu durante um diálogo;
quando voltou, o grupo já jogava a fase e a tela dela continuava na caixa,
esperando um avanço que nunca viria. Causa: `Host.sendDialogueTo`
(`network/dialogue.go`) tinha um `if !d.Active { return }` — escrito para quem
entra no **meio** de uma cena, calado justamente para quem volta **depois** que
ela acabou, já que `Active:false` é publicado uma vez, como evento, por
`DialogueDirector.end`. O cliente guardava o `Active:true` de antes da queda, e
`DialogueActive` é o portão de pausa de input e simulação: ela voltou para um
jogo congelado numa caixa que só o host podia avançar. Agora o host sincroniza
o estado seja ele qual for, e `triggerReconnect` (`reconnect.go`) zera
`CurrentDialogue` do lado do cliente ao perceber a queda — o que cobre também o
caso inverso (cair fora de um diálogo e voltar dentro de um).
Testes: `network/dialogue_rejoin_test.go`.

**Cemitério do Necromante deixou de passar por cima dos personagens.** Ele é
chão amaldiçoado, mas era desenhado inteiro depois dos corpos, como toda magia
de `ability.DrawAll`. Dividido em dois passes: `DrawGround` (solo, rim,
manchas, miasma e o buraco sob cada mão) antes dos corpos, junto com o selo do
Mago; `DrawOverlay` (os ossos das mãos e as almas) continua depois. Mover tudo
para antes seria uma linha só e custaria o efeito: é a mão agarrando a perna de
quem está dentro que faz a magia dizer o que ela faz.
Arquivos: `skill/graveyard_draw{,_ground,_overlay}.go`, `graveyard_manager.go`,
`ability/necromancer.go`, `game/renderer.go`. Regra registrada em
`doc/skill_performance.md`.

**Ritmo do jogo: `entity.GameSpeed = 1.35`.** A queixa não era de
balanceamento — as proporções entre lobo, orc e jogador estavam certas — era
que tudo se move devagar ao mesmo tempo. Um fator único, aplicado em quatro
pontos: `PlayerSpeed` (os bots vêm junto), `RegisterEnemy` (`Speed == 0` fica
0), o `FrameTime` das animações de caminhada (sem isso o orc patina: o
`0.156` dele é a solução de uma equação que tem a `Speed` dentro) e as
velocidades de projétil **com o tempo de vida dividido pelo mesmo fator**, para
o alcance não mudar. Esse último detalhe pegou um bug real em revisão: o
`CannonBallTTL` é um orçamento de voo calibrado contra `CannonRange`, e sem a
divisão o tiro de aviso do canhão passava a machucar 1589 px além do alcance
declarado (`TestBlindShotDiesAroundTheEdgeOfTheRange`). Recarga, dano, cadência
e duração de área **não** levam o fator. Novo doc: `doc/game_speed.md`.
Testes: `entity/gamespeed_test.go`.

**Texto de HUD e diálogo.** A "Escala do HUD" existia desde 04/09 e era lida
por um único lugar — `DrawHealthBar`. As outras ~58 chamadas de `rl.DrawText`
ignoravam a preferência do jogador. Agora `internal/ui/text.go` é o ponto
único (`TextSize` aplica escala + piso 18; `FitTextSize` aplica só o piso, para
tamanhos já derivados de geometria escalada), os tamanhos de fábrica subiram, e
o diálogo ganhou faixa por papel em vez de um piso 16/teto 34 para os três — o
teto era o que o impedia de crescer num monitor grande. A caixa de diálogo
passou a derivar a altura do texto que promete caber, e **encolhe a fonte em
vez de cortar a fala** quando ela ainda não cabe: perder tamanho é reversível,
perder o fim de uma fala de narrativa não é. O slider foi de `0.8..1.3` para
`0.8..1.8` (`settings.HUDScaleMax`) — as escalas de controle continuam nas
faixas antigas, porque aquelas mexem em alvo de toque. Texto em espaço de
mundo (`portal_counter.go`) ficou fora de propósito: já escala com o zoom.
Testes: `ui/text_test.go`, `ui/dialogue_geometry_test.go`.

**Verificação.** `go build ./...`, `go vet ./...` e `go test ./internal/...`
rodados de verdade no container de nuvem (receita em
`doc/running_desktop.md`/memória de build). As 16 falhas restantes são
pré-existentes e idênticas com `GameSpeed` em 1.0 e em 1.35 — a maioria por
assets ausentes naquele ambiente, mais `TestSliderStepUpDownClampAtBounds` e
`TestBossDodgeWindowMatchesThornTelegraph` (comparação de `float32` sem
tolerância) e `TestHiddenOnlyEverHidesWhatAnOpaqueCellCovers`. **Falta o teste
manual:** sessão de 3 máquinas derrubando o Wi-Fi de uma durante um diálogo e
religando depois do fim da cena; o toque real no celular com a escala em 0.8,
1.0 e 1.8; e um playtest com o grupo para confirmar o 1.35.

## 2026-09-14 — Cliente que nunca conecta: prazo no dial, estado visível e volta ao menu com o motivo

Relato do Gui: host no PC, jogador entrando pelo Android — o Android cai numa
fase vazia, só com ele, e o host vê o jogo normal mas sem o jogador do Android.
Android hospedando funciona. O diagnóstico é que a conexão **nunca abriu**, e que
esse estado não existia na máquina de estados do cliente:

- `connectToHost` punha `network.Role = "client"` antes de qualquer socket, e só
  atribuía `CurrentClient` quando o dial voltava. No intervalo, papel de cliente
  não simula nada (só o host chama `UpdateSimulation`), então a tela mostrava o
  mapa sem inimigos e sem os outros jogadores, com o jogador local andando —
  uma falha de rede desenhada como um jogo funcionando.
- `TickReconnect` devolvia na primeira linha enquanto `CurrentClient` fosse nil,
  então o relógio de silêncio de 5 s (`client_liveness.go`) nunca começava a
  contar. Ninguém vigiava.
- `ConnectClient` usava `net.Dial` **sem prazo**. Porta fechada responde RST na
  hora; porta bloqueada por firewall de entrada DESCARTA o SYN, e aí o dial
  ficava pendurado o tempo limite do sistema (~2 min no Android) antes de um
  `log.Fatalf` que **fechava o aplicativo**.

O que mudou:

- **`internal/network/client_connect.go` (novo):** `StartClientSession` é a porta
  única para abrir a sessão de cliente — guarda identidade, entra no estado
  "conectando", disca em segundo plano e manda o join. `Connecting()`,
  `AwaitingHost()` (conectando **ou** reconectando) e um teto folgado
  (`connectCeiling`, 10 s) vigiado por `TickReconnect` **antes** do teste de
  `CurrentClient`. Uma conexão que volte depois do teto é fechada em vez de
  adotada (geração de tentativa).
- **`ConnectClient` passa a usar `net.DialTimeout`** com o mesmo `dialTimeout`
  (3 s) do caminho de reconexão.
- **`network.SessionFailure`** (`FailureConnect` / `FailureReconnect`) substitui
  o booleano `gaveUp`: a camada de rede diz o QUE aconteceu, `ui/menu.go` escolhe
  a frase. `failSession` é o caminho único de fim de sessão — a janela de
  reconexão esgotada e o dial que nunca completou terminam iguais.
- **`ui.DrawConnectingOverlay`** ("Conectando ao anfitriao..."), irmã da de
  reconexão; o fundo escurecido e o título viraram `drawWaitBackdrop`
  compartilhado. `game/loop.go` congela o quadro por `AwaitingHost()` em vez de
  só `Reconnecting()`.
- **Sem `log.Fatalf` no caminho de conexão:** falhar volta ao menu com
  "Nao foi possivel conectar ao anfitriao (porta 9000 bloqueada?)".

Testes: `internal/network/client_connect_test.go` — join enviado ao conectar
(listener de verdade em loopback), falha quando ninguém escuta, teto encerrando
um dial pendurado, `TickReconnect` vigiando a conexão inicial com
`CurrentClient` nil (a regressão em si) e `AwaitingHost` cobrindo as duas
esperas.

Nota de rede, documentada em `doc/network.md`: **aparecer na lista não é ser
alcançável.** A sala chega ao Android pelo broadcast UDP de *saída* do host, que
nenhum firewall de entrada barra; entrar depende da TCP 9000 estar aberta na
máquina do host. A regra do Firewall do Windows é por caminho do executável, então
um binário renomeado/recompilado com nome novo pede permissão de novo.

## 2026-09-10 — Nova narrativa da campanha: os sete mapas reescritos, cena depois dos canhões, "Imperatriz das Trevas"

Pedido do Gui: substituir a narrativa do jogo pelo texto novo, mantendo
acentuação e pontuação exatamente como escritas.

- **Roteiros** (`assets/dialogues/world_01..07.json`): as 103 falas do texto
  novo, na ordem e com a pontuação originais — conferidas por script contra o
  texto do pedido, fala por fala. "Narrador"/"Narração" entra como `speaker`
  vazio (convenção de `doc/dialogue.md`). Os ids que já existiam ficaram
  (é o que marca a cena como vista).
- **Cenas novas:** fim de fase nos mapas 2, 3 e 5 (`world_0N_hordas_limpas`),
  abertura no mapa 7 (`world_07_abertura`) e o fim do mapa 6
  (`world_06_canhoes_destruidos`). As deixas do resgate continuam sendo a
  última fala de cada clímax ("Ergam-se.", "IRMÃOS, RECEBAM A LUZ DO
  DIVINO!", a prece do Arqueiro, "Que os céus caiam sobre eles!", "Eu sou a
  Avatar dos deuses!").
- **O mapa 6 não tinha como tocar uma cena "depois dos canhões".** O corredor
  não tem horda (nenhuma entrada em `waveRuns`), então o `WaveState` fica em
  Total 0 e `on_waves_cleared` nunca disparava. A condição do gatilho virou
  `stageFightOver` (`internal/game/dialogue_stage_cleared.go`): horda limpa,
  **ou** canhões declarados e todos caídos (`network.CannonsSilenced`) — a
  mesma regra que já abre o portal do corredor (`PortalsUnlocked`). Mapa sem
  horda e sem canhão continua sem disparar. Teste:
  `dialogue_stage_cleared_test.go` (corredor, reinício de fase, mapa de
  hordas, mapa quieto).
- **"Senhora das Trevas" → "Imperatriz das Trevas"** em todo texto visível: nome
  da chefe na barra (`entity/enemy_sprite.go`), anúncios de horda dos mapas 5 e
  7 (`network/wave_runs.go`) e subtítulo da vitória (`ui/victory.go`). O texto
  novo usa só "Imperatriz"; a barra dizendo outro nome na cena final seria
  contradição na tela. Nomes internos (`EnemyTypeDarkLady`, pasta
  `senhora_das_trevas`) não mudaram.
- **Cabe na caixa:** a fala mais longa (narração do mapa 6, ~290 caracteres)
  foi medida contra a geometria de `dialogueLayout` nas seis proporções de
  `dialogue_geometry_test.go` mais 1024x768 — sobra pelo menos uma linha em
  todas. Todo caractere usado está no Latin-1, que a fonte padrão desenha.

Verificado aqui: `go test ./internal/dialogue/` com os sete roteiros novos
carregando sem perder roteiro nem fala, ids únicos no jogo, um roteiro por
gatilho; `gofmt` limpo nos arquivos Go tocados. `game` e `network` não foram
compilados neste ambiente — rodar `go build ./...` e
`go test ./internal/game/ ./internal/network/ ./internal/dialogue/`.

## 2026-09-10 — Selo do Mago: um pentagrama de fogo marca quem chamou a Chuva de Meteoros

Pedido do Gui: "quando o mago usa a ultimate acontece a chuva de meteoros; eu
gostaria de adicionar um efeito de pentagrama flamejando em volta do mago,
enquanto a chuva está acontecendo, para sinalizar que ele é a origem de todo
aquele poder".

**O problema que ele descreve é real e é só desta suprema.** As outras três
nascem em cima de quem as lança — o altar da Sacerdotisa, o avatar dourado da
Paladina, a flecha que sai do arco. A Chuva de Meteoros sorteia alvos no mapa
INTEIRO (`skill.StepMeteorRains`), e cada pedra nasce 780 unidades acima do
próprio alvo: nada no que se vê na tela liga a tempestade ao conjurador.

**`skill.MeteorSigil`** (`internal/skill/meteor_sigil.go` + `_manager.go`) é a
resposta: um selo aceso no chão, aos pés do mago, pelos 15 s de
`MeteorRainDuration` — anel externo, anel interno, doze marcas de runa girando
ao contrário, a estrela de cinco pontas em três passadas (traço fundo, brasa,
núcleo claro), brasa acesa em cada vértice, ondas de calor a cada 1,6 s e
lufadas de brasa subindo. Zero regra de combate: não causa dano, não muda
alcance, não entra em teste de colisão nenhum.

**Ele é um CÍRCULO, não uma elipse.** A primeira ideia era achatar o Y para
"deitar" a figura no chão; `art_style.md` §12.2 diz o contrário — decalque de
chão é a única peça vista a 0°, "um círculo lê como círculo e não como elipse"
—, e é o que toda marca de chão do jogo já faz (anel do meteoro, poça do
Avatar, anéis do altar e do Santuário, chevron de mira). Achatar só este selo o
poria em perspectiva diferente da vizinhança, e em figura girante o achatamento
ainda distorce o ângulo desenhado (erro medido no indicador de mira em 02/09).

**A cor é brasa, não sangue.** `art_style.md` §12.1: "sigilo é giz, cinza e cera
— nunca pintado em vermelho". O selo é fogo (mesma família dos meteoros), então
o tom mais fundo da paleta é laranja queimado `(205,70,18)` e não vermelho.

**Uma mensagem nova no fio, e só uma.** Os meteoros continuam viajando um a um
sem dono (`broadcastUltimate("meteor_rain", "", alvo)`); o selo é o oposto — só
significa alguma coisa se estiver aos pés de UM jogador —, então `Cast`
transmite `"meteor_sigil"` com o `ownerID` UMA vez, no lançamento, e os dois
lados envelhecem a figura pelo mesmo relógio. A âncora segue o dono a cada
quadro por `SetMeteorSigilAnchor` (host: `tickMeteorSigils`; cliente:
`AdvanceClientSkills`), como a do Avatar. `ClearOwner` ganhou o sinal
`"meteor_sigil_end"` para a tomada de posse de bot; a chuva em si continua sem
dono depois de começada, como sempre foi.

**Pendência de estilo, para o Gui decidir.** `art_style.md` §11.6 proíbe símbolo
oculto e §12 abre a exceção "só no `world_07`". A regra nasceu para o TERRENO da
mata sombria (pentagrama no chão da floresta troca a história do bioma); este
selo é efeito de magia lançada pelo jogador, dura 15 s e aparece em qualquer
mapa. Se a proibição vale também para efeito de magia, a figura vira um círculo
rúnico sem a estrela — troca localizada em `Draw`. O doc ainda não diz qual das
duas leituras vale.

Verificado neste ambiente com `gofmt`, `go build`, `go vet` e
`go test ./internal/skill` (raylib compilou aqui desta vez, ver
`build_verification.md`); os pacotes `network` e `game` não foram compilados —
as mudanças ali são quatro inserções. Gui: rode `go build ./...`,
`go test ./internal/...` e olhe a suprema em jogo — o que se ajusta em um número
é `MeteorSigilRadius` (170) e `meteorSigilSpin` (11 °/s).

## 2026-09-10 — Dois defeitos visuais: anel de contato lendo como risco na barriga da chefe, e fio branco no atlas de tile

Relato do Gui: um "círculo vermelho no corpo" da Senhora das Trevas, e um risco
branco de 1 px atravessando o topo da tela na fala do Arqueiro. Diagnóstico
completo (medição de pixels, causas descartadas) fica fora do changelog; ver
`internal/skill/boss_contact.go` e `internal/tilemap/renderer.go`.

**Anel de contato (`skill.DrawBossContactRing`).** O raio, o centro
(`Enemy.HitCenter()`) e a ordem de desenho (depois do corpo, nos dois ramos de
`renderer.go`) já estavam certos desde a correção de 31/08 — o problema era só
o estilo: dois `DrawCircleLines` de 1 px contra o corpo dela (610 px) leem
como um traço na pele, não como aviso de área. Trocado por um halo — uma
pilha de `DrawCircle` concêntricos com alfa caindo para fora, ainda no raio
exato de `BossContactRadius` e ainda pulsando por `rl.GetTime()`. Custo:
6 `DrawCircle` por quadro em vez de 3 chamadas antes — ainda um punhado, não
dezenas (`doc/skill_performance.md`, Regra 5).

**Sangramento de atlas em tile layers (`tilemap.MapRenderer.drawTileLayer`).**
O retângulo de origem usava o tamanho de tile do MAPA (`tileW`/`tileH`) para
recortar do atlas, mas o offset dentro do atlas era calculado com o tamanho
de tile do TILESET (`ts.TileWidth`/`ts.TileHeight`). Quando os dois batem
(caso comum) não fazia diferença; quando não batem, a amostra invadia a
peça vizinha — um fio de 1 px vazando pela borda. Corrigido para usar as
dimensões do tileset também no tamanho do retângulo de origem. Ver
`doc/tilemap.md`, "O retângulo de origem usa o tamanho de tile do TILESET,
não do mapa".

Não confirmado ainda contra o jogo rodando (sem Go neste ambiente — ver
`build_verification.md`): o fio branco tinha três suspeitos e só o do atlas
foi corrigido por ser um bug de correção geral, não específico deste caso;
os outros dois (moldura de `ui.DrawHealthBar` e um fio assado num PNG de
tileset) seguem descartáveis por teste, não por código. Gui: rode
`gofmt -l internal`, `go build ./...`,
`go test ./internal/skill ./internal/ui ./internal/tilemap ./internal/network`,
e recapture as duas cenas — se o fio branco persistir, siga o procedimento a/b/c/d
do diagnóstico original (mover a câmera, comentar a barra de vida) para achar
qual dos outros dois suspeitos é.

## 2026-09-04 — Lentidão nos mapas 2 e 3: o terreno desenhava cinco camadas para mostrar uma

Relato do Gui: mapa 1 fluido; ao entrar no 2 o jogo fica lento e a música
começa a travar; no 3 continua; do 4 em diante a lentidão cai bastante mas a
música segue meio travada — melhor que no 2 e 3, pior que no 1. Já se havia
removido a fita de trilha desses dois mapas (31/08), e ainda assim sobrava
alguma coisa.

**A causa é `overdraw`, e ela nunca tinha sido medida.** `doc/performance.md`
inteiro contava *quads* e *draw calls*; R1 (culling) e R4 (batch por material)
cortaram os dois. O que ninguém contou foi **fragmento**. O terreno é desenhado
empilhado — a célula recebe a grama base mais uma camada por material abaixo do
próprio — e o quad de cada camada cobre a **célula inteira**. Então o custo por
célula é a altura do material dela na pilha:

| Mapa | Quads por célula | Quads/quadro a 1080p |
|---|---|---|
| `world_01` | 1,23 | 258 |
| `world_02` | 4,16 | 804 |
| `world_03` | **5,40** | **1.049** |
| `world_04` | 3,15 | 641 |
| `world_05`–`07` | 2,3–2,6 | 471–520 |

A coluna **é** a curva relatada, fase a fase. A laje negra do `world_03` são
oito quads de tela cheia empilhados na mesma célula.

Três hipóteses foram medidas e descartadas antes: **VRAM** (os mapas 4–7 usam
71–83 MB contra 27–59 MB do 2 e 3, e vão melhor), **área de props por área de
mapa** (0,23 / 0,27 / 0,34 / 0,12 nos mapas 1–4, não explica a curva) e
**número de spawns** (o `world_04` tem 22 postos contra 20 do `world_02`).

1. **Oclusão de camada** (`internal/tilemap/terrain_renderer.go`,
   `terrain_mask.go`). Quando os oito vizinhos de uma célula carregam o mesmo
   material, ela não tem borda para desbotar: `side()` devolve 1 nos quatro
   lados, nenhum termo de canto entra e o dither zera junto com a rampa. A
   máscara sai **1,0 exata**, as texturas de material são RGB (sem alfa), e o
   quad fica **opaco** — tudo sob ele é invisível. `fullyLinked` é o teste,
   `prepareCoverage` o roda uma vez por mapa dentro do `PrepareMasks`, `hidden`
   faz cada passe perguntar antes de desenhar. **Célula cercada desenha um quad
   só.** Entre 80% e 94% das células dos sete mapas estão nessa condição.

   A cobertura é calculada **antes** do teste de batch, de propósito: vale para
   os dois caminhos, e um aparelho que caísse no caminho por célula perderia
   justamente a otimização de que mais precisa.

   Resultado: 245–303 quads por quadro em **todos** os mapas — `world_03` 3,56×,
   `world_02` 2,80×, `world_04` 2,30×. O custo de terreno deixa de depender da
   fase.

2. **Atalho da célula cercada nos quatro shaders**
   (`assets/shaders/terrain_batch_100/330.fs`, `terrain_blend_100/330.fs`). O
   mesmo caso, pelo outro lado: com os oito vizinhos ligados, sai direto na
   amostra da textura, pulando oito `smoothstep`, quatro `length()` e o `sin()`
   do ruído. O item 1 corta as camadas **escondidas**; este barateia as
   **visíveis**, que depois dele são ~90% do que sobra.

3. **`terrain_blend_*.fs` convergiu com `terrain_batch_*.fs` no dither.**
   Faltavam duas coisas no caminho por célula: o ruído não era ancorado no
   texel (fervia com a câmera) e não era pesado pela rampa — ele furava o
   **interior** da célula com até 8% de alfa, deixando a camada de baixo
   aparecer onde nada deveria atravessar. Além de ser o defeito que era, isso
   teria feito a oclusão parecer erro de desenho nesse caminho.

4. **Sub-buffer de áudio** (`internal/sound/music.go`). A música travar era o
   mesmo defeito ouvido: `sound.Update` alimenta o stream uma vez por quadro e
   o padrão do raylib é `sampleRate/30` ≈ 33 ms, então um quadro mais longo que
   isso já produz falha audível — o que explica os três regimes de uma vez.
   `musicBufferFrames = 4096` (~93 ms) via `rl.SetAudioStreamBufferSizeDefault`
   antes de `InitAudioDevice`. Não conserta o quadro longo (item 1 conserta);
   remove o acoplamento entre fps e áudio.

5. **`QuadsOccluded` no painel do F3** (`tilemap/stats.go`, `game/perf_hud.go`).
   A oclusão não muda pixel nem draw call, então sem um contador ela não teria
   onde aparecer se parasse de funcionar. A linha virou
   `terreno N quads / M ocultos / K binds de shader`.

Conferido por `work/perf/verify_terrain_occlusion.py`, que replica a fórmula do
shader em Python e prova que ela devolve 1,0 exato para toda `edgeWidth`
declarada, recalcula a verdade a partir do JSON de cada mapa e prova que nenhum
quad pulado estava sob célula não cercada, e mede o quadro com o culling já
aplicado. Travado do lado do Go por `terrain_occlusion_test.go`. Os três
verificadores anteriores (`verify_go_static`, `verify_terrain_batch`,
`verify_viewport`) continuam passando. `doc/performance.md` (R8 e seção 1.5),
`doc/tilemap.md` ("Só se desenha a CAMADA que aparece") e `doc/audio.md`
atualizados. Falta `go build` / `go test` e a sessão em jogo no aparelho.

## 2026-09-02 — Indicador de mira: reta certa e angulo certo

Print do Gui: a bola de fogo saindo por uma reta e o chevron desenhado em
outra. Medido no proprio print (1080x2340, escala de tela 2,07 deduzida pela
asa do chevron): a bola voava a 181,5 graus e o chevron apontava 180,9 — o
**angulo batia**. O que nao batia era a **reta**, com um vao de 101 px de mundo
entre as duas, contra os 103,5 do `GroundOffset` do mago.

**Causa 1 — ancora.** O chevron nascia em `entity.GroundPoint` (o pe), porque
marca no chao le melhor que marca flutuando na altura do peito. Le mesmo, e
esta errado: `ProcessInput` mira em `p.Position + dir*100` e o host nasce o
projetil em `Position`. A marca era uma reta PARALELA um corpo abaixo da magia.
**Correcao:** a ancora passou a ser `p.Position`.

**Causa 2 — o achatamento.** A ponta caia sobre uma ELIPSE
(`aimChevronSquash = 0,55`) para parecer deitada no chao. Achatamento nao so
move a marca, ele a GIRA: 45 graus de mira saiam desenhados como 28,8, e toda
direcao que nao fosse exatamente horizontal ou vertical errava em ate 16 graus.
O unico numero que a coisa existe para comunicar era o unico que ela tinha
licenca para distorcer. **Correcao:** `aimChevronSquash` deixou de existir; a
ponta cai em `Position + Dir * distancia` e o chevron aponta exatamente `Dir`.

`aimChevronDistance` subiu de 96 para **120**, porque a distancia agora se mede
do peito e nao do pe: a figura chega a ~99 px acima de `Position` (topo da
cabeca) e ~103 abaixo (as solas), entao raio menor poria uma mira ao norte ou
ao sul dentro do personagem. De quebra isso resolveu o ponto fraco anotado no
dia anterior — a marca ao norte nao cai mais em cima da veste.

## 2026-09-02 — Cluster de botoes, um gesto por vez, e icones no lugar de letras

Tres pedidos do Gui no mesmo dia, todos sobre o mesmo canto da tela.

**1. Lancamento acidental enquanto mira.** Mirar no celular e um arrasto
SEGURADO, e o arrasto passa por cima dos outros botoes — eles ficam a um
polegar de distancia, que e o proposito de um agrupamento. Cada um se armava na
passagem e disparava quando o dedo subia: o jogador pedia um angulo e recebia
uma suprema.
**Correcao:** `input.ActionLock` (`internal/input/action_lock.go`). O primeiro
dos tres botoes a receber um toque fica com o gesto; os outros dois param de
responder e se desenham apagados ate ele soltar. Um dono so, e nao uma bandeira
por botao, porque a regra e sobre o GESTO. O joystick de movimento fica de
fora de proposito: andar enquanto mira e lanca e o jogo funcionando como deve,
nas duas plataformas. `game.cancelAndroidAim` libera o gesto em todo caminho
que para de atualizar os botoes (fim de run, corpo em portal, desktop, suprema
que deixou de estar disponivel no meio da mira) — sem isso o cluster ficaria
morto pelo resto da partida, porque a soltura e detectada varrendo toques.

**2. Distribuicao dos botoes.** Era uma coluna vertical de tres circulos do
mesmo tamanho: alcancavel, mas sem dizer qual e a acao principal, e com o
polegar tendo de percorrer a pilha inteira para chegar na suprema.
**Correcao:** o ataque basico virou o circulo grande no centro
(`attackRadiusFrac` 0,06 → **0,085**) e as duas habilidades orbitam ele
(`skillRadiusRatio` 0,62, `orbitGapRatio` 0,30, Q a 200 graus e R a 252). O
raio de orbita e ataque + habilidade + folga, entao um circulo de habilidade
nunca cruza o do ataque em tela nenhuma. O ataque passou a ser testado como
CIRCULO e nao mais como retangulo: a quina de um quadrado alcanca 1,41 raios,
contra 1,30 da borda interna das habilidades. Ver `doc/controls.md`.

**3. Icones.** Os botoes carregavam "FIRE", "Q" e "R" — letra de teclado em
aparelho sem teclado, e uma palavra em ingles. Agora sao glifos (espada,
estrela, estrela com anel) em `assets/ui/icon_*.png`, carregados por
`internal/ui/action_icons.go`. Sao PNGs BRANCOS com alfa: um arquivo serve ao
estado aceso, apagado e bloqueado por tingimento. Como eles trazem o proprio
disco translucido, o preenchimento colorido do botao ficou fraco
(`actionFillAlpha`) e a pista de cor foi para o ANEL — preenchimento forte
embaixo do icone transforma o botao num borrao e o glifo para de ler.

**Cor: os tres botoes usam UMA cor.** No primeiro corte cada slot tinha a sua
(vermelho, laranja, dourado). Numa tela que ja carrega barra de vida, numero de
dano, luz de magia e contador de onda, tres circulos saturados no canto leem
como tres ALERTAS — e a pista era redundante, porque os glifos ja diferem e o
botao do meio e uma vez e meia maior que os outros dois. Ficou
`ui.ActionNeutral`, o quase-branco do proprio icone; o que separa os botoes e
tamanho, posicao e desenho. `actionRingAlpha` subiu de 0,85 para 0,95 no
caminho: anel branco tem menos contraste contra grama clara do que anel
dourado. O chevron de chao perdeu o tingimento por botao pela mesma razao —
`AimHint.Accent` e o `fadeTowards` do `AimIndicator` deixaram de existir.

Os tres passaram a ser desenhados por uma funcao so, `ui.DrawActionButton`
(`internal/ui/action_button.go`). Antes eram dois pacotes e duas funcoes
(`input.DrawAimJoystick` e `SkillButton.Draw`), e ja tinham divergido:
preenchimentos diferentes, aneis diferentes, um com botao no dedo e outro sem.
`DrawAimJoystick` deixou de existir nas duas plataformas.

O botao de habilidade tambem parou de PULAR para a origem do toque enquanto
mira. Numa coluna aquilo era inofensivo; num agrupamento, um botao que salta
para debaixo do dedo cai em cima dos vizinhos. A direcao fica com a linha de
arrasto e com o chevron de chao (`ui.AimIndicator`).

## 2026-09-02 — Indicador de mira no chão (Android)

**Sintoma:** no celular, a direção do próximo tiro é invisível. Ela vive em
`AimJoystick.AimDir` e em `SkillButton.SkillDir`, e só era desenhada **ao lado
do botão**, no canto inferior direito — enquanto o jogador olha para o
personagem, do outro lado da tela. Um toque simples dispara numa direção
configurada há vários segundos e nada no personagem diz qual.

**Correção:** `ui.AimIndicator` (`internal/ui/aim_indicator.go`), um chevron
discreto desenhado em espaço de mundo na frente do personagem.

- **Ancorado no pé** (`entity.GroundPoint`), não no meio do sprite, e colocado
  sobre uma **elipse** (`aimChevronSquash = 0.55`) em vez de um círculo: os
  sprites são de perfil num mapa de cima, então uma marca que subisse 96 px
  inteiros ao norte flutuaria na altura do peito em vez de ficar no chão.
- **Quieto por padrão:** alfa `aimRestAlpha` em repouso, subindo para
  `aimActiveAlpha` enquanto um dedo está num dos três botões de ação, com
  a transição suavizada (`aimFadeRate`) para não piscar a cada toque.
  `aimRestAlpha` nasceu 0,16 e sumia na grama — só aparecia com zoom de 3x
  no preview; ficou em **0,24**, o menor valor que ainda se lê a 1:1.
- **Cor por botão:** branco em repouso, vermelho claro no ataque básico,
  laranja no Q, dourado no R — as mesmas cores dos botões.
- **Direção perseguida, não atribuída** (`aimTurnRate`): os três botões guardam
  três direções independentes, então soltar o Q devolve a marca à mira de
  ataque. Atribuída, ela **teleportaria** para o outro lado do personagem.

`game/aim_hint.go` decide qual das três direções mostrar (prioridade R → Q →
ataque; em repouso, a de ataque, que é o que um toque dispararia) e desenha em
`renderer.go` **depois dos corpos e antes das barras**: antes dos corpos, o
próprio sprite cobriria a marca justo quando a mira aponta para o norte.

Só no Android (`cfg.FullScreen`): no desktop a mira é o mouse e o cursor já diz
para onde ela vai. Nada foi enviado pela rede — é desenho local do jogador
local, e nenhum outro jogador vê a marca.

## 2026-08-31 — Flecha Celestial derruba o Orc; Chuva de Meteoros escala por mapa

Dois ajustes de suprema pedidos pelo Gui em 31/08/2026, com a mesma forma de
defeito: um número escrito contra um elenco ou um mapa que mudou depois, e
ficou para trás. Ver os comentários em `internal/skill/celestial.go` e
`internal/skill/meteor.go`.

**Sintoma (Flecha Celestial):** `CelestialDamage = 40` era dano da época do
slime de 100 de vida; contra o orc de guarnição (600 de vida) eram 15 flechas
por orc — sete ativações e três minutos e meio para derrubar um único inimigo
da guarnição.
**Correção:** `CelestialDamage` 40 → **600**, igual à vida do orc — uma
flecha derruba um orc, sem sobra. Teste novo:
`TestOneCelestialArrowFellsAnOrc` (`network/balance_test.go`), no mesmo
formato de `TestOneMeteorTakesFourFifthsOfAnOrc`.

**Sintoma (Chuva de Meteoros):** `MeteorRainInterval` era fixo em todo mapa
(0,015 s), mas o sorteio é sobre o mapa inteiro — o mesmo número de pedras
ficava espalhado por áreas que variam onze vezes. O achado que mudou a
leitura: pela tabela de progressão, o Mago só tem a suprema em campanha a
partir do `world_06`, e junto com o `world_07` são os dois MENORES mapas do
jogo — o padrão estava calibrado contra o `world_05`, onde a suprema nunca é
lançada em campanha, e rodava a 5×/11× essa densidade nos dois mapas onde ela
de fato existe.
**Correção:** `MeteorRainInterval` → `MeteorRainRefInterval` (referência,
calibrada no `world_05`, valor inalterado) mais `MeteorRainIntervalFor`
(`internal/skill/meteor.go`), que reescala o intervalo pela área do mapa para
manter constante a DENSIDADE (nascimentos por segundo por área) — que é o que
a chuva de fato controla, já que ela não mira, sorteia o mapa inteiro.
`MeteorRainMinInterval` (mesmo 0,015) vira o teto de desempenho explícito;
nenhum mapa pode furá-lo. `world_05` fica bit a bit idêntico; `world_06` e
`world_07` passam de ~1000 nascimentos por chuva completa (herdado do
`world_05`) para ~194 e ~90, e de ~16/~35 meteoros simultâneos na tela para
~3, igualando-se ao resto da campanha.
Testes novos: `internal/skill/meteor_scale_test.go` (cinco testes, função
pura) e `TestNoCampaignMapBreaksTheMeteorRainPerformanceCeiling`
(`internal/game/meteor_rain_ceiling_test.go`), que lê os mapas reais.

Docs atualizados: `doc/combat_rules.md` (contrato da flecha e a nova seção
"a chuva mede-se por densidade, não por contagem"), `doc/performance.md` §8
(nota de que `MeteorRainInterval` deixou de ser puramente decisão de
design) e `doc/skill_performance.md` (o regime de uma magia de área-inteira
tem de ser contado por mapa, não só no mapa de calibração).

**Pendente de veredito em jogo:** o corte de densidade no `world_06`/`world_07`
não tem teste automatizado — precisa ser jogado. Se "ficou fraco", o botão é
`MeteorRainRefInterval` (move todo mapa junto, nunca um caso especial por
fase).

## 2026-08-31 — Buff na Sacerdotisa: o raio sagrado ficou largo (13 → 26 → 20)

Pedido do Gui: "aumentar a área do ataque básico para ficar mais fácil de curar
os jogadores".

O ataque básico dela cura quem o tiro atravessa, e o teste é `raio do projétil +
PlayerSize`. Com `HolyAttackRadius = 13` a folga era de 33 px: curar exigia mira
de tiro certeiro num alvo em movimento, enquanto a horda chega — acidente feliz,
não uma coisa que a curandeira fazia.

Foi a 26 primeiro, e o playtest devolveu o outro limite: **o mesmo número que
resolve a cura desenha o projétil**, então dobrar o raio dobrou a bola de luz e
um tiro que sai várias vezes por segundo virou um farol na tela. Ficou em **20**,
com folga de **40 px** — e esse não é um meio termo arbitrário: 40 é exatamente
dois raios de corpo (`PlayerSize` 20), a promessa em uma frase ("perdoa um corpo
inteiro de erro de mira"). O teste cobra a relação, não o literal.

Buff no raio do PROJÉTIL, não num raio de cura separado (escolha do Gui): o
desenho sai do mesmo número, então o que a tela mostra continua sendo o que
acerta — e é essa mesma amarra que impede o número de crescer indefinidamente.
Custo aceito: o tiro também acerta inimigos mais fácil; ela segue com o menor
dano por tiro do elenco (18).

`drawHolyProjectile` foi reescrito em FRAÇÕES do raio: a versão anterior tirava
4 px fixos do núcleo, e com o raio maior isso viraria uma bola branca chapada.
Agora o núcleo sólido é 0,42x e o brilho aditivo cresce em cima dele — a regra da
paleta dela (o branco na menor área) aplicada ao projétil. Foi o que permitiu
crescer o raio de acerto sem o tiro pesar tanto na tela.

`internal/entity/projectile_holy_test.go` trava os dois lados da faixa: perdoar
pelo menos um corpo inteiro de erro de mira, e nunca virar aura — senão o ataque
básico faz o trabalho do Santuário sem recarga.

## 2026-08-31 — Skill nova: `preview-skill-effects`

Pedido do Gui depois do preview em canvas que usamos para afinar a pulsação do
Santuário: virar aquilo em método, para outra LLM conseguir refazer — e para
servir de base a uma futura bancada de visualização de efeitos.

O problema que a skill ataca é que **o agente que escreve o efeito não consegue
vê-lo**. Ele escreve `DrawCircleGradient` no escuro e espera o print do
usuário. Foram duas rodadas às cegas no Santuário no mesmo dia (miolo rosa; e
depois um contorno que parecia o Escudo Sagrado) — as duas teriam morrido em
dez segundos num preview.

A skill traz a tabela de porte raylib → canvas (com as armadilhas: graus x
radianos, gradiente que termina em alfa 0 e não em preto, `'lighter'` como
aproximação do `BlendAdditive`), o esqueleto do harness (constantes copiadas
verbatim do Go, ciclo de vida em laço, fundo igual ao do mapa onde a magia é
julgada, conjurador que anda, sliders só nos números em discussão) e — a parte
que mais importa — a **linha entre o que o preview prova e o que ele não
prova**: cor, silhueta, ritmo, proporção e parentesco visual com outras classes,
sim; fps, fill rate, culling e raio de jogabilidade, nunca. O juiz de custo
continua sendo o F3 do `doc/skill_performance.md`.

Fecha com o caminho para uma IDE de efeitos, se isso crescer: registro de
magias no mesmo formato do `ability.Registry`, scrubber de tempo (julgar um
estilhaço de 0,45 s exige parar dentro dele), fundos por bioma, modo lado a
lado para flagrar gramática repetida, e leitura das constantes direto do
arquivo Go em vez de cópia à mão — constante copiada envelhece, e bancada
desatualizada é pior que bancada nenhuma.

O preview do Santuário ficou em `skills/preview-skill-effects/references/santuario.html`
como exemplo pronto para servir de molde.


## 2026-08-31 — O Santuário virou aura móvel (e dourada)

Pedido do Gui, em duas partes que acabaram sendo a mesma — mais uma terceira,
depois do primeiro print.

**1. A cor.** A paleta declarada da Sacerdotisa sempre foi dourado → amarelo
divino → branco, mas o Santuário desenhava discos de alfa baixo em blending
NORMAL: amarelo fraco pintado por cima da grama não vira luz, vira tinta, e o
miolo aparecia rosado na tela. Agora o efeito inteiro é aditivo
(`sanctuary_draw.go`), que soma em vez de cobrir — e aditivo não consegue puxar
para o rosa, porque dourado não tem azul para somar. Junto vieram as regras do
`SKILL.md`: gradientes no lugar de discos de borda dura e a cor mais quente na
MENOR área (o branco só no núcleo e nas partículas).

**2. A forma.** O Santuário era um lugar: a Sacerdotisa o largava 120 px à
frente e quem quisesse a cura ia até o círculo. Isso punha a classe curandeira
em conflito com a própria magia — para continuar curando, ela tinha de ficar
parada num ponto que a luta já havia deixado para trás. Agora a aura é ancorada
NELA e reancorada todo quadro (`Host.anchorSanctuaries`, antes da cura, pelo
mesmo motivo de `tickAvatars`), e `SanctuarySpeedBonus` dá 25% de velocidade à
conjuradora enquanto durar. Aliados recebem a cura, não a mobilidade: o bônus é
o que transforma "a magia me segue" em "eu levo a magia", e valendo para o time
inteiro viraria ferramenta de deslocamento do grupo.

**3. A pulsação, e por que o contorno saiu (mesmo dia, depois do print).** A
primeira versão do visual ficou parecida demais com a Paladina: borda firme,
anel pulsante por fora, arcos contra-rotativos e motes em órbita achatada são,
item por item, o vocabulário do Escudo Sagrado e do Avatar. Duas classes com a
mesma paleta dourada não podem ter também a mesma **gramática** — de relance, o
Santuário virava "o escudo da Paladina, só que maior". O contorno saiu inteiro.
No lugar dele há ondas que **nascem na Sacerdotisa** e morrem exatamente no
alcance da cura: a borda estática dizia onde a magia acaba, a onda diz de onde
ela **vem**, e o limite aparece como consequência uma vez por batida em vez de
ficar riscado no chão. Cada onda são quatro anéis concêntricos (espessura
crescente, alfa decrescente) para ler como luz difusa e não como linha, com a
crista clareando para o branco perto dela e caindo para o dourado ao abrir. As
motes acendem quando a onda passa por cima delas (`waveBoost`) — sem isso, a
pulsação passaria *por cima* do santuário em vez de atravessá-lo. Regime: uma
onda a cada 0,9 s vivendo 1,4 s, no máximo duas no ar.

A coleção passou a ser indexada por DONO (como escudo e avatar): uma
Sacerdotisa tem uma aura, e relançar renova em vez de empilhar — no host e no
cliente. O bônus de velocidade não ganhou mensagem de rede: a aura já é
replicada no lançamento e envelhece com o mesmo relógio nos dois lados, então
cada máquina deriva o multiplicador do estado que já tem (Regra 6 do
`doc/skill_performance.md`). O movimento é previsto na máquina do jogador, e é
lá que o bônus é aplicado (`game.ApplyMoveBuff`, sempre recalculado a partir de
`entity.PlayerSpeed` para o fator não se acumular a 60 Hz); a bot Sacerdotisa
ganha o mesmo em `applyBotIntent`. O `Draw` também ganhou o culling que faltava
(Regra 2), com as motes se cullando uma a uma.


## 2026-08-31 — Correção pós-playtest: anel de contato e barra da chefe

Relato do Gui depois de testar a afinação do item anterior: não estava claro
quando o anel de contato começava a causar dano, uma "esfera" não identificada
aparecia perto da chefe na tela, e a barra de vida dela estava afastada demais
da borda esquerda.

A "esfera" era o próprio anel de contato (`skill.DrawBossContactRing`), e o
motivo de não parecer ligado à chefe é que ele nascia no lugar errado: centrado
em `Enemy.Position()`, que é o PÉ dela (por causa do `FootLine` da folha de
animação), enquanto o corpo visível fica 180 px ACIMA do pé (`HitOffsetY`).
Corrigido para `Enemy.HitCenter()` — a mesma correção de altura que a caixa de
acerto de espada já usa (`internal/entity/enemy.go`) — em `bosses.go`
(`Host.BossContactCenter()`, renomeado de `BossPosition()`),
`host_boss_contact.go` e `renderer.go` (o lado cliente aplica o mesmo
`HitOffsetY` manualmente sobre `X`/`Y`, já que só recebe a posição crua).

De caminho, `BossContactRadius` deixou de ser um número solto (150) e passou a
ler o `Radius` físico da própria chefe do `EnemyDef` (130) — o "corpo de
aranha" que `enemy_sprite.go` já documentava como deliberadamente menor que a
envergadura das patas — para que o dano comece de fato perto de onde a sprite
está, não antes.

A barra de vida (`ui/boss_bar.go`, `bossBarLeftFrac`) mudou de 0.32 para 0.24,
aproximando-a da borda esquerda com uma folga de 4 pontos percentuais acima de
onde `DrawHealthBar` termina (~20% da largura da tela).

## 2026-08-31 — Afinação da arena da Senhora das Trevas (seis ajustes)

Plano: `doc/plan_world07_arena_tuning.md`. Primeiro playtest mostrou a arena
fácil demais e a Legião Espectral derretendo a chefe sozinha; seis ajustes na
mesma direção, em três fases.

1. Espinhão a cada 10 s (era 15), aviso de 1,2 s (era 2,1) —
   `ThornTelegraph`/`ThornRise` (`skill/boss_thorn.go`) e `bossDodgeWindow`
   (`entity/enemy_boss_anim.go`), acoplados e travados por
   `TestBossDodgeWindowMatchesThornTelegraph`.
2. Gárgulas dobradas: `bossSentryStep` 2 → 4 (`network/host_boss.go`). Os 10
   postos do mapa se esgotam aos 140 s (decisão aceita, ver §10 do plano).
3. Névoa a cada 30 s (era 60) — `bossFogEvery`. Uma névoa em cada duas fica
   sem resposta de suprema, de propósito.
4. Vida da chefe 30000 → 45000 (`entity/enemy_sprite.go`).
5. **Anel de contato** (novo): a chefe agora cobra dano contínuo
   (`BossContactDPS` 18, raio 150) de quem encostar no corpo — jogador ou
   espectro da Legião Espectral, pelo mesmo caminho
   (`skill/boss_contact.go`, `network/host_boss_contact.go`,
   `skill.DamageSpectersInRadius`). Sem isso a Legião tirava sozinha quase
   metade da vida antiga numa ativação; agora fica perto de 10%. Consequência
   aceita: a Paladina não consegue golpear vindo de baixo do corpo dela.
6. A horda do cerco virou `BossDriven` (`network/host_waves.go`,
   `WaveDef.BossDriven`, `WaveRunner.ReleaseBatchNow`): perde o relógio
   próprio de 70 s e passa a sair no mesmo quadro em que a névoa entra no ar.
   `MaxConcurrent` subiu de 34 para 50 para as duas levas caberem sem virar
   parede.

Testes novos: `TestTheArenaBossOutlastsTheSpectralLegion`,
`TestBossDodgeWindowMatchesThornTelegraph`,
`TestBossDrivenWaveOnlyReleasesOnRequest`,
`TestWorld07FirstWaveArrivesWithoutTheBoss`,
`TestWorld07SentryPostsMatchTheStepCadence`. Docs atualizados:
`doc/combat_rules.md`, `doc/plan_world07_arena.md`, `doc/skill_performance.md`.


## 2026-08-31 — Fim de campanha: fogos, dialogo final e "Missao Concluida" na queda da chefe

Pedido do Gui: a queda da Senhora das Trevas (mapa 7, unico chefe do jogo)
precisava fechar a campanha em vez de so abrir o portal como qualquer outra
fase.

Tres pecas, todas lendo o mesmo `BossState` ja replicado pela barra de vida do
chefe (`internal/network/boss_state.go`), entao funcionam identicas no host e
no cliente sem mensagem de rede nova:

1. **Fogos de artificio** (`internal/ui/victory_fireworks.go`): emissor de
   tela propria (foguetes sobem, estouram em faiscas aditivas), ligado
   enquanto `boss.Present && boss.Health <= 0` for verdade. Comecam no
   instante em que a chefe cai, mesmo com o campo ainda sendo limpo — a festa
   nao espera o resto morrer.
2. **Dialogo final** (`assets/dialogues/world_07.json`, gatilho
   `on_waves_cleared` que ja existia no diretor de dialogo): uma cena com os
   cinco personagens quando a horda vira `cleared`.
3. **"MISSAO CONCLUIDA"** (`internal/ui/victory.go`): `drawMapCleared`
   (wave_hud.go) passa a checar `network.GetBossState().Present` e desenha o
   banner dourado pulsante no lugar do "Mapa limpo" generico quando o mapa que
   fechou tem chefe.

Nada disto sabe o nome do mapa — "tem chefe" ja significa "e o fim da
campanha", porque a Senhora das Trevas e o unico chefe do jogo.


## 2026-08-31 — Correção: o caminho assado nos mapas 2 e 3 estava claro demais

Relato do Gui na sequência do item 1 do dia (a trilha virou material pintado no
chão): o caminho ficou claro demais, perdendo a essência da floresta sombria.

Causa medida: `terrainForestPath` (material 8) desenhava com `terrain_dirt.png`
— a MESMA textura da pilha VERDE (vila), luminância média 151. Todo o resto da
pilha escura fica entre 55 e 102 (terra nua 58, grama densa 68, cascalho da
esplanada 67, laje 55) — mais que o dobro do brilho de qualquer vizinho.

Também não era material `span` como o resto da pilha escura (repetia a cada 128
px em vez de 512 px, perdendo ~60% do contraste local — o mesmo defeito que a
seção "spanMaterials" de `internal/tilemap/terrain_renderer.go` já descreve
para os outros materiais da pilha escura).

Conserto: `terrain_dirt.png` regrado por canal (mesma técnica de
`work/tiled-assets/match_biome_tint.py`) para a luminância dos vizinhos (151 →
~63) e ladrilhado a 512 px — vira `assets/tilesets/terrain_forest_path.png`,
com `_source.png` de auditoria. Não é arte nova: mostrei o candidato ao Gui
(faixa de comparação contra os vizinhos da pilha) antes de instalar, e ele
preferiu essa correção rápida a esperar por arte pintada sob medida.
`terrainForestPath` entrou em `spanMaterials`
(`internal/tilemap/terrain_renderer.go`).

## 2026-08-31 — Sete defeitos relatados em jogo: mobile, rede e fase 6/7

Sete relatos do Gui em 30/08/2026, todos com causa lida no código antes de
mexer (`doc/plan_correcoes_mobile_e_rede.md`; diagnóstico completo em memória
de projeto, `defeitos-30-08-2026.md`). Implementados na ordem sugerida pelo
plano.

1. **smoothstep degenerado apagava borda de célula no mobile.** A pilha do
   castelo declara `edgeWidth: 0.01` e os quatro shaders faziam
   `smoothstep(0.01, edgeWidth, uv)` com `edge0 == edge1` — indefinido pela
   spec GLSL. Desktop (`highp`) dá ±inf e o clamp salva; Android ES2
   (`mediump`) dá NaN e vira alpha 0 em muitos drivers, sumindo a beirada do
   tapete. `rampLo`/`rampHi` (e os equivalentes de canto) derivam limites
   seguros de `edgeWidth` nos quatro shaders (`assets/shaders/terrain_*.fs`);
   o dither dos dois shaders `_100` (mediump) trocou de constante para ficar
   dentro da faixa que não degenera. Travado por
   `TestStackEdgeWidthNeverDegeneratesTheShaderRamp`
   (`internal/tilemap/terrain_mask_test.go`).
2. **Cliente mobile preso na tela de Game Over, ou no mapa errado depois de
   reconectar.** `GameOver` e a geração da fase só viajavam como EVENTO
   (`MsgGameOver`/`MsgReset`); a tela bloqueando dispara `dropConn`, que
   descarta o buffer TCP — inclusive um `MsgReset` já entregue — e nada
   reconciliava depois. `StateUpdatePayload` ganhou `StageGen`/`GameOver`
   (`network/protocol.go`); o cliente reconcilia a cada snapshot em
   `client_stage_reconcile.go` (novo): geração avançada sem evento visto vira
   reset recuperado pela posição que o host já reportou para o jogador NESTE
   snapshot, e um `GameOver` divergente é adotado do host. `lastSeenStageGen`
   não é resetado num reconnect — só numa sessão de fato nova — para o
   primeiro snapshot pós-reconexão revelar o que foi perdido.
3. **Jogador novo entrava com o mapa do menu, não o do grupo.**
   `sendCurrentMapTo` só era chamado no ramo `isReconnect`; um join novo nunca
   recebia `MsgTravel` e ficava com os assets do `world_01` desenhados sobre
   coordenadas de outro mapa. Agora é chamado sem a condição, com um `bool`
   `isReconnect` para preservar a semântica do campo `Reconnect` do payload
   (`network/host_rejoin.go`).
4. **Caixa de acerto do canhão do mapa 6.** `damageCannonsInSweep` testava um
   PONTO (`y-70`, raio 80) contra o arco de espada em vez da caixa real da
   peça (184×285). `cannonPost.hitBox()` substitui o ponto único;
   `skill.NearestPointInRect` acha o ponto da caixa mais perto do jogador, o
   que conserta distância e ângulo de uma vez (`network/cannons.go`). De
   brinde: `bot.CannonHuntRange` (3200→4000) estava desatualizado desde que
   `CannonRange` subiu para 4000, quebrando um teste em silêncio.
5. **Portão de saída do mapa 6 quase impossível de atravessar.**
   `gates_gate_04` tinha `collisionFootprint` caindo DENTRO do retângulo do
   portal, e o único mecanismo que abre portão (`arenaGate`) exige zona
   `arena_lock` + `WaveState.Total > 0` — o mapa 6 não tem nenhum dos dois.
   `portal_exit_gate.go` (novo) desliga footprints sobrepondo o retângulo do
   portal quando `PortalsUnlocked()`, religa por `StageGeneration`. Achado de
   brinde, fora de escopo: `world_05` tem parede PINTADA (não footprint)
   tocando a borda do portal `portal_saida_do_salao` — vale o Gui olhar.
6. **A chefe do mapa 7 só existia no host.** Névoa e espinhões nunca eram
   replicados; um cliente via a chefe em idle sem aviso nenhum, o dano
   chegava só pelos eventos de combate. `MsgBossFog`/`MsgBossThorn`
   (`network/protocol.go`), irmãos de `MsgSentryOrb`/`MsgCannonBall` — magia
   de MONSTRO não viaja em `MsgSkill`. Host emite em `host_boss.go`; cliente
   trata em `client.go` e envelhece os efeitos por quadro
   (`AdvanceClientSkills`, sem o que a marca do espinho ficava congelada e a
   névoa nunca desenhava). `DrawThorns`/`DrawFog` saíram de dentro do ramo
   `if Role == "host"` em `renderer.go`. Barra de vida da chefe e aviso
   "PERIGO" no cliente ficaram de fora — bônus visual do plano, sem defeito
   associado.
7. **A trilha dos mapas 2 e 3 era fita de geometria cara no mobile.**
   `trail_renderer.go` desenhava uma faixa translúcida de ~384 px atravessando
   o mapa em modo imediato a cada quadro — o item mais caro do desenho numa
   GPU de blocos. Removido: a curva das duas trilhas (`trilha_mata`,
   `trilha_fortaleza`) foi reamostrada e suavizada OFFLINE (o mesmo algoritmo
   que rodava em tempo de execução) e assada direto na camada `ground` de
   `world_02.json`/`world_03.json` como material 8 (`terrainForestPath`) —
   textura que já estava na VRAM dos dois mapas, custo novo zero. No
   `world_03` a trilha passou a encostar na esplanada (materiais 9/10) em vez
   de parar 6 células antes dela, sem nunca sobrescrever pedra já pintada —
   decisão do Gui. `internal/tilemap/trail.go`, `trail_renderer.go`,
   `assets/tilesets/terrain_trail*.png`, o campo `Trail`/`trails` de
   `MapRenderer` e `Stats.TrailQuads` foram removidos. `doc/tilemap.md`
   ("Trilhas") e `doc/performance.md` atualizados.

## 2026-08-27 — Piscar de sobreposicao, gargulas do 4 acordadas cedo, climax do 3 travando e afinacao de dano

Seis relatos de jogo do Gui, na ordem em que ele os listou.

1. **Climax do mapa 3 nao abria.** O grupo de cinco vencia a emboscada
   "A emboscada do portao" sem cair abaixo de 30% ao mesmo tempo, e como ela e
   `Endless` a fase nao terminava — o MESMO defeito que o mapa 4 teve em 25/08.
   Mesma ordem de diais: teto 18->28, reposicao 8 orc + 8 lobo -> 16 + 12,
   leva 5/1,0s -> 7/0,8s (`network/wave_runs.go`).
2. **Gargulas do mapa 4 abriam fogo cedo.** `partyReachedTier` perguntava se
   ALGUM jogador vivo tinha alcancado o degrau 3, e um batedor — tipicamente um
   bot, que avanca lutando ate 700 px a frente da festa — cruzava a boca do
   saguao sozinho. Agora e a MAIORIA dos vivos: nem o batedor abre a porta nem
   um retardatario a tranca (`network/sentry_wake.go`, dois testes novos).
3. **Cura da Area Angelical (Sacerdotisa):** 20 -> 40 HP/s, e quem o altar
   reergue volta com 70% em vez de 50% (`skill/angelic.go`).
4. **Chuva de Meteoros (Mago):** dano de impacto 100 -> 480, que e 80% da vida
   do orc (600) e portanto DOIS meteoros para mata-lo; nascimento a cada
   0,015 s em vez de 0,025 (~67/s, regime ~93 no ar). Estes dois numeros ja
   estavam em `doc/combat_rules.md` e em `TestOneMeteorTakesFourFifthsOfAnOrc`
   desde 23/08 — a constante e que nunca foi trocada, e o teste estava
   vermelho. `doc/skill_performance.md` atualizado: se o cliente cair na chuva,
   o dial e lotear o broadcast, nao baixar o intervalo.
5. **Senhora das Trevas:** vida 2000 -> 30000 (15x). Ela e um alvo parado num
   cerco `Endless`; a vida dela e o cronometro da arena final.
6. **O PISCAR DE QUEM FICA NA FRENTE.** Dois personagens (ou duas magias)
   sobrepostos trocavam de camada a cada quadro. A causa e uma so, em tres
   lugares: `for range` sobre MAPA em Go percorre em ordem aleatoria a cada
   passagem, e todas essas colecoes sao `map[string]*Coisa`.
   - `game/render_order.go` (novo): corpos ordenados por Y e depois por ID, e o
     jogador local entra na fila na profundidade dele em vez de sempre por
     cima. Barras de vida e marcador de ausencia passaram para um segundo
     passe, acima de todos os corpos. Vale tambem para os monstros e projeteis
     espelhados no cliente.
   - `skill/draw_order.go` (novo): `orderedIDs` ordena cada colecao de efeito
     antes de desenhar (meteoro, bola de fogo, altar, santuario, escudo,
     flecha, cemiterio, legiao, avatar, espada, esfera da sentinela, bola de
     canhao).
   - `ability/registry.go`: `DrawAll` percorria o registro (mapa), entao ate a
     FAMILIA de magia que ficava por cima mudava por quadro. Agora ha uma lista
     `drawOrder` na ordem de registro.

## 2026-08-25 — Climax do mapa 4 travava: gargula 25->34 de dano (3 esferas derrubam), reposicao 8/12/8->14/14/6, teto 22->30, leva 5/1,1s->7/0,9s

O grupo aguentava a emboscada "O salao se fecha" sem que todos cassem abaixo
de 30% ao mesmo tempo, entao a cena do Arqueiro nunca abria e a fase (Endless)
travava. `TestFourSentryOrbsKillACharacter` virou `TestThreeSentryOrbsKillACharacter`;
nova propriedade `TestClimaxOrcIsNotAMinorityAgainstSlime` em
`garrisons_test.go`. Ver "O clímax não se vence matando o último" e a tabela
da gárgula em `doc/combat_rules.md`.

## 2026-08-23 — Paredes VERTICAIS do castelo: os corredores do 4, 5 e 6 ganharam parede

Os tres mapas de castelo com corredor nao tinham arte de parede nenhuma nas
laterais: `world_04`, `world_05` e `world_06` tinham so a faixa de colisao
pintada, e o corredor terminava no cascalho do lado de fora. A parede horizontal
aprovada nao resolve isso, e nao e questao de girar a peca.

- **A projecao decide.** O painel horizontal (`walls_wall_01`, 241x158 px)
  desenha a face SUL de frente, e o topo aparece so como faixa de 10 px. Nessa
  projecao, uma parede que corre norte-sul tem as faces leste e oeste de perfil
  — largura zero. O que sobra e o TOPO da parede visto de cima, com a face
  frontal aparecendo so na ponta SUL. E o que o kit novo desenha.
  Consequencia aceita e medida: um trecho vertical de 70 px ao lado de um painel
  de 158 px de altura le como mureta baixa. Nao e defeito da arte — e o que essa
  projecao produz. Mais presenca exigiria um kit com face lateral em obliqua
  paralela, que sao dois kits (leste e oeste) e nao um.
- **`castle_walls_vertical` (7 pecas).** `wall_v_cap_n`, `wall_v_middle_a/_b`
  (2 celulas), `wall_v_middle_a/_b_short` (1 celula), `wall_v_cap_s` e
  `wall_v_pier`. Qualquer trecho de N celulas fecha como
  cap_n + middles + cap_s, com pilastra a cada 4 celulas.
- **A folha veio com tres defeitos, e nenhum precisou de geracao nova**
  (`work/tiled-assets/build_castle_walls_vertical.py` reproduz o atlas):
  middles arrematados nas duas pontas (cintura de 20 px em cada junta, cortados
  entre juntas de bloco); folha clara e quente demais (lum 95,7 / p90 132,7
  contra 52,4 / 93,9 do painel aprovado, corrigida por casamento de histograma
  — o `match_biome_tint.py` trava em +-12% e o desvio era ~80%); e as duas
  fileiras da folha em escalas diferentes.
- **Os 4 cantos gerados ficaram de fora.** O braco horizontal deles nao e a
  parede horizontal aprovada: sem pilastra, sem o perfil de cimalha 10+8, fiada
  de 18 px contra 17. Precisam ser remontados por composicao a partir do painel
  aprovado, como as juncoes da cerca foram. Ate la, todo trecho colocado
  termina em tampa, que e o que um corredor precisa.
- **Ancora: canto SUDOESTE do footprint.** No `cap_s` ela fica na base da
  FAIXA, nao na base da arte — os 158 px de face sao altura, nao chao, a mesma
  leitura que o painel horizontal ja faz com os 47 px de rodape dele.
- **`collision: false` no manifesto, de proposito.** Todo trecho colocado hoje
  fica dentro da faixa de colisao ja pintada (o lado de fora do mapa, 4 celulas
  de espessura). Um footprint de 70 px la dentro nao bloqueia nada novo e faz o
  `validate_map.py` acusar dupla colisao em cada uma das 264 pecas. Se um
  trecho for colocado em chao livre um dia, e uma linha no manifesto.
- **Colocado por `work/tiled-assets/place_walls.py`**, que e onde os trechos de
  cada mapa estao declarados: `world_06` (o corredor inteiro, 50 celulas dos
  dois lados), `world_04` (salao de 55 celulas, vestibulo norte, escadaria) e
  `world_05` (corredor de 53 celulas e a entrada norte). O salao oval do
  `world_05` ficou de fora: e curvo, e o kit e reto.

**A conferir no jogo (F3):** andar o `world_06` de ponta a ponta e olhar as
juntas dos trechos e o encontro com a fileira de colunas; no `world_04`, o
encontro da parede vertical com a fileira horizontal na altura da escadaria
(linha 12) — e ali que falta o canto. `props` no F3 sobe ~24 por parede visivel.

**Defeito separado, encontrado e NAO corrigido:** no `world_07` as 44 pecas de
parede e coluna estao numa objectgroup chamada `structures_back`, mas o
`castle_manifest` esta registrado na camada `buildings` — entao elas nunca sao
desenhadas. A arena esta sem parede na tela hoje.

## 2026-08-22 (4) — Chuva de Meteoros sem culling; camera ancorada em pixel; doc de custo de magia

Terceira rodada do mesmo levantamento, agora na ultimate do Mago. O defeito e
irmao do da Legiao Espectral, por outra porta: nao e colisao, e DESENHO.

- **A chuva desenhava o mapa inteiro.** `MeteorRainInterval = 0.025` com 1,4 s
  de vida por meteoro da ~56 no ar ao mesmo tempo, cada um alimentando um
  rastro; em regime sao ~2.550 particulas vivas, e cada particula e um
  `DrawCircleGradient` (leque de ~36 triangulos no rlgl) — ~92.000 triangulos
  por quadro so de particula, em blending aditivo, em 4K. E os alvos sao
  sorteados no MAPA INTEIRO: no `world_02`, 83% do que era desenhado ficava
  fora da tela. `internal/skill/view.go` traz `SetDrawView` + `visible` /
  `visibleAny`, aplicados em `meteor.go`, `particle.go`, `explosion.go` e
  `fire_ground.go`. O renderer publica a MESMA janela que o mapa e os inimigos
  ja usavam.
- **O painel do F3 conta:** `magias: N desenhadas / M puladas pelo culling`.
  Um `M` parado em zero num mapa grande e culling faltando — o instrumento que
  teria pego este defeito na revisao.
- **`doc/skill_performance.md` (novo).** Sete regras com as contas que as duas
  ultimates quebraram, checklist de revisao e como medir no F3. Rota adicionada
  ao `AGENTS.md` como OBRIGATORIA para magia nova, e a
  `create-character-abilities` ganhou a secao que manda ler antes de escrever a
  primeira linha.
- **C2 implementado: camera ancorada em pixel.** O relato de "imagem quebrando
  ao caminhar" mesmo com vsync ligado nao era tearing: era a Causa 3 da §4½,
  documentada desde 09/08 e nunca corrigida. O terreno usa filtro POINT e a
  camera ficava em coordenada fracionaria, entao o chao saltava de texel em
  texel enquanto os sprites deslizavam. `Camera2DState.Update` arredonda
  `Target` e `Offset` depois do clamp.
- **O F3 responde a pergunta do vsync.** Cabecalho mostra `vsync ON/OFF` e o
  refresh do monitor, lidos de `rl.IsWindowState` — o estado REAL da janela e
  nao o que a `Config` pediu, porque um perfil de driver com vsync forcado em
  off ganha do jogo.

Nao foi mexido, de proposito: `MeteorRainInterval` e `MeteorRainDuration` sao
decisao de design. Culling nao muda um pixel do que o jogador ve.

## 2026-08-22 (3) — Correcao confirmada em jogo; retratos precarregados; vsync ligado

Refeitas as capturas do F3 depois da correcao da Legiao Espectral. **A mesma
cena do `world_02` que rodava a 16 fps com 96,7 ms de quadro roda a 60 fps com
16,7 ms**, e a linha nova diz onde estavam os 90 ms de "resto": `simulacao 0,0`.
Corroboracao independente: o `world_03` sustenta 156 inimigos vivos com
`simulacao 0,5 ms`. Ressalva honesta: as capturas mostram `espectros 0`, entao
ainda falta um print com os trinta espectros vivos para a prova direta.

Duas coisas novas apareceram nessas capturas:

- **O quadro de 40 ms do dialogo — CORRIGIDO.** `PIOR 40,2 ms` no `world_03`,
  e o sintoma relatado ("toda vez que passa um dialogo o ms piora, depois
  normaliza") e o retrato: a `reference.png` do orador (1536x1024, ~6 MB) era
  carregada na PRIMEIRA fala dele, dentro do quadro que abre a caixa. Mesmo
  defeito que `PreloadEnemyTextures` ja resolvia para as folhas de monstro, e
  mesma solucao: `dialogue.File.PortraitKeys()` lista o elenco do mapa e
  `ui.PreloadPortraits` sobe todos no `syncMap`, no quadro da troca de mapa. A
  lista sai do arquivo INTEIRO e nao do roteiro que vai tocar, porque o
  `on_last_stand` dispara no meio de uma luta perdida — o pior momento possivel
  para ler um PNG do disco.
- **Screen tearing: o jogo nunca pediu vsync.** `rl.SetTargetFPS` NAO e vsync —
  ele so faz o laco dormir, e o buffer continua sendo trocado no meio do
  desenho da tela. Era por isso que o painel mostrava `quadro 16,7 / PIOR 16,7`
  enquanto a imagem rasgava: a cadencia estava certa, o sincronismo nunca
  existiu, e um medidor de tempo de quadro nao ve tearing. `Config` ganhou
  `VSync bool` (true nas duas plataformas) e `Run` aplica
  `rl.SetConfigFlags(rl.FlagVsyncHint)` ANTES de `InitWindow`.

E um defeito de fiacao encontrado no caminho: **`Config.TargetFPS` nunca era
lido** — `Run` chamava `rl.SetTargetFPS(60)` com o numero cravado, entao o
mecanismo de "declarar 30 no Android" descrito no item mobile #6 nao teria
efeito nenhum. Agora e lido, e `0` significa "sem teto proprio: quem cadencia e
o monitor". O padrao continua 60 para nao mudar carga de GPU sem medida; num
monitor acima de 60 Hz vale trocar para `0` (ver `doc/performance.md`, "Vsync e
teto de quadro").

## 2026-08-22 (2) — A Legiao Espectral testava obstaculo contra o mapa inteiro: 16 fps no mapa 2

O painel de memoria adicionado horas antes respondeu a pergunta que motivou
ele, e a resposta foi "nao e vazamento". Duas capturas do mesmo host
(`world_01` fluido, `world_02` apos o climax a 16 fps) mostram heap vivo
praticamente igual (1,6 -> 2,5 MB), pico igual (3,5 -> 3,6), `Sys` igual
(17,5 -> 17,7), goroutines iguais (4) e espelho de rede zerado nos dois. Nada
acumula entre as fases.

O que a captura mostrava era **`GC 611` contra `GC 9.003`** e **90,1 dos 96,7 ms
do quadro no balde chamado "resto"** — churn de alocacao, nao retencao, dentro
de `UpdateSimulation`.

A causa: `LegionCount = 30`. Cada espectro se movia por `moveSpecter`, que
chamava `tilemap.IsColliding` ate quatro vezes, e `IsColliding` percorre a
lista INTEIRA de solidos do mapa — ~1.400 retangulos no `world_02` (1.132
celulas solidas mais os apoios das 179 pecas de vegetacao). Somando a
separacao, que movia os DOIS espectros de cada um dos 435 pares na hora, um
quadro com a legiao engajada fazia da ordem de **cinco milhoes de comparacoes
de retangulo**. E o custo cresce com o TAMANHO DO MAPA, que e por que a mesma
suprema rodava lisa na fase 1 e derrubava o jogo na fase 2 — a forma de um
vazamento, sem ser um.

- **As magias passam a falar com o `CollisionGrid`.** `StepLegions`,
  `StepArrows`, `StepFireballs` e `AdvanceLegions` recebem `collision.Solid` em
  vez de `[]rl.Rectangle`; `Host.SetCollisionRects` virou `Host.SetSolid`.
  `CollidesCentered` olha so as celulas que a caixa toca mais o indice espacial
  de apoios: uma ou quatro celulas, nao mil e quatrocentos retangulos. E a
  mesma porta que jogador e monstro ja usavam (`EntityManager.Solid`) — as
  magias e que estavam de fora, e a lista plana existia so para elas. O cache
  `World.collisionRects`, criado no mesmo dia, foi removido: com a lista plana
  sem uso, ele deixou de ter o que cachear.
- **Efeito colateral bom no portao da arena.** `UpdateArenaGate` remontava o
  snapshot de retangulos do host toda vez que o portao abria, para as magias
  enxergarem a passagem nova. Com a grade COMPARTILHADA, o portao muda os apoios
  dentro dela e quem le pela grade ve a mudanca no quadro seguinte: a chamada
  saiu. A malha de navegacao continua sendo avisada (`RebuildNavArea`), porque
  ela e derivada uma vez no carregamento e nao observa a grade.
- **`Legion.separate` soma os empurroes e move uma vez por espectro**: de ate
  870 `moveSpecter` por quadro para 30. Tambem mais estavel — aplicando par a
  par, o empurrao de A contra B mudava a posicao que o par seguinte lia.
- **O painel do F3 separa `simulacao` de `resto`** (`internal/game/perf_sim.go`)
  e ganhou `lixo N MB/s` (taxa de alocacao, o denominador que faltava para o
  `GC 9003`) e `espectros N` — a Legiao poe trinta entidades simuladas em campo
  e nao aparecia em contador nenhum, entao o painel mostrava "inimigos 10 vivos"
  num mapa onde trinta espectros pisavam.

Aritmetica das duas correcoes juntas: ~1.000 comparacoes por quadro contra
~5.000.000. Nao e numero medido — a medida vem da proxima captura, e a linha
`simulacao` existe para ela. Se `simulacao` cair e o `resto` continuar em 90 ms,
a causa e GPU (terreno em 4K, Faixa R4) e e outro trabalho.

Detalhes, tabela das capturas e a analise de por que paralelizar entre nucleos
NAO era a resposta (14% de CPU a 16 fps significa processo esperando, nao
processo sem nucleo) em `doc/performance.md`, secao 4⁹⁄₁₀.

## 2026-08-22 — O lag que cresce com a FASE: medidor de memoria no F3 e quatro retencoes fechadas

Sessao de teste com dois jogadores (host + cliente) reportou o jogo ficando
quase injogavel conforme a campanha avanca. A suspeita era acumulo de recurso.
O projeto nao media um unico byte, entao a primeira entrega e a MEDIDA e nao a
correcao: `internal/game/perf_mem.go` acrescenta tres linhas ao painel do F3 —
heap Go (atual, pico da sessao e `Sys`), objetos, contagem de GC, goroutines,
texturas na VRAM separadas por familia (mapa / folhas de inimigo / retratos de
dialogo) e o tamanho do mundo espelhado da rede. Comparar duas capturas, uma na
fase 1 e outra na fase em que engasga, ambas logo apos o carregamento, responde
sozinha se o defeito e retencao, pico de alocacao ou custo de GPU.

Quatro achados do mesmo levantamento, ja corrigidos:

1. **O cliente reconstruia a colisao inteira do mapa a cada quadro.**
   `loop.go` chamava `w.Collision.Rects()` sessenta vezes por segundo para
   entregar os obstaculos a `AdvanceClientSkills`; `Rects()` varre a grade
   inteira (42.000 celulas no `world_03`) e aloca um retangulo por celula
   solida. O host ja fazia certo — uma vez por mapa, via `SetCollisionRects`.
   Agora a lista mora no `World` (`World.collisionRects`), montada uma vez no
   carregamento, e os dois papeis leem a mesma. E o unico dos quatro cujo custo
   CRESCE COM O TAMANHO DO MAPA, entao e o que melhor explica "piora conforme
   avanca nas fases".
2. **`portraitCache` nunca descarregava** (`internal/ui/dialogue_box.go`, item
   M5 de `doc/performance.md`). Cada orador de dialogo prende o `reference.png`
   dele, ~6 MB na placa, ate o fim da sessao; sete fases de campanha acumulam o
   elenco inteiro. O cache mudou de arquivo (`internal/ui/portrait_cache.go`) e
   de escopo: vive por MAPA, e `travelTo` o descarrega junto do mapa que fica
   para tras.
3. **`Manager.Reset()` nao limpava os efeitos do chefe.** Espinhoes e nevoa so
   saiam pela morte da Senhora das Trevas (`host_boss.go`); sair do mapa 7 por
   portal ou por F8 os levava para a fase de destino e nada mais os tirava de
   la.
4. **`MapRenderer.Load` podia vazar referencia de textura.** Dois tilesets do
   mesmo mapa citando a MESMA imagem davam dois `AcquireTexture` e um so
   `ReleaseTexture` (o mapa `mr.Textures` e indexado por caminho), prendendo o
   atlas na VRAM pelo resto da sessao. Uma referencia por caminho agora. E
   `Unload` zera `Terrain` e `Manifests` depois de devolve-los, para uma
   segunda chamada nao devolver referencias que ja nao tem.

## 2026-08-20 — Bots atravessavam o mapa 3 sem lutar: o atalho do portal desligava o cerebro

`tickOneBot` pulava `rt.brain.Think(view)` inteiro sempre que o portal estava
ativo, assumindo que o campo estaria vazio nesse ponto — falso para
`world_03`, mapa de guarnicao sem `enemy_spawn_*`, cujo portal fica aberto
desde o primeiro quadro com a guarnicao inteira em campo. O atalho caiu: o
cerebro roda sempre, e o portal virou um destino que ele mesmo escolhe via
`travelDest` (`internal/bot/steering.go`), so quando nao ha inimigo engajavel
por perto **e** um humano ja esta na porta ou perto dela
(`HumansAtPortal`/`PortalEscortRadius`) — sem essa segunda trava o bot
marcharia sozinho no instante em que o portal de um mapa sem hordas abre.
`finishMove` zera o `Push` so quando o portal vence a cadeia de destino
(alvo engajado > recuo > `travelDest` > formacao); atacar continua
funcionando normalmente enquanto o bot caminha ate a porta. Ver
`doc/plan_avanco_bots_e_gargula.md` §A2 (causa 4) e `doc/tilemap.md`
("Portal trancado pelas hordas").

## 2026-08-20 — Fase B2/B4: sentinela visivel na View do bot, so o Arqueiro a mata

A gargula era descartada na montagem da `View` do bot (`buildBotView`), entao
nenhum bot jamais a via. Agora ela entra em `Foes` com `IsSentry: true` e
`HitCentre` (o ponto de acerto real, nao os pes). Toda funcao de selecao de
alvo comum do pacote `bot` (`nearestFoe`, `mostThreateningFoe`,
`clusterCentre`, `countFoesWithin`, `foeBlocksLine`, `foeBeyondAlly`,
`anyFoeWithin`) passou a pular `IsSentry` — nenhum bot gasta cadencia numa
estatua invulneravel. So o Arqueiro enxerga: com a suprema pronta e uma
sentinela viva, `huntSentry` vira a prioridade do quadro inteiro, aproxima ate
o alcance util da propria suprema (`View.UltimateRange`, que so a Rede
preenche para o Arqueiro) e mira o `HitCentre`, preferindo uma segunda
sentinela alinhada atras da primeira. `EnemiesLeft` passou a excluir
sentinelas — sao postos fixos, nao horda. Ver
`doc/plan_avanco_bots_e_gargula.md` §B2/§B4.

## 2026-08-20 — Fase B1: gargula de alcance global, esfera com TTL por distancia

`entity.SentryGlobalRange` (16000, maior que a diagonal do maior mapa)
substitui os 1900 fixos da gargula — o alcance so decide quem ela pode
escolher como alvo, o dano continua so chegando quando a esfera encosta. Sem
isso o TTL fixo de 9s nunca bastaria para uma esfera lancada do outro lado do
mapa: agora `network.sentryOrbTTLFor` calcula pela distancia
(`dist/velocidade*1,5 + 2s`, teto de 40s) e o TTL viaja no evento de rede
("cast") para o cliente nao podar a esfera antes da hora. `skill.SentryHasLiveOrb`
impede uma segunda esfera da MESMA sentinela enquanto a primeira ainda esta no
ar — sem isso, cadencia de 1,35s contra ate 40s de voo empilharia uma dezena
delas perseguindo o mesmo jogador. Ver `doc/plan_avanco_bots_e_gargula.md`
§B1/§B2.

## 2026-08-20 — Fase A3: recuo com histerese e alcance util do ataque basico

`retreatHysteresis` entra em recuo abaixo de 35% de vida e so volta a engajar
acima de 60% (Paladina: 25%, e so depois de Escudo gasto —
`paladinaRetreatHysteresis`), evitando o vaivem de um unico golpe cruzando o
limiar. Recuando, o destino e o posto de formacao empurrado 300px para tras
na direcao oposta ao inimigo mais proximo (`retreatDest`); Arqueiro, Mago,
Necromante e Sacerdotisa continuam atirando enquanto recuam, Paladina para de
golpear. De brinde (§A5): cada classe ganhou o alcance util do proprio
projetil (`arqueiroAttackRange` 1120, `magoAttackRange` 840,
`necromanteAttackRange` 800, Sacerdotisa reaproveitando `boltRange`) — o
ataque basico so sai dentro dele, em vez de gastar a cadencia numa flecha ou
bola de fogo que expira no caminho. Ver `doc/plan_avanco_bots_e_gargula.md`
§A4/§A5.

## 2026-08-20 — Fase A2: raio de engajamento e formacao por classe

Bots escolhiam alvo no mapa inteiro (`mostThreateningFoe`/`nearestFoe` sem
limite) e marchavam ate ele ignorando tudo pelo caminho. `engageableFoes`
(`internal/bot/steering.go`) filtra o alvo para dentro de `engageRadius`
(900px) do bot OU do `HumanCentre` — fora disso o inimigo simplesmente nao
existe para a decisao. `formationPost` da a cada classe um posto relativo ao
`HumanCentre` na direcao do avanco (`View.AdvanceDir`, media suavizada da
velocidade dos humanos vivos, calculada uma vez por quadro em
`Host.updateAdvanceDir` e mantida quando o grupo para). `followDest` passou a
usar o posto de formacao em vez do centro cru. Ver
`doc/plan_avanco_bots_e_gargula.md` §A2/§A3 (R2, R3).

## 2026-08-20 — Fase A1: `HumanCentre` tira o bot da propria referencia de avanco

Bots seguiam `PartyCentre`, a media de TODOS os vivos — bots inclusive. Um bot
que se adiantava puxava o centro consigo, e o resto seguia o centro que tinha
acabado de andar: um erro individual virava marcha do esquadrao (mapa 3, bots
atravessando ate o climax). `View` ganha `HumanCentre`/`HasHumans`, calculados
so a partir de jogadores humanos vivos; todo destino de "seguir o grupo" nos
cinco cerebros passa a usar isso (`followDest`, `internal/bot/steering.go`).
Sem humano vivo, o bot segura posicao em vez de escolher um novo lider entre
os bots. Ver `doc/plan_avanco_bots_e_gargula.md` §A1.

## 2026-08-20 — Trava de uma via da arena (mapa 5) agora vale para bots

`arena_gate.go` corrigia a posição só do jogador local; bots voltavam ao
corredor porque ninguém aplicava a regra a um corpo que só o host move.
`network.SetArenaLock` publica a zona por quadro (padrão de
`SetPartyPortals`), `botRuntime.arenaLocked` aplica o mesmo clamp por bot no
host, e `Intent.Dest` ao sul da soleira é projetado de volta, descartando a
rota velha. De brinde, `resolveBotMove` para de dar passe livre a um bot
encravado num footprint que acabou de religar (o portão de saída,
`SetFootprintsEnabledOverlapping`): empurra para a célula livre mais próxima
antes de resolver o passo, em vez de deixá-lo atravessar parede. Ver
`doc/tilemap.md` "Arena de mão única".

## 2026-08-20 — Recarga da Rajada de Flechas: 1,5 s -> 6 s

A Q do Arqueiro recarregava quatro vezes mais rapido que a Bola de Fogo, que e
a habilidade equivalente do Mago. Com o bot jogando a classe ficou evidente: a
rajada saia quase sem intervalo, porque a recarga quase sempre permitia.
`skill.ArrowVolleyCooldown` passa a 6.0, a mesma regua de `FireballCooldown`.
Vale para humano e bot — o portao e o mesmo (`Host.beginSkillCooldown`).
Tabela de recargas em `doc/combat_rules.md` atualizada.

## 2026-08-20 — Navegacao de bots e monstros (fases 1-5)

Bots empurrando arvore a caminho do portal, monstros socando a cerca do
mapa 3: os dois decidiam sem mapa, so "esta caixa colide aqui?". Ver
`doc/plan_navegacao_bots_monstros.md`.

- **Fase 1**: `internal/network/host_bot_move.go` trocou `collision.Resolve`
  por `collision.ResolveDetour` com direcao comprometida (`botRuntime.detourDir`)
  — o remendo imediato, sem malha.
- **Fase 2**: pacote novo `internal/nav` (puro — so `collision`, `world`,
  `rl.Vector2`): malha de 32px derivada da colisao, A* octil sem cortar
  quina, suavizacao por linha de visao, `Follower` por agente, orcamento de
  8 buscas/quadro compartilhado.
- **Fase 3**: bots navegam pela malha. `bot.Intent` trocou `Move rl.Vector2`
  por `Dest`/`HasDest`/`Push` — o cerebro diz PARA ONDE, `host_bot_tick.go`
  (via `nav.Follower`) decide POR ONDE. `EntityManager.Nav *nav.Grid`, irmao
  de `Solid`.
- **Fase 4**: monstros so consultam a malha quando `Enemy.moveTowardTarget`
  detecta, por uma JANELA de distancia real (nao o `collision.Progressed`
  por passo — esse continua contando como "progresso" um monstro deslizando
  ao longo de uma face sem se aproximar do alvo de verdade), que a
  distancia ao alvo nao encolheu por `FoeStuckBefore` (0,4s). Investe direto
  por padrao, como antes; contorna so quando bater prova que o rumo direto
  nao funciona.
- **Fase 5**: `Host.RebuildNavArea` reconstroi so a area do portao da arena
  quando ele abre/fecha (`game/arena_gate.go`), em vez da malha inteira.
  Overlay de depuracao **F4**: malha (livre/bloqueada) e rota de cada bot e
  monstro (`game/nav_debug.go`).
- `nav.Follower` reancora no waypoint mais distante ainda visivel antes de
  pagar por um replan inteiro, para sobreviver a um empurrao de
  `ResolveEnemyOverlap` sem perder a rota.
- Testes novos: `internal/nav` (caminho existe/nao existe, sem cortar
  quina, `RebuildArea`, `NearestWalkable`, orcamento, benchmarks de
  `Build`/`FindPath` na escala do `world_05`); `internal/entity` (monstro
  atras de barricada em L, matilha de 20 atravessando um vao distante,
  monstro em campo aberto nunca consulta a malha); `internal/network`
  (bot contornando arvore, bot usando a malha quando o vao esta longe,
  `RebuildNavArea` seguindo o portao).

## 2026-08-20 — A matilha do `world_02` nao acaba antes do climax

Relato do Gui: no mapa 2 os jogadores estavam ELIMINANDO a ultima horda antes
do climax comecar. A corrida terminava, o mapa ficava limpo e a cena do
Necromante nunca tocava — a fase liberava o portal sem entregar a suprema.

- `world02Waves`, horda 3 ("a matilha"): `Endless: true`. Mesmo contrato do
  climax do mapa 3 e da horda 5 do mapa 5 — enquanto `LastStandDone()` for
  falso, a composicao volta inteira para a fila cada vez que esvazia. Depois do
  resgate a reposicao para, o que sobrou em campo fica finito, e matar o ultimo
  conclui a corrida e libera o portal.
- A composicao (12 slime + 84 lobo) deixa de ser um TOTAL e passa a ser a
  mistura de reposicao. `MaxConcurrent` 42 continua sendo o teto de pressao: o
  ritmo nao muda, o fim e que nao chega.
- `TestClimaxWaveWaitsForTheRescue` (novo, `wave_runs_test.go`): para todo mapa
  com janela `ClimaxWindowWaveIndex`, a horda declarada em `FromWave` tem de
  ser a ultima da corrida E tem de se repor. Uma horda finita maior parece a
  mesma coisa e nao e — este e o teste que reprova a proxima tentativa de
  "subir a dificuldade" trocando `Endless` por numero.
- `TestWorld02HasTwoWaves` -> `TestWorld02HasThreeWaves`: o teste ainda cobrava
  duas hordas, de antes de a horda 2 (desgaste) existir.

## 2026-08-20 — Travessia de portal: quem entra some e espera

- Corrigida a travessia de portal com bots no grupo: um bot entrando e
  saindo do retângulo o tempo todo impedia "todos dentro ao mesmo tempo" de
  ficar verdadeiro. Agora quem pisa na zona do portal (humano ou bot) some
  da tela e congela até o grupo inteiro estar dentro, liberando o retângulo
  para os demais. Novo `PlayerState.InPortal` (estado de tique, como
  `IsDead`), decidido pelo host em `Host.tickPortalPresence`
  (`internal/network/host_portal_presence.go`) contra os retângulos que
  `game.UpdatePortal` publica via `Host.SetPartyPortals`
  (`internal/network/host_bot_portal.go`, que trocou de guardar só o centro
  para guardar os retângulos inteiros). `tickBots` pula por inteiro quem
  está `InPortal`; o humano local congela via `network.LocalPlayerInPortal`
  em `game.ProcessInput`, com ESC/botão SAIR (`game/portal_cancel.go`,
  `ui/portal_wait.go`) para cancelar. `bot.SeekPortal` parou de usar
  separação entre aliados — o portal só existe com a fase já limpa, então
  não há golpe em área para evitar ali, e a separação era o que empurrava os
  bots para fora do mesmo retângulo pequeno. `InPortal` é limpo em
  `PlaceEveryoneAtSpawn` e `ResetStage`. Ver `doc/network.md`,
  `doc/tilemap.md` e `doc/plan_bots_de_classe.md` §5/§14.

## 2026-08-20 — IA da Sacerdotisa (bot): cura primeiro, ataca depois

- `internal/bot/sacerdotisa.go` reescrito: a mira agora prioriza a reta que
  atravessa o aliado mais ferido dentro do alcance do tiro (`boltRange`),
  recusa a reta quando um monstro bloqueia o caminho (ataca o bloqueador
  em vez disso) e só cai para o alvo mais ameaçador quando ninguém precisa
  de cura. `backLine` subiu para 420 e ganhou `panicLine` (fuga imediata) e
  `calmRadius` (sem monstro por perto = janela de recuperação: aproxima dos
  feridos e atira até encher todos, ou lança Santuário em si mesma se for a
  ferida). O gatilho do Santuário agora conta ela mesma na soma de feridos e
  exige proximidade do aglomerado antes de lançar. Ver
  `doc/plan_bots_de_classe.md` §5.

## 2026-08-20 — Reconexao do cliente: correcao de bugs de compilacao/integracao

- Fechado o gap entre `internal/network/host_absence.go`/`host_rejoin.go`/
  `reconnect.go` (escritos por outro agente) e o resto da base: adicionados
  `PlayerState.Absent`/`absentSince`, `TravelPayload.Reconnect` e
  `ClientConn.superseded` (`protocol.go`, `host.go`), sem os quais o pacote
  `internal/network` nao compilava. `handleClient` (`host.go`) agora delega
  `MsgJoin` a `Host.handleJoin` (antes ainda recriava o `PlayerState` do zero
  a cada join, ignorando o codigo de reconexao) e marca ausencia em vez de
  apagar o slot no defer, checando `superseded`. `checkGameOver` passou a
  ignorar jogadores ausentes dos dois lados (nem segura, nem decide sozinho).
  `game/dialogue.go`'s `partyIsFalling` trocou `GetAllPlayers` por
  `PresentPlayers`. `UpdateSimulation` ganhou a chamada a `tickAbsence`.
  Reescritos `Client` (mutex unico para troca de conexao e escrita,
  `readLoop(conn)` capaz de distinguir erro da conexao ATUAL de uma conexao
  ja substituida) e criado `keepalive.go` para `setKeepAlive`, ambos exigidos
  por `reconnect.go` mas nunca implementados. `internal/entity/enemy_manifest_test.go`
  corrigido para comparar com `reflect.DeepEqual` (struct com slice nao
  compara com `!=`), bloqueava `go test ./...` do modulo inteiro.
- `go build ./...` e `go vet ./...` limpos. Os seis testes exigidos pela
  tarefa de reconexao passam. Falhas remanescentes em `go test ./...`
  (`internal/game`, `internal/network`, `internal/entity`) sao de outras
  frentes inacabadas — `world_02.json` invalido, balanceamento de hordas/
  canhoes/sentinelas do `world_07`, `PlayOrder` da Senhora das Trevas — e
  fora do escopo desta correcao.

## 2026-08-19 — Barra de vida do aliado no espaco de mundo

- `internal/entity/player_health_bar.go`: `DrawAllyHealthBar` desenha a vida
  de cada jogador remoto acima do proprio quadro (geometria de `CharacterDef`,
  nao a do monstro). Mesmos limiares de cor do inimigo (0.5/0.25), paleta mais
  clara/suave para nao confundir aliado com alvo; 5px de altura, fundo e borda
  que a barra do monstro nao tem. Chamada em `internal/game/renderer.go`, no
  laco de `allPlayers`, pulando jogador morto.

## 2026-08-20 — Senhora das Trevas (chefe final do world_05)

- **Arte**: seis folhas em `assets/sprites/enemies/senhora_das_trevas/`
  (`idle` 8q, `idle_scan`, `cast_loop`, `cast_release`, `attack_windup`,
  `attack_strike`, 4q cada), 38,8 MB de VRAM, `RenderScale 1.0` — as folhas
  foram montadas no tamanho de tela, entao o runtime nao reamostra nada.
  Manifesto emitido por `work/enemy-sprites/senhora-das-trevas/montar_folhas.py`.
- **Reproducao de animacao** (`internal/entity/enemy_anim_playback.go`): dois
  conceitos novos no `EnemyAnimDef`. `PlayOrder` separa PASSO de COLUNA, para
  que uma animacao possa tocar quadros fora de ordem — o `idle_scan` usa
  (0,1,2,3,2,1) e nao paga pelos quadros do retorno. `OneShot` marca o ciclo que
  toca uma vez e trava no ultimo quadro. Os dois tem zero value compativel com
  todo inimigo anterior, que por isso nao mudou.
- **Maquina de estados de chefe** (`internal/entity/enemy_boss_anim.go`):
  `enemyAnimFor` decide por velocidade e a chefe tem `Speed 0` — ficaria em idle
  para sempre. A maquina nova decide por tempo e intencao, com a janela de
  desvio (`attack_windup`) sendo um LACO com duracao propria em vez de um quadro
  segurado.
- **Avanco visual**: o `idle_scan` desloca o DESENHO em ate 46 px e nunca
  `Position` nem a hitbox. O ciclo de patas foi desenhado no lugar, seguindo
  `enemy_creation_guide.md` §7.
- Testes: `internal/entity/senhora_manifest_test.go` amarra a `EnemyDef` ao
  manifesto e reprova se um `PlayOrder` apontar para coluna inexistente.

## 2026-08-20 — `world_07`, a arena da Senhora das Trevas (passo 1 de 6)

- **Mapa**: `assets/maps/world_07.json`, 40x30 celulas (5120x3840). Sala fechada
  de castelo: `dark_flagstone` no miolo, `siege_gravel` e `bare_soil` desfazendo
  o piso na borda (mesma pilha, entao a transicao nao e pintada). Muralha de
  `fortress_wall` no perimetro, dois portoes fechados a leste e oeste, quatro
  braseiros. Miolo deliberadamente VAZIO — a auditoria avisa 6 blocos 8x8 sem
  detalhe e isso e o desenho: e o espaco de desvio dos espinhoes.
- Auditoria: 100% do piso alcancavel a pe, nenhuma janela 3x3 poluida.
- **Registro**: `campaignMaps` (setima fase), `waveRuns` (`world07Waves`),
  `climaxWindows` (janela na horda 1 — a fase inteira e o climax) e
  `sentryPosts` (dez postos, dos cantos para o meio das paredes).
- **A horda**: uma so, `Endless`, `Ambush`, 10 slime + 10 lobo + 8 orc, leva
  inteira a cada 70 s pelos dois portoes (5/5/4 de cada lado), teto de 34 vivos.
  As gargulas entram por `Sentries`, nao pela composicao — Speed 0 exige posto.

Falta (passos 2 a 6 de `doc/plan_world07_arena.md`): nascer a chefe no
`boss_anchor`, a barra de chefe, os tres relogios (`host_boss.go`), os espinhoes,
a nevoa com as isencoes de Area Angelical / Avatar, e o fim da corrida por morte
da chefe (`EndsWithBoss`).

### Ajuste (mesma data): o kit e de CASTELO, e as gargulas vao para as extremidades

- O grupo ainda esta dentro do castelo, entao o salao usa o mesmo kit do
  `world_04`/`world_05`: chao de `castle_stone` (14) com `castle_blocks` (11) na
  faixa do perimetro e `castle_carpet` (13) num tapete que sai da entrada sul e
  abre no estrado da chefe — numa sala vazia de 5120x3840 o tapete e a unica
  coisa que diz para onde olhar, e ele aponta para ela. Pecas do
  `castle_manifest`: paredes com estandarte, dois portoes, duas fileiras de
  colunas, duas estatuas ladeando o estrado, quatro braseiros.
- **`paint_terrain.py` nao conhece os materiais de castelo** (as opcoes de
  `--type` param no 10), entao o chao e escrito direto pelo
  `work/tiled-map-world07/build_world07.py`. Os quatro materiais formam a
  TERCEIRA pilha (`terrain_mask.go`) com `edgeWidth` 0,01: eles quase nao
  desbotam um no outro, que e o certo para chao construido — o tapete tem borda
  reta, nao serrilhada.
- **Corrigido `render_map.py`**: o `TERRAIN_TEXTURE` parava no material 10, e o
  sintoma era um mapa de castelo renderizando com o chao PRETO. O motor sempre
  soube desenhar 11-14; era so a ferramenta que nao.
- **Postos de sentinela nas extremidades.** A gargula tem `Speed 0`: onde nasce
  e onde fica. Os dez postos passaram para as paredes leste e oeste, acima e
  abaixo de cada portao, guardando a boca por onde a horda entra. A ordem
  alterna os lados, entao cada horda arma uma a oeste e uma a leste.

### Passo 2 de 6: a chefe nasce e ganha barra propria

- `internal/network/bosses.go`: tabela `bossOfMap` (so o `world_07`), `InstallBoss`
  na ancora `boss_anchor` do mapa, `RestoreBoss` no reinicio de fase e
  `updateBossState`, publicado uma vez por quadro. Mesma forma de `waveRuns`,
  `garrisons` e `sentryPosts` — a FASE declara e o resto so le.
- `internal/network/boss_state.go`: `BossState` publicado pelo host e lido pelo
  HUD, igual ao `WaveState`. Host e cliente desenham a mesma barra sem nenhum dos
  dois ir procurar a criatura no EntityManager.
- `internal/ui/boss_bar.go`: barra no HUD, em espaco de TELA (regra do
  `doc/camera.md`), com o nome acima, moldura de ouro envelhecido e marcas a
  cada 25% — 400 de vida sem subdivisao nao da sensacao de progresso.
- A barra flutuante da chefe foi suprimida: duas barras para a mesma criatura e
  ruido, e com trinta inimigos em campo a flutuante e uma entre trinta.
- `RestoreBoss` entrou no `host_reset.go` pelo motivo CONTRARIO ao das gargulas:
  sem repor a chefe, a segunda tentativa nao ficaria mais facil, ficaria
  impossivel de terminar — a corrida daquele mapa so para quando ela cai.
- **Marcadores da horda renomeados para `enemy_spawn_gate_*`.** Eram
  `climax_spawn_*`, que e a porta de `StartClimax` (a emboscada roteirizada do
  mapa 3). `StartWaveRun` monta a corrida a partir de `enemy_spawn_*`, e um mapa
  sem nenhum simplesmente nao roda horda: a arena teria ficado silenciosa.

### Passos 3 a 6 de 6: os tres relogios, os espinhoes, a nevoa e o fim da fase

- `internal/network/host_boss.go`: espinhao a cada 15 s, nevoa a cada 60 s,
  primeiro compasso aos 6 s. Nao cabiam em `AttackCooldown`, que e UM numero.
  Os tres se alinham a cada 420 s (MMC de 60 e 70, e 15 divide os dois) — e isso
  fica, porque da a uma luta longa uma onda em vez de uma parede.
- `internal/skill/boss_thorn.go`: a marca no chao PULSA (perigo) e o anel FECHA
  (quanto falta) — duas informacoes distintas de proposito. `ThornTelegraph` e
  2,0 s e nao um numero solto: sao os 1,8 s do laco `attack_windup` mais os
  0,21 s ate o terceiro quadro do `attack_strike`. A posicao e uma FOTO tirada
  no instante em que ela levanta os bracos, e o espinho nao segue ninguem: um
  espinho que perseguisse tiraria do desvio a condicao de decisao.
- `internal/skill/boss_fog.go`: 30/s, ~4 s para matar. Cobre os limites do
  MUNDO e nao a zona `arena` do mapa — se dependesse da zona, um mapa novo com
  chefe e sem ela teria uma conjuracao que nao machuca ninguem, em silencio.
- `internal/skill/angelic_contains.go`: `AngelicContains(ponto)`, que e a
  pergunta que a nevoa faz. `HasAngelic(id)` responde "a Sacerdotisa TEM altar",
  e quem se salva e quem esta EM CIMA dele — inclusive quem nao o conjurou.
- `WaveDef.EndsWithBoss`: `Endless` so conhecia uma saida, o resgate do mapa 3.
  A arena tem outra. Ela para a REPOSICAO e nao a horda: o que esta em campo
  continua vivo, porque matar os trinta no instante da morte da chefe some com
  a limpeza final, que e o momento em que a vitoria assenta.

**Lacuna conhecida: nada disto e replicado.** Num cliente a chefe fica em idle e
nem a marca do espinhao nem a nevoa aparecem — o dano chega pelos eventos de
combate, mas sem aviso. Fechar exige o `Anim` da chefe no protocolo e dois
eventos novos. E o que falta antes de a fase ser jogavel em rede.

### Fechamento: o portal e o acumulo das gargulas

- **O `world_06` ja esperava esta fase.** O portal dele chama-se
  `portal_para_o_chefe` e o `target_map` estava provisorio, apontando para o
  `world_01` — a campanha voltava ao comeco em vez de chegar na arena. Agora
  aponta para o `world_07`.
- **Gargulas por relogio, nao por `WaveDef.Sentries`.** Aquele campo e por
  HORDA, e a arena tem uma horda so, infinita: `Sentries: 2` arma duas no comeco
  e para para sempre. O acumulo virou um terceiro contador em `host_boss.go`,
  com o mesmo periodo da leva (70 s), de modo que cada cerco novo chega com uma
  torre nova em cada portao. Comeca em um ciclo cheio porque a primeira dupla ja
  veio da propria horda — armar mais duas nos primeiros segundos poria quatro
  torres em campo antes de o grupo ter visto uma. Teto: os dez postos.

### Correcoes do primeiro teste em jogo

1. **A chefe "perseguia" o jogador — e nao era perseguicao, era EMPURRAO.**
   `moveTowardTarget` com `Speed 0` nao move nada, mas `ResolveEnemyOverlap`
   dividia a correcao de sobreposicao MEIO A MEIO. Com trinta inimigos entrando
   nela, a chefe era arrastada para fora da ancora. Agora quem tem `Speed 0` nao
   e empurrado: o corpo movel absorve a correcao inteira. Vale para a gargula
   tambem, que sofria do mesmo em menor escala.
2. **Ritmo das animacoes.** A causa nao era o numero de desenhos, era o TEMPO
   por pose: quatro quadros a 0,07 s davam 0,28 s para o golpe inteiro, e o
   agachamento ficava 70 ms na tela. `PlayOrder` sustenta a pose sem gerar arte:
   o `attack_strike` virou `[0,0,1,2,2,2,2,3]` a 0,10 s, com o agachamento
   ocupando QUATRO dos oito passos. Nos LACOS repetir quadro e matematicamente
   igual a dobrar o `FrameTime`, entao neles o ajuste foi so no tempo.
   **Isto sustenta a pose, nao cria intermediarios** — se ainda ficar duro entre
   uma pose e outra, ai sim e geracao de quadros novos.
3. **Aviso de PERIGO.** A danca e o telegrafo desenhado na propria chefe, e ela
   acontece do outro lado de uma arena de 5120 px: um grupo segurando um portao
   nao ve a chefe. O aviso vai para o HUD com moldura vermelha na borda da tela
   (visao periferica, sem tapar o campo de jogo) e o piscar ACELERA conforme o
   tempo acaba — a frequencia e a segunda informacao.
4. **O espinhao redesenhado.** A marca tinha um pulso e um anel; pulso diz
   "perigo" e nao diz quando, e a marca ficava identica do primeiro ao ultimo
   instante, o que ensina o jogador a ignora-la. Agora sao quatro camadas: poca
   escura (ONDE, no raio exato do dano), anel que fecha (QUANTO FALTA),
   rachaduras que crescem do centro (vem DE BAIXO) e as pontas espiando nos
   ultimos 25% (O QUE vem). A urgencia sobe ao quadrado do tempo. O espinho
   virou um tufo de sete lascas de alturas diferentes, deterministicas por
   indice — sorteadas, elas piscariam entre quadros.

### 31/08/2026 (segunda rodada) — sequência de vitória e mapa da arena reduzido

- **Sequência de vitória corrigida.** Fogos e "Missão Concluída" liam direto
  de `Health <= 0` e apareciam antes de a horda parar, dos caídos se
  erguerem e da cena final falar. `BossState.VictoryReady` agora guarda
  esse instante; só `(*Host).MarkBossVictoryReady`, chamado pelo
  `DialogueDirector` quando a cena do gatilho `TriggerWavesCleared` termina,
  o liga. `(*Host).reviveFallenForVictory` (novo `host_boss_victory.go`)
  ressuscita com vida cheia quem morreu na luta, uma vez por queda da chefe.
  De brinde: `BossState` passou a viajar no `EnemyUpdatePayload` — antes
  nunca era replicado a clientes.
- **Mapa do `world_07` reduzido pela metade:** 40×30 células (5120×3840 px)
  viraram 20×15 (2560×1920 px). Toda a geometria (muros, pilares, portões,
  sentinelas, âncora da chefe, zona `arena`) escalada por 0,5; ver
  `doc/plan_world07_arena_tuning.md` §11 para o detalhe da reamostragem.
- Validação apenas sintática (`gofmt -e`) — sem Go funcional neste ambiente
  para compilar/testar; rodar `go build ./...` e `go test ./...` localmente
  antes de jogar.

### 31/08/2026 (quarta rodada) — bug: horda presa em "restam N" depois da chefe cair

- **Causa:** `EndsWithBoss` só parava a reposição da horda; o que já estava em
  campo (até a leva inteira, 28) continuava vivo e precisava ser morto na mão
  para a corrida chegar a `WavePhaseCleared` — sem isso o diálogo final e a
  vitória (que dependem dessa fase) nunca chegavam. Reportado com print: chefe
  morta, HUD preso em "restam 28", sem diálogo nem "Missão Concluída".
- **Correção:** `(*Host).endHordeOnVictory` (novo `host_boss_victory.go`) mata
  toda a horda e as gárgulas dos portões no instante da queda da chefe (mesmo
  `IsActive = false` de uma morte normal) e zera `WaveRunner.pending`
  (`clearPending`, novo). Ver `doc/plan_world07_arena_tuning.md` §13.

## Arena da Senhora das Trevas: mobília e a camada morta do `world_07`

- **Arte nova (3 atlas):** `arena_throne` (trono cortado em base/encosto na linha
  do assento, casulos, degrau), `arena_candles` (8 peças de vela de ritual) e
  `arena_sigils` (círculo de invocação, sigilos e marcas de chão). Prompts,
  scripts de montagem e renders em `work/tiled-assets/`; regra de estilo da
  câmara em `doc/art_style.md` §12.
- **Defeito corrigido (o que fazia a fase parecer crua):** 50 objetos do
  `world_07` nunca eram desenhados. 44 muros e colunas estavam num objectgroup
  chamado `structures_back` — nome de *role*, não de camada lida — e 6 props na
  camada `props`, que pertence ao `siege_defenses_manifest`. `manifestSources`
  é uma lista de pares (manifesto, objectgroup) e não tem fallback nem aviso:
  objeto em camada não registrada some em silêncio. Todos foram para
  `buildings`, como nos mapas 4, 5 e 6. `decor_brazier_01`, que não existia em
  manifesto nenhum, virou `candle_stand_iron`.

## Arena: papel `ground_detail` passa a ser desenhado, e perímetro refeito

- **Bug de motor:** `MapRenderer` só desenhava os papéis `structures_back` e
  `foreground` dos manifestos. `ground_detail` existia no schema do manifesto e
  no `validate_manifest.py` desde sempre, mas nunca chegava à tela — todo objeto
  com esse papel sumia em silêncio. Foi o que aconteceu com os dez sigilos da
  arena. `drawVisibleLayers` e `drawLayersUpTo` agora chamam
  `drawManifests("ground_detail")` antes de `structures_back`, que é a ordem
  certa: peça pintada no chão fica acima do terreno e abaixo de jogador, chefe e
  decalque de aviso.
- **Perímetro do `world_07` refeito** (`work/tiled-assets/rework_arena_walls.py`):
  os trechos norte-sul passaram a usar `castle_walls_vertical_manifest`
  (`wall_v_cap_n`, `wall_v_middle_a/b`, `wall_v_pier`), como nos mapas 4, 5 e 6 —
  antes usavam `walls_wall_04`, que é parede de frente. As horizontais deixaram
  de ser espaçadas em 2 células fixas (o que abria vão entre peças de larguras
  diferentes) e passaram a emendar por `x + (w_atual + w_seguinte)/2 - 6`, a
  mesma regra do `world_04`. As colunas verticais foram alinhadas à **camada de
  colisão** (área andável x 256..3200), o que fechou 140 px de chão livre atrás
  da parede leste e tirou o jogador de dentro da parede oeste.
- **Poluição visual removida:** 6 `pillars_column_01` soltos, 2
  `statues_statue_01` ao lado do trono e as fileiras de vela do corredor
  (`candle_row_h` e `candle_row_v`).
- **Menu inicial redesenhado:** os acessos para criar e entrar em partidas
  agora usam um painel de fantasia com moldura, ícones, estados de hover e
  textos em português, mantendo as mesmas ações de rede local.
- **Fluxo de entrada unificado:** a seleção de herói e a busca de partidas
  passaram a usar cartões e controles no mesmo estilo do menu inicial, sem
  alterar a navegação, a escolha de classe ou a descoberta na rede local.
- **Clique do menu inicial corrigido:** os botões de criar e entrar em partida
  deixaram de compartilhar a margem de toque expandida; o alvo clicável agora
  coincide exatamente com cada botão desenhado.
- **Cartão de herói ampliado:** a seleção de personagem recebeu mais espaço
  vertical e a seta de navegação à esquerda foi corrigida para aparecer.
- **Caixa de diálogo redesenhada:** as falas passaram a usar um painel de
  pergaminho com placa de nome, retrato sobreposto e cenário mais escurecido,
  mantendo o congelamento e os controles de avanço existentes.
- **Retrato de diálogo solto na tela:** o orador deixou de ser desenhado dentro
  de uma moldura com borda dourada e passou a aparecer recortado sobre a cena,
  com o busto subindo acima da caixa e o painel cruzando por cima do quadril.
  Cada personagem ganhou uma arte própria de retrato (`portrait.png`, meio-corpo)
  além da arte de seleção (`reference.png`, corpo inteiro), as duas agora com
  fundo transparente e tela padronizada.
- **Card de seleção sem serrilhado:** as texturas de retrato e de seleção passaram
  a usar filtro bilinear. A arte entrava com 1536 px de altura e era desenhada com
  uns 340 no card; sem filtro, a redução era amostrada por vizinho mais próximo e
  fio de ouro, malha e cabelo serrilhavam. O slot e a moldura do card também
  ficaram fixos, então o cartão parou de mudar de forma a cada seta apertada.

## Som das magias

- **Cada habilidade ganhou efeito sonoro, ouvido só por quem está perto.** O
  disparo mora em `internal/skill`, no ponto em que o efeito nasce — o mesmo
  código que host e cliente já executam ao replicar a magia —, então nenhuma
  mensagem de rede foi criada: cada máquina mede a distância do próprio
  jogador (`sound.SetListener`) e decide o volume, com silêncio além de 1600
  unidades de mundo. `internal/sound` passou a ter um banco de vozes (várias
  cópias carregadas por arquivo, porque um `rl.Sound` toca um disparo por vez)
  e uma fila, porque `PlayAt` é chamada da goroutine de rede e toda chamada de
  áudio do raylib tem de ficar na thread do laço. Sons mais longos que a magia
  desvanecem no fim dela (`PlayAtFor`), e a Chuva de Meteoros tem intervalo
  mínimo entre estrondos para não virar ruído branco. Ver `doc/audio.md`.

- **Ataque básico com som, um por personagem.** O efeito virou propriedade do
  personagem (`CharacterDef.AttackSound`, ao lado de `AttackSpeed`) e é cortado
  com desvanecimento na cadência dele — todos os cinco arquivos são mais longos
  que o intervalo entre dois golpes, e o do Necromante tem 8 s para uma cadência
  de 0,77 s. A Paladina soa em `skill.NewSwordSweep`, que host e cliente já
  executam; os quatro personagens de projétil soam no host em `HandleAttack` e,
  no cliente, por dedução do instantâneo de projéteis (um id que não estava no
  anterior é um disparo novo), porque o ataque básico não tem evento de rede e
  criar um seria a mensagem mais frequente da partida.

## 2026-09-04 — Tela de Configurações: áudio por barramento, vídeo/controles/interface locais

Botão CONFIGURAÇÕES na tela inicial (`ui/main_menu.go`) e a tela em si
(`internal/ui/settings_screen.go` e companhia), alcançável também de dentro
da partida — ESC no desktop, um botão discreto no HUD no celular
(`game/settings_overlay.go`) — sem pausar a partida, exatamente como a
sobreposição de reconexão.

**Áudio.** `sound.Volume`/`sound.SFXVolume` deixaram de ser constantes: viraram
quatro barramentos atômicos (geral/música/habilidades/ataque, `sound/bus.go`),
cada um com mudo próprio que preserva o valor. Slider com trilha grossa,
salto ao tocar, degrau de 5%, prévia sonora ao SOLTAR (`sound.PlayUI`, sem
espacialização) e roda do mouse no desktop — matemática pura em
`ui/slider.go`, testada em `ui/slider_test.go` sem abrir janela.

**Novo pacote `internal/settings`**: modelo, clamp por campo, `Load`/`Save`
em JSON. Falha ao carregar (arquivo ausente, JSON corrompido, campo fora da
faixa) cai em silêncio no padrão — nunca trava o jogo. Persistência real só
no desktop (`os.UserConfigDir()`); no Android, sem um caminho gravável
confirmado neste ambiente, a configuração vale a sessão (documentado, não
fingido — ver `doc/settings.md`).

**Vídeo/controles/interface**: teto de quadros e tela cheia aplicam na hora;
VSync avisa que só vale na próxima abertura; o painel de desempenho (F3)
ganhou uma opção alcançável no celular; tamanho/opacidade dos botões de ação,
modo canhoto (espelha joystick E o ângulo das habilidades) e raio do
joystick no mobile; mira contínua vs. só-no-clique no desktop (o chevron de
mira nunca refletia o mouse antes desta mudança); escala do HUD (por ora só
a barra de vida) e indicador de mira alternável.

**Cortado desta rodada, documentado em `doc/settings.md`**: qualidade de
efeitos por partícula (exige ler `doc/skill_performance.md` antes),
remapear teclas (fase 2), joystick dinâmico, e mostrar nome/nomes dos
jogadores — o nome escolhido é só local por enquanto: a restrição
não-negociável do pedido ("Configuração nunca entra na rede") e o pedido de
"aparecer na descoberta de partidas" se contradizem, e a restrição venceu.

`gofmt -l -e ./internal/` não pôde ser executado nesta rodada (sem toolchain
Go neste ambiente de execução, mesmo remoto) — revisão manual linha a linha
no lugar; falta rodar de verdade na máquina do Gui antes de dar por
verificado, junto com o toque real no celular (lista em `doc/settings.md`).

## 2026-09-28 — Jogo online pela Oracle Always Free (relay proprio)

- Novo relay em Go: `internal/relay` (protocolo, servidor, cliente) e `cmd/relay`. So o app do jogo e necessario: o host publica a sala de saida, os amigos escolhem "ONLINE <nome>" na tela de entrar. Testes: `internal/relay/relay_test.go` e `internal/network/host_online_test.go`.
- `network.OnlineServer` (vazio = desligado), `dialGame` como ponto unico de discagem do cliente (connect e reconexao), `Host.StartOnline`, lista de salas online (`online_rooms.go`, `ui/discovery_entries.go`).
- `Host.acceptLoop` recebe o listener e para quando ele fecha (antes girava para sempre com erro depois de `Host.Close`).
- `infra/oci/legiao.tf`: VM Always Free em `sa-saopaulo-1` + bucket com o binario do relay; a VM baixa o binario a cada inicio do servico.
- Nao verificado ainda: `terraform apply` e uma partida real pela internet.

## 2026-09-28 — Jogo online: "CRIAR PARTIDA ONLINE", sala morre com o host, limites do gratuito

- Menu: "CRIAR PARTIDA" volta a ser so rede local; novo "CRIAR PARTIDA ONLINE" com nome de sala num teclado desenhado (`ui/onscreen_keyboard.go`, `ui/online_room_setup*.go`). A sala so e aberta no relay no inicio da partida, e uma falha aparece na tela.
- Relay `LEGIAO2`: sinal de vida PING/PONG ligado ao laco do jogo (`network.MarkFrame`), sala e jogadores derrubados em 45 s sem host, `ShutdownOnline` no fim do jogo; limites de salas, jogadores, conexoes, ociosidade e teto mensal de trafego; `STATS`; tag de versao do jogo na lista (`network.OnlineGameTag`).
- Lista de salas mostra codigo e ocupacao, sala cheia nao responde ao toque, subtitulo diz se o servidor esta fora do ar.
- `infra/oci/legiao.tf`: alerta de custo opcional (`TF_VAR_alert_email`), output `health_check`.
- Verificado: build de `./internal/...`, `./cmd/...`, testes de `internal/relay` (com -race) e os novos de `internal/network`. Pendente: partida real depois de publicar o binario novo do relay.

## 2026-09-28 — Online: endereco do relay fora do app, 10 amigos por sala, capacidade calculada

- O app nao tem mais o IP do relay: le `online.json` num bucket publico (`network.OnlineConfigURL`, `online_config.go`), com cache em memoria e em disco, e `"enabled"` como interruptor de emergencia. Testes em `online_config_test.go`.
- Relay: 10 amigos por sala, 24 no servidor (limite de banda da VM gratuita), 50 salas, 40 conexoes por IP. Conta em `doc/network.md`, "Capacidade do relay".
- `infra/oci/legiao.tf`: bucket `legiao-public` (leitura sem listagem) com `online.json` gerado do IP da VM, variavel `online_enabled`, output `online_config_url`.

## 2026-09-28 — Online: fila de envio por jogador, compressao negociada, mensagens de erro

- `host_peer_writer.go`: o host nao escreve mais na rede dentro do quadro; cada conexao tem fila (snapshot substitui snapshot, evento nunca some), lote com um Flush, e conexao travada por 5 s e fechada. Um jogador com rede ruim nao trava mais a partida dos outros.
- Compressao deflate (BestSpeed) negociada no `join` (`compress`), com marcador `compress` e troca de leitor no cliente (`client_inflate.go`). Medido: ~6,7x menos bytes, 57 us por tique por amigo fora da thread do jogo.
- `online_errors.go`: frases para cada falha online (criar, entrar, reconectar), dica na tela de reconexao, selo "SALA ONLINE #codigo" / "SERVIDOR ONLINE FORA" para o host (`ui/online_badge.go`). Relay devolve `RefusedError` com motivos em constantes.
- Testes: `host_peer_writer_test.go`, `online_errors_test.go`, relay (recusa tipada, `Connected`). Pendente: medir no celular real e depois subir `MaxPlayers` do relay.

## 2026-09-28 — Interpolacao consertada, ping no HUD, protecoes do host

- Bug antigo: a interpolacao nunca interpolava (dois snapshots de 50 ms de historia desenhados 100 ms atras = fator sempre 0; remotos andavam aos saltos de 20 Hz). Agora buffer de snapshots com relogio do host (`HostT`, `host_time.go`) e atraso adaptativo a oscilacao da rede (`interpolation_clock.go`). Testes com LAN e 4G simulados em `interpolation_test.go`.
- Ping: `ping`/`pong` a cada 2 s, "PING N ms" no HUD do cliente (`client_ping.go`, `ui.DrawNetworkBadge`).
- Host: linha maxima de 64 KiB, 120 mensagens/s por jogador, `MaxPartySize` 11, recusa de versao diferente com `join_rejected` e frase no menu (`host_guard.go`).
- Lista de salas online ordenada (com vaga primeiro, mais cheias antes).

## 2026-09-29 — Relay: teto de 24 para 64 amigos, taxa no STATS, tag net2

- Medido com gerador de carga: ~0,12% de nucleo e ~0,27 Mbps por amigo com fluxo comprimido. `MaxPlayers` 24 -> 64, `MaxConns` 256 -> 512 (`relay.NewServer`). `STATS` ganha `taxa_kbs`.
- `OnlineGameTag` "net1" -> "net2": builds sem compressao nao veem mais as salas novas.
- Exige publicar o binario novo do relay (`infra/oci/legiao-relay`, `terraform apply`, `update_relay`).

## 2026-09-29 — Gráficos: resolução que dá para ver, teste automático que mede a GPU

Relato do Gui num Galaxy S25 Ultra: trocar a resolução não mudava nada, e o
TESTAR escolheu Baixa (30 fps, sem cinzas) num aparelho de ponta.

- Resolução: a escolha só vale quando a janela nasce, e nada disso era
  visível — a aba só gravava no VOLTAR, sair pelo Home não reabre o processo
  e a tela não dizia em que tamanho o jogo desenhava. Agora a aba Gráficos
  grava a cada toque, a linha mostra o tamanho valendo e aparece REINICIAR
  AGORA quando ele difere do escolhido (`ui/render_restart.go`, pacote novo
  `internal/apprestart` — JNI, só Android).
- Teste automático: media o tempo do QUADRO (vsync de 120 Hz, simulação da
  partida atrás, a própria tela), então nem 540p "segurava 60". Passa a
  medir a cena entre duas cercas de GPU (`ui/perf_bench_gpu.go`), mediana,
  orçamento de 12 ms (60 fps) / 24 ms (30 fps); o resultado mostra os ms e
  cada resolução vai ao logcat (`[Graficos]`). `perf_bench.go` dividido em
  `perf_bench_result.go`.
- A conferir no aparelho: REINICIAR AGORA reabre sozinho; logcat mostra
  `Upscaling required` com o tamanho escolhido depois de reabrir.

## 2026-09-29 — REINICIAR E VOLTAR: reiniciar pela resolução não perde a partida

Pedido do Gui depois de testar o REINICIAR AGORA: quem troca a resolução no
meio da partida tem de poder voltar a ela.

- `internal/resume`: `resume.json` na pasta do app, gravado no clique e
  apagado ao ler (uso único, vale 4 min — abaixo dos 5 do slot do host).
- Cliente: volta com o MESMO `PlayerID` ao mesmo endereço; o host trata como
  reconexão e devolve posição, vida, recargas e mapa (`host_rejoin.go`).
- Host/solo: volta no começo da fase atual (`Host.StageMap()`), mesmo
  personagem e cor; a sala online reabre com o mesmo nome
  (`network.OnlineRoomName()`). A fase é aplicada antes do primeiro quadro
  em `playSession` — no laço, o diálogo do mapa 1 abriria antes da viagem.
- Host com outras pessoas (`network.OtherHumansPresent`) vê o aviso ao lado
  do botão. `ui.startHost` ganhou `startHostAs` (cor já decidida) e o menu
  ganhou `adoptLocalPlayerID`.

## 2026-09-29 — Volta online pelo mesmo código; HUD do mesmo tamanho em qualquer resolução

- Host online que reinicia: a sala voltava com código NOVO e os amigos
  ficavam 5 min reconectando a um código morto. `relay.HostAs` (o servidor já
  aceitava; só o cliente não pedia) + `network.OpenOnlineRoomAs`; o bilhete
  guarda código e segredo. Sem republicar o relay. Testes de ponta a ponta com
  relay local: `network/session_resume_test.go` (cliente reinicia e mantém o
  slot; host reinicia e reabre o mesmo código) e `relay.TestHostAsReclaimsCodeAfterRoomDied`.
- HUD "inchava" em 540p: números fixos de interface eram pixels da janela,
  que em 540p valem o dobro da tela. `ui.Px`/`SetRenderScale` (mesmo fator do
  zoom da câmera) em `TextSize`, no piso de fonte, na HUD da partida (vida,
  selo de rede, horda, chefe, setas das gárgulas, diálogo, vitória, game
  over, portal, reconexão), no `renderer.go` e nos tetos/pisos de menus e
  Configurações. Regra em `doc/coding_patterns.md`, "Pixels de interface".

## 2026-09-30 — Reorganização do repositório (branch feature/reorganizacao-repositorio)

- `plans/` esvaziada: todos os planos já executados foram removidos; as referências a eles nos docs de área saíram.
- `*_source.png` (43 folhas-fonte, ~88 MB) saíram de `assets/tilesets/` para `art/source/tilesets/`: continuam versionadas, mas não vão mais para dentro do APK. Skill `create-tiled-assets` e `doc/tilemap.md` apontam para o caminho novo.
- Removidos sem uso: pacote `internal/system/` (ninguém o importava), `Mago_attack_old.mp3`, `fireball-impact_old.mp3`, `fireball-impact_oold.mp3`, `arqueiro/portrait_old.png`.
- Deixaram de ser versionados (continuam no disco): `work/` (já estava no `.gitignore`) e os `liblegiao.so` gerados pelo build Android (fica um `.gitkeep` por ABI).
- `.gitignore`: `Claude outputs/`, `*.pptx`, `~$*`, `__pycache__/`.
- `doc/project_structure.md` reescrito com o layout atual (pacotes `bot`, `nav`, `relay`, `settings`, `sound`, `skill`…, `cmd/relay`, `infra/oci`, `art/source`).
