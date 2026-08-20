# Plano: o orc de guarnição do mapa 3

Plano de integração do pacote importado em `work/x320p_Spritesheets`. Documento
de trabalho: quando a implementação acontecer, o que sobreviver dele vira
seção em `doc/combat_rules.md` (regra de território) e comentário de registro
em `internal/entity/enemy_sprite.go` (arte e stats).

---

## 1. O que o pacote é, medido

Não é uma sprite sheet. São **21 animações × 16 direções**, uma folha PNG por
par (animação, direção), em grade de células de **320×320**.

| Fato | Medida |
|---|---|
| Células por folha | 4×4, 5×4, 6×4 ou 6×5 conforme a animação; **nenhuma célula vazia** |
| Direções | 16 arquivos por animação: `000, 022, 045, ... 337` (passo de 22,5°) |
| Camadas | `_Body_` e `_Shadow_` separados, mesma geometria |
| Movimento | **Em lugar fixo.** Em `Walk_Armed_180` o centro horizontal fica entre 158 e 161 nos 20 quadros |
| Pivô | Estável. Linha do pé oscila 203–231 px dentro da célula; centro em x ≈ 160 |
| Ocupação | O orc tem ~**110 px de altura** dentro da célula de 320 |
| Variante em disco | **`WithSkulls`** — crânio pendurado no cinto, placas em ombros e canelas. A pasta `work/x320p_Spritesheets` não diz isso no nome; **renomear para `work/WithSkulls_x320p_Spritesheets`** antes que a informação se perca |

### A célula grande não é desperdício

Parece que a arte foi exportada pequena dentro de uma célula folgada. Não foi:
a célula tem que caber o **arco do espadão**, e no `Attack_01` o conteúdo
alcança x 12–308 dos 320 disponíveis. O corpo parado ocupa um terço da célula
porque o golpe ocupa a célula inteira.

Isso importa para o orçamento: os 110 px do corpo **são** a resolução real
disponível, e nenhum tier do fornecedor melhora isso (§6).

### O ângulo → direção na tela

```
        000 = ↑ (nuca)
   315 = ↖        045 = ↗
270 = ←                    090 = →
   225 = ↙        135 = ↘
        180 = ↓ (rosto)
```

O ângulo cresce **no sentido horário** na tela.

> **Isto já esteve errado duas vezes no código, e as duas vezes o erro passou
> por uma conferência visual.** A leitura decisiva é o `Roar`: nas poses
> curvadas de `Idle` e `Walk` o rosto fica na sombra dos ombros, e costas
> musculosas passam por peitoral com muita facilidade. Em `Roar` a cabeça
> levanta e o focinho sai da sombra. A evidência, verificável em
> `verify_facing.py --anchor`:
>
> | dir | o que se vê | logo |
> |---|---|---|
> | 000 | nuca; ombros e costas da cabeça, **nenhum rosto** | anda para cima |
> | 090 | perfil com olho e focinho para a **direita** | anda para a direita |
> | 180 | rosto; olhos e mandíbula virados para a câmera | anda para baixo |
> | 270 | perfil espelhado do 090, focinho para a **esquerda** | anda para a esquerda |

Isso dá **simetria de espelho perfeita no eixo vertical**: `022↔337`,
`045↔315`, `067↔292`, `090↔270`, `112↔247`, `135↔225`, `157↔202`. Guardar
`000`–`180` (9 arquivos, a metade que varre do norte pelo **leste** até o sul) e
espelhar os 7 do meio reconstrói as 16 direções — a mesma economia que os
personagens jogáveis já fazem com `shouldMirrorWalkRow`.

O espelho troca a mão que segura a espada. O jogo já aceita essa troca nos
personagens; a 170 px de altura ela não se lê.

### O personagem

Bruto humanoide de couraça de placas com espadão de uma mão, postura curvada,
paleta terrosa (couro avermelhado, metal esbranquiçado, lâmina com ferrugem).
Lê bem contra o chão escuro do mapa 3 pela regra da §11.4b de `doc/art_style.md`
— é claro e dessaturado sobre cenário saturado e escuro, que é exatamente o
contrato do guia.

