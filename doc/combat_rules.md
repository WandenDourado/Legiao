# Combat Rules

Cadence, cooldowns, death, revival, Game Over and the test mode. Everything
here is decided by the host; clients display it.

## Attack Speed

The basic attack has a cadence: a character can only land `AttackSpeed` hits
per second, so holding the fire button or tapping faster changes nothing.

| Character | Attacks/s | Interval |
|---|---|---|
| Mago | 1.2 | 0.83s |
| Sacerdotisa | 1.4 | 0.71s |
| Paladina | 1.5 | 0.67s |
| Necromante | 1.3 | 0.77s |
| Arqueiro | 2.5 | 0.40s |

`AttackSpeed` is a field of `entity.CharacterDef`, so tuning a character's
rhythm is one line in `entity.RegisterCharacter` — no combat code changes.
`CharacterDef.AttackInterval()` converts it to seconds; a character declaring
`0` has no cadence limit.

The gate itself is `Host.beginAttackCooldown`
(`internal/network/host_attack_gate.go`), consulted by `Host.HandleAttack`, so
it applies identically to the host's own attacks and to every `MsgAttack` a
client sends.

## Skill Cooldowns

Each skill declares `Cooldown()` in the ability registry, gated by
`Host.beginSkillCooldown` (`internal/network/host_skill_gate.go`). Skills with
charges (`ability.Charged`) only arm the cooldown after the last charge.

A skill returning `0` has **no gate at all** — the host lets it through every
frame. `internal/ability/registry_test.go` fails on any skill that does, which
is how the Mago's fireball is kept from shipping without a recharge again.

| Character | Q | R |
|---|---|---|
| Mago | Bola de Fogo 6s | Chuva de Meteoros 60s |
| Sacerdotisa | Santuario 14s | Area Angelical 60s |
| Arqueiro | Rajada de Flechas 6s | Flechas Celestiais 30s (2 charges) |
| Paladina | Escudo Sagrado 6s | Avatar dos Deuses 60s |
| Necromante | Cemiterio 12s | Legiao Espectral 60s |

Each value is one constant in `internal/skill/` (`FireballCooldown`,
`SanctuaryCooldown`, ...), so tuning is a one-line change.

The counters are drawn from the host's own timers: `Host.broadcastCooldowns`
ships a `cooldown` message every simulation tick, and both roles read it back
through the shared store in `internal/network/cooldowns.go`. Nothing counts
down locally, so a client's counter can never disagree with the host that will
refuse the cast.

| Platform | Where the counter is drawn |
|---|---|
| Android | On top of the Q and R buttons (`ui.SkillButton.Draw`), plus a dimming ring on the fire button for the attack cadence. |
| Desktop | Pip bar at the bottom of the screen, one per bound ability (`ui.DrawSkillBar`, fed by `game.DrawSkillCooldownBar`). Slots are spaced by whatever is wider, the pips or the skill names, so long names like "Chuva de Meteoros" cannot print over their neighbour. |

## Death And Revival

A dead player stays on the field: the body is drawn from the same sprite sheet
through a grayscale shader (`internal/entity/death_tint.go`,
`assets/shaders/grayscale_*.fs`), frozen on the frame and direction it died
facing. It does not move, attack or cast.

| Rule | Value |
|---|---|
| Revive delay | `entity.RespawnDelay` = 30s |
| Health on revive | `entity.RespawnHealthPercent` = 30% |
| Revive position | Where the player fell |

The body gets up where it fell: `tickRespawns` never touches X/Y. A dead player
stops sending input, so the position the host still holds is the spot where it
died. The map spawn is only used by the stage reset, which is a fresh run.

The countdown is armed by `Host.markPlayerDead` and ticked by
`Host.tickRespawns` (`internal/network/host_death.go`). It rides to the clients
in `PlayerState.RespawnIn`, and `game.SyncLocalPlayer`
(`internal/game/death.go`) mirrors the verdict onto the local player entity —
which is why the loop only has to ask `p.IsDead` before moving it.

Timers freeze during a Game Over: reviving into a finished run would quietly
undo it.

## Game Over And Stage Reset

The run ends when every player is dead (`checkGameOver`). The host announces it
once and `Host.UpdateSimulation` returns immediately from then on, so waves,
enemies and skill effects all stop — the world the players see is the one they
lost in.

Only the host can restart, with **F5** or the button on the Game Over overlay
(`ui.GameOverPanel`); everyone else is told they are waiting on the host.
`Host.ResetStage` (`internal/network/host_reset.go`) clears the field and the
skill manager, rewinds the wave runner to wave 1, drops every cooldown, and
puts all players back on the spawn at full health. Clients mirror it from the
`reset_stage` message.

## Corrida de hordas

**A corrida pertence ao mapa**, não ao pacote. `internal/network/wave_runs.go`
tem uma entrada por mapa, com a chave sendo o caminho a partir da raiz — a
mesma que `World.Path` e `campaignMaps` usam. `Host.StartWaveRun(path, points)`
monta o runner com a corrida daquele mapa.

Isso já foi uma variável de pacote, e vazava: qualquer mapa carregado rodava as
três hordas do `world_01`. O `world_02` herdava a proporção de slime e lobo da
vila e anunciava "os lobos chegaram" na fase errada.

