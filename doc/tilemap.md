# Tilemap

Use this doc for Tiled map files, tilesets, map loading, render layer order, and collision objects.

## Runtime Map

Runtime starts on:

```text
assets/maps/world_01.json
```

and swaps to any other map through a portal (see "Portais").

Maps and tilesets needed at runtime must live under `assets/` so Android can bundle them.

Current expected layout:

```text
assets/
  maps/
    world_01.json
    world_02.json            # mapa de validacao do bioma escuro
    world_03.json            # mapa 3: da mata a fortaleza (materiais 9 e 10)
  tilesets/
    terrain_grass.png        # pilha verde: base do terreno (material 1)
    terrain_dirt.png         # material 2
    terrain_stone.png        # material 3
    terrain_dark_grass.png        # pilha escura: grama cheia (material 4)
    terrain_dark_grass_sparse.png # grama rala (material 5)
    terrain_bare_soil.png         # terra nua (material 6)
    terrain_siege_gravel.png      # material 9  (mapa 3: pátio de obra da fortaleza)
    terrain_dark_flagstone.png    # material 10 (mapa 3: laje da esplanada)
    terrain_trail.png             # fita da trilha, RGBA (alpha = perfil do caminho)
    forest_trees.png              # atlas do forest_manifest (carvalho grande)
    forest_pine.png               # atlas do forest_pine_manifest (pinheiro dourado)
    terrain_toppings.tsx     # unico tileset referenciado pelo mapa (firstgid 400)
    terrain_toppings.png
    village_buildings.png    # atlas do buildings_manifest
    village_fence_v2.png     # atlas do fences_manifest
    fortress_wall.png        # atlas do fortress_wall_manifest (mapa 3: muralha, portao, torre)
    siege_defenses.png       # atlas do siege_defenses_manifest (mapa 3: barricadas, estacas, braseiro)
    terrain_toppings_fortress.png/.tsx  # 3o kit de topping (firstgid 464): cascalho e laje
    village_vegetation.png   # atlas do vegetation_manifest
    *_source.png             # folhas geradas, nunca sobrescritas (trilha de auditoria)
```

Atlas de manifesto **não tem `.tsx`**: o recorte vem do manifesto, não de
aritmética de grade. Só o tileset de toppings, que é de tiles 1×1 de verdade,
precisa de `.tsx` e de entrada em `tilesets` no mapa.

## Ground Detail (toppings)

`terrain_toppings.tsx` (32 tiles de 128 px, `firstgid` 400) quebra a repetição
do terreno: 8 variações de grama, 8 de terra, 8 de pedra e 8 sutis. São tiles
comuns na camada `ground_detail`, passáveis, espalhados por bioma com
`work/tiled-assets/scatter_toppings.py` (grama 5,5%, terra 16%, pedra 22%;
nunca sobre célula ocupada nem sobre o spawn). Cada tile tem uma propriedade
`name` no TSX para facilitar a escolha no Tiled.

## Terreno em camadas

A camada `ground` guarda um material por célula e o motor desenha isso em
**camadas empilhadas**, não célula a célula: uma célula recebe o próprio
material e todos os que estão abaixo dele na mesma pilha. Cada camada só desbota
nas bordas voltadas para algo abaixo dela (`terrain_mask.go`), e o shader
devolve a máscara como alpha, então a camada de cima se dissolve sobre a de
baixo que já está na tela.

Isso é o que faz uma junta terra↔pedra funcionar. Com cada célula desenhada
como um quad fechado sobre grama, os dois lados recuavam da borda comum e a
grama aparecia no meio da junta. Empilhado, a borda de um calçamento lê
grama → terra pisada → pedra, e a trilha encosta no calçamento sem costura.

### As pilhas

`terrainStacks` tem uma pilha por chão. Material de pilhas diferentes nunca se
pinta um sobre o outro, então um caminho de terra não arrasta um forro de bioma
junto.

| Pilha | De baixo para cima | Rampa de borda |
|---|---|---|
| Chão verde (a vila) | `1` grama → `2` terra → `3` pedra | 0.34 de célula |
| A mata (mapa 2+) | `8` caminho → `7` grama viva → `4` grama escura → `5` grama rala → `6` terra nua → `9` cascalho de cerco → `10` laje negra | 0.50 de célula |

Os dois últimos são chão **construído** e mesmo assim moram na pilha da mata
(mapa 3, a fortaleza). Uma pilha própria teria terminado a esplanada num degrau
duro contra a terra morta em volta; nesta, a laje se dissolve no cascalho e o
cascalho na terra nua sozinhos. O preço é herdarem a rampa de 0.50, que é fade
de bioma e não a borda feita de um calçamento de vila — a borda reta do lugar
vem da arte da muralha apoiada na divisa, não do terreno.

A rampa é da pilha, não global. Estrada e calçamento têm borda feita e ficam
com a rampa curta; bioma não termina numa linha, então desbota sobre meia
célula e a borda deixa de ler como escada de células inteiras.

**A ORDEM da pilha da mata é o desenho, não detalhe de implementação.** Ela é
a mata adoecendo, e nessa ordem: grama viva, grama escura densa, grama rala,
terra nua. Atravessar o mapa lê como *uma* floresta se apagando, porque
estágios vizinhos são vizinhos na pilha.

Já esteve errada: a terra nua ficava no fundo, o que punha a grama **rala**
encostada na viva — a mata parecia perder o mato antes de escurecer, de trás
para frente.

Duas posições que custaram bug e valem entender:

- **`7` grama viva mora na pilha da MATA, não na verde.** A grama da vila (`1`)
  está na pilha verde, e material de pilhas diferentes nunca se pinta um sobre
  o outro. Enquanto o lado claro de um mapa de transição usava a grama da vila,
  a fronteira era **um degrau só**, qualquer que fosse o formato do recorte.
  Nenhum ruído conserta isso; a pilha conserta.
- **`8` caminho fica ABAIXO da grama.** Caminho é onde a grama não está, então
  a grama desbotando na borda dissolve *dentro* dele — que é o que uma beira
  pisada parece. Acima da grama ele teria todos os estágios do bioma pintados
  por baixo e imprimiria um anel escuro em volta de si; na pilha verde (como
  `2` terra) ele desbota contra a grama da **vila** e imprime um halo verde
  claro, que é o defeito que motivou o material próprio.

Um bioma novo é: uma constante de material, uma textura em `terrainTextureFiles`
e uma pilha em `terrainStacks` — mais uma entrada em `terrainOverlays`, na
ordem em que ele deve ser pintado.

### Textura cobrindo várias células (`span`)

`drawPlain` já espremeu a textura inteira dentro de **uma** célula de 128 px.
Uma folha de 1254 px perdia ~60% do contraste local no caminho até a tela, e
toda célula de um material ficava idêntica à vizinha.

Subir a resolução da arte **não resolve**: o gargalo é o destino de 128 px, não
a origem. O que resolve é a mesma textura cobrir N×N células, com cada célula
recortando a janela da sua posição — pixels 1:1, e o padrão passa a repetir a
cada N células em vez de a cada uma.

