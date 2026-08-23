# Changelog

## 2026-08-23 (2) — A linha do setor virou escudo do jogador

**Relato:** *"os jogadores estao conseguindo matar os monstros de longe, sem
precisar entrar no posto"* (mapa 3).

**Causa:** `Guard.covers` exigia o jogador DENTRO do retangulo do setor para o
guarda sequer olhar. As faixas do mapa 3 tem **1280 px** de altura e a flecha
alcanca **1120**: dava para ficar na faixa de tras, atirar por cima da linha e
limpar o posto seguinte um a um. O raio de visao de 2600 nao ajudava — ele nem
chegava a ser perguntado, porque a pergunta do setor vinha antes e ja devolvia
"nao".

**Correcao — nao foi mexer no raio nem apagar o setor**, foi acrescentar uma
SEGUNDA porta de aquisicao, mais curta que a primeira:

| Porta | Pergunta | Raio | Respeita o retangulo? |
|---|---|---|---|
| Setor | "entrou no pedaco que eu guardo, e eu o vejo?" | 2600 (3400 no orc) | sim |
| **Ameaca** | "consegue me **acertar** daqui?" | ~1370 | **nao** |

Mais restritiva em distancia, mais permissiva em geometria — as duas coisas
juntas sao o ponto. A acumulacao faixa por faixa continua valendo (um guarda nao
acorda por alguem longe noutra faixa), mas **um posto nao deixa de ser defendido
porque quem atira ficou do lado de fora**.

O raio sai de `entity.LongestAttackReach()` — velocidade x tempo de vida do
projetil, nao um numero escrito a mao. Para isso, dois numeros cravados dentro
de `NewProjectile` viraram constantes com nome: `ProjectileLifetime` (2,0 s) e
`ArrowLifetime` (1,6 s).

**E levar dano virou notar.** `Enemy.TakeDamage` marca o guarda como engajado,
qualquer que seja a origem do golpe. A porta de ameaca cobre a geometria
previsivel; esta e a rede embaixo dela — magia de area, flecha celestial, um
angulo que ninguem previu. Fica no `TakeDamage` porque ele e o FUNIL por onde
todo dano a inimigo passa (dez pontos de chamada, verificados), e porque um
caminho novo de dano nao pode depender de alguem lembrar de avisar a IA.

### A armadilha que quase entrou junto

O raio de ameaca e uma **variavel de pacote**, e os personagens entram no
registro por um `init()`. Go inicializa toda variavel de pacote **antes** de
rodar qualquer `init()`, entao a primeira versao — que varria `AllCharacters()`
para achar o maior alcance — teria calculado sobre um registro vazio: alcance 0,
raio de 250 px, e o defeito de volta **em silencio**, sem nada acusar.

`LongestAttackReach` percorre as **constantes** dos projeteis. Quem cobra que o
elenco caiba nelas e um teste, que roda depois do `init()`.

### Um teste mudou de lado

`TestGuardIgnoresTrespassersOutsideItsSector` afirmava que "quem esta do outro
lado da barricada nao e problema deste monstro" a 600 px do posto. Essa frase
deixou de ser verdadeira de proposito: ela agora exige o alvo fora do setor **e**
fora do alcance. O caso que ela cobria virou o teste irmao,
`TestGuardDefendsItsPostAgainstSomeoneShootingFromOutsideTheSector`.

## 2026-08-23 — O portal abria antes da emboscada; o ultimo suspiro perdeu os NPCs

Nove ajustes pedidos em jogo. Tres deles sao a mesma historia contada de tres
lugares, entao vale comecar por ela.

### A fase 3 travava, e o portal era a causa

**Sintoma:** o grupo chegava a fortaleza e a horda infinita nao comecava.

**Causa:** o `world_03` nao tem um unico marcador `enemy_spawn_*` — de
proposito, a jogabilidade dele e de guarnicao. Entao `WaveState.Total` fica em
zero, e `game.PortalsUnlocked` lia isso como "mapa quieto, nao tranque a
saida": **o portal se materializava no primeiro quadro da fase**. Bastava um
jogador entrar nele para a fase parar de vez — quem espera dentro de um portal
e congelado e nem desenhado (`host_portal_presence.go`), e a porta do climax
exige TODOS os vivos dentro da zona da fortaleza. Aquele corpo nunca chegava, a
emboscada nunca armava, e nada na tela dizia por que.

**Correcao, em dois lugares porque sao dois defeitos:**

- `network/climax_pending.go` (novo): enquanto um mapa ainda DEVE a emboscada
  roteirizada (`climaxRuns`), ele nao e um mapa quieto — e um mapa cuja luta
  ainda nao comecou, e o portal fica trancado como em qualquer corrida de
  hordas. Instalada a emboscada, `WaveState.Total > 0` e a regra normal volta a
  valer sozinha; por isso o cliente nao precisa de mensagem nova de protocolo —
  ele descobre o mapa no carregamento igual ao host.
- `game/climax_gate.go`: quem esta dentro de um portal nao conta para a porta
  do climax, nem para segurar nem para abrir. Hoje e cinto e suspensorio, mas a
  porta nao deve depender de o portal estar fechado para funcionar.

### O personagem nao sumia no portal e o aviso piscava