| Mapa | Hordas | Teto de simultâneos |
|---|---|---|
| `world_01` | 24 slimes → 30 slimes + 18 lobos → 36 slimes + 30 lobos | 12 / 18 / 26 |
| `world_02` | 28 slimes + 10 lobos → 20 slimes + 28 lobos → 12 slimes + **84 lobos** (**Endless**) | 16 / 22 / 42 |
| `world_05` | 40 → 56 → 60 + 2 gárgulas → 62 + 4 → **Endless** + 6 | 18 / 22 / 26 / 30 / 38 |

No `world_02` e no `world_05`, a última horda — a que abre a janela do
clímax — só se repõe até o resgate. Depois de `LastStandDone()`, os inimigos
que já estão em campo ficam finitos; derrotar o último muda a corrida para
`cleared` e libera a saída.

A matilha do `world_02` era finita, e com cinco jogadores isso virou defeito em
jogo: o grupo **limpava** a horda 3 antes de cair aos 30% de vida, a corrida
terminava e o Necromante nunca se erguia — a fase entregava o portal sem
entregar a suprema. Uma horda finita, por maior que seja, é sempre uma questão
de aguentar; é a reposição que faz do resgate a única saída. Com `Endless`, a
composição de 12 slimes + 84 lobos deixa de ser um total e passa a ser a
**mistura de reposição**: ela volta inteira para a fila cada vez que esvazia, e
`MaxConcurrent` continua sendo o teto de pressão — o ritmo não muda, o fim é que
não chega. `TestClimaxWaveWaitsForTheRescue`
(`internal/network/wave_runs_test.go`) guarda isso para todo mapa cuja janela
seja `ClimaxWindowWaveIndex`.

**As tabelas são escritas para CINCO jogadores.** Não existe fator dinâmico:
nenhum número de horda ou de guarnição depende de quantos estão na sala. Isso é
uma decisão, não um esquecimento — testar sozinho fica inviável, e a ferramenta
para isso é o F8 (e o F2 para as supremas).

O que decide **dificuldade** é `MaxConcurrent`; a composição decide **duração**.
Subir só a composição faz a fase ficar longa, não difícil: o grupo mata na mesma
velocidade, por mais tempo. Por isso o teto sobe primeiro em toda a tabela.

A régua muda a cada suprema liberada, e é ela que explica a composição de cada
fase — do mapa 3 em diante a massa fraca deixa de ser preço e vira combustível
da Legião, então o preço migra para o orc; do mapa 4 em diante o corpo a corpo
deixa de ser preço porque a Área Angelical o responde, então o preço migra para
a gárgula, que bate de fora do raio dela. O planejamento completo está em
`doc/plan_dificuldade_5_jogadores.md`.

Mapa sem entrada na tabela fica **quieto** (`WaveState.Total` 0, portal
liberado), que é o que um mapa de validação de terreno quer. Mapa com
marcadores `enemy_spawn_*` e sem corrida quase sempre é esquecimento, então
`StartWaveRun` registra um aviso.

`SpawnInterval` zero solta um lote **por quadro** — `wave_runs_test.go` reprova.

### Os inimigos

| | Vida | Velocidade | Dano | Cadência | Alcance |
|---|---|---|---|---|---|
| Slime | 100 | 100 | 10 | 1,0 s | 25 |
| Lobo | 40 | 240 | 18 | 0,7 s | 30 |
| Orc | 600 | 130 | 30 | 1,5 s | 70 |
| Gárgula | 40 | **0** | 14 | 1,35 s | **1900** |

A gárgula (`castle_sentry`) é a única criatura que não se move e a única que
`checkProjectileCollisions` recusa machucar com **projétil comum** — espada e
espectro passam. Alcance 1900 contra os 720 de raio da Área Angelical: ela é o
monstro desenhado para bater de **fora** da resposta que o grupo tem, e é por
isso que a fase seguinte à que a introduz é a que entrega as Flechas Celestiais
(40 de dano perfurante contra os 40 de vida dela).

Onde ela aparece está em `network/sentries.go`, e as duas portas de entrada são
diferentes porque os mapas são: no `world_04`, travessia de território, ela é
**guarnição** e nasce com o mapa; no `world_05`, corrida de hordas, ela entra
**por horda** (`WaveDef.Sentries`, um total acumulado) e nunca na primeira.

O posto ocupado **não é reocupado**: o cursor conta postos usados, não gárgulas
vivas, então o que o jogador derruba fica derrubado.

O lobo é desenhado para ser **a presa certa da Legião Espectral e a ameaça
errada para o jogador**: pouca vida, muita velocidade, muito dano. A 240 ele
passa o jogador (200) e alcança — era 180, "escapável por pouco", que é o
oposto do que a matilha do mapa 2 precisa ser.

**Vida e dano do lobo se afinam juntos.** O duelo que eles decidem:

| | Tempo para matar |
|---|---|
| Espectro sobre o lobo | 40 de vida ÷ 61 dano/s = **0,65 s** |
| Lobo sobre o espectro | 60 de vida, 18 por golpe a cada 0,7 s = **2,63 s** |

Cada espectro vale cerca de **quatro lobos**, então os trinta dão conta da
matilha de sessenta — que é o que a cena do mapa 2 exige da ultimate.

