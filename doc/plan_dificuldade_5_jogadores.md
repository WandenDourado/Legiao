# Plano de dificuldade para o grupo de cinco

> **Estado: implementado em 12/08/2026.** Este documento fica como o *porquê* de
> cada número. Os valores em vigor estão no código (`network/wave_runs.go`,
> `network/garrisons.go`, `network/sentries.go`, `game/progression.go`) e o
> resumo operacional em `doc/combat_rules.md`; quando os dois divergirem, o
> código é a verdade e este arquivo é a intenção.
>
> Duas diferenças em relação ao plano original, ambas decididas na
> implementação: o mapa 4 ficou com **quatro** gárgulas (duas por ilha, e não
> duas ao todo), e as do mapa 5 ficaram em **chão pisável dentro do oval** em vez
> de sobre as estátuas decorativas — a seção 7 explica.

Planejamento, não implementação. Cada seção diz **o que muda, quanto, e por
quê** — o "por quê" importa mais que o número, porque os números vão ser
reafinados em jogo e a razão é o que impede a reafinação de desfazer o desenho.

---

## 1. O diagnóstico: por que está fácil

### As tabelas foram escritas para um jogador, não para cinco

Não existe nenhum lugar no código onde a quantidade de monstros dependa de
quantos jogadores estão na sala. `WaveDef.Composition`, `WaveDef.MaxConcurrent`
e `GarrisonSquad` são números fixos. Um grupo de cinco enfrenta exatamente a
mesma fase que um jogador sozinho — com cinco vezes o dano e cinco alvos para o
monstro dividir a atenção.

**Decisão tomada:** as tabelas passam a ser escritas **para cinco**. Sem fator
dinâmico. Testar sozinho fica inviável e isso é aceito; o F8 continua sendo a
ferramenta de quem quer atravessar a campanha para validar outra coisa.

### A conta que mede o tamanho do problema

Dano por segundo do grupo, somando o elenco pelos números que já estão no
código (`ArrowDamage 15` a 2,5/s, `SwordDamage 30` a 1,5/s, bola de fogo 25 mais
20/s de chão a 1,2/s, mais o crânio e o projétil da Sacerdotisa):
**algo entre 200 e 250 de dano por segundo**, sustentado.

Vida total que cada fase põe em campo hoje:

| Fase | Monstros | Vida somada | Tempo de dano puro para cinco |
|---|---|---|---|
| Mapa 1 | 53 | 4.280 | ~20 s |
| Mapa 2 | 78 | 6.200 | ~28 s |
| Mapa 3 (guarnição) | 83 | 20.500 | ~90 s |
| Mapa 4 (guarnição + clímax) | 55 | 12.900 | ~57 s |
| Mapa 5 (4 hordas finitas) | 133 | 17.900 | ~80 s |

O mapa 1 inteiro é **vinte segundos de combate real**. Não é uma fase, é uma
introdução que ninguém pediu para durar tão pouco. O mapa 3 e o 5 seguram
melhor porque têm orc — 600 de vida cada um — e é isso que aponta para onde a
correção tem de ir: **massa fraca não custa tempo de cinco jogadores**.

### O teto de simultâneos é o botão de dificuldade; a composição é o de duração

`MaxConcurrent` decide quanta pressão chega ao mesmo tempo; `Composition`
decide quanto tempo a horda dura. Subir só a composição faz a fase ficar
**longa**, não difícil — o grupo mata na mesma velocidade, por mais tempo.
Praticamente todo o ajuste abaixo sobe os dois, mas o teto sobe primeiro.

Referência para ler os tetos propostos: com cinco jogadores, `MaxConcurrent 25`
é **cinco monstros por cabeça** se todos engajarem — que na prática nunca
acontece, porque o grupo se espalha e a matilha escolhe o mais perto.

---

## 2. A régua muda a cada suprema liberada

Este é o eixo do plano. O que era "difícil" numa fase vira "fácil" na seguinte
**não porque os monstros mudaram, mas porque o grupo ganhou uma resposta**.

