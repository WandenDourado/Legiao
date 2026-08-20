---
name: create-tiled-map
description: Plan and build Tiled maps for Legiao — biome zoning, landmark and path layout, contextual topping scatter, collision and spawn. Use when asked to create a new map or phase, redesign an existing one, or populate an area with terrain, vegetation, buildings, fences or ground detail. Produces a plan for approval first, then the map, then the audit.
---

# Create Tiled Map

Build a place, not a grid of assets. A map reads well when every patch of
ground has a reason to be that material and every detail belongs to what
happens there. This skill turns a brief into a plan, the plan into a map, and
checks the result by measurement.

Companion skill: `create-tiled-assets` owns making and integrating art. This
one owns *where things go*. When the plan needs art that does not exist, ask
for it (see **Pedido de assets**) instead of forcing an existing piece into a
role it does not fit.

## Ordem obrigatória: planejar → aprovar → construir

Never start writing the map file from a brief. Refazer layout custa muito mais
do que discutir um plano. The plan is a short document with five parts:

1. **Leitura do briefing.** What the player must be able to do (start here,
   reach there), and what the place *is* (a cared-for hamlet, an abandoned
   outpost, a forest edge). Everything else follows from that.
2. **Zonas.** Divide the map into named regions with a purpose and an
   approximate cell range: `lote cercado (24-32, 30-38)`, `praça da casa`,
   `estrada`, `orla da floresta`, `ermo`. Every cell of the map should belong
   to a zone; if a region has no purpose, it will look empty in game.
3. **Biomas por zona**, each with a justification (table below).
4. **Elementos por zona**: buildings, trees, rocks, fences, props — with
   counts, not just names. Plus the path network that connects them.
5. **Orçamento de densidade e assets faltando.**

Present the plan, get approval, then build. Only skip the gate if the user
explicitly asks for a direct build.

**Creativity is expected.** The brief is the skeleton, not the ceiling: propose
the woodpile beside the house, the rock outcrop that explains the stone path,
the clearing before the forest. Never contradict the brief, never move what it
fixed, and say in one line why each addition exists.

## Âncora, porta e alinhamento

Quatro fatos do motor que decidem layout. Todos já custaram refação:

- **A porta abre para o SUL.** O footprint de colisão de uma casa fica *acima*
  da âncora (a massa opaca da `_base`: `offsetY -192`, altura 192) e a arte da
  base é desenhada para cima. O chão livre em frente à porta é o que está
  **abaixo** da âncora.
  Logo: casa no fundo (sul) de um lote tapa a saída; casa ao norte com a rua
  ao sul é o arranjo que funciona. Se a rua precisa passar na frente das
  casas, ela vai na linha logo abaixo delas.
- **`add_object` grava `y = (linha + 1) * tile`.** A âncora cai na divisa
  entre a célula pedida e a de baixo. Qualquer conferência que leia
  `obj["y"] // tile` recebe `linha + 1` — inclusive `audit_layout.py`. Ao
  testar se uma célula está livre, teste as duas.
- **Casa não precisa mais de alinhamento de grade.** Já precisou: a colisão era
  quantizada em células de 128 px, então a arte tinha que ficar centrada num
  número inteiro delas ou a margem saía torta. A colisão é retângulo em pixels
  agora (`doc/tilemap.md`), e a casa pode parar onde o layout pedir. Se algum
  script ainda empurrar `x` para múltiplo de 128 por causa de colisão, é
  sobra dessa época.
- **Cerca não se emenda por largura de source.** Várias peças têm 5–7 px de
  margem transparente antes do poste, e emendar por `width` abre vão na junta.
  Use `place_fence.py`, que mede no alfa onde o trilho começa e termina.

## Bioma por função do chão

Terrain is not decoration — it says what happened on that ground. The engine
draws grass as the base and blends dirt/stone over it, so a map is grass plus
what people did to it.

| Terreno | Significa | Usar em |
|---|---|---|
| Grama (1) | chão intocado | ermo, campo, quintal, orla da floresta |
| Terra (2) | chão pisado, trilha de passagem | estradas, caminhos entre casas, currais, área de trabalho |
| Pedra (3) | chão construído, mantido | soleira e entorno de casas, praça, poço, ponte, escadaria |
| Grama escura (4) | mata fechada, bioma corrompido | interior de floresta sombria |
| Grama rala (5) | a mata rareando | onde a mata começa a abrir |
| Terra nua (6) | solo exposto, sem vida | desmatamento, chão morto |
| Grama viva (7) | mata **saudável** | a floresta normal de um mapa de mata |
| Caminho da mata (8) | chão pisado dentro da mata | estrada de terra num mapa de mata |