Os números vieram de jogar: com 35 de vida e 28 dano/s o espectro ganhava por
0,23 s e levava um lobo junto, então a legião se gastava na metade da matilha.
A identidade da skill sobrevive ao buff porque ela é uma **razão**, não um
número: muito dano por segundo contra pouca vida. Contra um alvo de 300 de vida
o espectro ainda entrega só ~160 de dano antes de cair.

Vida e dano do lobo continuam se afinando **juntos**: baixar a vida ajuda o
espectro, subir o dano atrapalha.

## Ultimates: ganhas por fase

O grupo **começa a campanha sem as supremas** e ganha as suas ao passar de
fase — **uma por personagem, não todas de uma vez**. A regra é uma frase: *o
personagem ganha a dele na fase seguinte à cena em que ele se ergue.*

`game.UltimatesGrantedOn(mapPath)` não é uma tabela; ele **deriva** o conjunto
das duas listas que já existem: `campaignMaps` (a ordem das fases) e
`lastStandHeroes` (quem se ergue em cada uma). Uma fase nova entra na lista e
herda a regra sozinha.

| Mapa | Supremas disponíveis |
|---|---|
| `world_01` | nenhuma |
| `world_02` | nenhuma (a cena do Necromante é aqui) |
| `world_03` | Necromante |
| `world_04` | Necromante, Sacerdotisa |
| `world_05` | Necromante, Sacerdotisa, Arqueiro |
| `world_06` | Necromante, Sacerdotisa, Arqueiro, Mago |

Isto já foi um booleano — "a partir do mapa 2, todo mundo tem a sua". Com ele o
grupo chegava ao `world_03` com as três supremas na mão, e a dificuldade de cada
fase, que é calibrada **contra** as supremas que existem ali, media outra coisa.
Pior: o herói da fase chegava com a ultimate que a cena dela existe para
entregar, o que esvazia o resgate.

O gate é do **host** (`Host.skillUnlocked`), consultado antes do cooldown para
que um lançamento recusado não queime uma carga. Cliente que desenhar o botão
mesmo assim recebe o não.

**F2 libera tudo para aquele jogador**, o que é o que torna uma fase adiantada
testável desde o mapa 1 — junto com o F8, que leva até ela.

Mapa fora da campanha (sandbox, teste de terreno) conta como liberado: travar
um mapa por onde ninguém progride só atrapalharia quem está experimentando.

**A exceção é o resgate do próprio último suspiro**, que concede a suprema do
herói da fase por cima desta tabela — só para a corrida em jogo, sem alterar o
que `UltimatesGrantedOn` diz para a fase seguinte. Ver "O resgate" abaixo.

## O último suspiro (`on_last_stand`)

O terceiro gatilho de diálogo, e o único que dispara **durante** a luta. Ele
existe para o momento em que um mapa é desenhado para ser perdido, de modo que
a cena aconteça em vez do Game Over.

Condição (`game.partyIsFalling`): a **janela do clímax do mapa está aberta** E
ninguém mais de pé, ou todo mundo que ainda está de pé abaixo de **30% da
vida** (`dialogue.LastStandHealth`). As duas condições, sempre — uma sem a
outra não é o último suspiro.

### A janela do clímax é declarada por mapa, não avaliada solta

`internal/network/climax_window.go` (`climaxWindows`) é a tabela — mesmo
padrão de `waveRuns`, `climaxRuns` e `lastStandHeroes`: a FASE declara a
janela, e `game.partyIsFalling` só lê a declaração através de
`network.ClimaxWindowOpen`.

Isto corrige um defeito real: antes, a única guarda de MOMENTO era
`WaveState.Total > 0` e fase de luta, que **qualquer horda** de um mapa com
corrida satisfaz. Bastava o grupo cair de vida na horda 1 do mapa 2 ou do mapa
5 para a cena tocar ali — bem antes da horda que a fase de fato desenha como
clímax.

| Mapa | Natureza da janela | Quando abre |
|---|---|---|
| `world_01` | nenhuma | nunca (não tem cena de clímax) |
| `world_02` | por horda (`ClimaxWindowWaveIndex`) | a partir da horda 3 de 3 (a matilha) |
| `world_03` | emboscada (`ClimaxWindowAmbush`) | enquanto a corrida da emboscada da fortaleza está no ar |
| `world_04` | emboscada (`ClimaxWindowAmbush`) | enquanto a corrida da emboscada do salão está no ar |
| `world_05` | por horda (`ClimaxWindowWaveIndex`) | a partir da horda 5 de 5 (a última) |
| `world_06` | checkpoint (`ClimaxWindowCheckpoint`) | quando o grupo alcança a zona `corridor_checkpoint` |

Nos mapas de emboscada (`world_03`, `world_04`) a janela reaproveita o mesmo
`WaveState` que os mapas de horda usam, mas com uma diferença que a torna
segura: esses dois mapas **não têm entrada em `waveRuns`**, só em
`climaxRuns` (ver "Defesa de território" abaixo), então a única fonte capaz de
deixar `WaveState.Total` sair de zero ali é a própria emboscada roteirizada —
"a corrida está no ar" já significa "o clímax está no ar".

