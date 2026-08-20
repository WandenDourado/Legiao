# Plano: o bot que avança lutando, e a gárgula de alcance global

> **Estado: implementado (fases A1–A3, B1–B3).** Pedido por Gui em
> 20/08/2026, depois de ver os bots atravessarem o mapa 3 até o ponto do
> clímax sem lutar. Duas partes independentes; a segunda depende de a
> primeira existir para ser jogável, mas o código não se cruza. **Fase C**
> (medição do mapa 7 em jogo e afinação de `sentryPosts`/`AttackCooldown`)
> continua em aberto — exige uma sessão jogada, não código. Ver
> `doc/changelog.md` para o que cada fase entregou de fato. **20/08/2026,
> depois:** jogando o mapa 3, apareceu uma quarta causa da causa 3 (§A2) — o
> atalho do portal em `tickOneBot` desligava o cérebro inteiro sempre que o
> portal estava ativo, e o `world_03` (mapa de guarnição, sem hordas) abre o
> portal desde o primeiro segundo. Corrigido — ver a causa 4 e `travelDest`
> abaixo.

---

# Parte A — Avançar lutando, recuar ferido

## A1. O sintoma

No mapa 3 os bots não brigam: eles marcham até o ponto do clímax, passando
pelos monstros. O desejado é que **caminhem em direção ao objetivo lutando pelo
caminho**, e **recuem quando estiverem com pouca vida**.

## A2. O diagnóstico — três causas somadas, todas verificadas

**1. O alvo não tem raio.** `mostThreateningFoe` e `nearestFoe`
(`internal/bot/steering.go`) varrem **o mapa inteiro**. O Arqueiro, com
`dist > arqueiroKeepRange`, faz `dest = target.Pos`; a Paladina interpola entre
o aliado mais frágil e o alvo. Ou seja: escolhido um monstro do outro lado do
mapa, o bot **atravessa o mapa** — e, no caminho, ignora tudo, porque só existe
um alvo por vez.

**2. `PartyCentre` inclui os próprios bots.** Em `buildBotView`
(`host_bot_tick.go`) o centro é a média de **todos** os vivos, bots inclusive.
Um bot que se adianta puxa o centro; os outros seguem o centro; o centro anda
mais. Um erro individual vira marcha do esquadrão — é isso que faz parecer que
"os bots foram embora juntos".

**3. Ninguém tem a noção de frente.** O objetivo do mapa (a zona de clímax, o
portal) pertence ao `game`, e o bot não o conhece — por desenho
(`doc/plan_bots_de_classe.md` §4). Então a única coisa que ele pode perseguir é
alguém. Quando esse alguém é um monstro distante, "avançar" e "atacar" viram a
mesma decisão errada.