---

## 2. Por que ele "foge do padrão" — e por que isso é uma feature

O slime e o lobo usam `EnemyModeRadial`: **uma** vista superior verdadeira que
o renderizador gira em direção à velocidade. O orc **não pode** usar isso, e a
própria `skills/create-enemy-sprites/SKILL.md` já diz por quê:

> Radial fails on anything whose appeal is a face seen head-on, and on rigid
> shapes (armour, shields, skulls) where a baked highlight visibly spins.

Couraça, escudo de ombro e crânio: girar essa arte faz o brilho embutido rodar
junto e denuncia o truque no primeiro quadro. Ele exige `EnemyModeDirectional`
— que está **declarado e não implementado** em `internal/entity/enemy_sprite.go`.

Três lacunas do motor separam o pacote do jogo:

1. **Modo direcional para inimigo não existe.** `EnemyDef` tem `Columns` mas
   não `Rows`; `drawEnemySprite` lê sempre `y = 0` da folha.
2. **Inimigo não tem máquina de estados de animação.** Ele tem um único loop
   (`advanceEnemyAnim`) e nada mais. Não há ataque, não há flinch, não há
   morte: `TakeDamage` põe `IsActive = false` e o monstro some no mesmo quadro.
3. **A rede não carrega estado de animação.** `EnemyState` tem posição e vida,
   e o cliente deriva o facing do delta de posição (`trackRemoteEnemy`). Isso
   basta para um slime girando; não basta para um golpe que precisa acertar no
   mesmo quadro nos dois lados.

Nenhuma das três é opcional. Elas são o plano.

---

## 3. Quais animações fazem sentido

Veredito sobre as 21, com o motivo:

### Entram

| Animação | Quadros | Papel no jogo |
|---|---|---|
| `Idle_Armed` | 16 | Guarnição parada no posto. É o estado padrão do mapa 3: o monstro **já está em campo** quando o grupo entra |
| `Walk_Armed` | 20 | Perseguir dentro do setor e voltar ao posto ao desengajar |
| `Attack_02` | 30 | O golpe. Escolhido entre as três por leitura: o arco é o mais telegrafado (levanta acima do ombro, corta na diagonal) e tem faísca de impacto num quadro — o jogador consegue **aprender a esquivar**, que é o que uma guarnição estática pede |
| `Hit_Armed` | 20 | Flinch. Sem ele, bater no orc não dá retorno nenhum |
| `Death_Armed` | 24 | Hoje o inimigo pisca e some. Um bruto que cai devia cair |

### Ficam de fora agora, com motivo

| Animação | Por quê |
|---|---|
| `Attack_01`, `Attack_03` | Variantes do mesmo golpe. Custam 1 Mpx cada por variedade que só aparece quando houver mais de um orc na tela. Segunda leva |
| `Run_Armed` | O orc de guarnição não corre: ele **guarda**. Um perseguidor lento é o que dá ao jogador a opção de atravessar o vão. Se um capitão de elite existir, este é o sprite dele |
| `Roar` | Bom candidato para o grito de engajamento ao cruzar a borda do território, mas é adorno antes de a IA de território existir. Terceira leva |
| `Idle_Unarmed`, `Death_Unarmed`, `Hit_Unarmed`, `Jump_*`, `Run_Unarmed`, `Walk_Unarmed` | Existem para um estado desarmado que o jogo não tem |
| `Idle_Block`, `Hit_Block`, `Idle_Transition_To_Block`, `Hit_Block_To_Idle`, `Idle_Get_Weapon` | O pacote está oferecendo uma **mecânica de desarme**: golpe pesado derruba a espada → o orc bloqueia ou corre buscá-la. É bonito e é um sistema de combate novo, não uma animação. Anotado, não planejado |
| `_Shadow_` (todas) | O jogo não desenha sombra de inimigo. Usar dobraria a memória por uma mancha que o chão escuro do mapa 3 quase engole. Descartar; se algum dia houver sombra, um blob procedural custa zero |

---

