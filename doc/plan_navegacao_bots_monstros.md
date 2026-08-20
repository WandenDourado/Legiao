# Plano: navegação de bots e monstros

> **Estado: implementado (fases 1-5).** Decisões tomadas com Gui em
> 20/08/2026, depois de ver bots empurrando uma árvore a caminho do portal e
> monstros socando a cerca do mapa 3. Fase 6 (afinação em jogo) continua em
> aberto — ver `doc/changelog.md` para o que cada fase entregou de fato.

Decisões desta conversa:

- **Malha de navegação para todos** (bots e monstros), não remendo local.
- **O monstro só planeja quando percebe que não avança** — ele investe direto
  como hoje, bate, olha e dá a volta. O bot planeja sempre que precisa.

---

## 1. O sintoma, e o que o código diz

**O bot empurra a árvore.** `resolveBotMove`
(`internal/network/host_bot_move.go`) usava `collision.Resolve` — o deslize
simples. O contorno existe no repositório e é bom (`collision.ResolveDetour`,
com direção comprometida e varredura da face), **mas o bot não o usava**. O
único socorro dele era `unstickMove`: depois de 1,5 s parado, um único quadro
de passo a 90° e volta a empurrar. Era exatamente o vaivém observado.

**O monstro soca a cerca.** Esse já usa `ResolveDetour`
(`entity.(*Enemy).step`), e ela resolve o caso para o qual foi escrita: uma
barricada reta com um vão em linha de vista. `openingSide` varre até 96 passos
**ao longo de um eixo só**, a partir de onde a criatura está. Ela não vê:

- vão que exige contornar um canto (barricada em L, casa);
- aglomerado de árvores, onde a face não é uma reta;
- caminho que começa **afastando-se** do alvo antes de aproximar.

**E há um terceiro efeito, de matilha:** `ResolveEnemyOverlap` desempilha os
monstros depois que todos andaram. Num engarrafamento contra a cerca, esse
empurrão devolve a criatura para a face do obstáculo e zera o progresso que o
contorno tinha começado.

A causa comum é uma só: **os dois decidem sem mapa**. A única pergunta que o
sistema sabe fazer é "esta caixa colide aqui?", e nenhuma quantidade de
esperteza local responde "por onde se chega lá".

---

## 2. O princípio: três camadas, e faltava a do meio

| Camada | Pergunta | Quem responde |
|---|---|---|
| Decisão | **Para onde** eu vou? | `internal/bot` (cérebros) e a IA do monstro |
| Rota | **Por onde** se chega lá? | `internal/nav` |
| Passo | O que o mapa **deixa** eu fazer neste quadro? | `collision.Resolve` / `ResolveDetour` |

O contorno parecia burro porque a camada do passo estava fazendo o trabalho da
camada da rota. Ela não tinha como fazê-lo: ela enxerga uma caixa, um passo, um
quadro.

**O rumo direto continua sendo o caminho normal.** A rota é o plano B — o que
entra quando a linha reta não serve. Isso mantém o custo perto de zero no caso
comum (campo aberto, que é a maior parte do tempo de jogo) e preserva a
sensação de bicho investindo.

---

## 3. `internal/nav`

Puro: sem `network`, sem `game`, sem `entity`, sem textura. Depende de
`collision.Solid`, `world.Bounds` e `rl.Vector2`, nada mais. Testável sem abrir
janela.

### 3.1 A malha

Derivada da colisão no carregamento do mapa. Célula de 32 px; uma célula é
livre quando uma caixa de agente de 48×48 centrada no centro dela não colide.

- Os pés que andam neste jogo são todos do mesmo tamanho (jogador 40×40, orc e
  slime ~40,5, lobo 45). Uma caixa de 48 cobre todo mundo com folga.
- 32 px é meia largura de um footprint: garante que um vão real de ~60 px
  tenha pelo menos uma célula livre dentro dele.
- Monstro andante com footprint maior que 48 precisaria de uma segunda malha
  — não existe hoje.

### 3.2 Invalidação

`CollisionGrid.SetFootprintsEnabledOverlapping` muda a colisão em jogo (o
portão da arena, `internal/game/arena_gate.go`). Quando isso acontece,
`Host.RebuildNavArea` reconstrói só as células que tocam a área do portão.

### 3.3 A API implementada