| Onde | O quê |
|---|---|
| `spanMaterials` | Quem participa. Só o bioma escuro: dar span à grama da vila dobraria o tamanho de cada folha no `world_01`, que é mapa aprovado. |
| `spanFor` | O span é **derivado do tamanho da textura** (512 px contra célula de 128 = 4×4), não declarado. Arte de resolução maior sobe o span sozinha; arte pequena nunca é ampliada. |
| `tileWindow` | O recorte da célula, em pixels e em UV. |
| uniform `tileRect` | Nos dois shaders. |

O `tileRect` existe porque `fragTexCoord` fazia **dois** trabalhos ao mesmo
tempo: amostrar a textura e calcular a máscara de borda. Recortar o `src` sem
separar os dois apagaria o desbotamento entre materiais. Com span 1 ele é
`(0,0,1,1)` e o comportamento antigo é reproduzido exatamente.

Textura de bioma nova entra a **512 px** — reduzir para 256 derruba o span para
2×2 e joga metade do ganho fora.

| File | Responsibility |
|---|---|
| `internal/tilemap/terrain_renderer.go` | Passes de desenho do terreno e texturas por material. |
| `internal/tilemap/terrain_mask.go` | Pilhas de material, rampa de borda e máscara de 8 vizinhos. |
| `assets/shaders/terrain_blend_*.fs` | Recebe `edge`/`corner`/`edgeWidth` e devolve o material com alpha na borda. |
| `work/tiled-map-world02/preview_terrain.py` | Renderiza a camada `ground` fora do jogo, com a mesma máscara — para conferir uma transição sem compilar. |

Vizinho fora do mapa conta como **ligado**: não há chão lá fora para dissolver,
então desbotar na borda do mundo só expunha a base de grama como uma moldura.

Limite conhecido: a camada `ground` guarda um material por célula, então uma
célula não registra em que bioma ela está. Terra (`2`) ou pedra (`3`) cruzando
a mata ainda mostra a base de grama clara na junta, porque elas são da pilha
verde.

**A saída já existe e é o `8` caminho:** um bioma que precisa de chão pisado
declara o dele na **própria pilha**, em vez de emprestar o da vila. Foi assim
que o halo verde em volta da estrada do mapa 2 sumiu. Mais um material custa
uma constante, uma textura e uma posição na pilha; uma camada de bioma separada
custaria muito mais e ainda não foi necessária.

## Loader Files

| File | Responsibility |
|---|---|
| `internal/tilemap/tilemap.go` | Tiled JSON/TMJ structs, map loading, tileset resolution. |
| `internal/tilemap/tsx.go` | External TSX parsing. |
| `internal/tilemap/renderer.go` | Tileset texture loading and layer rendering. |
| `internal/tilemap/collision_grid.go` | Tiled collision grid and projectile rectangles. |
| `internal/tilemap/spawn.go` | Player spawn object lookup. |
| `internal/tilemap/filereader_desktop.go` | Desktop file reads. |
| `internal/tilemap/filereader_android.go` | Android APK asset reads via raylib. |

## Android Asset Rule

Do not use `os.ReadFile` directly for runtime map/tileset data. Use the tilemap `readFile()` abstraction so Android can read from APK assets.

Texture paths still go through `assets.Path()` before raylib loading.

## Asset Manifests

Vegetation and buildings are rendered from explicit manifests, not from
implicit atlas cell arithmetic. `manifestSources` in
`internal/tilemap/vegetation.go` pairs each manifest with the Tiled
objectgroup layer whose named objects it draws:

| Manifest | Object layer |
|---|---|
| `assets/vegetation_manifest.json` | `vegetation` |
| `assets/forest_manifest.json` | `vegetation` |
| `assets/forest_pine_manifest.json` | `vegetation` |
| `assets/buildings_manifest.json` | `buildings` |
| `assets/fences_manifest.json` | `fences` |

Dois manifestos podem dividir a mesma camada: `Draw` ignora objeto cujo nome não
está no manifesto dele. Foi assim que a árvore grande entrou sem tocar no
`village_vegetation.png`, que o world_01 usa.

Adding a category is one entry there plus its manifest file. Each manifest
identifies its atlas by resolved path (so the generic tile renderer skips
those GIDs) and supplies every explicit collision footprint; CollisionGrid
merges them with the `collision` layer and never relies on a GID range.

### Footprint é retângulo, não célula

**Um footprint de manifesto é medido em pixels contra a arte e colidido em
pixels.** `footprintIndex` guarda o retângulo e o indexa nas células que ele
toca; a busca consulta só as células da caixa consultada, então o custo é da
caixa e não do número de props no mapa.

Já foi quantizado na grade de 128 px — a célula virava sólida com 50% de
cobertura — e era isso que fazia a colisão deixar de bater com o desenho.
Somando o arredondamento à folga que os footprints declaravam, a casa pequena
bloqueava 34 px de grama de cada lado e **64 px na frente da própria porta**.
A cerca era pior: célula cheia de 128 px para um corrimão de ~40 px, pintada à
mão na camada `collision`.

Duas regras que vieram daí:

- **O footprint fica dentro da arte.** Chão que o jogador vê livre e não pode
  pisar é o defeito. `validate_manifest.py` falha com mais de 8 px fora dos
  pixels opacos da peça.
- **Peça que não cabe num retângulo usa `collisionFootprints` (plural).** Um
  canto de cerca é um L e um portão aberto é dois postes com vão no meio;
  descrever qualquer um deles como uma caixa só bloqueia o miolo vazio do canto
  ou fecha o portão. A forma singular `collisionFootprint` continua válida.

A casa é a `_base` recortada na linha do beiral, e o bloco é exatamente a massa
opaca dela: do topo da parede ao pé, sem folga. Beiral, copa e hera flutuam
acima do chão e nunca colidem.

**A altura da arte nem sempre é profundidade de chão, e essa é a decisão que
não dá para automatizar.** Uma pedra em pé de 260 px é 260 px *para cima*:
bloquear a altura dela viraria um muro de 260 px de fundo. Um tronco compacto é
o oposto — a arte dele é raiz espalhada no chão, então a altura da arte é
profundidade e o bloco cobre tudo. A cerca é o caso do meio: a faixa vai do
topo do corrimão até a base dos postes, porque com só a base bloqueada o
jogador plantava o pé entre os corrimãos.

| Peça | Bloco vertical | Por quê |
|---|---|---|
| Casa | a massa opaca da `_base` | a parede é o obstáculo |
| Tronco compacto (`tree_trunk_*`) | a arte inteira, +8 px de folga | é raiz no chão, não altura |
| Pedra em pé, tronco morto alto | só a base | altura é altura |
| Cerca (trecho horizontal) | topo do corrimão até a base | senão o pé entra entre os corrimãos |
| Cerca (trecho vertical) | a coluna inteira | o trecho corre no eixo N–S |

A largura de um tronco vem do **perfil de massa** da arte, não da caixa
delimitadora: as colunas com pelo menos 40% da massa máxima. A caixa pegaria as
pontas de raiz translúcidas e bloquearia grama.