### Duas pilhas, e a ordem delas é o desenho

Os materiais formam **duas pilhas** (`internal/tilemap/terrain_mask.go`), e
material de pilhas diferentes **nunca se pinta um sobre o outro**:

| Pilha | De baixo para cima |
|---|---|
| Chão verde (a vila) | `1` grama → `2` terra → `3` pedra |
| A mata | `8` caminho → `7` grama viva → `4` grama escura → `5` grama rala → `6` terra nua |

Isso é o que impede um caminho de terra de arrastar um forro de bioma junto.

**A ordem da pilha da mata é a mata adoecendo**: viva, escura densa, rala, terra
nua. Estágios vizinhos são vizinhos na pilha, e é isso que faz o motor rampar a
transição sozinho. Já esteve errada, com a terra nua no fundo: a rala encostava
na viva e a mata parecia perder o mato *antes* de escurecer.

> **Regra que economiza mais trabalho que qualquer outra: transição não se
> pinta, se declara.** Não existe material de transição para inventar nem arte
> de borda para gerar — basta pôr os dois estágios na mesma pilha, em ordem.

**Num mapa de mata, use `7` e não `1`.** A grama da vila está na pilha verde:
usá-la no lado claro faz a fronteira virar **um degrau só**, qualquer que seja
o formato do recorte. Mesma coisa para o caminho — `8`, não `2`, senão ele
imprime um halo verde-claro em volta de si.

### `--band` e `--ramp`: dois problemas diferentes

| | Para quê | Ruído |
|---|---|---|
| `--band LIN,LIN` | Contorno de faixa: entrante, península, ilha | 1D suave, 5–12 células |
| `--ramp ZERO,CHEIA` | **Misturar** dois biomas célula a célula | 2D coerente sobre o mapa |

`--band` desenha uma fronteira ondulada; `--ramp` faz um bioma **tomar** o
outro ao longo de várias linhas.

Numa transição de verdade use `--ramp`. Uma fronteira, por mais ondulada que
seja, é uma curva suave **quantizada em células** — e isso lê como escada. A
rampa entremeia e a escada some.

O ruído da rampa é 2D e coerente **de propósito**. A primeira versão usava 1D
por linha com semente própria em cada uma: cada linha virava uma corrida
horizontal independente, e o empilhamento delas produzia degraus retangulares —
pior que a fronteira reta que ela veio substituir.

```bash
# A mata adoecendo, do bioma mais EXTERNO para o mais interno.
python $S/paint_terrain.py $M --type dark_grass   --ramp '38,31' --seed 77
python $S/paint_terrain.py $M --type sparse_grass --ramp '24,17' --seed 5
python $S/paint_terrain.py $M --type bare_soil    --ramp '8,4'   --seed 13
```

**A ordem das chamadas importa, e erra fácil.** A rampa **satura** além da
linha cheia: dali em diante ela pinta tudo. Rodada depois de um material mais
interno, ela apaga o que já estava lá — numa versão do `world_02` sobraram
**2 células** de 1649 de grama escura. Vá do externo para o interno.

**`tidy_terrain.py` come rampa.** As ilhas de uma célula que ela produz são o
desenho, não ruído. Se precisar do tidy, rode-o **antes** da rampa.

Rules that follow:

- **Toda casa tem chão de pedra na frente**, ainda que pequeno (3×2 células
  bastam): ninguém constrói uma porta abrindo para o mato.
- **Terra conecta**, pedra fica. Uma estrada é de terra; ela vira pedra ao
  entrar no núcleo construído da vila e volta a ser terra ao sair.
- Uma transição de material sempre tem um motivo visível: uma casa, um portão,
  uma ponte. Pedra no meio do nada é erro de layout.
- Não pinte transição na arte. O motor desenha o terreno em **camadas
  empilhadas por prioridade** (grama sob tudo, terra sobre a grama, pedra
  sobre as duas), e cada camada só desbota na borda voltada para algo abaixo
  dela. Materiais podem se encostar à vontade: a pedra se dissolve na terra, e
  a borda de um calçamento lê grama → terra pisada → pedra sozinha.
- **Borda irregular só em chão natural.** `--ragged` e `--jitter` existem para
  campo, clareira e trilha batida. Calçamento, praça e piso de casa são chão
  *construído*: retângulo de borda reta. Ragged em chão construído produz
  retalho — pedaços de grama mordendo o calçamento — e é o erro que mais
  aparece em revisão.