| Fase | Quem já tem a suprema | O que essa suprema apaga | Logo, a fase precisa de |
|---|---|---|---|
| Mapa 1 | ninguém | — | volume e cerco puros |
| Mapa 2 | ninguém | — | desgaste antes do clímax |
| Mapa 3 | Necromante | massa fraca: slime e lobo em quantidade | **orc** nos vãos |
| Mapa 4 | + Sacerdotisa | o desgaste: ela segura o grupo vivo contra muita coisa forte | **dano de fora do altar** — gárgula |
| Mapa 5 | + Arqueiro | a gárgula, e qualquer coisa a longa distância | **tudo junto**, e em quantidade |

As três contas que sustentam a tabela, todas tiradas do código:

- **Legião Espectral** (`skill/specter.go`): 30 espectros, cada um vale ~4
  lobos, recarga de 60 s. Um lançamento apaga uma matilha inteira. Mas
  `entity/orc_legion_test.go` prova que **cinco orcs limpam a legião** — o orc
  é o antídoto declarado, e está testado.
- **Área Angelical** (`skill/angelic.go`): raio 720, cura 20/s por 12 s,
  ressuscita, recarga de 60 s. Ela responde a *tudo que está dentro do raio*.
  Não responde ao que bate **de fora** dele.
- **Flechas Celestiais** (`skill/celestial.go`): alcance 4.800, perfura, 40 de
  dano — e a gárgula (`castle_sentry`) tem 40 de vida. Uma flecha, uma gárgula.
  `checkProjectileCollisions` recusa **todo projétil comum** contra ela; espada
  e espectro passam. Alcance de ataque dela: **1.900** — mais que o dobro do
  raio do altar da Sacerdotisa.

A gárgula é, literalmente, o monstro desenhado para não ter resposta antes do
mapa 5. É por isso que ela é o instrumento do mapa 4.

### Correção de pré-requisito: a liberação hoje está errada

`UltimatesUnlockedOn` (`internal/game/stage_skip.go`) libera **todas** as
supremas a partir do mapa 2, para todo mundo. O desenho da campanha é outro: o
personagem ganha a dele **na fase seguinte à cena em que ele se ergue**
(`lastStandHeroes`). Sem essa correção, a tabela acima não descreve o jogo — o
grupo chega ao mapa 3 com Área Angelical e Flechas Celestiais na mão, e toda a
progressão de dificuldade abaixo perde o sentido.

Proposta: a função vira `UltimateUnlockedFor(char, mapPath)`, lendo uma tabela
por mapa, e `SetUltimatesUnlocked` passa a carregar o conjunto liberado em vez
de um booleano.

| Mapa | Supremas disponíveis |
|---|---|
| world_01 | nenhuma |
| world_02 | nenhuma (a cena do Necromante acontece aqui) |
| world_03 | Necromante |
| world_04 | Necromante, Sacerdotisa |
| world_05 | Necromante, Sacerdotisa, Arqueiro |

Paladina e Mago ficam de fora: as cenas deles são o mapa 5 e o que vier depois.
Se isso deixar os dois sem suprema pela campanha inteira, é uma decisão de
roteiro a tomar — mas é uma decisão, não um efeito colateral.

---

## 3. Mapa 1 — a vila

**Régua:** ninguém tem suprema. A dificuldade aqui é **volume e cerco**, e nada
mais. Não entra orc: ele é a criatura da fortaleza e aparecer aqui adianta o
mapa 3 sem nenhum ganho.

| Horda | Hoje | Proposto |
|---|---|---|
| 1 | 8 slime · teto 4 · 3,0 s · 2 | **24 slime · teto 12 · 2,4 s · 4** |
| 2 | 12 slime + 5 lobo · teto 7 · 2,5 s · 2 | **30 slime + 18 lobo · teto 18 · 1,9 s · 5** |
| 3 | 16 slime + 12 lobo · teto 12 · 2,0 s · 3 | **36 slime + 30 lobo · teto 26 · 1,5 s · 6** |

53 monstros e 4.280 de vida viram **138 monstros e 10.920** — de vinte segundos
de dano puro para cinquenta e um, e com o cerco três vezes mais denso, que é o
que faz a diferença ser sentida em vez de contada.

A forma da rampa não muda e é de propósito: a horda 1 ainda ensina o slime
sozinho, a 2 ainda apresenta o lobo como acento, a 3 ainda inverte a proporção.
O que muda é a escala de cada degrau.