A building is two pieces cut at the eave line sharing one world anchor:
`*_base` in `structures_back` (with the collision footprint) and `*_roof` in
`foreground` (no collision) — the same trunk/canopy pattern used by trees.

### Arte de referência com fundo assado

`work/tiled-assets/build_forest_tree.py` existe por causa de um caso que vai se
repetir: a arte da árvore grande veio com um tapete de grama assado na base, um
blob de bordas suaves na cor do próprio bioma escuro. Sobre grama escura quase
não aparecia; sobre grama rala e terra nua seria um retângulo verde sob cada
árvore.

A remoção é por **matiz**, não por canal de cor: madeira fica em hue ~23 e a
grama daquele tapete em ~62, e mesmo o pixel escuro entre as raízes guarda o
matiz. Saturação baixa conta como madeira, porque aquela grama é saturada (~1.0)
e sombra não é. O que sobra entre os dedos das raízes fica transparente de
propósito — é o chão do mapa que tem que aparecer ali.

## Quem respeita a colisão

A grade de colisão é uma só, e quem anda no mapa é resolvido pelo mesmo código:
`internal/collision`. A regra tem três degraus, nessa ordem:

1. move inteiro, se o destino estiver livre;
2. se não, desliza no eixo que continua livre (raspa na parede em vez de parar);
3. se o deslize não rende progresso na direção pretendida (menos de 15% do
   passo), contorna: caminha ao longo da **face** do obstáculo e **se compromete
   com uma direção do mundo** até o caminho direto abrir. Sem esse compromisso o
   monstro troca o deslize pelo desvio a cada quadro e fica pairando ao lado da
   casa.

**O compromisso é com uma direção, não com um sentido de rotação**, e essa
distinção é o conserto de "os monstros ficam batendo de frente na cerca em vez
de dar a volta". A versão anterior guardava um sinal `+1/-1` aplicado a
`perp = (-delta.Y, delta.X)` — uma rotação de 90° do **rumo atual**. O rumo gira
enquanto a criatura caminha, então o mesmo sinal apontava para lados diferentes
do mundo ao longo do contorno. Medido contra uma barricada com um vão: o monstro
deslizava certinho até 60 px da abertura, a componente lateral do rumo virava
quase zero, o contorno começava, o `+1` fixo o mandava para o lado **oposto**, e
ele caminhava 27 s para longe do vão que já tinha alcançado.

Agora a face sai de **qual eixo ficou bloqueado** (numa barricada horizontal a
face é sempre X, venha a criatura de frente ou de esguelha), e o lado sai de
`openingSide`: uma varredura que olha ao longo da face **nos dois sentidos ao
mesmo tempo** (+1, −1, +2, −2, …) e devolve o sentido onde a abertura aparece
primeiro — a mais próxima, não a de um lado escolhido a dedo. "Abertura" é ter
onde pisar **e** o passo seguinte na direção pretendida estar livre; esse
segundo teste tem de ser de **um passo**, porque sondar fundo demais pula por
cima do obstáculo e responde "livre" em cima da própria parede.

Custo: até 192 consultas ao mapa, e só no quadro em que um contorno **começa** —
enquanto a criatura está contornando, a direção guardada é seguida sem
varredura. Medido nos 32 postos do mapa 3 contra um alvo do outro lado da
barricada mais próxima: 32 de 32 chegam.

| Quem | Footprint | Onde fica | Onde |
|---|---|---|---|
| Jogador | `Radius * 2` (40 px) | **nos pés**, ~105 px abaixo de `Position` | `game.PlayerFootprint` / `entity.PlayerGroundBox` |
| Monstro | 45% do diâmetro de combate (40–45 px) | centrado em `Position` | `entity.EnemyFootprint` / `Enemy.step` |

### A malha de navegação (`internal/nav`)

Os três degraus acima resolvem "o que o mapa deixa eu fazer NESTE quadro" —
uma caixa, um passo. Eles não sabem "por onde se chega lá": `openingSide`
enxerga a face de UM obstáculo, não o mapa inteiro, e um vão a mais de ~320 px
de distância (96 passos de varredura) é invisível para ele. `internal/nav`
existe para essa pergunta — uma malha de células de 32 px, derivada da mesma
`collision.Solid` uma vez por carregamento de mapa, com A* octil (sem cortar
quina) e suavização por linha de visão.

O rumo direto continua sendo o caminho normal — a malha é o plano B:

- **Bot**: pede rota sempre que o destino não está em linha de visão livre.
- **Monstro**: investe direto, como sempre — bate, desliza, contorna
  localmente. Só pede rota quando a distância até o alvo **não encolhe** por
  `FoeStuckBefore` (0,4 s) seguidos — o "bateu, olhou, deu a volta" que a tela
  mostra, não um caçador oniscente.

Ambos soltam a rota assim que o alvo volta a estar visível em linha reta.
Orçamento de **8 buscas por quadro**, compartilhado por todo mundo — um
engarrafamento de matilha nunca trava um quadro esperando A*.

O portão da arena (`game/arena_gate.go`) muda a colisão EM JOGO: quando isso
acontece, `Host.RebuildNavArea` recalcula só as células da área do portão —
reconstruir a malha inteira a cada abertura seria o mesmo custo do
carregamento do mapa, para uma mudança de meia dúzia de células.

**F4** (desktop) desenha a malha (verde = livre, vermelho = bloqueada) e a
rota atual de cada bot e monstro. Ver `doc/plan_navegacao_bots_monstros.md`
para o desenho completo.

### A caixa do jogador fica nos pés, não no centro

O sprite é desenhado **centrado** em `Position`: quadro de 192 px com a sola em
187, escala 1,15, então o pé cai ~105 px abaixo do ponto que o jogo colidia.
Esse vão era visível — passando acima de um tronco, a caixa já tinha limpado o
obstáculo enquanto o pé ainda era desenhado sobre as raízes.

`entity.GroundOffset` deriva a distância de `FootLine` e `RenderScale` do
`CharacterDef`, então personagem novo com outro rig acerta sozinho (sem
`FootLine`, cai para o fundo do quadro). A caixa **mantém o tamanho de sempre**
(`Radius * 2`): todo vão por onde o jogador passava continua passando, só mudou
onde ela fica. A borda de baixo dela é a linha da sola.

Quem testa o jogador contra o mapa usa `game.PlayerFootprint`, que devolve o
**centro** junto com o tamanho. Usar `p.Position` como centro é testar o peito
do personagem contra o chão — foi assim que o portal e o overlay F3 discordaram
do resolvedor antes. `entity.PositionFromGroundCenter` faz a volta, e as duas
conversões são exatamente inversas: se não fossem, o jogador derivaria alguns
pixels por quadro.

Monstro continua centrado de propósito: o lobo gira em torno de `Position` e o
slime é uma bolha que já assenta no centro — nenhum dos dois é uma figura em
pé com o pé longe do meio do sprite.