- Ilha de terreno com menos de 5 células é ruído da borda irregular, não
  layout. `tidy_terrain.py` devolve essas células à grama.

## Topping por contexto

Toppings existem para quebrar a repetição da textura, e são o lugar onde o mapa
ganha ou perde credibilidade. A regra é uma só: **o detalhe conta a mesma
história do lugar.**

| Contexto | Sobe | Some |
|---|---|---|
| Sob e ao redor de árvore (raio ~3) | folhas, galhos, cogumelos, folhiço na pedra | grama rala, flor de campo aberto |
| Vila cuidada, perto de casa (raio ~4) | flores, grama aparada | poça, fuligem, rachadura, caco, raiz |
| Dentro de lote cercado / curral | grama seca, terra batida, palha | flores, cogumelo |
| Beira de estrada | cascalho, pedrinhas | nada específico |
| Ermo longe de tudo | rachadura, raiz, mato alto, pedra solta | nada específico |

Two more rules, both learned from bad maps:

- **Frequência, não exclusividade.** Folha perto de árvore é *mais provável*,
  não obrigatória; longe da árvore ela aparece de vez em quando e isso é bom.
  Regra binária produz anel de folhas em volta de cada tronco.
- **Coerência de estado.** Uma vila bem construída não tem pedra rachada com
  poça d'água no calçamento inteiro. Decadência é um recurso narrativo: se o
  lugar não está em ruína, ela aparece em uma ou duas células, ou em nenhuma.

Isso tudo vive em `references/topping_rules.json` como pesos e multiplicadores.
**Ajuste o arquivo por mapa** — uma ruína inverte exatamente esses pesos.

## Densidade: nem salada de frutas, nem deserto

| Medida | Alvo | Verificação |
|---|---|---|
| Grama | 6–10% das células com topping | `audit_layout.py` |
| Terra | 15–22% | idem |
| Pedra | 20–28% | idem |
| Janela 3×3 | no máximo 4 elementos (topping + objeto) | falha na auditoria |
| Peça igual repetida | mínimo 4 células de distância | `scatter_toppings.py` |
| Bloco 8×8 sem nada | nenhum, fora de área bloqueada | aviso na auditoria |

Área alta de tráfego (a célula onde o jogador nasce, a soleira da porta, o vão
do portão) fica limpa de propósito: `keep_clear_radius_around_spawn` para o
spawn, `clear_traffic.py --zone` para as outras.

Objetos grandes (árvore, pedra, casa) contam como elemento na janela 3×3. Três
arbustos e duas árvores no mesmo canto é salada, mesmo com zero toppings.
O espalhamento decide célula a célula e não enxerga a janela inteira, então
`clear_traffic.py` faz o teto valer no fim, tirando o topping que tem mais
vizinhos até a janela caber.

Se um bioma cobre poucas células — um calçamento de 50 células, por exemplo —
quem manda na densidade não é `density`, é `min_same_tile_distance`: com 10
peças disponíveis e distância mínima 4, a área satura muito antes do alvo.
Baixe o espaçamento antes de subir a densidade.

**Nada de vegetação em chão que não é grama.** Mato no meio do calçamento, ou
dentro do footprint de uma casa, denuncia peça posta por coordenada em vez de
leitura do chão. Escolha a célula lendo a camada `ground`; em vila cuidada,
exija ainda uma célula de folga da beira da rua (na orla da floresta, não —
árvore encostada na trilha é o certo). `audit_layout.py` mede isso.

## Ordem de plantio: quem conta a história vem primeiro

Com espaçamento mínimo generoso o mapa **satura**, e quem for sorteado por
último não acha vaga. Medido, e mais de uma vez: dos 6 carvalhos-marco pedidos,
**zero** eram plantados; das 15 árvores mortas, 6; dos 39 props, 7.

Plante nesta ordem:

1. **Marcos** (a maior silhueta, o maior espaçamento)
2. **Moldura de objetivo** — o anel em volta do portal, o círculo de pedras
3. **Peças que contam a história** — árvore morta, ossada, resto de corte
4. **Paredes** de mata
5. **Preenchimento** — o pinheiro comum, arbustos, toppings

Duas armadilhas do espalhador:

- **Sem jitter, tudo cai no centro exato da célula.** Duas peças na mesma célula
  ficam sempre à mesma distância, e o espaçamento mínimo rejeita em bloco.