O mapa tem 15 marcadores `enemy_spawn_*`, o suficiente para a bússola de oito
setores distribuir levas de 6 sem repetir o lado.

---

## 4. Mapa 2 — a mata sombria

**Régua:** ninguém tem suprema, e a horda 2 é a cena que **precisa derrubar o
grupo**. Com cinco jogadores, derrubar ficou mais difícil — e a resposta óbvia
(subir a matilha de 35 para 60 simultâneos) esbarra no joelho de desempenho que
o próprio código documenta.

**A resposta escolhida é outra: desgaste.** Entra uma horda intermediária cuja
função é fazer o grupo chegar à matilha **sem recargas e sem vida cheia**.
Assim a queda acontece por acúmulo, e não por um teto de simultâneos que o
motor pode não aguentar.

| Horda | Hoje | Proposto |
|---|---|---|
| 1 | 10 slime · teto 5 · 3,0 s · 2 | **28 slime + 10 lobo · teto 16 · 2,2 s · 4** |
| 2 — *nova* | — | **20 slime + 28 lobo · teto 22 · 1,7 s · 5** — "a mata se fecha" |
| 3 (a matilha) | 8 slime + 60 lobo · teto 35 · 1,2 s · 5 | **12 slime + 84 lobo · teto 42 · 1,0 s · 6** |

O 42 é o **número a medir, não um número fechado**. O comentário em
`wave_runs.go` diz que 35 foi medido com 30 espectros em campo varrendo a lista
de inimigos por quadro. Se o quadro cair, baixar **aqui primeiro** — 38, depois
35 — e compensar com `BatchSize` e intervalo. Nunca compensar cortando a
composição: o tamanho da matilha é o que a cena exige.

---

## 5. Mapa 3 — a defesa de território

**Régua:** o Necromante tem a Legião. Ela apaga slime e lobo em quantidade —
e isso não é um problema a corrigir, é a função dela. O que precisa mudar é
**onde está o preço da fase**: se o preço for massa fraca, um lançamento a cada
60 s compra o mapa inteiro. O preço tem de ser **orc**.

Hoje: 22 orcs, 44 lobos, 17 slimes — 83 monstros.
Proposto: **36 orcs, 70 lobos, 30 slimes — 136 monstros.**

| Linha | Hoje | Proposto | Por quê |
|---|---|---|---|
| Setor mata | 4 slime + 2 lobo | 6 slime + 4 lobo | abertura, continua sendo abertura |
| **Vão A** | 2 orc + 2 lobo | **4 orc + 4 lobo** | a primeira mordida tem de custar |
| Setor trilha | 6 lobo + 2 slime | 10 lobo + 4 slime | combustível da legião, travessia em movimento |
| **Vão B** | 3 orc + 3 lobo | **5 orc + 5 lobo** | |
| Setor corte | 10 lobo + 5 slime | 15 lobo + 8 slime | campo aberto, matilha |
| **Vão C1 (caro)** | 5 orc + 4 lobo | **7 orc + 6 lobo** | **sete orcs. Cinco limpam a legião** (`orc_legion_test.go`) — o vão caro continua caro *mesmo depois da suprema*, que é a única forma de a escolha entre os dois vãos continuar existindo |
| **Vão C2 (barato)** | 3 orc + 3 lobo | **4 orc + 5 lobo** | o desvio pelo mapa continua sendo a opção barata, mas deixa de ser gratuita |
| Setor pátio | 2 orc + 10 lobo + 6 slime | 6 orc + 14 lobo + 9 slime | |
| **Vão D** | 6 orc + 4 lobo | **10 orc + 6 lobo** | a boca da fortaleza é a linha mais cara do trajeto e permanece sendo |

**Clímax — a emboscada do portão** (`world03Climax`, `Endless`, é a cena da
Sacerdotisa). Ele agora tem de levar **cinco** jogadores abaixo de 25% de vida,
e o grupo chega com a Legião na mão.

| | Hoje | Proposto |
|---|---|---|
| Reposição | 6 orc + 6 lobo | **8 orc + 8 lobo** |
| Teto | 12 | **18** |
| Leva | 4 a cada 1,2 s | **5 a cada 1,0 s** |