O `Radius` do inimigo cobre a silhueta inteira porque é hitbox de combate;
andar com essa caixa deixaria um slime de 90 px preso em vãos de 128 px por
onde o jogador passa. Por isso o footprint de movimento é uma fração dele,
na mesma faixa da caixa do jogador.

Quem simula é o host: `EntityManager.Solid` recebe a `CollisionGrid` do mapa
carregado em `World.ApplyToHost`, e o cliente só desenha as posições que
recebe. A separação de horda (`ResolveEnemyOverlap`) também passa pelo
resolvedor — desempilhar um amontoado não pode empurrar ninguém para dentro de
uma árvore. Quem já nasce dentro de um sólido se move livre até sair.

F3 desenha o footprint testado do jogador e o de cada monstro
(`game.DrawFootprintDebug`), laranja quando está sobre espaço sólido — e a do
jogador aparece **nos pés dele**, que é onde ela é testada.

## Trilhas

Uma trilha é uma **fita de arte ao longo de uma curva** — nem tile, nem material.

Medindo a arte de referência: o maior degrau de verdor entre colunas vizinhas é
2.12 num perfil que vai de 19.2 a 8.6, e o miolo do caminho *ainda tem grama*
(verdor 10.8 contra 16.5 na grama cheia). Não existe borda para recortar. Então
a arte é desenhada como está, e o próprio perfil dela, medido coluna a coluna,
virou o canal **alpha** de `terrain_trail.png` — a fita desenha o caminho e
deixa de fora as margens de grama da imagem, que eram o que a fazia arrastar
verde para dentro dos outros biomas.

**Por que não tile.** Um tile de caminho reto precisa de peça de canto desenhada
para cada curva: rotação só produz horizontal a partir de vertical, ela não
inventa canto. E prenderia o traçado à grade de 128 px.

**Por que não campo de distância por pixel.** Perguntar a cada pixel qual o ponto
mais próximo da curva quebra na virada: no lado de fora, um leque inteiro de
pixels responde o mesmo vértice, a distância percorrida trava ali e a textura
roda em catavento. Com geometria, o `v` é um número por vértice e linear entre
eles por construção.

Objeto polilinha na objectgroup `trails`, com uma propriedade só:

| Propriedade | Padrão | Significado |
|---|---|---|
| `width` | 512 px | largura da fita, que é também a cada quanto a textura repete ao longo do trajeto |

Duas regras que a fita impõe:

- A normal de cada junta é a média das duas direções e cada quad reaproveita a
  aresta do anterior. Sem isso haveria vão na parte de fora da curva, e
  sobreposição misturaria a arte semitransparente contra ela mesma, imprimindo
  um degrau claro em cada emenda.
- A fita dobra sobre si mesma no lado de dentro de uma curva de raio menor que a
  meia-largura. `Trail.Path` reamostra a 24 px e suaviza com raio de meia-
  largura para arredondar os cantos da polilinha, mas o traçado no Tiled ainda
  precisa de folga: com 512 px de largura, pelo menos 2 células entre viradas.

| File | Responsibility |
|---|---|
| `internal/tilemap/trail.go` | Leitura das polilinhas, reamostragem e suavização da curva. |
| `internal/tilemap/trail_renderer.go` | Emite os quads texturizados da fita. |
| `work/tiled-assets/build_trail.py` | Gera `terrain_trail.png` a partir da referência, medindo o alpha dela. |
| `work/tiled-map-world02/preview_terrain.py` | Espelha bioma e fita fora do jogo, rasterizando cada quad em dois triângulos com UV por baricêntricas. |

A trilha é desenhada logo depois do terreno e antes do `ground_detail`: ela é
chão, e tudo que fica em pé sobre o chão vem por cima dela.

## Portais

Um portal é um objeto na objectgroup `portals`. Quem pisa nele troca de mapa.
O destino mora no arquivo do mapa, não numa tabela em código:

| Propriedade | Obrigatória | Valor |
|---|---|---|
| `target_map` | sim | caminho do mapa destino a partir da raiz (`assets/maps/world_02.json`) |
| `target_spawn` | não | nome do objeto de chegada na camada `spawn` do destino (padrão `player_spawn`) |

Objeto sem `target_map` é ignorado — portal que não leva a lugar nenhum é erro
de mapeamento, não portal.

No `world_01` o portal (`portal_fim_da_trilha`, célula 44,2 — x 5632, y 256)
fica **no fim da estrada de terra**, entre os dois carvalhos que ladeiam a
trilha. O percurso é o roteiro do mapa: cerca inicial → rua de pedra da vila →
estrada de terra rumo ao norte → portal. São ~54 células livres do
`player_spawn` até ele, todas caminháveis (conferido por busca em largura na
camada `collision`); é a razão de o portal não ficar mais dentro da cerca.

| File | Responsibility |
|---|---|
| `internal/tilemap/portal.go` | Leitura da camada `portals` e teste de sobreposição. |
| `internal/game/world_state.go` | `World`: mapa carregado (renderer, colisão, bounds, spawns, portais) e o que o host precisa saber dele. |
| `internal/game/portal.go` | `UpdatePortal` (a decisão) e `ApplyPendingTravel` (a chegada). |
| `internal/game/portal_party.go` | A conta do grupo: quantos vivos pisam em cada portal. |
| `internal/game/portal_counter.go` | O "2/3" desenhado sobre o portal enquanto ele espera. |
| `internal/game/portal_shape.go` | Geometria do óvalo (medida a partir do retângulo do objeto) e mistura de cores. |
| `internal/game/portal_draw.go` | Ordem de desenho, poça de luz da base e halo. |
| `internal/game/portal_vortex.go` | Corpo do portal: superfície translúcida, redemoinho, ondulação e borda. |
| `internal/game/portal_sparks.go` | Partículas ao redor do portal (sem estado, derivadas do relógio). |

O portal é desenhado só com primitivas de raylib, nunca com asset de imagem —
`doc/art_style.md` deixa todo brilho para o runtime. A forma é um **óvalo em
pé**, medido contra o personagem (~186 px de altura, como uma porta) e não uma
poça no chão: o retângulo do objeto continua sendo só a área de gatilho, e o
desenho sobe a partir do centro dele. Tudo gira devagar (`portalSpin`, 0,45
rad/s) e pulsa em seno — portal é feitiço permanente, então nenhum movimento
pode dar salto. O corpo é desenhado antes das entidades, então o jogador sempre
passa na frente dele.

Uma exceção importante no blend: poça, halo, redemoinho, borda e partículas são
**aditivos**, mas a superfície do óvalo é **alpha**. Luz somada sobre a grama
saturada do mapa não apaga o verde por baixo, e a névoa azul sai esverdeada;
além disso, porta que deixa ver o chão do outro lado para de ler como porta.

### Portal trancado pelas hordas

O portal é a recompensa por terminar o mapa: enquanto a corrida de hordas não
acaba ele **não existe** — não é desenhado e não teleporta.