Um mapa com roteiro `on_last_stand` e **sem** entrada em `climaxWindows` é
esquecimento, não silêncio proposital: o diretor de diálogo registra um aviso
no log (o mesmo cuidado que `StartWaveRun` já tem para um mapa com marcadores
`enemy_spawn_*` e sem corrida) — roteiro que nunca toca, em silêncio, é o pior
defeito que esta cena pode ter.

O checkpoint do mapa 6 continua sendo lido do mesmo jeito de antes —
`Zone.Contains` lê a posição guardada do jogador mesmo morto, de propósito: um
corpo que caiu dentro do checkpoint ainda conta como "alcançou o ponto", que é
o "... e estejam com uma certa quantidade de vida ou mortos" do pedido
original.

Três guardas que continuam valendo e não são decoração:

- **A janela tem de estar aberta.** É a tabela acima; sem ela nada mais
  importa.
- **Grupo vazio não é grupo caindo.** Antes do primeiro estado de jogador
  chegar a lista está vazia, e "todos abaixo de 30%" é verdade sobre ninguém —
  a cena tocaria no primeiro quadro.
- **Não é zero de propósito.** Esperar o grupo inteiro morrer poria a cena
  depois do Game Over que o host anuncia, e um resgate que chega depois do fim
  não é resgate.

## Defesa de território (mapa 3)

O mapa 3 **não é uma corrida de hordas**. Não há onda que chega: o monstro já
está em campo quando o grupo entra, e ele **pertence a um lugar**. A fase é
uma travessia, e o que a estrutura são as barricadas.

### As barricadas são a regra, não o cenário

Cada barricada é uma **linha que atravessa o mapa inteiro**, com um ou dois
vãos de 4 células:

```text
#######----#################----##
#----#############################
##########----##########----######
```

Os vãos ficam em colunas **diferentes** de uma linha para a outra, alternando
de lado. É isso que impede subir reto pela trilha e obriga a atravessar o mapa
na largura — e é o que dá ao defensor um lugar certo para esperar.

Isso já esteve errado das duas maneiras opostas, e as duas foram pegas por
medição e não pelo olho:

- **Peça solta não é barricada.** A primeira versão pôs três peças atravessadas
  na trilha; o jogador contorna sem perceber que havia algo ali.
- **Vão de 3 células é vão fechado.** A peça vizinha avança o footprint dela
  para dentro do vão, e o `audit_layout.py` mediu 49% do mapa alcançável com o
  portal do lado de fora. Quatro células, e o footprint da barricada passou a
  ser o vão das **estacas** (384 px) em vez da caixa com as travessas (452):
  as pontas que sobram são altura, não chão.

### Quem defende: orc no vão, lobo e slime no campo

| Onde | Quem | Por quê |
|---|---|---|
| **Vão de barricada** | Orc de guarnição (+ 1–2 lobos) | O vão é uma decisão. Enfrentar um orc custa mais do que dar a volta pelo mapa até a outra passagem — quando há outra. |
| **Campo entre as linhas** | Lobo e slime | Travessia: coisa rápida e frágil, que o grupo atravessa lutando em movimento. |

Composição por vão, do sul para o norte — `internal/network/garrisons.go`:

| Barricada | Vão | Orcs |
|---|---|---|
| A (linha 50) | leste | 4 |
| B (linha 40) | oeste | 5 |
| C (linha 30) | **oeste** | **7** |
| C (linha 30) | leste | 4 |
| D (linha 20) | centro | 10 |

**Os sete do vão C-oeste não são um número redondo qualquer:** cinco é o que
derrota a Legião Espectral, e passar desse número é o ponto. Esta é a primeira
fase em que o Necromante **tem** a suprema, e um vão que a ultimate compra
deixaria de ser uma decisão. A linha de dois vãos existe para propor a
escolha — pagar ali, ou atravessar o mapa inteiro até a passagem de quatro.

Ao todo o mapa põe **136 monstros** em campo: 36 orcs, 70 lobos, 30 slimes.

O preço da fase está deliberadamente no **orc**, e não na massa. A Legião apaga
slime e lobo em quantidade — é a função dela — então um mapa cujo preço fosse
massa fraca seria comprado por um lançamento a cada 60 s. Com o preço no orc, a
massa fraca vira o combustível da ultimate em vez do obstáculo.

### O orc é a contrapartida da Legião Espectral

O alvo: **cinco orcs limpam os trinta espectros**, morrendo ou não.

A primeira coisa que a conta mostra é que **a vida não é a alavanca**. Trinta
espectros somam 30 × 11 ÷ 0,18 = **1 833 de dano por segundo**; com os stats
provisórios (220 de vida, 30 de dano, 1,8 s) os cinco orcs caíam em **1,1 s sem
matar um único espectro**.

A segunda é que **o dano tem um teto, e ele é baixo**. Em `StepLegions` o
inimigo revida contra **cada espectro engajado**, cada um no próprio
`hurtTimer` — e esse timer nasce **zerado**, então o primeiro revide sai no
quadro em que o espectro encosta. Com dano ≥ 60 (a vida cheia de um espectro)
todo espectro morre ao engajar e **um orc sozinho limpa a legião em quatro
quadros**. Isso não cumpre o alvo: destrói a ultimate.