`Endless` é o que faz isso funcionar sem exagero: a Legião não **encerra** a
horda, só compra sessenta segundos. Depois deles a pressão volta inteira, e o
grupo cai — que é o que a cena espera.

---

## 6. Mapa 4 — o castelo

**Régua:** o grupo tem Legião e Área Angelical. A Sacerdotisa cura 20/s num
raio de 720 por 12 segundos e ressuscita os caídos. Enfileirar mais orcs contra
isso é enfileirar mais do que ela foi feita para segurar.

**O que ela não responde é o que vem de fora do raio.** É esse o instrumento
desta fase, e ele já existe no jogo: a **gárgula** (`castle_sentry`), alcance
1.900 — mais que o dobro do altar —, imune a todo projétil comum, ancorada em
ilha inalcançável. E é exatamente por isso que a cena desta fase é a do
Arqueiro: `castCastleJudgment` já percorre **todas** as sentinelas ativas, então
aumentar o número delas não exige nenhuma mudança na cena.

### 6.1 As gárgulas entram na fase, não só no clímax

Hoje as duas sentinelas nascem dentro de `StartClimax`. Proposta: elas passam a
ser **guarnição**, em campo desde o portão se fechar atrás do grupo — a fase
inteira vira um corredor sob fogo, em vez de um corredor de corpo a corpo com
uma surpresa no fim.

Número proposto: **4** (duas por ilha). O mapa só declara dois marcadores
`climax_spawn_stream_*`; **os outros dois precisam ser criados** no builder em
`work/castle-map4`, e as ilhas de 4x4 comportam.

### 6.2 Guarnição: 37 → 67

| Faixa | Hoje | Proposto |
|---|---|---|
| Vestíbulo | 4 slime | **8 slime** |
| Corredor | 4 slime + 4 lobo | **6 slime + 8 lobo** |
| Boca do saguão | 2 orc + 6 lobo | **4 orc + 10 lobo** |
| Antessala | 4 orc + 4 lobo + 3 slime | **6 orc + 8 lobo + 5 slime** |
| Pé da escadaria | 4 orc + 2 lobo | **8 orc + 4 lobo** |

A regra que já está escrita em `garrisons.go` continua valendo e é boa: **não
alivia depois do clímax**. As duas faixas de cima seguem sendo as mais caras.

### 6.3 Clímax: o mais fraco de toda a campanha

`world04Climax` hoje é 10 slime + 8 lobo com teto 8, e não é `Endless`. É a
horda mais fraca do jogo inteiro — numa fase onde o grupo tem **duas** supremas
e cuja cena é uma das quatro do roteiro. Ela não derruba cinco jogadores; nem
derrubaria um.

| | Hoje | Proposto |
|---|---|---|
| Composição | 10 slime + 8 lobo | **8 orc + 12 lobo + 8 slime** (reposição) |
| `Endless` | não | **sim** |
| Teto | 8 | **22** |
| Leva | 3 a cada 1,3 s | **5 a cada 1,1 s** |
| Fogo de fora | — | **as 4 gárgulas, durante o clímax inteiro** |

O contrato de `Endless` é o mesmo do mapa 3: quem encerra é `LastStandDone()`.
**Verificar antes de implementar** que o portão do mapa 4 depende de `cleared`
como o do 3 — se não depender, `Endless` tranca a fase.

---

## 7. Mapa 5 — o salão da senhora

**Régua:** o grupo tem as três. É a última fase e pode usar tudo ao mesmo
tempo. As gárgulas entram aqui **pelo contexto**, como você pediu: não na
primeira horda, e sim quando a fase já se declarou.

| Horda | Hoje | Proposto | Gárgulas |
|---|---|---|---|
| 1 | 10 slime · teto 5 | **30 slime + 10 lobo · teto 18 · 2,2 s · 4** | — |
| 2 | 15 slime + 8 lobo · teto 8 | **30 slime + 26 lobo · teto 22 · 1,8 s · 5** | — |
| 3 | 20 slime + 15 lobo · teto 12 | **20 slime + 32 lobo + 8 orc · teto 26 · 1,5 s · 5** | **2** |
| 4 | 25s + 20l + 10 orc · teto 18 | **16 slime + 30 lobo + 16 orc · teto 30 · 1,2 s · 6** | **4** |
| 5 (`Endless`) | 20s + 20l + 20 orc · teto 35 | **12 slime + 24 lobo + 20 orc · teto 38 · 1,0 s · 6** | **6** |