## 4. Onde ele entra no mapa 3 — o buraco já está aberto

Isto não é encaixe forçado. O código **pede** este monstro por nome:

```go
// internal/network/wave_runs.go
// TODO(monstros): world_03 ainda nao declara corrida ...
// o mapa 3 usa `enemy_post_*` e `climax_spawn_*` de proposito, porque a
// jogabilidade dele e de guarnicao e nao de horda.
// Quando os monstros da fortaleza existirem, a corrida do climax entra aqui.
```

E `doc/combat_rules.md` §"Defesa de território (mapa 3)" já descreve a regra
inteira, sem uma linha de Go que a execute:

- **23 marcadores `enemy_post_*`** em `assets/maps/world_03.json`, camada
  `spawn`. Nenhum arquivo `.go` lê esse prefixo — `internal/tilemap/spawn.go`
  só conhece `enemy_spawn`.
- **5 retângulos `territorio_*`** na camada `zones`, tier 1 a 5 do sul para o
  norte. Também não lidos.
- **9 marcadores `climax_spawn_*`** para a emboscada da fortaleza.

Consequência hoje: o mapa 3 roda sem nenhum inimigo, o portal já nasce
liberado, e a cena `world_03_climax` (que existe em `assets/dialogues/world_03.json`,
com a Sacerdotisa consagrando a esplanada) **nunca toca**.

O orc é quem fecha esse circuito.

---

## 5. Plano de implementação

Seis fases. Cada uma termina em algo verificável; nenhuma depende de a
seguinte existir.

> **Estado em 07/08/2026.** As Fases 0 e 1 estão feitas, e a Fase 2 começou:
> `Idle_Armed` e `Walk_Armed` instaladas (2,63 Mpx = 11 MB a 1×), modo
> direcional funcionando, `enemy_post_*` lido, 23 orcs em campo no mapa 3, e a
> troca idle↔walk em `internal/entity/enemy_anim_state.go`. Falta ataque,
> flinch, morte, IA de território e rede. **A Fase 0a virou urgente**: a
> caminhada está em uso.

### Fase 0a — Resolver o `FIX_Walk_Armed` antes de tocar em qualquer coisa

O fornecedor publica um `FIX_Walk_Armed.zip` (43 MB) — uma correção que existe
**exatamente para uma das cinco animações escolhidas**. Não há como saber, pelo
que está no disco, se o `Walk_Armed` de `work/` é o corrigido ou o defeituoso.

Medi o que dá para medir e não achei anomalia: as 16 direções têm 20 quadros
cada, a linha do pé fica em 201–231, e a deriva horizontal é **perfeitamente
simétrica entre os pares espelhados** (022↔202 = 8,5 px; 045↔225 = 7,0;
067↔247 = 6,0; 090↔270 = 4,5; 135↔315 = 1,5; 157↔337 = 2,5). Um pacote com
direção quebrada não teria essa simetria. Mas isso não descarta um defeito
visual — pé deslizando, espada atravessando a perna, quadro repetido.

**Gate:** baixar o `FIX_Walk_Armed.zip` e comparar arquivo a arquivo. As somas
MD5 das folhas atuais, para essa comparação:

| Direção | MD5 (`Walk_Armed_Body_*`) | | Direção | MD5 |
|---|---|---|---|---|
| 000 | `02b061c8` | | 180 | `5d1579f7` |
| 022 | `2baf085d` | | 202 | `ab1bf894` |
| 045 | `fef4b1be` | | 225 | `fea9dd3a` |
| 067 | `db272b95` | | 247 | `819e2b24` |
| 090 | `1f641ed3` | | 270 | `45c8d8e2` |
| 112 | `792b4b4a` | | 292 | `478f56ec` |
| 135 | `bf6e2cad` | | 315 | `54ccbaf8` |
| 157 | `587aa18a` | | 337 | `19d0bb18` |

(prefixo de 8 dígitos do MD5 do bitmap decodificado, não do arquivo — imune a
recompressão do PNG)

Descobrir isso depois de construir as folhas custa uma reexecução do
`build_orc.py`. Descobrir depois da Fase 4 custa uma sessão inteira procurando
um bug de animação que nunca esteve no código.