| Onde | O que faz |
|---|---|
| `internal/game/portal_gate.go` | `PortalsUnlocked()` lê o `WaveState` publicado pela rede; `advancePortalReveal` anima a abertura. |
| `World.portalReveal` | 0 a 1, mora no `World` para zerar sozinho quando o mapa troca. |
| `World.DrawPortals` | Avança o reveal e não desenha nada enquanto ele for 0. |
| `UpdatePortal` | Recusa a transição enquanto `portalReveal < 1`. |

Regras que valem a pena não reinventar:

- O gate lê `network.GetWaveState()`, que o host produz e espelha para os
  clientes em todo update de inimigo. Host e cliente abrem o portal no mesmo
  evento **sem mensagem nova de protocolo**.
- Mapa **sem** corrida de hordas (`Total == 0`) fica liberado. Travar por
  padrão deixaria um mapa de validação de terreno sem saída.
- O gatilho é o desenho estar aberto (`portalReveal >= 1`), não só a fase estar
  limpa: o que ainda não apareceu não funciona, que é a regra que o jogador vê.
- A abertura leva `portalRevealTime` (1,6 s) em fade e crescimento a partir de
  55% do tamanho, com a base fixa no chão — portal não dá pop.
- `ui/wave_hud.go` fecha o ciclo: ao limpar o mapa o subtítulo diz onde o portal
  abriu, porque a luta termina longe do fim da trilha.

`work/portal-preview/preview_portal.py <tempo>` reproduz o desenho fora do jogo
(mesma matemática, blend aditivo e alpha separados) para conferir proporção e
cor contra a grama sem compilar.

### Arena de mão única (mapa 5)

A zona `castle_climax` do `world_05` declara `arena_lock`. Ao cruzar a borda
sul dela, o jogador não pode retornar ao corredor de entrada. O portão físico
na borda norte permanece fechado enquanto a horda final se repõe; depois do
clímax, quando `LastStandDone()` deixa os inimigos restantes finitos e a
`WavePhase` vira `cleared`, o footprint de colisão do portão é desativado e o
portal para o mapa 6 fica acessível. Reiniciar a fase restaura o portão fechado.

**Não é colisão.** A trava é uma correção de posição (`entity.MoveByGroundCorrection`),
não uma parede: não há nada na borda sul que bloqueie o movimento por si só.
Isso importa porque a regra vale por CORPO, não por cliente — cada máquina
aplica a correção ao corpo que ela move.

- **Humano**: o próprio cliente roda `World.UpdateArenaGate` todo quadro para
  o seu jogador local (`game/loop.go`), e por isso obedece — a autoridade da
  posição de um humano é o cliente dele (`MsgInput`).
- **Bot**: é um corpo que só o host move (`doc/plan_bots_de_classe.md` §2).
  Nenhum cliente aplica a regra a ele, então a mesma correção tem de rodar no
  host, por bot, keyed em `botRuntime.arenaLocked` — o espelho de
  `arenaGate.returnLocked`, mas por agente em vez de por travessia local.
  `network.SetArenaLock` (`internal/network/host_arena_lock.go`) é o canal:
  `UpdateArenaGate` publica a zona e o `armed` todo quadro que já os calcula
  (mesmo padrão de `SetPartyPortals`/`host_bot_portal.go`), e
  `network.applyArenaLock` (`host_bot_move.go`) aplica o clamp depois do
  passo do bot resolver, e `clampDestToArenaLock` projeta um `Intent.Dest` ao
  sul de volta para a soleira, descartando a rota velha (`Follower.Clear()`).
  **O host nunca clampa um humano** — a posição dele já vem do cliente, e as
  duas autoridades brigando pela mesma coordenada faria o personagem tremer
  entre duas respostas.
- `arenaLocked` cai em `ResetBotArenaLocks`, chamado do reinício de fase
  (F5) e de todo carregamento/troca de mapa — nunca de `ReconcileBots`, que
  também roda em join/reconexão e apagaria o estado de bots que ainda
  legitimamente seguem trancados no meio de uma corrida.

**Um footprint que liga e desliga em jogo pode virar passe livre.**
`collision.Resolve` devolve o movimento inteiro sem checar nada quando o
agente já começa dentro de sólido — pensado para tirar quem nasceu em cima de
um obstáculo ou foi empurrado por uma multidão. O footprint do portão de
saída (`SetFootprintsEnabledOverlapping`, alternado por `arenaGate.bind`) é
exatamente um caso disso: um bot parado na faixa do portão no instante em
que ela volta a ser sólida ganha esse passe livre e atravessa paredes até
sair. `resolveBotMove` cobre isso ao testar a caixa atual antes de resolver o
passo: se já está bloqueada, empurra para a célula livre mais próxima
(`nav.Grid.NearestWalkable`) primeiro, e só então resolve — o passe livre
ainda existe (o ponto reencaixado já é livre), mas não sobra nada para
atravessar.

### O portal leva o GRUPO

A transição só acontece quando **todos os jogadores vivos estão pisando no mesmo
portal**. É a mesma pergunta que `climax_gate.go` já fazia na fortaleza do mapa
3 — "o grupo chegou?" — e agora ela tem uma resposta só.

| Onde | O que faz |
|---|---|
| `internal/game/portal_party.go` | `countParty` conta vivos dentro de cada portal; `readyPortal` acha o que pode levar. |
| `internal/game/portal.go` | `UpdatePortal` decide (só host/solo) e chama `network.StartTravel`. |
| `internal/network/travel_local.go` | `StartTravel` anuncia e enfileira; `ConsumeLocalTravel` é lido pelo laço. |
| `internal/network/client_travel.go` | O cliente limpa o mundo espelhado ao ser levado. |
| `internal/network/host_travel.go` | `PlaceEveryoneAtSpawn`: o host põe a festa inteira no ponto de chegada. |

Regras que valem a pena não reinventar:

- **A caixa testada é a dos pés, para todo mundo.** `entity.GroundBoxAt` existe
  porque um jogador remoto é uma posição num snapshot, nunca um `*Player`.
  Testar o local pelos pés e os outros pelo peito faria o contador discordar de
  si mesmo dependendo de quem está olhando.
- **A posição vem de `GetAllPlayers()`, nunca da interpolada.** O desenho está
  100 ms atrasado de propósito; decidir uma troca de mapa com ele seria decidir
  com 100 ms de erro. Ver `doc/network.md`.
- **Morto viaja junto e não segura ninguém.** Ele chega morto do outro lado e
  espera o mesmo ressurgimento que esperaria aqui. Por isso `ApplyPendingTravel`
  fica **fora** do bloco `if !p.IsDead` do laço — dentro dele, a máquina de quem
  morreu ficaria sozinha no mapa que o grupo acabou de deixar.
- **Contar por PORTAL, não por mapa.** Dois portais com destinos diferentes
  somariam um grupo que nunca esteve junto.
- **Grupo vazio não chegou.** No primeiro quadro a lista está vazia e "todos
  dentro" seria verdade sobre ninguém.
- **`World.partyTally` é contado uma vez por quadro e lido pelo desenho.**
  Recontar no renderer é como o número na tela e a porta que decide passariam a
  dizer coisas diferentes.