- **Objetivo precisa de raio livre, como o spawn.** Um `dead_branch_curved`
  caiu a 143 px do portal e o footprint dele — que cobre a arte inteira, por ser
  peça comprida — tapou a própria célula do portal. **O mapa ficou sem saída, e
  nada disso aparece no render**: só o flood fill do `audit_layout.py` viu.

## Ordem de construção

Toppings por último, sempre: eles reagem à posição de tudo o mais.

1. `new_map.py` — mapa vazio com as camadas na ordem que o motor espera e o
   tileset dos toppings já registrado.
2. `paint_terrain.py` — zonas de terreno (retângulo, disco, caminho).
3. `tidy_terrain.py` — tira as ilhas soltas que a borda irregular deixou.
4. Objetos: `place_fence.py` para os lotes; casas, vegetação e props via
   manifesto, seguindo `create-tiled-assets` (âncora, ponto de conexão,
   escala). Vegetação escolhida lendo a camada `ground`, nunca por coordenada
   solta.
5. Colisão: vem dos footprints de manifesto, inclusive a da cerca — cada peça
   tem o dela (`collisionFootprints`), com o vão do portão aberto livre por
   construção. **A camada `collision` é só para o que não é peça de manifesto**
   (borda do mundo, mata fechada); no `world_01` ela está vazia. Pintar célula
   sob uma cerca hoje é bloquear duas vezes: o script já não pinta (`--paint-collision` é legado).
6. Spawn: `player_spawn` em célula livre, com folga em volta.
7. `scatter_toppings.py` — espalhamento contextual.
8. `clear_traffic.py` — limpa as zonas de tráfego alto e faz valer o teto por
   janela.
9. Auditoria e render.

## Gates de aceitação

```bash
S=skills/create-tiled-map/scripts
python $S/audit_layout.py <mapa> --manifest assets/vegetation_manifest.json \
    --manifest assets/buildings_manifest.json --manifest assets/fences_manifest.json \
    --goal <objeto_objetivo>
python $S/render_map.py <mapa> --manifest ... --out /tmp/mapa.png --scale 0.25 --collision
python skills/create-tiled-assets/scripts/validate_map.py <mapa> --manifest ...
go build ./cmd/desktop && go test ./...
```

### Dois kits de topping no mesmo mapa

Um mapa de transição usa o kit da vila no lado claro e o próprio no lado
escuro. `firstgid` por bioma existe para isso — rodar o espalhamento duas vezes
**não compõe**, porque a limpeza da segunda rodada apaga a primeira.

Registre o segundo tileset no mapa, ou os gids dele não resolvem e a camada
`ground_detail` some em jogo justamente nas zonas novas. E confira o índice: um
id além do tileset vira gid que ninguém resolve, **sem erro nenhum** —
`scatter_toppings.py` agora recusa, depois de um mapa ter saído com as peças 36
e 37 num tileset de 32.

### Conferir a transição sem compilar

`render_map.py` desenha borda dura. Para julgar uma **transição de bioma** use
`work/tiled-map-world02/preview_terrain.py`, que espelha o `terrain_renderer.go`
e o shader — inclusive o span. Se a junta está feia lá, está feia no jogo.

`audit_layout.py` responde por medição as quatro perguntas que o olho erra:
o jogador alcança o objetivo a pé (flood fill do spawn com a colisão real),
existe janela poluída, existe vegetação plantada em caminho ou dentro de casa,
existe bloco morto. Em mapa de vila, rode também com `--vegetation-margin 1`
e olhe o que ele aponta: perto de rua é erro, na orla da floresta é acerto.
**Depois olhe o render** — layout é julgado com o mapa inteiro na tela, não
célula a célula.

O gate final é o usuário no jogo, com F3. Nunca declare o mapa correto sozinho.

## Pedido de assets

Quando o plano precisa de algo que não existe, peça — com descrição suficiente
para virar prompt. Formato:

> **Assets que faltam para este mapa**
> - `placa_madeira` — placa de estrada: poste de madeira escura de ~150 px com
>   uma tábua horizontal presa por dois pregos de ferro, madeira e ferragem da
>   mesma família das cercas, sem texto legível na tábua. Usada na bifurcação
>   da estrada. Colide.
> - `poste_lampiao` — poste de ferro de ~220 px com lanterna de vidro fosco no
>   topo, base de pedra igual à das cercas, sem brilho pintado (o efeito é do
>   runtime). Usado nas duas pontas do calçamento. Colide.

Peça no fim do plano, junto com o que dá para fazer sem eles. Se o usuário
aprovar, `create-tiled-assets` gera e integra; o mapa é construído com o que
existe e completado depois.

## Exemplo de plano

