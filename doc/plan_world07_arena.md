# Plano — `world_07`, a arena da Senhora das Trevas

Plano para aprovação antes de construir, como a `create-tiled-map` exige.
Decisões travadas: **mapa novo (`world_07`)**, **2 portões opostos**,
**névoa mata em ~4 s**, **arena de 40x30 células**.

---

## 1. O mapa

**40x30 células de 128 px = 5120x3840 px.** Retângulo fechado, sem saída: a
fase termina quando a chefe cai.

```
        ┌──────────────── 40 células (5120 px) ─────────────────┐
        │  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓    │
   30   │  ▓                                                ▓   │
  cél.  │ ╔═╗                      ◆                       ╔═╗  │  ◆ chefe (centro)
 (3840) │ ║O║  ← portão oeste             portão leste →   ║L║  │  ╔═╗ portão
        │ ╚═╝                                              ╚═╝  │
        │  ▓                    ▲ jogadores                 ▓   │
        │  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓    │
```

| Elemento | Posição | Porquê |
|---|---|---|
| Chefe | `(2560, 1920)`, centro exato | Fixa. `Speed 0`. É o eixo em torno do qual tudo gira. |
| Portão oeste | `x ≈ 384`, meia altura | ~2170 px do centro |
| Portão leste | `x ≈ 4736`, meia altura | mesma distância, simétrico |
| Entrada do grupo | borda sul, centro | Chegam vendo a chefe de frente |

**Distância chefe → portão: ~2200 px, uns 11 s de corrida** (jogador a 200/s).
É a régua que faz a horda demorar a chegar e dá ao grupo a escolha de qual
frente segurar.

**Bioma:** salão de castelo corrompido — `terrain_castle_stone` e
`terrain_dark_flagstone` no chão, `fortress_wall` no perímetro. Reaproveita o
que o `world_04`/`world_05` já carregam; **zero atlas novo**, o que importa
porque `doc/performance.md` mede 82–86 MB por mapa e a chefe sozinha já custa
38,8 MB.

**Camadas:** `ground`, `ground_detail`, `structures_back` (os dois portões),
`collision` (perímetro + os dois portões fechados como parede intransponível
para o jogador), `foreground`, e os objectgroups de sempre.

**Zonas e marcadores:**

| Objeto | Camada | Uso |
|---|---|---|
| `boss_anchor` | `spawn` | onde a chefe nasce |
| `wave_spawn_west`, `wave_spawn_east` | `spawn` | os dois portões |
| `player_spawn` | `spawn` | entrada sul |
| `arena` (zona cobrindo tudo) | `zones` | área da névoa e janela do clímax |

**Sem prop no miolo.** A arena é uma sala vazia de propósito: com 30 inimigos,
a chefe e os espinhões, qualquer coluna vira lugar onde o jogador trava no
momento errado. Detalhe fica no perímetro.

---

## 2. Barra de vida de chefe

Hoje `enemyBarLayout` desenha a mesma barra para todo inimigo. A chefe precisa
de outra, e não é enfeite: com 30 inimigos em tela, a barra dela tem que ser
identificável antes de o jogador procurar.

**Barra fixa no topo da tela**, em espaço de tela e não de mundo (a regra do
`doc/camera.md` sobre o que fica fora da transformação de câmera):

- Largura de 60% da tela, centralizada, com o nome **"Senhora das Trevas"**.
- Moldura em ouro envelhecido, o mesmo acento da personagem.
- **Marcas de fase a cada 25%** — não porque haja mudança de fase agora, mas
  porque uma barra de 400 de vida sem subdivisão não dá sensação de progresso.
- A barra flutuante sobre a cabeça dela **some**: duas barras para a mesma
  criatura é ruído.

Arquivo novo: `internal/ui/boss_bar.go`. Fica em `ui` e não em `entity` porque é
HUD, e o `enemy_draw_box.go` continua cuidando só do que é mundo.

---

## 3. Os três relógios

O que o motor já tem: `AttackCooldown` (um só) e a corrida de hordas. O que
falta é um **relógio por habilidade**, porque 15 s e 60 s são independentes.

Arquivo novo: `internal/network/host_boss.go`, ao lado de `host_garrison.go`.
Ele é do host, e só do host — o cliente recebe o resultado.

| Relógio | Período | O que dispara |
|---|---|---|
| Espinhão | **15 s** | `TriggerStrike()` + os espinhões |
| Névoa | **60 s** | `TriggerCast()` + a névoa |
| Horda | **70 s**, com o **primeiro em 10 s** | uma leva completa |

O deslocamento de 10 s que você pediu é o que impede os três de coincidirem: em
70 s o ciclo é névoa (60), horda (70), espinhão a cada 15. **Em 420 s os três
voltam a bater juntos** — 60 e 70 têm MMC 420, e o espinhão de 15 divide os
dois. É um pico de dificuldade que se repete a cada 7 minutos, e é bom que
exista: dá à luta um clímax periódico em vez de uma parede constante.

### 3.1 Espinhão — 15 s

Sequência, encaixada na animação que já existe:

1. `TriggerStrike()` → `attack_windup` (braços cruzados tremendo, **1,8 s**).
   Esse é o tempo de leitura: o jogador vê a pose e sabe o que vem.
2. **No começo do windup**, o host tira uma foto da posição de cada jogador e
   marca o chão ali — um decalque de aviso, sem dano.
3. `attack_strike` toca; **no quadro 3** (o agachamento, ~0,21 s dentro da
   folha) o espinhão irrompe em cada marca e causa dano uma vez.

A janela real de desvio é ~2 s, e ela é honesta: a marca aparece antes do golpe
e não segue o jogador. Quem se move sai.