- **`arrived` só limpa quando NINGUÉM está num portal.** O grupo chega junto: se
  limpasse assim que o primeiro saísse, o portal dispararia de novo debaixo de
  quem ainda estivesse na chegada.

### Quem entra no portal some e espera

Com bots no grupo (`doc/plan_bots_de_classe.md`) a porta quase não abria: um
bot entrava e saía do retângulo o tempo todo, e "todos dentro ao mesmo tempo"
nunca ficava verdadeiro por mais de um quadro. A regra agora: **quem pisa na
zona do portal some da tela e congela**, liberando o pequeno retângulo para o
resto do grupo entrar. Vale para humano e para bot; a conta continua sendo do
grupo inteiro (`countParty`, acima) — essa parte não mudou.

| Onde | O que faz |
|---|---|
| `internal/network/host_bot_portal.go` | `SetPartyPortals` guarda os retângulos (não só o centro) e se estão ativos — `game.UpdatePortal` chama a cada quadro. |
| `internal/network/host_portal_presence.go` | `tickPortalPresence`: testa `entity.GroundBoxAt` de cada jogador vivo contra os retângulos, a MESMA caixa que `countParty` usa. |
| `internal/network/protocol.go` | `PlayerState.InPortal` — estado de tique, como `IsDead`; não vai em `slimPlayer`. |
| `internal/game/input_handler.go` | Congela o humano local: direção zero, sem ataque, sem habilidade, enquanto `network.LocalPlayerInPortal`. |
| `internal/game/portal_cancel.go` | ESC / botão SAIR empurra o jogador para fora do retângulo e devolve o controle. |
| `internal/game/renderer.go` | Não desenha quem está `InPortal` — nem remoto, nem o local. |

Regras que valem a pena não reinventar:

- **Morto nunca entra em espera.** Ele viaja junto e continua caído, do mesmo
  jeito que já viajava antes desta mudança.
- **Ativo exige `portalReveal >= 1` E `!arrived`.** Chegar em cima de um
  portal (por exemplo, o spawn de um mapa perto de um) não pode fazer o grupo
  sumir na hora que aterrissa.
- **Quem move o corpo continua sendo quem sempre moveu.** O host decide a
  flag; um bot congela porque `tickBots` pula quem está `InPortal` por
  inteiro, e um humano continua enviando a própria posição — o host nunca
  move o corpo de ninguém, só recusa dar-lhe um destino novo enquanto a flag
  estiver ligada (bot) ou o cliente mesmo se recusa a mover (humano,
  `ProcessInput`).
- **A limpeza é obrigatória em dois lugares**: `PlaceEveryoneAtSpawn`
  (travessia) e `ResetStage` (F5). Esquecer um dos dois deixa o grupo
  invisível e sem controle no mapa seguinte — o pior defeito que esta
  entrega poderia ter.

### Pular de fase (F8, desktop)

**F8 leva direto para a próxima fase**, sem limpar as hordas e sem procurar o
portal. **Shift+F8 leva direto para a ÚLTIMA fase**, de qualquer mapa — a
campanha começa sempre no `world_01` (`loop.go`), então chegar a uma fase
funda (o corredor final, mapa 6) exigia apertar F8 cinco vezes, uma por fase,
toda vez que o jogo abria. Shift+F8 chega lá num só toque, e como
`game.UltimatesGrantedOn` deriva as supremas liberadas do **índice** da fase
na lista (não de ter passado por ela), o grupo chega no mapa 6 já com as
quatro supremas que a fase pede — sem precisar andar as cinco fases
anteriores. É chave de desenvolvimento, como F2 e F5: não existe equivalente
de toque e não se quer um.

| Onde | O que faz |
|---|---|
| `internal/game/stage_skip.go` | `campaignMaps` (a ordem das fases), `UpdateStageSkip`, `jumpToLastCampaignMap` (Shift+F8). |
| `internal/game/world_travel.go` | `travelTo`: a troca de mapa em si, compartilhada com o portal. |

F8 percorre a lista `campaignMaps` em vez de seguir o `target_map` do mapa
atual, e a diferença não é acadêmica: o portal do `world_02` aponta de volta
para o `world_01` enquanto o `world_03` não existe, então seguir o portal
mandaria o testador **para trás** — o oposto do que a tecla serve. A lista é
também o único lugar onde a ordem da campanha está escrita.

F8 leva o **grupo inteiro**, como o portal: ele anuncia o destino por
`network.StartTravel` em vez de trocar o mapa só nesta máquina. Só o host pula,
pelo mesmo motivo que só o host reinicia a fase — é a máquina que simula. Num
cliente a tecla não faz nada.

**Ao adicionar uma fase nova, incluir o mapa em `campaignMaps`.**
`stage_skip_test.go` cobra que todo caminho da lista exista no disco e que não
haja repetido (mapa repetido tornaria a segunda metade da lista inalcançável).

Mapa fora da lista e última fase não pulam, e os dois casos registram no log —
tecla que não faz nada parece quebrada.

Trocar de mapa troca o `World` inteiro de uma vez. Renderer, grade de colisão,
`world.Bounds` e spawns sempre vêm do mesmo arquivo, então nada fica apontando
para o mapa de onde o jogador acabou de sair. O host é reapontado no mesmo
instante (`World.ApplyToHost`), e mapa **sem marcadores `enemy_spawn_*` é mapa
sem hordas**: `Host.Waves` fica nil e nada nasce — que é o que um mapa de
validação de terreno quer.

### Os cinco prefixos da camada `spawn`

São cinco coisas diferentes, e o prefixo é a única forma de distingui-las:

| Prefixo | O que é | Quem lê |
|---|---|---|
| `enemy_spawn_*` | de **onde uma horda chega**, fora da tela | `WaveRunner` |
| `enemy_post_*` | **guarnição**: quem já está em campo quando o grupo entra, com setor e patrulha | `garrisons.go` |
| `enemy_sentry_*` | **posto de gárgula**: posição fixa de uma criatura estática de alcance 1900 | `sentries.go` |
| `climax_spawn_*` | de onde a **emboscada** vem, depois de o grupo alcançar o objetivo | `StartClimax` |
| `enemy_cannon_*` | **canhão fixo** (mapa 6): não é `entity.Enemy`, é decoração do Tiled com uma arma de host atrás | `cannons.go`, `host_cannon.go` |

`enemy_sentry_*` é separado de `enemy_post_*` porque a gárgula não patrulha, não
volta ao posto e não tem setor — passá-la pelo caminho da guarnição lhe daria um
`Guard` com trecho de patrulha e território, perguntas que ela não responde, e o
setor poderia recusar o único alvo que ela tem. E é separado de `climax_spawn_*`
porque um posto é uma **posição**, não uma origem: dois monstros nunca ocupam o
mesmo. Enquanto as gárgulas do `world_04` eram marcadores de clímax, o host
tinha de reconhecê-las pelo **nome** (`stream_`) para não nascer um lobo dentro
da água.