**4. O atalho do portal desligava o cérebro.** Descoberta por Gui em
20/08/2026, jogando o mapa 3 depois das fases A1–A3 e B1–B3: os bots
atravessavam a guarnição inteira sem lutar e morriam no caminho.
`tickOneBot` (`internal/network/host_bot_tick.go`) tinha um atalho:
`if view.PortalActive { intent.Dest = view.Portal } else { intent =
rt.brain.Think(view) }`. **Com o portal ativo, o cérebro nunca era chamado**
— sem alvo, sem ataque, sem recuo, sem formação, todo o trabalho das fases
A1–A3 ficava desligado. O comentário que justificava isso ("o campo está
vazio quando um portal abre") é falso para mapa de guarnição:
`game.PortalsUnlocked()` devolve `true` com `WaveState.Total == 0`, e o
`world_03` não tem um único marcador `enemy_spawn_*` — a jogabilidade dele é
guarnição, `Waves` fica nil, e o portal fica aberto desde o primeiro segundo
com 83 monstros em campo. A geometria fecha o caso: `player_spawn` do
`world_03` está em y=8448 e `portal_portao_da_fortaleza` em y=1024, dentro do
clímax — "ir direto para o ponto do clímax" era literalmente "ir direto para
o portal".
*O conserto:* o atalho caiu, e o portal virou um DESTINO que o próprio
cérebro escolhe — `travelDest` (steering.go), consultado por todos os
cérebros, só devolve o portal com três condições juntas: `PortalActive`,
nenhum inimigo engajável por perto (`engageableFoes`), e `HumansAtPortal` —
pelo menos um humano vivo já dentro de um portal ou perto dele
(`PortalEscortRadius`, 1200 px). Sem essa terceira condição o bot voltaria a
marchar sozinho no instante em que o portal de um mapa sem hordas abre.
`finishMove` (steering.go) aplica o resultado: **sem `Push`** quando o portal
vence — separação é exatamente o que empurraria todo mundo para fora do
pequeno retângulo do portal (`doc/tilemap.md` "Quem entra no portal some e
espera"). A cadeia de destino de cada cérebro ficou: alvo engajado > recuo >
`travelDest` > posto de formação — e ir para o portal muda só o destino, não
o ataque: um alvo dentro do alcance útil da arma continua saindo mesmo
enquanto o bot caminha para a porta.

## A3. O desenho: o avanço é dos humanos; o bot escolta

Três regras, e nenhuma delas ensina ao bot o que é o clímax:

**R1 — A referência de avanço é o centro dos HUMANOS vivos.** `View` ganha
`HumanCentre rl.Vector2` e `HasHumans bool`, e todo destino de "seguir o grupo"
passa a usar isso em vez de `PartyCentre`. Sem humano vivo, o bot **segura a
posição e luta** em vez de escolher um novo líder — a partida, nesse ponto, está
sendo decidida pelo ressurgimento, não por deslocamento.
*Por quê:* corta a realimentação da causa 2 pela raiz. O grupo avança quando o
jogador avança, que é a definição de escolta.

**R2 — Raio de engajamento.** Um inimigo só é alvo se estiver a menos de
`engageRadius` (900 px) **do bot** ou **do centro humano**. Fora disso ele não
existe para a decisão. O bot marcha; quem entra no raio vira alvo; morto o
alvo, ele volta a marchar.
*Por quê:* é a diferença entre "há um monstro no mapa" e "há um monstro no meu
caminho". 900 px é pouco mais que o alcance do Arqueiro (600) e cobre a largura
de uma tela.

**R3 — Formação por classe em torno da frente humana.** O destino de marcha não
é o centro humano, é um ponto relativo a ele na direção do avanço:

| Classe | Posto |
|---|---|
| Paladina | 160 px **à frente** |
| Arqueiro / Mago | 250 px atrás, deslocados lateralmente |
| Necromante | 250 px atrás, do lado oposto |
| Sacerdotisa | 350 px atrás |

A direção do avanço é a velocidade média dos humanos vivos (suavizada); parado
o grupo, a formação usa a última direção conhecida.
*Por quê:* é o que faz "caminhar até o clímax lutando" acontecer sem o bot
saber o que é o clímax. E é o que impede a Sacerdotisa de chegar antes da
Paladina.

## A4. Recuar ferido

Regra compartilhada (`steering.go`), com **histerese**, porque um limiar seco
faz o bot entrar e sair do combate a cada tiro:

- entra em recuo abaixo de `retreatUnder` = **0,35** de vida;
- volta a engajar acima de `rejoinAbove` = **0,60**;
- **Paladina recua em 0,25**, e só depois de ter usado o Escudo: a linha da
  frente que recua cedo entrega o grupo.

Recuando, o destino é **atrás da linha** — o posto de formação dele deslocado
mais 300 px para trás, na direção oposta ao inimigo mais próximo. E o que ele
faz enquanto recua depende da arma:

- Arqueiro, Mago, Necromante e Sacerdotisa **continuam atirando** enquanto
  andam (é o que um jogador faz);
- a Paladina para de nadar contra a maré: sem golpe corpo a corpo enquanto
  recua, ou ela recua andando de costas e apanhando.

A Sacerdotisa não precisa de regra nova para socorrer: ela já mira o mais
ferido (`work/prompts/ajustes_sacerdotisa_e_portal.md`), e um bot recuando é o
mais ferido.

## A5. Uma correção que apareceu no caminho

**Não gastar cadência atirando no que não se alcança.** Hoje o Arqueiro pede
`HandleAttack` a qualquer distância; a flecha tem `Lifetime 1.6` a 700 de
velocidade — **1120 px de alcance real**. Todo tiro além disso é uma recarga
jogada fora. `tuning.go` ganha o alcance útil por classe e o ataque básico só
sai dentro dele.

---

# Parte B — Gárgula de alcance global, e a resposta do Arqueiro

## B1. O que muda

A gárgula (`entity.EnemyTypeCastleSentry`) passa a alcançar **o mapa inteiro**:
`AttackRange` deixa de ser 1900 e vira `SentryGlobalRange`, uma constante
nomeada maior que a diagonal do maior mapa (world_05 é 8192×11520 → diagonal
≈ 14.100; use **16000**).

**Não use `math.MaxFloat32`**: o teste é `dist > e.AttackRange+e.Radius`, e uma
soma com MaxFloat32 estoura para +Inf. Um número grande e explicado é mais
honesto que um infinito.

## B2. Sem isto, o resto é ruído

`SentryOrbSpeed` é **300** e `SentryOrbTTL` é **9 s** — a esfera percorre, no
melhor caso, **2700 px**. Uma gárgula com alcance global e esfera de 2700
dispara sem parar e nunca acerta ninguém do outro lado do mapa: o jogador vê
esferas roxas nascendo e morrendo no meio do caminho, e nada mais.

Então o alcance global exige duas mudanças na esfera:

**1. TTL calculado no nascimento**, e não constante:
`ttl = dist/SentryOrbSpeed * 1.5 + 2 s`, com teto de 40 s. O 1,5 paga a
perseguição (o alvo se move; a esfera curva). A **lentidão permanece de
propósito** — ela é o "tempo de ver, tempo de correr" que o comentário de
`host_sentry_orb.go` defende. De longe, a esfera vira uma inevitabilidade lenta
atravessando a tela, e isso é bom.

**2. Uma esfera por gárgula no ar.** Com cadência de 1,35 s e viagem de 20 s,
cada gárgula teria ~15 esferas perseguindo o mesmo jogador. A regra é: a
gárgula não dispara de novo enquanto a esfera dela estiver viva (acertou,
expirou ou o alvo morreu = pode disparar). É isso que transforma "chuva de
esferas" em "ameaça constante".

## B3. A consequência que precisa ser medida antes de comemorar

| Mapa | Gárgulas |
|---|---|
| world_04 | 4 (guarnição, nascem com o mapa) |
| world_05 | por horda |
| **world_07** | **10 postos**, alternando leste e oeste |

Com alcance global, as dez do mapa 7 batem sempre, de qualquer lugar. A resposta
desenhada é a suprema do Arqueiro: 2 flechas por ativação, 30 s de recarga, 40
de dano contra 40 de vida — **uma flecha por gárgula**. Dez gárgulas custam
~2,5 minutos só de torres.

Isso pode estar certo (a arena final é para ser dura) ou pode ser demais. O
botão de ajuste **não é a criatura**: é a lista de postos por mapa
(`sentryPosts`, `internal/network/sentries.go`) e o `AttackCooldown` da
`EnemyDef`. Meça o mapa 7 antes de mexer em qualquer um dos dois.

## B4. O Arqueiro bot precisa ENXERGAR a gárgula

Hoje ele não pode: `buildBotView` (`host_bot_tick.go`) **descarta**
`EnemyTypeCastleSentry` ao montar `View.Foes`. Para o bot, a gárgula não existe
— é por isso que a suprema nunca sai contra ela.

**A mudança:** a gárgula entra em `Foes` com `IsSentry: true` e com o **ponto de
acerto** (`HitCentre`, que é `e.HitCenter()` — a gárgula tem `HitOffsetY -67`,
então o pé não serve como mira).

**A regra de prioridade do Arqueiro, acima de tudo o que ele faz:**

> Suprema pronta **e** gárgula viva ⇒ matar gárgula.

E quatro detalhes que a implementação tem de respeitar, ou a regra não funciona:

1. **Alcance da flecha é 4800** (`CelestialRange`). Gárgula mais longe que isso
   ⇒ o bot **anda até ~4200** antes de lançar. Lançar de 6000 é gastar a
   ativação numa flecha que morre no caminho.
2. **São duas cargas** (`CelestialCharges 2`, recarga só depois da segunda). Se
   duas gárgulas estiverem quase alinhadas a partir dele, uma flecha resolve as
   duas — ela perfura e 40 mata cada uma. Senão, uma flecha por gárgula.
3. **Mire o `HitCentre`.** A flecha testa contra `e.HitCenter()`
   (`celestial_manager.go`); mirar no pé ainda acerta por folga de raio, mas é
   folga, não desenho.
4. **Nenhum bot pode gastar cadência numa gárgula.** Projétil comum é recusado
   contra ela (`host.go`, `checkProjectileCollisions`) e o dano por proximidade
   também (`checkEnemyPlayerCollisions`). Logo: `IsSentry` sai de toda seleção
   de alvo — de todas as classes — **exceto** da suprema do Arqueiro. Um Mago
   bot bombardeando uma estátua invulnerável é o defeito espelho deste.

Enquanto a suprema não estiver liberada pela campanha (o Arqueiro só a ganha a
partir do mapa 5 — `game/progression.go`), a regra simplesmente não dispara, e
está certo: nos mapas anteriores a gárgula é problema do grupo, não dele.

---

# Fases

| Fase | Entrega | Critério de aceite |
|---|---|---|
| A1 | `HumanCentre`/`HasHumans` na View + todos os destinos de "seguir" migrados. | Um bot adiantado não puxa mais os outros; sem humano vivo, os bots seguram posição. |
| A2 | Raio de engajamento + formação por classe. | No mapa 3, com o host andando para o norte, os bots avançam junto, engajam a guarnição que entra no raio e não somem no horizonte. |
| A3 | Recuo com histerese + alcance útil do ataque básico. | Bot abaixo de 35% sai da linha, volta acima de 60%, e ninguém atira além do alcance real da arma. |
| B1 | Alcance global da gárgula + TTL por esfera + uma esfera no ar por gárgula. | No mapa 4, a gárgula acerta do outro lado do mapa, e há no máximo quatro esferas em campo. |
| B2 | Gárgula visível na View (`IsSentry`, `HitCentre`) e ignorada por todas as classes. | Nenhum bot ataca gárgula com ataque básico ou magia comum. |
| B3 | Regra da suprema do Arqueiro. | No mapa 5, o Arqueiro bot aproxima-se e derruba as gárgulas assim que a suprema fica pronta. |
| C | Medição do mapa 7. | Tempo de limpeza das dez torres e dano recebido registrados; só então afinar `sentryPosts`/`AttackCooldown`. |

# Testes

- **`bot`** (puros, sem janela): alvo fora do `engageRadius` é ignorado; posto
  de formação por classe; recuo entra em 0,35 e só sai em 0,60; Arqueiro escolhe
  gárgula quando a suprema está pronta e a ignora quando não está; nenhuma
  classe mira `IsSentry` no ataque básico; alvo além do alcance útil não vira
  ataque.
- **`network`**: `HumanCentre` ignora bots e mortos; `buildBotView` inclui
  gárgula com `IsSentry` e `HitCentre`; uma esfera por gárgula; TTL cresce com a
  distância.
- **`entity`/`skill`**: gárgula com alcance global escolhe o jogador mais fraco
  do mapa; esfera lançada a 6000 px chega ao alvo.

# Contrato de execução

1. `AGENTS.md` + `doc/coding_patterns.md` antes de escrever; `git status
   --short` antes e depois.
2. `internal/bot` continua sem importar `network`: tudo o que a regra nova
   precisa (centro humano, `IsSentry`, ponto de acerto, alcance da suprema)
   chega pela `View`.
3. Um arquivo, uma responsabilidade; dividir antes de ~150 linhas. Nada em
   `loop.go`.
4. Por fase: `gofmt -l .` (vazio), `go vet ./...`, `go build ./...`,
   `go test ./...` e uma rodada no desktop (F8 para chegar ao mapa da fase).
5. Documentar: `doc/plan_bots_de_classe.md` (§5, o comportamento por classe),
   `doc/combat_rules.md` (a gárgula e o alcance global), `doc/performance.md`
   se a medição do mapa 7 mudar algum número, e **uma linha em
   `doc/changelog.md` por fase**.