Arquivo novo: `internal/skill/thorn.go` + `thorn_manager.go`, seguindo o padrão
de `fire_ground` (que também é dano de área ancorado no chão) — telegrafia,
erupção, dano único, e desenho por primitivas de raylib, **nunca por asset**,
como manda a regra de efeitos do `art_style.md`.

### 3.2 Névoa — 60 s

`TriggerCast()` → `cast_loop` dança por **2,4 s** → `cast_release` → a névoa
cobre a arena inteira e dura **8 s**.

**Dano: 30/s.** Com a vida dos heróis, isso mata em ~4 s, que foi o que você
escolheu: sobra tempo de correr até a Área Angelical, não sobra tempo de
ignorar.

**As duas únicas salvações**, e as duas já são consultáveis:

```go
protegido := m.HasAvatar(playerID) ||          // Paladina ultada
             angelicContains(m, playerPos)     // dentro da Área Angelical
```

`HasAngelic(ownerID)` responde "a Sacerdotisa está com a área ativa"; falta uma
função que responda "**este ponto** está dentro da área", porque a proteção é
de quem está em cima, não de quem lançou. É a única função nova de skill.

**Aviso de 2,4 s.** A dança é o telegrafo — o jogador tem o tempo do
`cast_loop` para chegar na área. Somado ao cooldown da Sacerdotisa, é o que
torna a coordenação possível em vez de sorte.

### 3.3 Hordas infinitas — 70 s

```go
{
    Name: "O cerco da senhora",
    Composition: map[entity.EnemyType]int{
        entity.EnemyTypeBasic:        10,  // slime
        entity.EnemyTypeFast:         10,  // lobo
        entity.EnemyTypeGarrison:      8,  // orc
        entity.EnemyTypeCastleSentry:  2,  // gargula
    },
    Endless:       true,
    MaxConcurrent: 34,
    ...
}
```

**Divisão pelos dois portões:** 5 slime, 5 lobo, 4 orc, 1 gárgula de cada lado.
Todos pares, todos dividem exato — é o que sua contagem já garantia.

`Endless` já existe (o clímax do `world_03` usa) e é exatamente a semântica
pedida: a composição volta inteira para a fila quando esvazia. **Só para quando
a chefe morre**, o que exige uma condição nova — hoje a corrida termina por
contagem, não por morte de um inimigo específico.

`MaxConcurrent: 34` é o teto: 30 da horda mais folga. Sem teto, hordas a cada
70 s que não são limpas empilham indefinidamente e a fase vira uma parede em
vez de uma luta.

**As gárgulas são acumuladas, não repostas.** O campo `Sentries` do `WaveDef`
já trata disso no `world_05`: 2 por horda, somando. Depois de cinco hordas há
10 torres de alcance 1900 no salão, e é isso que dá à luta longa uma escalada
sem eu inventar um sistema de dificuldade.

---

## 4. Fim da fase

A corrida de hordas hoje termina quando a contagem zera. Aqui ela termina
quando **a chefe morre**, e a corrida precisa saber disso.

Proposta mínima: um campo `EndsWithBoss entity.EnemyType` no `WaveDef`. Quando
preenchido, `Endless` só é reciclado enquanto existir um inimigo daquele tipo
vivo; quando ele cai, a corrida para de repor e a fase termina depois que o que
está em campo é limpo.

Alternativa que eu descartei: matar a horda no instante em que a chefe cai.
Fica anticlimático — trinta inimigos sumindo no ar tira do grupo a limpeza
final, que é o momento em que a vitória assenta.

---

## 5. Arquivos

**Novos**

| Arquivo | Responsabilidade |
|---|---|
| `assets/maps/world_07.json` | a arena |
| `internal/network/host_boss.go` | os três relógios, do lado do host |
| `internal/skill/thorn.go` + `thorn_manager.go` | espinhão: telegrafo, erupção, dano |
| `internal/skill/dark_fog.go` + `dark_fog_manager.go` | névoa: área, dano por segundo, isenções |
| `internal/ui/boss_bar.go` | a barra de chefe |

**Editados**

| Arquivo | Mudança |
|---|---|
| `internal/game/stage_skip.go` | `world_07` em `campaignMaps` |
| `internal/network/wave_runs.go` | `world07Waves`, com `Endless` e `EndsWithBoss` |
| `internal/network/climax_window.go` | janela do mapa 7 |
| `internal/network/host_spawn.go` | nascer a chefe no `boss_anchor` |
| `internal/skill/angelic_manager.go` | `AngelicContains(ponto)` |
| `internal/ui/hud.go` | chamar a barra de chefe |
| `internal/entity/enemy_boss_anim.go` | ligar `TriggerCast`/`TriggerStrike` ao host |
| `doc/combat_rules.md`, `doc/changelog.md` | registrar |

**Duas lacunas conhecidas que este plano não fecha**, e vale decidir se entram
agora ou depois:

1. **Replicação.** O estado de animação da chefe é local do host; num cliente
   ela fica em idle. Precisa do `Anim` no protocolo.
2. **Multiplayer da névoa.** A verificação de proteção roda no host, que já tem
   a posição de todos — isso funciona. Mas o *desenho* da névoa no cliente
   precisa da mesma mensagem de ativação que as outras skills usam.

---

## 6. Ordem de construção

1. O mapa (`world_07.json`) e o registro nas tabelas — dá para carregar e andar.
2. Nascer a chefe e a barra de chefe — dá para vê-la e bater nela.
3. O relógio dos espinhões, que é o mais simples e o que mais define a luta.
4. As hordas com `Endless` + `EndsWithBoss`.
5. A névoa e as isenções, que é a peça com mais dependência (duas skills).
6. Replicação, se você quiser a fase jogável em rede já nesta rodada.

Cada passo é jogável sozinho, o que permite parar em qualquer um.