`enemy_cannon_*` é o mais separado de todos: não passa por nenhum sistema de
inimigo. `World.EnemyCannons` lê os marcadores e `Host.InstallArrivalCannons`
os arma como `cannonPost` — posição fixa e um temporizador, nada mais. Sem
sprite própria (a estátua de gárgula que ocupa a mesma célula é só cenário do
Tiled) e sem HP: o único jeito de tirar um canhão de campo é o julgamento
roteirizado da Paladina no resgate do mapa 6 (`castCannonJudgment`, ver
`doc/combat_rules.md`). Hoje só `world_06.json` declara algum.

O nome vem sem o prefixo (`enemy_sentry_ilha_oeste_sul` → `ilha_oeste_sul`), e a
ordem alfabética não é cosmética: no `world_05` as gárgulas entram aos poucos,
horda a horda, percorrendo a lista do início.

Duas limitações atuais, ambas deliberadas:

- A transição é local. Mapa não faz parte do protocolo de rede, então um host
  que atravessa um portal continua simulando para clientes que ficaram no mapa
  antigo. Vale igual para o F8.
- Quem chega em cima de um portal só o dispara de novo depois de sair dele
  (`World.arrived`).

## F3 Debug Overlay

Press F3 to toggle `tilemap.DebugEnabled()`, shared by every debug drawing:

- Magenta: rendered source rectangle of a manifest piece.
- Cyan: the 128px anchor cell of the object.
- Red cross: the exact world anchor.
- Red translucent boxes: solid space in `CollisionGrid` — painted cells as full
  128 px cells, prop footprints as the pixel rectangles they are.
- Green box: the player footprint actually tested by movement resolution
  (`game.PlayerFootprint`), drawn at the character's feet and turning orange
  while it overlaps solid space.

## Tileset References

External tilesets should be referenced relative to the map file inside `assets/`, for example:

```json
{ "source": "../tilesets/village_set.tsx" }
```

TSX image paths are resolved relative to the TSX file directory.

## Rendering Order

`DrawFrame` uses `MapRenderer.DrawWithCamera` so map layers and entities share one camera block.

Current layer split:

- Draw lower map layers up to `foreground`.
- Draw world entities.
- Draw upper layers from `foreground`.

If map layer names change in Tiled, update the `DrawWithCamera` layer arguments in `internal/game/renderer.go`.

## Só se desenha o que a câmera mostra

Todo passe percorria o **mapa inteiro** a cada quadro. No `world_03` eram
23.044 quads de terreno para mostrar as ~127 células que cabem numa tela de
1080p, e 18.844 deles embrulhados num `Begin/EndShaderMode` próprio — cada um
esvazia o batch do rlgl, então eram 18.844 draw calls por quadro. Nenhum passe
consultava a câmera.

`tilemap.Viewport` (`viewport.go`) converte a `rl.Camera2D` no retângulo de
mundo visível e no intervalo de células que ele cobre, e `DrawWithCamera` o
distribui para todos os passes.

| Passe | Como é cortado |
|---|---|
| Terreno (`TerrainRenderer.eachCell`) | intervalo de células |
| Tile layers (`drawTileLayer`) | intervalo de células |
| Peças de manifesto (`ManifestRenderer.Draw`) | interseção do **retângulo desenhado** |
| Fita de trilha (`TrailRenderer.ribbon`) | interseção da caixa de cada quad |

Três regras que valem entender antes de mexer:

- **Cullar não pode mudar um pixel.** O desenho de uma célula nunca sai da
  própria célula, e a máscara de borda lê a camada `ground` **do mapa**, não o
  que já está na tela — então um vizinho cortado continua contando como ligado.
  Foi isso que permitiu cortar sem tocar no resultado visual.
- **Peça de manifesto é testada pelo desenho, não pela âncora.** Ela é ancorada
  longe do que desenha: a árvore ancora no tronco e desenha a copa centenas de
  pixels acima, a casa ancora no pé da parede. Cortar pela célula âncora faria
  a copa sumir enquanto o tronco ainda está abaixo da tela. O retângulo
  desenhado é exato e por isso não leva margem.
- **O terreno leva margem de 1 célula, a trilha e os props não.** A margem
  cobre o quad que atravessa a borda da tela; quem é testado pelo retângulo
  exato não precisa dela.

A trilha é cortada apesar de tudo caber num batch só, porque cada quad custa
doze chamadas de cgo (cor, coordenada e vértice por canto) e uma trilha longa
emite centenas por quadro. O `travelled` e a aresta anterior continuam
avançando mesmo no quad pulado — sem isso a textura saltaria ao reentrar na
tela e a emenda abriria.

`MapRenderer.Draw` (sem câmera) usa `EverythingVisible()`: quem não tem câmera
não pode cullar, e desenhar tudo é o comportamento que existia antes.

Medido em `work/perf/verify_viewport.py`, que varre 121 posições de câmera por
mapa e **calcula a verdade de forma independente** (interseção exata da célula
com a tela, sem passar pela conta do `Viewport`) para provar que nenhuma célula
visível é cortada:

| Mapa | Antes | Depois (média) | Pior caso | Redução |
|---|---|---|---|---|
| `world_01` | 3.323 | 258 | 398 | 13× |
| `world_02` | 12.702 | 825 | 1.065 | 15× |
| `world_03` | 23.044 | 1.071 | 1.530 | **22×** |

## Uma troca de shader por MATERIAL, nao por celula

`drawBlended` embrulha **cada celula** num `Begin/EndShaderMode` proprio, porque
a mascara de vizinhos vai em `uniform` e uniform so muda entre draw calls. Cada
um desses esvazia o batch do rlgl: no `world_03` eram **1.104 draw calls por
quadro** so de chao, mesmo depois do culling.

O caminho em batch (`internal/tilemap/terrain_batch.go`) tira a mascara do
uniform:

| O que | Antes | Depois |
|---|---|---|
| Mascara de 8 vizinhos | `uniform vec4 edge/corner`, por celula | **textura** do tamanho da grade, 1 texel por celula |
| Qual celula | implicito (uma draw call por celula) | **tint** do `DrawTexturePro`, que vira cor de vertice |
| `edgeWidth`, `spanF` | por celula | uniform por material |
| Draw calls (`world_03`) | 1.104 | **~10** |

Por que o tint: e o unico canal que o `DrawTexturePro` deixa variar por quad
**sem quebrar o batch**. O preco e que o shader de batch **nao multiplica
`fragColor`** no resultado — se multiplicasse, pintaria o chao com as proprias
coordenadas.

Detalhes que custaram pensamento e nao devem ser desfeitos:

- **O uv local vem do span, nao de `fract()`.** `fract(uv*span)` devolveria 0
  na aresta final da celula em vez de 1, imprimindo uma linha de um pixel em
  cada emenda. O shader faz `fragTexCoord * spanF - window`, exato nas duas
  pontas.
- **A textura de mascara e `FilterPoint`, obrigatoriamente.** Ela e dado
  indexado por celula, nao imagem: interpolar entre dois texels misturaria a
  vizinhanca de duas celulas e desbotaria bordas que nao existem.
