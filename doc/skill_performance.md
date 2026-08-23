# Desempenho de magias

Como escrever uma magia nova sem derrubar o jogo, e por que as duas ultimates
que derrubaram tinham a mesma forma de defeito.

Este documento é **obrigatório** ao criar ou revisar qualquer magia
(`skills/create-character-abilities/SKILL.md` aponta para cá). Ele não fala de
arte nem de equilíbrio: fala do que a magia **custa por quadro**.

---

## Por que ele existe

Duas ultimates derrubaram o jogo em agosto de 2026, com sintomas idênticos
("o fps cai depois que fulano usa a suprema") e causas diferentes:

| | Legião Espectral (Necromante) | Chuva de Meteoros (Mago) |
|---|---|---|
| O que punha em campo | 30 espectros simulados | ~56 meteoros + ~2.500 partículas |
| O erro | colisão contra **lista plana** de ~1.400 retângulos | **desenhar o mapa inteiro**, sem culling |
| A conta | ~5.000.000 comparações/quadro | ~92.000 triângulos/quadro de partículas, 83% fora da tela |
| Onde aparecia | `simulacao` | `resto` (GPU) |

Nenhuma das duas era um vazamento. As duas eram **trabalho proporcional a algo
que ninguém tinha contado**: o número de retângulos do mapa numa, o tamanho do
mapa na outra. E as duas escalavam com a FASE, então apareciam como "o jogo
piora conforme avança" — a forma de um vazamento, sem ser um.

**A lição que generaliza:** uma magia é escrita e testada com UMA instância no
mapa 1. Ela vai para produção com trinta instâncias no mapa 7. Toda decisão
abaixo é sobre essa diferença.

---

## As três contas que toda magia paga

Antes de escrever a primeira linha, responda as três. Em número, não em
adjetivo.

### 1. Simulação — quantas entidades, e o que cada uma consulta

```
entidades em campo  ×  consultas por entidade por quadro  ×  custo da consulta
```

A Legião: `30 × 4 × 1.400 = 168.000`, e isso só no movimento — a separação par
a par multiplicava por mais 29.

**Quantas entidades em campo** não é o que a magia cria de uma vez, é o
**regime**: taxa de nascimento × tempo de vida. A Chuva de Meteoros cria um a
cada 0,025 s e cada um vive 1,4 s, então o regime é 56 — nunca "um meteoro".

### 2. Desenho — quantas primitivas, e quantas estão na tela

```
entidades desenhadas  ×  primitivas por entidade
```

E `entidades desenhadas` tem de ser **as visíveis**, não todas. Ver a Regra 2.

### 3. Rede — quantas mensagens por segundo

Toda entidade que o cliente precisa replicar custa uma mensagem. A Chuva de
Meteoros transmite **uma por meteoro**: 40 por segundo, 600 ao longo da
ultimate, cada uma com marshal de JSON e escrita para cada peer.

---

## As sete regras

### Regra 1 — colisão nunca contra lista plana

**Nunca** receba `[]rl.Rectangle` para testar obstáculo. Receba
`collision.Solid` e chame `CollidesCentered`.

```go
// ERRADO — varre TODOS os sólidos do mapa a cada chamada
func StepMinhaMagia(m *Manager, rects []rl.Rectangle, dt float32) {
    if tilemap.IsColliding(pos, w, h, rects) { ... }
}

// CERTO — consulta só as células que a caixa toca
func StepMinhaMagia(m *Manager, solid collision.Solid, dt float32) {
    if blocked(solid, pos, size) { ... }   // skill/obstacle.go
}
```

Por quê: `IsColliding` é O(retângulos do mapa) — 1.400 no `world_02`, e cresce
com o mapa. `CollidesCentered` é O(células que a caixa toca) — uma ou quatro.
É a mesma porta que jogador e monstro sempre usaram (`EntityManager.Solid`).

O ajudante `blocked` já existe em `internal/skill/obstacle.go`. Use-o.

### Regra 2 — todo `Draw` começa com culling

Primeira linha de todo método de desenho, **antes** de `BeginBlendMode`:

```go
func (x *MinhaCoisa) Draw() {
    if !visible(x.Position, x.raioVisualTotal) {
        return
    }
    ...
}
```

`visible` está em `internal/skill/view.go`. O raio é o **alcance visual
inteiro**, não o de dano: se a magia joga partículas a 3,5× o raio do anel, o
teste usa 3,5×. Geometria comprida (um rastro, um raio) usa `visibleAny` com as
duas pontas.

Partículas se cullam **uma a uma**, não pelo emissor: o emissor pode estar fora
da tela cuspindo partículas para dentro dela.

Por quê: a tela mostra 8,3 Mpx; o `world_02` tem 49,2 Mpx de mundo. Sem
culling, 83% do desenho é jogado fora depois de pago.

### Regra 3 — conte o regime, não o caso único

Escreva o número no comentário da constante:

```go
// MeteorRainInterval é o intervalo entre nascimentos.
// REGIME: 1/0,025 × 1,4 s de vida = ~56 meteoros no ar ao mesmo tempo.
MeteorRainInterval float32 = 0.025
```

Se o regime passa de ~20 entidades, a magia é um sistema e não um efeito:
volte e releia as três contas.

### Regra 4 — interação entre N entidades é O(n²): acumule, aplique uma vez

Separação, empurrão, atração mútua — qualquer laço "cada um contra cada um".
Com 30 entidades são 435 pares.

```go
// ERRADO — move os dois em cada par, na hora
for i, a := range xs {
    for j := i+1; j < len(xs); j++ {
        mover(a, empurrao)          // 870 chamadas de movimento
        mover(xs[j], -empurrao)
    }
}

// CERTO — soma os pares, move uma vez por entidade
for i := range push { push[i] = rl.Vector2{} }
for i, a := range xs {
    for j := i+1; j < len(xs); j++ {
        push[i] = sub(push[i], e); push[j] = add(push[j], e)
    }
}
for i, x := range xs { mover(x, push[i]) }   // 30 chamadas
```

Além de 29× mais barato, é **mais correto**: aplicando par a par, o empurrão de
A muda a posição que o par seguinte lê, e a ordem do laço vira parte do
resultado.

### Regra 5 — partícula é cara; `DrawCircleGradient` é um leque de triângulos

Cada `DrawCircleGradient` vira ~36 triângulos no rlgl. 2.500 partículas são
~92.000 triângulos por quadro, todos em blending aditivo, todos em 4K.

- Corte partículas antes de cortar qualidade: metade das partículas com o dobro
  do raio parece igual e custa metade.
- Não emita por quadro sem teto. `Emit` num laço de quadro precisa de um limite
  de emissor.
- Reaproveite o `ParticleEmitter` da entidade; não crie um por evento.

### Regra 6 — não transmita uma mensagem por entidade nascida

Se a magia cria N entidades por segundo e cada nascimento vira um broadcast,
são N mensagens por segundo × peers. Alternativas, na ordem:

1. **Determinismo com semente.** Transmita o CAST uma vez, com uma semente, e
   deixe host e cliente sortearem a mesma sequência. Uma mensagem no total.
2. **Lote.** Junte os nascimentos do tique de rede (20 Hz) numa mensagem só.
3. **Só o que dói.** Visual puro pode ser derivado no cliente; só o que resolve
   dano precisa de evento.

### Regra 7 — nada de `rl.Load*` dentro do quadro

Toda textura, shader ou som que a magia usa sobe no **carregamento do mapa**,
não na primeira vez que aparece. Precedentes:
`entity.PreloadEnemyTextures`, `ui.PreloadPortraits`.

Uma `rl.LoadTexture` de 6 MB dentro do quadro custa ~40 ms — e a ultimate é
justamente o momento em que ninguém pode perder um quadro.

---

## Checklist de revisão

Antes de dar uma magia por pronta:

- [ ] Nenhuma assinatura em `internal/skill` recebe `[]rl.Rectangle`.
- [ ] Todo `Draw` novo começa com `visible` / `visibleAny`.
- [ ] O regime (nascimento × vida) está escrito no comentário da constante.
- [ ] Se há interação par a par, os empurrões são acumulados.
- [ ] Emissores de partícula têm teto.
- [ ] O número de mensagens por segundo está contado e justificado.
- [ ] Nenhum `rl.Load*` fora do carregamento de mapa.
- [ ] **Medido no F3**, com a magia ativa, no mapa mais pesado em que ela pode
      ser usada.

## Como medir (F3)

Conjure a magia e leia o painel:

| Linha | O que ela responde |
|---|---|
| `cpu: ... simulacao X` | O custo de host da magia. Deve ficar abaixo de 1 ms. |
| `magias: N desenhadas / M puladas` | Se `M` é 0 num mapa grande, **falta culling**. |
| `resto` | GPU + espera. Sobe com a magia ativa? É fill rate: partículas ou raios grandes em blending. |
| `lixo N MB/s` | Alocação por quadro. Um salto com a magia ativa é `append` em laço quente. |
| `PIOR` | Um pico isolado é carregamento dentro do quadro (Regra 7). |

A regra de aceite: **com a magia no auge, o quadro não passa do alvo** no mapa
mais carregado onde ela pode aparecer (hoje, `world_03` com 156 inimigos ou
`world_07` com o chefe).

---

## Leitura relacionada

- `doc/performance.md` §4⁹⁄₁₀ — o diagnóstico completo das duas ultimates, com
  as capturas medidas.
- `skills/create-character-abilities/SKILL.md` — como escrever a magia
  (camadas, cor, movimento). Este documento é o custo; aquele é a forma.
- `internal/skill/view.go` e `internal/skill/obstacle.go` — os dois ajudantes
  que as Regras 1 e 2 exigem.