| | Valor | Por quê |
|---|---|---|
| `AttackDamage` | **30** | Dois golpes por espectro. Não pode chegar a 60. |
| `AttackCooldown` | **1,5 s** | Os dois golpes saem em 1,5 s. |
| `Health` | **600** | ~366 de dano por segundo em cada um dos cinco; 600 dura 1,6 s e eles precisam de 1,5. A margem é de décimos, e é ela que faz cinco ser o número. |
| `Speed` | 130 | Abaixo dos 200 do jogador, **de propósito**: atravessar o vão precisa continuar sendo melhor do que brigar. Um perseguidor que alcança transforma defesa de território em horda. |

Medido: **1, 2, 3 e 4 orcs morrem sem arranhar a legião** (30 de pé);
**5 limpam em 1,5 s**. O degrau é em cinco e não há meio-termo.

O dano continua em 30 — o mesmo de antes — então o orc **não** ficou mais letal
contra o jogador do que já era. A versão anterior desta seção dizia o contrário
porque a simulação supunha um cleave de alvos fixos em vez do revide-contra-
todos que o código faz; corrigido o modelo, o custo desapareceu junto.

`internal/entity/orc_legion_test.go` guarda a conta com quatro verificações:
cinco vencem, **quatro perdem** (senão o número perde o sentido), um perde, e o
dano fica **abaixo** da vida do espectro.

### Território: setor **e** raio

| Camada | O que é | Onde vive |
|---|---|---|
| **Setor** | A faixa entre duas barricadas. Cinco deles, `tier` 1 a 5 do sul para o norte. É a unidade de **dificuldade**: quanto mais ao norte, mais guarnição. | retângulos `territorio_*` na camada `zones` |
| **Raio** | O alcance individual de cada guarnição em volta do próprio posto. | `enemy_post_*` na camada `spawn` |

O monstro persegue quem entra no território dele. **Fora dele, não.**

O setor de cada posto **não** é repetido no nome: sai da geometria, do
retângulo que contém o ponto. Guardar o mesmo dado em dois lugares é como as
duas metades divergem.

### Patrulha, e não sentinela

Cada monstro tem um **trecho** de 260 px centrado onde nasceu, e caminha entre
as duas pontas com 1,4 s de pausa em cada uma. Não é enfeite: a primeira versão
mandava o monstro "voltar ao posto", e o posto era um ponto exato. A separação
entre vizinhos empurrava, a distância passava da tolerância, ele corrigia um
passo, era empurrado de novo — cinco monstros no mesmo posto viraram um enxame
vibrando parado. O trecho próprio, a tolerância de 48 px e a pausa nas pontas
são as três coisas que trocam o tremor por vai-e-vem.

### Notar é a única pergunta. Perseguir não termina.

| | Valor | |
|---|---|---|
| Raio de **visão** | 2600 px, piso por tipo | Foi 640, 1100 e 1700, e as três vezes ficou curto pelo mesmo erro: tratar a distância entre barricadas (1280 px) como teto. Esse teto não existe — quem impede o monstro de reagir a quem está do outro lado é o **setor**, não a distância. As faixas do mapa 3 têm 1280 px de altura, então 2600 acorda o posto quando o grupo entra na faixa e não depois de ele passar. |
| Visão do **orc** | 3400 px | O mais lento do elenco (130) contra um jogador a 200. Notar junto com o lobo (240) e nunca alcançar ninguém lia como distraído, não como pesado. `EnemyDef.Vision` é um **piso**: o mapa pode declarar mais, nunca menos. |
| Fim da perseguição | **não existe** | Sem prazo, sem coleira, sem linha de setor. A única saída é não sobrar ninguém vivo para perseguir. |
| Sair do setor | irrelevante depois de notado | O setor decide **quem ele nota**, e nada mais. |

**Por que o teto saiu.** O jogador anda a 200 e o orc a 130, então *qualquer*
teto — prazo, distância do posto, divisa de setor — era uma saída gratuita:
bastava recuar alguns passos e a guarnição inteira soltava o alvo no mesmo
quadro. Havia um prazo de 5 s renovado a cada golpe e uma coleira de 2,6 × a
visão, e em jogo os dois se resolviam andando para trás.

**O que impede o mapa inteiro de acordar** não é mais um teto, é a geometria: o
setor limita quantos *notam* de uma vez, então o grupo acumula perseguidores
faixa por faixa conforme sobe, em vez de puxar 156 monstros numa fila só.

### Recuar tem preço: a retaguarda

Com a perseguição permanente, o buraco passou a ser o mapa: todo posto olhava
para o **norte**, porque a fase é uma subida, e atrás do jogador não havia
ninguém. Fugir para trás era saída garantida.

Os dois mapas de território ganharam postos de **retaguarda** — nas costas do
ponto de entrada, no trecho entre a mata e a primeira barricada, e na faixa logo
ao sul de cada linha vencida. Eles não adensam vão nenhum: cobrem os vazios por
onde se recua. Composição pequena e quase toda de **lobo**, porque a 240 ele é o
único que alcança quem está correndo.

| Mapa | Postos | Monstros |
|---|---|---|
| `world_03` | 24 → **32** | 136 → **156** |
| `world_04` | 12 → **18** | 67 → **80** |

O miolo do saguão do mapa 4 fica de fora de propósito: é onde os 22 simultâneos
da emboscada circulam, e enchê-lo tira o chão da cena do Arqueiro.

### Desengajar: **sem** curar