- **A ordem dos bits e a de `terrain_mask.go`** (bordas N, L, S, O; cantos NO,
  NE, SE, SO). Trocar dois bits produziria um desbotamento espelhado — o tipo
  de defeito que passa numa captura estatica e so incomoda em movimento.
- **A grade tem de caber em 255 celulas por lado**, porque o indice viaja em
  dois canais de 8 bits. Passar disso cai no caminho antigo em vez de imprimir
  lixo.

**O caminho por celula continua no codigo, como fallback.** Isso e a mesma
politica que o terreno ja adotava com `t.enabled` (shader indisponivel cai para
borda dura): se o shader de batch nao carregar, faltar um uniform ou a grade
nao couber, o desenho volta ao de sempre — mais caro, resultado identico.

Conferido fora do jogo por `work/perf/verify_terrain_batch.py`, que prova que
empacotar e desempacotar a mascara devolve exatamente o que
`edgeMask`/`cornerMask` calculam (36.083 celulas nos quatro mapas) e que o uv
local fecha em 0 e 1 sem erro. Ele diz no proprio docstring o que **nao** prova:
que o GLSL compila, que o rlgl mantem o sampler ligado durante o bloco, e que o
tint chega como esperado — as tres so a execucao responde, e sao a razao de o
fallback existir.

## Assets: so o que o mapa usa, e uma vez so

Duas coisas que eram por mapa e nao precisavam ser:

**Carregava o acervo inteiro em todo mapa.** `NewManifestRenderers` subia os
nove atlas e `NewTerrainRenderer` as treze texturas de terreno, sempre — 78,6 MB
de VRAM fixos, dos quais o `world_01` (uma vila verde) usa ~30. Ele pagava pela
muralha da fortaleza, pelas defesas de cerco e pelo chao de castelo do mapa 4.

Agora os dois recebem o mapa e filtram:

- manifesto entra se o mapa **cita alguma peca dele pelo nome** (nao da para
  filtrar por camada: `vegetation` e dividida por cinco manifestos);
- material de terreno entra com **a pilha inteira abaixo dele**, porque o chao e
  desenhado empilhado — uma celula de laje pinta o cascalho e a terra nua por
  baixo. A grama base entra sempre, porque o passe de fundo a desenha sob todas
  as celulas de qualquer mapa.

**Recarregava tudo a cada portal.** `internal/tilemap/texture_cache.go` guarda
uma textura por arquivo com contagem de referencia: `AcquireTexture` reaproveita
e soma, `ReleaseTexture` subtrai e so descarrega no zero. O que o mapa de origem
e o de destino usam igual **nem chega a sair da VRAM**, e como `travelTo` carrega
o destino antes de descarregar a origem, isso tambem elimina a duplicata que
existia durante a transicao.

Sem mutex de proposito: carregar textura e chamada de GPU, que so pode acontecer
na goroutine que detem o contexto OpenGL.

## Medidor de quadro (F3)

`tilemap/stats.go` conta o que cada quadro desenhou e cronometra a submissão;
`entity/draw_stats.go` conta os inimigos; `game/perf_hud.go` desenha o painel
no canto superior direito, atrás do mesmo F3.

**O painel é feito para ser lido numa captura de tela e diagnosticado depois**,
sem quem lê ter de lembrar em que mapa estava. Essa regra é explícita porque
foi violada: a primeira captura chegou sem o nome do mapa e a fase teve de ser
deduzida pelos números (`trilha 0 quads` só acontece no `world_01`).

```text
world_01   host   2560x1440          <- mapa, papel na rede, resolução
fps 60   quadro 16.7 ms   PIOR 18.2 ms
cpu: mapa 2.1   entidades 0.8   resto 13.8 ms
terreno 500 quads / 155 binds de shader
tiles 35   props 22   trilha 0 quads   celulas 5520
inimigos 83 vivos / 14 desenhados   projeteis 3
pos 3456,2789   celula 27,21
```

Cada linha responde uma pergunta que a ausência dela deixaria em aberto:

| Linha | Para que serve |
|---|---|
| mapa / papel / resolução | identifica a captura; host e cliente rodam caminhos de desenho **diferentes**, e a resolução é o que dá sentido à contagem de células |
| fps / quadro / **PIOR** | o pior quadro é o número que importa; laranja acima de 20 ms |
| cpu: mapa / entidades / resto | separa submissão de CPU do que é GPU e vsync |
| terreno / tiles / props | verifica o culling: se subir, algum caminho escapou dele |
| inimigos vivos / desenhados | separa "caro de desenhar" de "caro de simular" |
| pos / célula | situa a amostra; duas capturas do mesmo mapa só se comparam sabendo de onde vieram |

Duas leituras que valem saber de cor:

- **`cpu: mapa` e `entidades` medem SUBMISSÃO, não a GPU.** O raylib acumula em
  batch e o trabalho real acontece no `EndDrawing`. É por isso que os dois
  números valem tanto: se a soma deles for pequena e o quadro ainda assim for
  longo, o gargalo está na GPU ou no vsync e **culling não resolve mais nada**.
  Se `mapa` sozinho for metade do quadro, resolve.
- **`83 vivos / 14 desenhados` não é o desenho.** Se quase todos estão fora da
  tela e o quadro continua caro, o custo é a simulação, que roda para os 83 de
  qualquer jeito (`ResolveEnemyOverlap` é O(n²)).

O pior quadro está lá de propósito: 58 fps de média com um quadro de 33 ms a
cada meio segundo é exatamente o padrão que faz a câmera trancar e cansar a
vista, e a média esconde isso por completo.

A amostragem roda mesmo com o F3 desligado — ligar o overlay no meio de uma
travada tem de mostrar a travada, não uma janela vazia enchendo por dois
segundos.

Os contadores não têm mutex de propósito: são escritos e lidos dentro do passe
de desenho, que roda na goroutine principal entre `BeginDrawing` e
`EndDrawing`.

## Collision

Solid space has two sources and they keep their own shape:

- the invisible Tiled tile layer named `collision`, authored per cell and
  stored as cells;
- manifest collision footprints, authored in pixels against the art and stored
  as pixel rectangles (`footprintIndex`).

Visual layers never define walkability. `tilemap.CollisionGrid` tests only the
cells a footprint touches and the rectangles it overlaps, and exposes both as
rectangles for projectile checks.

No `world_01` a camada `collision` está **vazia**: as 46 células que existiam
eram a cerca, e a cerca passou a ter footprint próprio. `world_02` ainda pinta
células (borda do mundo e mata fechada), que é o uso legítimo dela.

Player collision resolution is handled in `internal/game/collision.go`.

## Map Changes Checklist

1. Put runtime files under `assets/maps` and `assets/tilesets`.
2. Keep TSX/image references relative and Android-safe.
3. Verify map dimensions still produce valid `world.Bounds`.
4. Verify the `spawn` object layer has a `player_spawn` object inside bounds.
5. Verify layer names used by `DrawWithCamera`.
6. Run `go test ./...`.