### Fase 0 — Ferramenta offline: `work/orc-guarnicao/build_orc.py`

Um script, reproduzível, no mesmo espírito de `work/tiled-assets/build_*.py`:
a folha é o script que a gera.

Ele deve:

1. Ler as 9 direções `000`–`180` das 5 animações escolhidas.
2. **Subamostrar quadros** (ver tabela da §6) mantendo a duração total: pular
   1 em 2 e dobrar `FrameTime` preserva o ritmo original.
3. **Medir** a caixa de recorte por animação (união dos bboxes dos quadros
   daquela animação, em todas as direções guardadas) — nunca escolher um
   número redondo.
4. **Ampliar 2× com Lanczos** e montar uma folha por animação, linhas =
   direções, colunas = quadros.
5. Emitir `assets/sprites/enemies/orc/orc_manifest.json` com o que foi medido:
   caixa por animação, `FrameWidth`/`FrameHeight`, `Columns`, `Rows`, o
   deslocamento do recorte em relação ao pivô da célula, e a **`FootLine`**
   (linha do pé dentro do quadro recortado — o mesmo papel que
   `CharacterDef.FootLine` faz para o jogador).

O manifesto é obrigatório, não opcional: com recortes diferentes por animação,
trocar de `Idle` para `Attack` sem o deslocamento medido faz o orc **saltar**
no quadro da transição. É a mesma lição de `create-tiled-assets` — manifesto
medido, nunca aritmética de grade.

Validação da fase: um GIF de contato por animação, e um teste que confirma que
o pé fica na mesma linha de mundo nas 5 animações.

### Fase 1 — `EnemyModeDirectional` no renderizador

`internal/entity/enemy_sprite.go`:

- `EnemyDef` ganha `Rows`, e a geometria passa a ser **por animação**, não por
  inimigo. Sugestão: `EnemyDef.Anims map[EnemyAnim]EnemyAnimDef` carregado do
  manifesto, com `EnemyAnim` sendo `AnimIdle | AnimWalk | AnimAttack | AnimHit
  | AnimDeath`.
- Uma textura por animação, cache preguiçoso como o atual (`enemyTexture` vira
  chaveado por `(tipo, animação)`).
- `enemyRowForHeading(vx, vy) (row int, mirror bool)`: 16 setores de 22,5°,
  devolve linha 0–8 e se espelha. Isso é o análogo de
  `player_sprite_direction.go` e merece o mesmo tratamento — arquivo próprio,
  tabela-verdade em teste.
- `drawEnemySprite` passa a ler `y = row * FrameHeight`, a negar
  `source.Width` quando `mirror`, e a **ancorar pelo pé** em vez do centro,
  reusando a ideia de `GroundOffset`/`GroundPoint` de `character_ground.go`.

Cuidado a registrar no código: `Mode == EnemyModeDirectional` **não pode**
passar pelo caminho de `Angle`/`TurnRate`. Girar uma sprite direcional é
exatamente o bug que o modo existe para evitar.

Validação: o slime e o lobo continuam idênticos (`go test ./internal/entity/...`
mais uma partida). Regressão no radial aqui é o risco real desta fase.

### Fase 2 — Máquina de estados de animação do inimigo

Arquivo novo: `internal/entity/enemy_anim_state.go` (não engordar `enemy.go`,
que já passa de 400 linhas).

```
Idle ──alvo no setor──> Walk ──em alcance──> Attack ──fim──> Idle
  ▲                       │                     │
  └───────────────────────┴──dano──> Hit ───────┘
                                       │
                       vida ≤ 0 ──> Death (uma vez, sem loop)
```

Três regras que decidem se isto vai parecer certo:

- **O golpe acerta num quadro, não no fim do cooldown.** `AttackCooldown`
  hoje devolve `true` de `Update` e o dano sai imediatamente. Com animação, o
  dano tem que sair no **quadro de impacto** (o da faísca em `Attack_02`), e o
  orc fica travado no lugar durante o golpe. Isso é o que torna o telegrafo
  jogável em vez de decorativo.