O monstro que perde o alvo (porque não sobrou ninguém vivo) volta a patrulhar
onde estiver, e **mantém o dano que levou**. Recuar e voltar é estratégia
legítima de desgaste, e o mapa fica mais fácil a cada tentativa de propósito.

### O clímax não se vence matando o último

A emboscada da fortaleza **se repõe**: a composição volta inteira para a fila
toda vez que esvazia, e ela só para quando `LastStandDone()` — ou seja, quando
a Sacerdotisa ergue o altar. É a tradução em código de uma frase do briefing:
*o clímax tem de ser impossível de passar sem a ultimate dela.*

Uma horda finita, por maior que fosse, seria uma questão de aguentar — bastaria
matar o último. Repondo-se, ela não tem último.

| | |
|---|---|
| Mistura de reposição | 6 orcs + 6 lobos |
| Onde nasce | **nos marcadores `climax_spawn_*`**, sem filtro de distância |
| Simultâneos | 10 |
| Fim | `LastStandDone()`, e nada mais |

O orc é a massa porque é a criatura mais forte, e é a única cujos números foram
calibrados contra uma ultimate. Os lobos existem porque um cerco só de orcs, a
130 de velocidade, deixaria o grupo circular indefinidamente — e circular não
pode ser uma saída.

**Depois do resgate a horda vira finita.** A Área Angelical não mata ninguém:
ela ressuscita os caídos uma vez e cura quem está dentro. O que a ultimate faz
é parar a reposição — o que restou em campo passa a ter fim, e aí sim matar o
último encerra a fase. A ultimate vira o jogo, que é o que uma ultimate deve
fazer.

`TestClimaxIsUnwinnableWithoutTheRescue` guarda essa propriedade. Ela é fácil
de perder sem querer: basta alguém trocar `Endless` por um total maior achando
que está subindo a dificuldade, e a diferença entre as duas coisas não aparece
em nenhum outro lugar do código.

### O gatilho do clímax é uma porta, não um limiar de vida

No mapa 2 o clímax dispara quando o grupo está caindo. **Aqui não.** Ele arma
quando **todos os jogadores vivos estão dentro da zona `fortaleza`** — o
retângulo de 44×11 células à frente do portão. Chegar é a condição.

- Morrer no caminho não abre nada. O resgate é do clímax, não da travessia.
- Um jogador morto volta pela regra normal (30 s, 30% da vida) e o grupo
  espera; o portão não abre sozinho.
- **Grupo inteiro morto antes da fortaleza é Game Over**, como em qualquer
  fase. O último suspiro pertence à luta que acontece *depois* de o portão ser
  alcançado.

Só então vem a emboscada, e só dentro dela o `on_last_stand` volta a valer com
a Sacerdotisa (ver a tabela de heróis acima).

## O resgate (`host_last_stand.go`)

Quando a cena do último suspiro fecha a última linha, o resgate acontece. Ele
tem duas formas, e quem decide é o grupo:

| Se… | Então |
|---|---|
| Alguém joga com o herói da fase | Volta em pé com **vida cheia** e 2 s de imunidade, e **lança a ultimate por conta própria** |
| Ninguém joga | Um **NPC** daquele personagem aparece no centro do grupo e lança ali mesmo |

A cena entrega o momento ao jogador; ela não joga por ele. O NPC existe porque
o resgate não pode depender das escolhas de personagem do grupo.

### Quem se ergue é da FASE (`last_stand_heroes.go`)

| Mapa | Herói | Ultimate | Sustentada? |
|---|---|---|---|
| `world_02` | Necromante | Legião Espectral | sim |
| `world_03` | Sacerdotisa | Área Angelical | não |
| `world_04` | Arqueiro | Flechas Celestiais (julgamento) | não |
| `world_05` | Mago | Chuva de Meteoros | sim |
| `world_06` | Paladina | Avatar dos Deuses | sim |
| qualquer outro | Necromante | Legião Espectral | sim (padrão) |

Isto era o Necromante escrito no código em oito lugares. A correção é a mesma
que `waveRuns` levou, e pelo mesmo motivo: **uma fase nova não pode depender de
alguém lembrar de editar uma constante global.** A fase declara o herói e o
resto do sistema lê a declaração — inclusive o desenho do NPC, que recebe o
personagem junto da posição.

**`sustained` separa as duas naturezas de ultimate, e a diferença é de
manutenção e não de poder.** A Legião Espectral é um bando que *segue* o dono:
o host reancora a legião a cada quadro e tira o NPC de campo quando ela se
gasta. A Área Angelical é um **altar** consagrado no chão — ela não segue
ninguém, e esperar por ela na manutenção prenderia o NPC no campo até o fim da
fase, porque `HasLegion` nunca fica verdadeiro para quem não tem legião.

**O herói é revivido mesmo estando vivo.** A cena dispara com o grupo
abaixo de 30% da vida, então "vivo" ali quer dizer *por pouco*, e mandá-lo
para o resgate naquele fiapo desperdiçaria o momento.

