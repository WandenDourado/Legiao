# Ritmo do jogo (`entity.GameSpeed`)

Uma constante multiplica tudo que se desloca. Ela existe porque os testadores
de 20/09/2026 não disseram "o personagem anda devagar em relação aos monstros"
— disseram que o **jogo** é lento. A queixa não era de balanceamento (as
proporções entre lobo, orc e jogador estavam certas), era de ritmo: tudo se
move devagar ao mesmo tempo. Um fator único preserva as proporções; uma revisão
número a número as mudaria sem querer.

```go
// internal/entity/gamespeed.go
const GameSpeed float32 = 1.20
```

Jogador: 200 → 240 px/s.

**Histórico do número.** A primeira rodada (20/09/2026) foi a 1.35 (270 px/s).
No dia seguinte o Gui jogou e achou rápido demais; o fator baixou para 1.20
(21/09/2026). Nenhum outro ponto deste documento muda com o valor — é
justamente a vantagem de ser um fator único: reafinar é trocar um número, não
revisar o elenco.

## Onde ele entra (quatro pontos, e só quatro)

| # | Onde | O quê |
|---|---|---|
| 1 | `entity.PlayerSpeed` (`player.go`) | Os bots vêm junto: `botSpeed` deriva dele (`network/host_bot_move.go`). |
| 2 | `RegisterEnemy` (`enemy_sprite.go`) | Multiplica `def.Speed`. `Speed == 0` continua 0 — a Sentinela e a Imperatriz não saem do lugar. |
| 3 | `RegisterEnemy` e `RegisterCharacter` | **Divide** o `FrameTime` da animação de caminhada. |
| 4 | `entity/projectile.go` e `internal/skill/` | Multiplica velocidade de projétil e **divide** o tempo de vida. |

Os números escritos em cada registro continuam sendo os autorais — o 130 do
orc contra os 200 do jogador, que é a relação que os comentários daqueles
arquivos explicam. O fator é aplicado num ponto só, e não espalhado pelos
literais, para não perder esse fio.

`internal/skill` usa `gameSpeed` (minúsculo, `skill/gamespeed.go`), que é só um
espelho de `entity.GameSpeed` para os seis arquivos de projétil não precisarem
importar `entity` por causa de um número.

## As duas invariantes que não podem quebrar

**1. O pé não desliza.** O `FrameTime` do `AnimWalk` do orc não é gosto: é a
única solução de `2 × 63,5 × RenderScale / (Speed × 10)`, medida na arte. Subir
a `Speed` sem encurtar o ciclo devolve o monstro patinando — o defeito que
aquele número nasceu para consertar. Vale para o jogador também
(`CharacterDef.FrameTime`/`SprintTime`).

Guarda: `TestWalkCycleKeepsUpWithTheFeet` (`entity/gamespeed_test.go`).

Os multiplicadores de passo que **não** são o `GameSpeed` — a aura do
Santuário (1,25×) e o fôlego do renascer (`entity.RespawnGraceSpeed`, 1,4×,
27/09/2026) — respeitam a mesma invariante por outro caminho: o jogador anda a
`PlayerSpeed × network.MoveSpeedMultiplier`, e `Player.updateAnimation`
(`entity/player_sprite.go`) divide o tempo de quadro pela razão
`Speed / PlayerSpeed`. Antes disso o Santuário deslizava 25%.

**2. O alcance não muda.** Velocidade × tempo de vida é a **distância**, e é
ela que `entity.AttackReach`, o guarda de território e os alcances dos bots
(`bot/tuning.go`) leem. `420 × 2,0` e `567 × 1,48` dão os mesmos 840 px: o tiro
chega mais rápido ao mesmo lugar. Sem a divisão, o Arqueiro passaria a acertar
de 1512 px e o contrato numérico das supremas mudaria sozinho, em silêncio.

Guarda: `TestAttackReachSurvivesGameSpeed`.

Atenção ao caso disfarçado: um TTL que parece "rede de segurança" mas está
calibrado contra um alcance declarado **é** um orçamento de voo e precisa ser
dividido. Foi o que aconteceu com `CannonBallTTL` — com a velocidade escalada e
o TTL intacto, o tiro de aviso do canhão passou a machucar 1589 px além do
alcance que o corredor anuncia. `TestBlindShotDiesAroundTheEdgeOfTheRange`
pegou. `SentryOrbMaxTTL` é o mesmo caso; o TTL calculado por distância
(`sentryOrbTTLFor`) se ajusta sozinho, porque ele já é `dist/Speed`.

Alcance escrito como **distância** (`ArrowRange`, `FireballRange`,
`CelestialRange`) não leva o fator: é invariante por construção.

## O que ele deliberadamente NÃO toca

Recarga, dano, cadência de ataque, duração de magia de área, tempo de
ressurreição. O pedido era fluidez de deslocamento; multiplicar a cadência
junto transformaria isto numa mudança de dificuldade que ninguém pediu.

**Ponto que o playtest de 20/09 não confirmou:** com todo mundo mais rápido e
a cadência igual, o combate fica mais fácil — dá para orbitar o orc. Ainda em
aberto no valor atual (1.20); se aparecer, o ajuste é em `AttackCooldown` e
alcance, monstro a monstro, e **não** no `GameSpeed`.

## Por que é constante de compilação, e não configuração

O jogo é host-autoritativo, mas o jogador local é simulado no próprio quadro
(`network/interpolation.go`). Com fatores diferentes nas duas pontas o cliente
prevê 240 e o host corrige para 200: rubber-banding contínuo. Por isso ela
nunca entra na tela de Configurações, e por isso **host e clientes precisam da
mesma build** — ver `doc/network.md`, "Todo mundo na mesma build".

## Interpolação

A 240 px/s, os 100 ms de `interpDelay` viram ~24 px de defasagem visual, contra
~20 px na versão sem fator — bem abaixo do corpo de qualquer personagem.
`SnapshotHz = 20` e `interpDelay = 100 ms` ficam como estão. Reavaliar só se o
`GameSpeed` algum dia passar de ~1.6.