- **`Hit` não pode cancelar `Attack`.** Um bruto que perde o golpe toda vez
  que leva dano vira inofensivo com dois jogadores atirando.
- **`Death` adia a remoção.** `TakeDamage` continua zerando a vida, mas
  `IsActive` só cai no último quadro da morte. Todo caminho que percorre
  inimigos (`ResolveEnemyOverlap`, alvo de projétil, contagem de horda) precisa
  parar de mirar num orc morrendo — vale um `IsTargetable()` explícito em vez
  de reusar `IsActive` para duas perguntas diferentes.

### Fase 3 — IA de guarnição: posto e setor

O que `doc/combat_rules.md` já promete e ninguém executa.

- `internal/tilemap/spawn.go`: ler o prefixo `enemy_post_` e os retângulos
  `territorio_*` de `zones`.
- **O setor sai da geometria, não do nome.** O documento é explícito: o setor
  de cada posto é o retângulo `territorio_*` que **contém** o ponto. Guardar o
  mesmo dado duas vezes é como as duas metades divergem.
- `Enemy` ganha `HomePost rl.Vector2` e `Sector rl.Rectangle`.
- Sistema novo em `internal/system/` (a IA é regra de jogo, não desenho):
  perseguir quem entra no setor; ao sair, parar na borda e caminhar de volta ao
  posto **mantendo o dano levado**. O documento é claro que não curar é a regra,
  não um esquecimento — recuar e voltar é desgaste legítimo.

**Esta fase também é a correção do amontoamento**, e isso só ficou visível
depois de os orcs entrarem em campo. Com 23 deles convergindo no mesmo jogador,
a direção de separação (peso `enemySeparationWeight` 1.6) domina a direção de
perseguição e produz deriva lateral constante — um empurrão de módulo 2 já
supera o vetor unitário do alvo. O olhar já foi desacoplado disso
(`Enemy.Facing`), então eles não perseguem mais de costas; o que continua é o
corpo derivando.

Baixar o peso de separação seria tratar o sintoma, e mexeria no slime e no lobo,
que não têm o problema. A causa é o mapa inteiro cair em cima do grupo de uma
vez, e a regra que o documento de combate já descreve — cada orc preso ao
próprio setor — é o que impede isso por construção.

### Fase 4 — Rede

`EnemyState` precisa de dois campos novos em `internal/network/protocol.go`:

```go
Anim  string  `json:"anim,omitempty"`   // "idle"|"walk"|"attack"|"hit"|"death"
Frame int     `json:"frame,omitempty"`  // quadro dentro da animação
```

O facing continua derivado do delta de posição no cliente (`trackRemoteEnemy`),
que já funciona e não custa banda. Mas **animação não pode ser derivada**: o
cliente não tem como adivinhar que o orc começou um golpe, e um golpe que sai
em quadros diferentes nos dois lados é a coisa que mais denuncia dessincronia.

Ponto a medir e não presumir: com 20 orcs em campo, dois campos por inimigo por
snapshot. Se pesar, `Frame` pode virar `uint8` num formato binário — mas medir
antes.

### Fase 5 — Registro e balanceamento

`RegisterEnemy` de `EnemyTypeGarrison` em `enemy_sprite.go`, no mesmo estilo
comentado do lobo — o comentário deve dizer **qual duelo** os números decidem,
não repetir os números.

Ponto de partida a afinar em jogo:

| Stat | Valor inicial | Por quê |
|---|---|---|
| `Health` | 220 | Aguenta mais que o slime (100) e muito mais que o lobo (40). Guarnição é obstáculo, não presa |
| `Speed` | 130 | **Abaixo dos 200 do jogador**, de propósito. Se ele alcançasse, atravessar o vão deixaria de ser opção e a fase viraria horda |
| `AttackDamage` | 30 | Um golpe é evento. Dois seguidos sem esquivar é quase metade da vida |
| `AttackRange` | 70 | Alcance do espadão, maior que o do lobo (30) |
| `AttackCooldown` | 1.8 | Lento o bastante para o telegrafo de `Attack_02` caber inteiro dentro dele |
| `Radius` | ~45 | **Medir** contra a arte, como `EnemySlimeRadius`/`EnemyWolfRadius`. E lembrar de `doc/colisão`: a caixa fica nos **pés**, dentro da arte, não transbordando |