Mesmo tema, outro defeito. `game/loop.go` republicava o jogador local a cada
quadro com `network.UpdatePlayerState`, que **substitui a entrada inteira** —
apagando `InPortal` e `Absent`, que sao veredictos do host e chegam no snapshot
a 20 Hz. Sessenta quadros por segundo apagando, vinte por segundo repondo: o
corpo continuava desenhado e "Aguardando o grupo" piscava.

Agora existe `network.UpdateLocalPlayerState`, que preserva os campos que a
maquina local apenas espelha, e `network.LeaveLocalPortal`, que o ESC/SAIR usa
para limpar a flag **e** o espelho de uma vez (so a flag seria reposta no quadro
seguinte).

### Os NPCs do ultimo suspiro sairam

O "heroi invocado" existia para um caso que nao existe mais: ninguem no grupo
jogando com a classe da fase. Desde que toda classe vaga virou um BOT
(`host_bots.go`), a cena sempre encontra um corpo de verdade para reerguer.

Foram embora `summonHeroNPC`, `tickLastStand`, `LastStandNPC`,
`game/last_stand_npc.go` e cinco campos de `lastStandHero` (`npcID`, `cast`,
`alive`, `anchor`, `endSignal`) que so serviam para um personagem invocado ser
dono de um efeito. O que a fase ainda declara e QUEM se ergue e QUAL suprema
devolver carregada.

Junto foi o **julgamento celestial do mapa 4** (`castCastleJudgment`), que era o
outro item do relato: mesmo com um Arqueiro em campo, a cena disparava as
flechas nas sentinelas sozinha. Agora nao — a cena reergue o Arqueiro (humano ou
bot), devolve a suprema carregada e destravada, e **quem atira e ele**. O bot ja
sabe cacar gargula (`bot/arqueiro.go`). O julgamento do mapa 6
(`castCannonJudgment`) FICOU: o Avatar dos Deuses e imunidade total, nao um
ataque a distancia, entao sem ele o resgate devolveria o grupo vivo dentro do
mesmo corredor bombardeado.

### As duas flechas do Arqueiro iam para a mesma torre

A suprema dele tem duas cargas e a recarga so arma depois da segunda
(`ability.Charged`), entao entre um disparo e o outro `UltimateReady` continua
verdadeiro — o bot reavaliava "qual e a gargula mais perto", achava a MESMA (a
flecha ainda estava no ar) e gastava a segunda carga nela. Duas flechas, uma
torre, quando uma flecha ja resolve (40 de dano perfurante contra 40 de vida).

`arqueiroBrain` ganhou uma memoria de 3 s por alvo — o voo maximo de uma flecha
(4800 de alcance a 1600/s). Passado esse tempo, um alvo ainda de pe quer dizer
que ela errou, e ai ele pode voltar a mirar nele. Um tiro que atravessa DUAS
torres alinhadas marca as duas.

### As gargulas do mapa 4 nao atiram mais desde o vestibulo

`network/sentry_wake.go` (novo): a fase declara a partir de qual degrau de
territorio (`tilemap.Zone.Tier`) as sentinelas dela abrem fogo. O mapa 4 declara
o **degrau 3**, a boca do saguao — que e onde a emboscada arma e onde o anuncio
"As sentinelas despertaram" acontece. Antes disso elas estao em campo, sao
desenhadas e podem morrer; so nao atiram.

Uma vez acordadas, nao dormem mais (mesmo principio de `Enemy.chasing`): recuar
nao pode ser uma forma de desligar a fase. Um corpo caido la na frente tambem
nao acorda ninguem — a fase reage a quem esta avancando. Mapa fora da tabela
atira desde o primeiro quadro, que e o certo para os mapas 5 e 7, onde as
gargulas entram por horda e a fase ja escolheu o momento delas.

### Numeros

| O que | De | Para | Por que |
|---|---|---|---|
| `MeteorImpactDamage` | 100 | **480** | Uma pedra tira 80% da vida do Orc (600). Os 100 vinham de quando o inimigo mais duro do jogo tinha 100 de vida: desde o orc de guarnicao a chuva passava por cima do elenco pesado sem arranhar. Duas acertam de morte; uma sozinha nunca resolve. |
| `MeteorRainInterval` | 0,025 s | **0,015 s** | ~40/s para ~67/s. A chuva e a unica suprema que nao escolhe alvo — sorteia pontos no mapa inteiro —, entao o que ela controla de verdade nao e quanto cada pedra tira, e sim quantas caem perto de alguem. **Custa quadro**: os simultaneos sobem de ~56 para ~93, cada um com o proprio emissor de particulas. Se o F3 acusar, e este numero que afrouxa primeiro, nao o dano. |
| Sentinela `AttackDamage` | 14 | **25** | Quatro esferas derrubam um personagem (100 de vida), em vez de oito. A uma esfera por vez, oito acertos era uma ameaca que o grupo absorvia andando. |
| Senhora das Trevas `Health` | 400 | **2000** | Ela deixou de ser o chefe no fim de uma corrida de cinco hordas e virou quem comanda a ARENA: a vida dela nao e um bolo de dificuldade, e o CRONOMETRO da fase (a corrida so para quando ela cai). Com 400, o cerco final acabava antes de a fase mostrar do que e capaz. |

As tres relacoes acima estao travadas em `network/balance_test.go`: um numero de
equilibrio que so significa alguma coisa em relacao a outro numero precisa de um
teste que defenda a RELACAO, nao o valor.

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