```go
package nav

type Grid struct{ /* cell, agent, origin, w, h, free []bool, scratch, budget */ }

func Build(s collision.Solid, bounds world.Bounds, cell, agent float32) *Grid
func (g *Grid) RebuildArea(s collision.Solid, area rl.Rectangle)
func (g *Grid) Walkable(p rl.Vector2) bool
func (g *Grid) NearestWalkable(p rl.Vector2, maxRadius float32) (rl.Vector2, bool)
func (g *Grid) LineOfSight(a, b rl.Vector2) bool
func (g *Grid) FindPath(from, to rl.Vector2, out []rl.Vector2) ([]rl.Vector2, bool)
func (g *Grid) ResetFrameBudget()
func (g *Grid) ForEachCell(fn func(center rl.Vector2, half float32, free bool)) // F4 only

type Follower struct{ /* path, idx, goal, hasGoal, replanIn */ }
func (f *Follower) Desired(g *Grid, pos, goal rl.Vector2, dt, replanEvery float32) (dir rl.Vector2, following bool)
func (f *Follower) Active() bool
func (f *Follower) Clear()
func (f *Follower) Path() []rl.Vector2 // F4 only
```

Regras do algoritmo:

- **A\* octil, sem cortar canto**: a diagonal só é permitida quando as duas
  ortogonais correspondentes estão livres.
- **Suavização por linha de visão** (*string pulling*): de trás para frente,
  liga o ponto atual ao waypoint mais distante que ainda tem linha livre,
  recuando um de cada vez quando bloqueado.
- **Sem alocação por busca**: heap em slice reaproveitado (`astarScratch`),
  `seen`/`gScore`/`cameFrom` como slices indexados por célula com selo de
  geração — zerar dezenas de milhares de posições a cada busca custaria mais
  que a busca.
- `NearestWalkable` existe porque um agente pode estar DENTRO de sólido —
  parte do vizinho livre mais próximo em vez de responder "não há caminho".
- **Reancoragem**: se o waypoint atual deixou de ter linha de visão (um
  empurrão de `ResolveEnemyOverlap`, a separação de outro agente), `Follower`
  procura o waypoint mais distante do caminho já calculado que ainda enxerga,
  antes de pagar por um replan inteiro — só cai para replan se nem o waypoint
  atual segue visível.

### 3.4 Orçamento

- No máximo 8 buscas por quadro (`Grid.ResetFrameBudget` + reserva interna),
  em ordem de chegada. Quem não coube espera.
- Cada agente pede no máximo uma busca a cada 0,4 s (bot) / 0,7 s (monstro).
- Enquanto a busca não sai, o agente segue o rumo direto (nunca para).

---

## 4. Quando cada um planeja

**Bot — sempre que precisa.** Pede rota quando o destino não está em linha de
visão livre (`Follower.Desired` testa isso na entrada). Solta a rota assim que
o destino volta a estar visível em linha reta.

**Monstro — só quando percebe que não avança.** Investe direto, como sempre.
A implementação NÃO usa o `collision.Progressed` por passo para essa decisão
— ele é leniente demais (15% do passo pretendido já conta como "progresso", e
um monstro deslizando ao longo da face de uma barricada longa relata
progresso a cada quadro sem nunca se aproximar do vão real). Em vez disso,
`Enemy.notMakingHeadway` mede uma **janela de distância real** ao alvo: a cada
`FoeStuckBefore` (0,4 s), compara a distância de agora com a de 0,4 s atrás; se
não encolheu pelo menos 25% do que uma corrida livre cobriria nesse tempo, o
monstro pede rota. Volta ao rumo direto assim que o alvo reaparece em linha de
visão (`Follower.Desired` cuida disso sozinho).

O efeito na tela é o pretendido: um bicho que bate, olha e dá a volta — não um
caçador onisciente que já sai andando pelo caminho ótimo.

---

## 5. Onde o código toca (o que foi feito de fato)