Depois: `waveRuns["assets/maps/world_03.json"]` deixa de ser um `TODO` e passa
a declarar a **corrida do clímax** — a emboscada dos `climax_spawn_*`, que é a
única horda do mapa 3. Só com ela o `on_last_stand` volta a ser avaliado e a
cena da Sacerdotisa consegue tocar.

### Fase 6 — Verificação

- `go test ./...` — em especial `internal/entity` (radial não regrediu) e
  `internal/game/progression_test.go`.
- Teste novo de tabela para `enemyRowForHeading`: 16 direções × espelho.
- Um teste que carrega `orc_manifest.json` e confirma que toda animação
  declarada tem folha no disco com as dimensões que o manifesto afirma. É a
  checagem que impede o manifesto de mentir depois de alguém reexecutar o
  `build_orc.py` com outro parâmetro.
- Sessão de jogo no mapa 3: engajar, atravessar o vão, confirmar o desengajar
  sem cura, chegar na `fortaleza` e ver o clímax tocar.
- Build Android (`skills/legiao-android-build`) **é gate**, não formalidade —
  ver §6.

---

## 6. O orçamento medido, e o risco que ele carrega

Medido nas 9 direções escolhidas, com a subamostragem proposta:

| Animação | Direções | Quadros/dir | Total | Caixa do quadro | Mpx @1× | Mpx @2× |
|---|---|---|---|---|---|---|
| `Idle_Armed` | 9 → 16 | 8 de 16 | 72 | 108×116 | 0,90 | 3,61 |
| `Walk_Armed` | 9 → 16 | 10 de 20 | 90 | 90×117 | 0,95 | 3,79 |
| `Attack_02` | 9 → 16 | 10 de 30 | 90 | 154×156 | 2,16 | 8,65 |
| `Hit_Armed` | 5 → 8 | 7 de 20 | 35 | 119×125 | 0,52 | 2,08 |
| `Death_Armed` | 5 → 8 | 10 de 24 | 50 | 196×143 | 1,40 | 5,61 |
| **Total** | | | **337** | | **5,93** | **23,7** |

Em memória de textura RGBA: **24 MB a 1×, 95 MB a 2×.**

Para comparar: **os cinco personagens jogáveis juntos ocupam 20 MB.** O orc a
2× custaria quase cinco vezes o elenco inteiro.

`Hit` e `Death` levam só 5 direções (+espelho = 8) de propósito: um flinch dura
3 quadros e uma morte acontece uma vez. Precisão angular ali não se lê, e são
as duas animações com as caixas mais caras.

### A recomendação: 2× é o alvo, mas com portão