A curva de composição inverte de propósito: começa em slime e termina em orc.
A horda 1 é o momento do Necromante — ele apaga trinta slimes e paga com
sessenta segundos de recarga, que é a horda 2 inteira. A horda 5 é quase toda
orc porque é a única coisa que a Legião não compra.

Teto 38 na horda 5 pelo mesmo motivo do mapa 2: é o número a **medir**.

### O que precisa ser construído para as gárgulas do mapa 5

Elas não cabem no sistema atual. Hoje sentinela só nasce dentro de
`StartClimax`, com nome de marcador escrito no código, e só para o mapa 4.

Proposta, seguindo a filosofia que o repositório já aplicou a `waveRuns`,
`garrisons` e `lastStandHeroes` — **a fase declara, o sistema lê**:

- `WaveDef` ganha `Sentries int`: quantas sentinelas essa horda põe em campo.
- Um `sentryPosts` novo (arquivo `internal/network/sentries.go`), mapa → lista
  de marcadores `enemy_sentry_*`, na ordem em que devem ser ocupados.
- O `WaveRunner` instancia as sentinelas em `startWave`, **nos postos
  declarados** — nunca pelo anel de distância, pela mesma razão que a emboscada
  usa `Ambush`: quem escolheu onde elas ficam foi quem desenhou o mapa.
- **Não** entram em `Composition` nem em `waveTypeOrder`. Elas são cenário
  armado, não fila de spawn.

O mapa precisa de **6 marcadores `enemy_sentry_*`** na camada `spawn`, no
perímetro do oval. As três gárgulas decorativas já desenhadas — (5555, 4453),
(3008, 4721), (2363, 4117) — são os três primeiros postos naturais; os outros
três saem do builder em `work/castle-assets`.

**Duas consequências a aceitar de propósito:**

1. No mapa 5 a gárgula fica em **chão pisável**, ao contrário das ilhas do mapa
   4. Espada da Paladina e espectros a matam; só projétil não passa. Isso é
   melhor que torná-la intocável: sem ninguém no Arqueiro a fase ainda fecha,
   mas alguém tem de atravessar o salão sob fogo para calá-la. Com o Arqueiro,
   são duas flechas por lançamento a cada 30 s, e a decisão vira "guardar a
   suprema para as gárgulas ou gastá-la na matilha".
2. A horda só termina com `aliveCount == 0`, então **gárgula viva segura a
   horda**. É o objetivo, não um efeito colateral: ela deixa de ser um
   incômodo e vira objetivo.

---

## 8. Ordem de implementação e riscos

1. **Liberação de suprema por personagem** — pré-requisito de tudo. Sem ela a
   régua da seção 2 não descreve o jogo.
2. **Mapas 1 e 2** — só tabela, risco baixo. É onde a diferença aparece primeiro
   num teste com o grupo.
3. **Mapa 3** — só tabela (`garrisons.go` e `world03Climax`).
4. **Mapa 4** — tabela mais duas mudanças de verdade: gárgulas viram guarnição,
   e o clímax vira `Endless`. **Verificar o gate do portal antes.**
5. **Mapa 5** — tabela mais o mecanismo de sentinela por horda, mais os
   marcadores no builder do mapa. É o item mais caro; deixar por último.

**Riscos nomeados:**

- **`waveTypeOrder` é uma armadilha conhecida.** Tipo usado numa composição e
  ausente da lista faz `pendingTotal()` nunca zerar e a horda **nunca terminar**
  — foi o que aconteceu com o orc. As composições propostas usam só os três
  tipos já listados, e `TestWaveTypeOrderCoversEveryComposition` cobre.
- **Desempenho** nos tetos de 42 (mapa 2) e 38 (mapa 5). Medir com F3; baixar o
  teto antes de qualquer outra coisa.
- **`Endless` sem gate de fim tranca a fase.** Vale para o clímax novo do mapa 4.
- **Não há toolchain Go neste ambiente**: `go build` e `go test` vão precisar
  ser rodados na sua máquina depois de cada etapa.