**A ultimate dele vem liberada com o resgate, só para esta corrida.** A
campanha continua entregando a suprema do herói só na fase seguinte
(`game.UltimatesGrantedOn` — ver "Ultimates: ganhas por fase" acima), mas o
resgate depende exatamente dela: `Host.reviveHero` chama
`network.GrantUltimateForRun`, que soma ao conjunto da campanha em vez de
substituí-lo, e `Host.BroadcastUltimateGrant` replica a concessão para todo
peer (`MsgUltimateGrant`, ver `doc/network.md`) — sem isso o botão da ultimate
no celular continuaria apagado e a tecla no desktop continuaria muda, porque
os três lugares que perguntam "esta suprema está liberada?" (o HUD, a
entrada, o gate do host) leem o mesmo `network.UltimateUnlockedFor`. A
concessão é apagada na troca de mapa e no reinício de fase — ver
"Por corrida, não por sessão" abaixo.

### O NPC não é um jogador

Ele nunca entra em `h.players`. Se entrasse, contaria no HUD de jogadores, na
checagem de Game Over, e teria que morrer, ressuscitar e ser sincronizado como
todo mundo. Tudo o que ele precisa é **ser dono de um efeito e ser visto** — e
toda ultimate é indexada por um `ownerID` que é só uma string. Cada fase tem o
próprio `npcID` (`npc_necromante`, `npc_sacerdotisa`): é por ele que o cliente
descobre que há um NPC em campo, sem mensagem nova de protocolo.

O cliente aprende que existe um NPC pela **mesma mensagem** que já traz a
legião (`spectral_legion` com dono `npc_necromante`), então nada de protocolo
novo para algo que acontece uma vez por fase. Ele é desenhado pelo caminho de
um jogador remoto — mesma folha, mesma escala, mesma linha de contato — para
não se denunciar como peça de outra natureza.

Ele sai de campo quando a legião se gasta: veio para uma coisa só.

### Invulnerabilidade concedida

`host_invulnerability.go` é uma janela curta que a **narrativa** concede, ao
lado (e não dentro) do Avatar dos Deuses da Paladina, que é habilidade com
efeito próprio. As duas são consultadas no mesmo ponto do caminho de dano.

Duas regras que os testes travam: a janela **expira** (senão é imunidade
permanente esquecida em campo), e uma concessão curta **não encurta** uma longa
que ainda corre.

### O resto do grupo também é reerguido

Quem **não** é o Necromante volta a **30% da vida com 1 s de imunidade**. Sem
isso a cena devolvia o jogo com o grupo ainda abaixo do limiar do último
suspiro, e o primeiro lobo desfazia o que ela acabou de fazer.

Os dois tamanhos são deliberados: o Necromante é quem **age**, então volta
inteiro e com dois segundos para sair dos dentes, escolher a direção e lançar.
O resto só precisa não morrer no golpe seguinte.

Jogador morto também é reerguido. A cena é um resgate — deixar um corpo no chão
pelos 30 s de respawn enquanto a legião limpa o campo faria o jogador assistir
ao próprio resgate.

### Por corrida, não por sessão

Quatro coisas entram no reset de fase, e todas já causaram bug ou causariam:

| O quê | Se não fosse limpo |
|---|---|
| `ResetLastStand` | A cena continuaria "gasta" e nunca mais rearmaria |
| `clearInvulnerability` | A fase recomeçaria com alguém invencível |
| Cenas **por corrida** no diretor de diálogo | **Bug real:** o roteiro do clímax seguia marcado como tocado, a cena não abria na segunda tentativa, e como é ela que segura o Game Over o grupo perdia direto |
| `network.ClearRunGrantedUltimates` | A suprema que o resgate liberou ficaria destravada numa tentativa nova da MESMA fase, e uma fase perdida depois do resgate recomeçaria com uma suprema a mais do que ela espera. A troca de mapa já limpa isto de graça — `World.ApplyToHost` chama `SetUnlockedUltimates`, que apaga a concessão da corrida anterior —, mas o F5 fica no mesmo mapa e nunca passa por ali, então precisa da chamada própria em `Host.ResetStage` (e do lado do cliente, em `applyStageReset`). |

O diretor sabe que a fase reiniciou por `network.StageGeneration()`, um contador
que vive em `RequestLocalReset` — o único ponto por onde host e cliente passam
ao reiniciar. Contar em outro lugar teria que ser contado duas vezes.

E ele esquece **por gatilho**, não tudo: `Trigger.PerRun()` diz que o clímax e
o fim de fase são da corrida, e que a abertura é do mapa. Reiniciar a fase não
devolve o grupo à floresta, então repetir a conversa de chegada a cada
tentativa só cansaria.

Dentro da **mesma** corrida a cena não rearma, senão cada queda devolveria a
legião e a fase viraria imperdível.

## O corredor final (mapa 6)

O mapa 6 não é uma corrida de hordas nem uma defesa de território: é um único
corredor reto, do vestíbulo até a porta do chefe, guardado por **dois
canhões** parados na sala atrás da porta. Eles disparam desde o instante em
que o grupo chega — não há onda, não há guarnição, e por isso o mapa **não**
aparece em `waveRuns`, `climaxRuns`, `garrisons` nem `sentryPosts`
(`TestMap6HasNoHordeOrGarrison` guarda isso).

### O canhão não é um `entity.Enemy`