Você escolheu pré-ampliar 2× offline com Lanczos, e o motivo é bom — é a regra
que o próprio código já segue ("quadros autorados maiores do que são
desenhados"), e é o que corrigiu o lobo. Mas os 95 MB só apareceram depois da
medição. Então:

1. **Construir a 1× primeiro** (`RenderScale` ≈ 1,55 → orc com ~170 px, à
   altura da Paladina). 24 MB.
2. **Olhar na tela.** A pergunta é uma só: a ampliação bilinear de 1,55× está
   visível como moleza nas bordas da couraça?
3. **Se estiver, subir a 2× só `Idle`, `Walk` e `Attack`** — as três que ficam
   na tela o tempo todo e perto. `Hit` e `Death` seguem a 1×. Isso dá
   **18,0 Mpx = 72 MB**, contra 95 do pacote inteiro a 2×.

`build_orc.py` deve receber a escala **por animação** como parâmetro, para que
esse degrau seja uma reexecução e não uma reescrita.

### As três variantes: a escada de guarnição sai de graça — em trabalho

O fornecedor entrega o mesmo personagem em **`NoArmor`, `NoSkulls` e
`WithSkulls`**. Mesmo rig, mesmas 21 animações, mesma grade, mesmas direções.
Ou seja: o manifesto, o mapeamento de linha por direção, a máquina de estados e
a IA — tudo das Fases 1 a 4 — servem às três sem uma linha a mais.

E elas caem exatamente em cima do que o mapa 3 já pede. `doc/combat_rules.md`
diz que o setor é "a unidade de **dificuldade**: quanto mais ao norte, mais
guarnição", com `tier` 1 a 5. Uma escada visual óbvia:

| Variante | Papel | Onde |
|---|---|---|
| `NoArmor` | Bruto raso, o que morre rápido | Setores tier 1–2, ao sul |
| `NoSkulls` | Guarnição de verdade | Tier 3–4 |
| `WithSkulls` | Veterano; os crânios são patente | Tier 5 e a emboscada do clímax |

**Mas de graça só em trabalho, não em memória.** Textura não se divide por
quantos inimigos aparecem: três variantes são 3× a tabela acima, ou seja
**72 MB a 1×**, que é o mesmo preço do plano de uma variante com ampliação 2×
parcial (§ acima).

Então a decisão real do projeto é essa, e é um ou outro pelo mesmo orçamento:

> **Três orcs diferentes e um pouco moles, ou um orc nítido.**

Recomendação: **uma variante agora**, a `WithSkulls` que já está no disco, e
`build_orc.py` recebendo a variante como parâmetro desde o primeiro dia. Assim
a escada de tiers é uma reexecução do script, decidida depois que houver um orc
rodando no mapa 3 e um número de Android real na mão — e não antes, com base em
como a ideia soa.

### O risco que decide tudo: Android

72–95 MB de textura para um monstro é o tipo de número que passa no desktop e
mata o APK. O build Android tem que ser gate da Fase 0, não da Fase 6: se o
orçamento não couber, o que muda é a **tabela acima** (menos direções em
`Attack`, menos quadros em `Death`), e mudar isso depois das Fases 1–5 é
barato; descobrir depois do release não é.

### A saída fácil não existe: 320p é o teto

O catálogo do fornecedor foi conferido. Os tiers são **180p, 256p e 320p** — e
só. Não há `x640p`. O que está no disco já é a maior resolução que esse
personagem tem, e os outros dois tiers são o mesmo render reduzido.

Isso fecha a questão: **a ampliação Lanczos offline é o único caminho**, não
uma preferência. O portão de medição acima deixa de ser cautela e vira o
mecanismo que decide o orçamento.

O formato também já é o certo. O pacote existe como `Spritesheets` (grades, 42
MB) e como `Frames` (PNGs soltos, 113 MB). São os mesmos pixels; `Frames` só
ajudaria um empacotador que medisse quadro a quadro, e o plano usa caixa
uniforme por animação. Nada a baixar aí.

Fica anotado um item barato que pode ser útil um dia:
`WithSkulls_Spritesheets_FrontView` (9,1 MB) é o análogo do `reference.png` dos
personagens. Não serve para nada hoje — inimigo não tem retrato — mas se
aparecer bestiário ou ícone de HUD, é essa folha e ela é barata.

---

## 7. O que ficou anotado e não planejado

- **Desarme.** `Idle_Block`, `Hit_Block`, `Idle_Get_Weapon` e as variantes
  `_Unarmed` são um sistema inteiro esperando: golpe pesado derruba a espada, o
  orc bloqueia ou corre buscá-la. Seria o gancho de um **capitão** no clímax do
  mapa 3, não da guarnição comum.
- **Capitão de elite.** Agora tem cara: a variante `WithSkulls` com `Run_Armed`
  e `Roar`, contra uma guarnição `NoSkulls`. Segunda `EnemyDef`, mesma
  geometria, custo de trabalho quase zero depois da Fase 1 — o custo é de VRAM
  e está medido acima.
- **`Roar` como grito de engajamento** ao cruzar a borda do território. É o que
  faria "ele pertence a um lugar" ser algo que o jogador **ouve** em vez de
  algo que o documento afirma.