| Arquivo | Mudança |
|---|---|
| `internal/nav/*` | `grid.go`, `build.go`, `astar.go`, `smooth.go`, `los.go`, `follower.go`, `budget.go`, `debug.go`, `tuning.go` + `nav_test.go` + `bench_test.go`. |
| `internal/entity/manager.go` | `EntityManager.Nav *nav.Grid`, irmão de `Solid`. |
| `internal/entity/enemy.go` | `MoveEnv{Solid, Nav}` substitui o `collision.Solid` cru em `Update`/`MoveTowardTarget`/`moveTowardTarget`. `Enemy.follower nav.Follower` + `notMakingHeadway` (janela de distância) decidem quando consultar a malha. `Enemy.Route()` expõe a rota para o F4. |
| `internal/collision/resolve.go` | `Progressed` exportado (a versão testada acabou não sendo a base do gatilho do monstro — ver §4 — mas fica exportada para quem precisar do mesmo teste por passo). |
| `internal/network/host_bot_move.go` | `resolveBotMove` usa `ResolveDetour` com `botRuntime.detourDir` comprometido (fase 1). `desiredBotMove` converte `Intent.Dest`/`Push` em direção via `Follower`. |
| `internal/network/host_bots.go` | `botRuntime.nav nav.Follower`. |
| `internal/network/host_bot_tick.go` | Reseta o orçamento da malha uma vez por quadro (`tickBots`); a busca do portal também passa a ser um `Intent.Dest`. |
| `internal/network/host_nav.go` | `Host.RebuildNavArea`, `Host.BotRoute` (F4). |
| `internal/bot/intent.go` e os cinco cérebros | `Intent` carrega `Dest`/`HasDest`/`Push` em vez de `Move`. `moveTo`/`flee` (steering.go) substituem `seekAndSeparate` como o caminho normal. |
| `internal/game/world_state.go` (`ApplyToHost`) | Constrói a malha logo depois de `host.EntityManager.Solid = w.Collision`, loga o tempo uma vez. |
| `internal/game/arena_gate.go` | Quando o portão muda, chama `Host.RebuildNavArea` na área dele. |
| `internal/game/nav_debug.go`, `input_handler.go`, `renderer.go` | **F4**: desenha a malha e as rotas. |

---

## 6. Riscos, e a regra que cobre cada um

1. **Malha desatualizada.** `RebuildArea`/`RebuildNavArea` cobrem o portão da
   arena; testado (`TestRebuildAreaOpensAndClosesAGate`,
   `TestRebuildNavAreaFollowsTheArenaGate`).
2. **Tempo de carregamento.** `[Nav] ... malha construida em ...` loga uma vez
   por mapa. Não medido em produção ainda — fica para a fase 6.
3. **Pico de busca.** Orçamento por quadro (`PathBudgetPerFrame = 8`) e
   relógio por agente, sempre.
4. **Empurrão de matilha desalinha a rota.** Coberto pela reancoragem em
   `Follower.Desired` (§3.3) — sem ela, o teste de 20 monstros contra um vão
   distante (`TestPackGetsThroughABarricadeGapWithinBudget`) não passava.
5. **Cliente não navega.** A malha só existe no papel host (`EntityManager.Nav`
   nasce em `World.ApplyToHost`, chamado só do lado que simula).
6. **A navegação sugere; ela nunca move.** O passo continua sendo
   `collision.Resolve`/`ResolveDetour`.
7. **Log.** Nenhum log por quadro em caminho de agente; só o log de construção
   da malha, uma vez por carregamento.

---

## 7. Testes escritos

- **`nav`**: caminho existe / não existe, não corta quina, suavização não
  atravessa sólido, `LineOfSight`, `RebuildArea`, `NearestWalkable` de dentro
  de sólido, orçamento por quadro, fallback do `Follower` sob orçamento
  esgotado, benchmarks de `Build` e `FindPath` na escala do `world_05`.
- **`entity`**: monstro atrás de barricada em L alcança o alvo; monstro em
  campo aberto nunca consulta a malha; matilha de 20 atravessa um vão distante
  dentro do orçamento compartilhado.
- **`network`**: bot contorna um obstáculo finito (fase 1); bot usa a malha
  quando o vão está fora do alcance do contorno local; `RebuildNavArea` segue
  o portão da arena.

## 8. Em aberto (fase 6)

Afinação em jogo: pesos de separação, relógios (`BotReplanEvery`,
`FoeReplanEvery`, `FoeStuckBefore`), o limiar de 25% em `notMakingHeadway`, e
medição real do tempo de `Build` contra `CollisionGrid` (não o `nil`/stub dos
benchmarks do pacote `nav`) em `doc/performance.md`.