Briefing: *"mapa com uma cerca na parte inferior central e uma casa dentro; o
jogador começa dentro da cerca; fora dela um caminho de pedra que vira terra e
leva a uma floresta, que é o caminho da próxima fase."*

> **Leitura.** Vilarejo cuidado na borda de uma floresta. O jogador nasce em
> casa e sai para a estrada; a floresta é o objetivo e precisa se anunciar de
> longe.
>
> **Zonas** (mapa 60×45):
> - `lote` (23–36, 30–42): cerca com portão aberto ao norte e fechado ao sul.
>   A casa fica no **fundo** do lote, ao sul — a porta abre para baixo, então
>   é dali que o jogador sai e atravessa o quintal até o portão. Casa na borda
>   norte taparia a saída.
> - `terreiro` (24–30, 34–38): chão batido em volta da casa, borda reta.
> - `quintal`: o resto do lote, grama.
> - `trilha` (29–30, 29–38): terra, do terreiro até duas células além do
>   portão — o portão é o motivo visível da troca de material.
> - `estrada` (28–32, 19–28): calçamento, borda reta; vira terra na altura da
>   orla e some dentro da floresta, na borda do mapa.
> - `orla` (borda norte inteira): árvores adensando até virar parede de copas
>   na linha 0, com um corredor livre de 3 colunas na trilha.
> - `ermo` (o resto): campo aberto, densidade baixa.
>
> **Biomas.** Terra no terreiro e na trilha (chão pisado todo dia); pedra na
> estrada (chão mantido); grama no resto. As duas trocas têm motivo à vista: o
> portão e o fim do calçamento.
>
> **Elementos.** 1 casa média no lote (x alinhado à grade); cerca por
> `place_fence.py`; 2 arbustos no quintal, longe da trilha; 9 árvores na orla
> (3 de carvalho, 6 de pinheiro, adensando ao norte); 3 tocos marcando a boca
> da floresta; 1 tronco caído como portal visual, se o asset existir.
>
> **Toppings.** Regras padrão, com `vila cuidada` em raio 4 da casa (sem poça,
> sem fuligem) e `sob árvore` em raio 3 (folhiço forte). Estrada em 18%,
> praça em 24%, grama em 8%, ermo em 6%.
>
> **Faltando.** `placa_madeira` na bifurcação e `tronco_caido` na orla — sem
> eles o mapa funciona, mas a entrada da floresta fica sem marco.

## Scripts

| Script | Para quê |
|---|---|
| `new_map.py` | Mapa vazio com ground / ground_detail / structures_back / foreground / collision, todas as camadas de objetos (inclusive `trails` e `portals`) e o tileset dos toppings registrado. |
| `paint_terrain.py` | Pinta terreno por retângulo, disco, caminho ou faixa de bioma (`--band`), com borda irregular (`--ragged`, `--jitter`) — só em chão natural. |
| `tidy_terrain.py` | Devolve à grama as ilhas de terreno soltas deixadas pela borda irregular. |
| `place_fence.py` | Fecha um lote emendando as peças pelos pontos de conexão medidos no alfa e informa o vão do portão. Não pinta colisão: a cerca colide pelo manifesto, e pintar célula por baixo dela bloquearia 128 px onde a peça ocupa ~48. `--paint-collision` é legado. |
| `scatter_toppings.py` | Espalhamento contextual com pesos por bioma, multiplicadores por contexto, espaçamento mínimo e teto por janela (objetos contam). Aceita `firstgid` **por bioma**, para um mapa usar dois kits de topping ao mesmo tempo. |
| `clear_traffic.py` | Limpa zonas de tráfego alto e faz valer o teto de elementos por janela 3×3. |
| `audit_layout.py` | Alcançabilidade do spawn, objetivos, janelas poluídas, vegetação fora de lugar, blocos mortos, densidade por bioma. |
| `render_map.py` | Render do mapa inteiro (terreno, toppings, objetos, colisão, grade, spawn). `--toppings PNG:firstgid` repete, para os dois kits. **Desenha célula fechada**, sem a mistura de borda do shader — julgue LAYOUT aqui e transição no `preview_terrain.py`. |
| `map_utils.py` | Camadas, células, objetos, colisão efetiva. `blocked_cells` **superestima de propósito** — o motor colide contra retângulos, e aqui a célula conta como bloqueada assim que um footprint a toca; é a direção segura para checar se um caminho existe. `footprints_of` lê as duas formas do manifesto. |
| `references/topping_rules.json` | Pesos e contextos padrão — copie e ajuste por mapa. |