Ao contrário da gárgula sentinela (`enemy_sentry_*`, alcance 1900, ataca por
esfera e pode ser ferida por espectro), o canhão do corredor é **decoração do
Tiled com uma arma de host atrás**: a estátua de gárgula já existente no
manifesto do castelo, reaproveitada como marco visual (decisão do Gui,
13/08/2026 — uma sprite de canhão de verdade fica para depois), pareada com um
marcador `enemy_cannon_*` na camada `spawn`. Ele nunca entra no
`EntityManager`, não tem sprite própria, não pode ser ferido por espada nem
por projétil, e só sai de campo quando o julgamento roteirizado da Paladina o
destrói (ver abaixo). `internal/network/cannons.go` e `host_cannon.go`
carregam a lógica; `internal/skill/cannon_ball*.go` a bola de fogo em si —
reta, sem perseguição, no molde da Bola de Fogo do Mago, e não da esfera da
gárgula.

| Constante | Valor | Por quê |
|---|---|---|
| `CannonDamage` | 45 | Quase metade da vida cheia (100): dói o bastante para o Escudo Sagrado (absorve 50) e a cura da Sacerdotisa serem a resposta certa, não esquiva pura. |
| `CannonRange` | 3200 px (25 células) | O corredor tem ~48 células do spawn até a sala dos canhões; o alcance chega um pouco além do meio — "até próximo da metade" do briefing. |
| `CannonCooldown` | 2,2 s por canhão | Os dois postos disparam juntos: uma bola por canhão forma uma parede de fogo de ponta a ponta do corredor a cada salva. |
| `CannonBallRadius` | 480 px | O diâmetro de 960 px cobre a faixa caminhável de 768 px; as duas bolas se sobrepõem, sem rota lateral de esquiva. |

`TestCannonsOutrunEscudoSagrado` trava a relação entre esses três números e o
Escudo Sagrado: um jogador parado sob fogo, recarregando o escudo a cada
`ShieldCooldown`, ainda perde mais vida por ciclo do que o escudo consegue
repor. Perder essa margem sem querer — aliviando o dano, ou acelerando o
cooldown do escudo — devolveria o corredor atravessável na força bruta, e o
clímax deixaria de ser necessário.

### O gatilho é uma zona, não a horda

Todo outro mapa com resgate pergunta ao `WaveState` se o grupo está "em
horda". O mapa 6 não tem horda para perguntar (`WaveState.Total` fica em zero
o tempo todo), então ele declara uma zona `corridor_checkpoint`
(`tilemap.CorridorCheckpointZone`) cobrindo do fim do alcance dos canhões até
a sala deles. `game.partyIsFalling` passa a perguntar por essa zona em vez do
`WaveState` quando o mapa a declara — ver "O último suspiro" acima.

### O resgate: o julgamento da Paladina

Quando a cena fecha a última fala, `ResolveLastStand` concede o Avatar dos
Deuses **e** chama `castCannonJudgment`, que destrói diretamente os dois
canhões declarados pelo mapa — o mesmo desenho que o julgamento do Arqueiro
usa no mapa 4 (`castCastleJudgment`) para alcançar as sentinelas das ilhas: um
resgate roteirizado precisa de um alvo **preciso**, não de algo que o jogador
teria de mirar, e a Paladina não teria como mirar duas estátuas atrás de uma
porta fechada de qualquer forma. `RestoreCannons` os repõe se a fase reiniciar.

### O portal fecha o ciclo, por enquanto

Não existe mapa de chefe ainda (fase 7). O portal no fim do corredor aponta de
volta para `world_01.json` — a mesma regra que a última fase da campanha já
usava antes deste mapa existir (decisão do Gui, 13/08/2026) — e
`campaign_portals_test.go` cobra a cadeia inteira. Não há bloqueio de
`WaveState` no portal: o que impede alcançá-lo cedo demais são os dois
canhões, não uma porta trancada (o mesmo padrão dos mapas 3 e 4, ver
`game/portal_gate.go`).

## Teclas de desenvolvimento (desktop)

| Tecla | O que faz | Onde |
|---|---|---|
| F2 | Modo teste: tira todo cooldown e cadência do jogador local | `game/test_mode.go` |
| F3 | Overlay de debug (colisão, footprints, âncoras de manifesto) | `tilemap/renderer.go` |
| F5 | Reinicia a fase depois do Game Over (só o host) | `game/reset.go` |
| F8 | Pula para a próxima fase da campanha | `game/stage_skip.go`, ver `doc/tilemap.md` |
| Shift+F8 | Pula direto para a última fase da campanha, com as supremas correspondentes | `game/stage_skip.go`, ver `doc/tilemap.md` |

Nenhuma tem equivalente de toque, e nenhuma deve ganhar: são chaves de
desenvolvimento, não recursos de jogo. Todas ficam desligadas quando
`cfg.FullScreen` (Android).

## Test Mode (F2, desktop)

**F2** toggles test mode for the local player: no skill cooldowns (charges
included) and no attack cadence. It is per player and host-applied — a client
pressing F2 sends `test_mode` and the host relaxes that player's gates only, so
one person can test while the rest of the group plays normally. The screen
shows `MODO TESTE` while it is on.

## Verification

The rules above are covered by `work/combat-verify` (copies of the real files
compiled against raylib/host stubs, in the style of `work/dialogue-verify`):

    cd work/combat-verify && go test ./...

It does not replace `go build ./cmd/desktop` — the stubs mirror signatures, not
raylib itself.
