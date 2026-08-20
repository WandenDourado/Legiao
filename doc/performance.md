# Performance

Diagnóstico medido do custo de render, memória e rede, e o plano de correção em
ordem de impacto. Números produzidos por `work/perf/measure_cost.py` contra os
mapas e assets em disco — nenhum número aqui foi estimado no olho.

Parte do plano já foi implementada; os itens trazem a data e o que foi medido.

---

## 0. A pergunta direta: os três mapas ficam na memória?

**Não.** Um mapa por vez. `World` é trocado inteiro em `travelTo`
(`internal/game/world_travel.go`) e o anterior chama `Unload()`, que descarrega
as texturas do `MapRenderer`, do `TerrainRenderer` e dos manifestos.

Mas a resposta certa esconde três problemas reais:

1. **Todo mapa carrega o acervo inteiro do jogo, não o que ele usa.**
   `NewManifestRenderers()` carrega os 9 atlas de manifesto sempre, e
   `NewTerrainRenderer()` carrega as 10 texturas de terreno sempre. O
   `world_01` (uma vila verde) paga pela muralha da fortaleza e pelas defesas
   de cerco do `world_03`.
2. **A troca de mapa tem pico de 2×.** `travelTo` carrega o destino inteiro
   **antes** de descarregar a origem. Por alguns quadros as duas estão na VRAM.
3. **A troca de mapa re-decodifica tudo do zero**, inclusive os atlas
   idênticos que os dois mapas usam. Não há cache entre mapas.

Nada disso é o gargalo principal. O gargalo principal é o **render do
terreno**, e é por ele que o plano começa.

---

## 1. Render — o custo real por quadro

### 1.1 Medida

| Mapa | Células | Quads de terreno / quadro | Trocas de shader / quadro | Células visíveis em 1080p | Desperdício |
|---|---|---|---|---|---|
| `world_01` | 60×45 = 2 700 | 3 323 | 623 | ~127 | 26× |
| `world_02` | 60×50 = 3 000 | 12 702 | 9 702 | ~127 | 100× |
| `world_03` | 60×70 = 4 200 | **23 044** | **18 844** | ~127 | **182×** |

A tela de 1920×1080 mostra 15×8,4 = ~127 células de 128 px. O `world_03`
desenha 23 mil quads para mostrar 127.

### 1.2 De onde vem

`TerrainRenderer.Draw` (`internal/tilemap/terrain_renderer.go`) faz
**10 varreduras completas da grade** por quadro: uma de grama base e uma por
material em `terrainOverlays`. Cada varredura visita as 4 200 células do
`world_03` para descobrir quais recebem aquela camada.

Pior que a varredura é o que acontece em cada célula pintada:

```go
rl.BeginShaderMode(t.shader)   // flush do batch
t.drawPlain(...)               // 1 quad
rl.EndShaderMode()             // flush do batch
```

`Begin/EndShaderMode` **esvazia o batch do rlgl**. Cada célula com camada é uma
draw call própria. No `world_03` são **18 844 draw calls por quadro**, ou
**1,13 milhão por segundo a 60 fps**. Placa de desktop aguenta mal; GPU móvel
não aguenta — é o número que explica queda de fps no mapa 3 e no Android.

E o custo de CPU acompanha: `edgeMask` + `cornerMask` fazem 8 consultas de
vizinho por célula pintada, e cada consulta chama `paintedWith` → `stackRank`,
que é **varredura linear das pilhas**. São ~150 mil `stackRank` por quadro no
`world_03`, recalculando uma máscara que **nunca muda** enquanto o mapa não
muda.

### 1.3 Os outros passes, pelo mesmo defeito

- `drawTileLayer` percorre todas as células de cada tilelayer visível, mesmo
  as vazias, e para cada tile pintado chama `manifestOwnsGID` → varre os 9
  manifestos × `TilesetForGID` (varredura linear dos tilesets). O
  `ground_detail` do `world_03` são 424 tiles, cada um pagando ~9 comparações
  de caminho de arquivo (`filepath.Clean` em `UsesGID`, que **aloca**, por tile
  por quadro).
- `ManifestRenderer.Draw` é chamado uma vez por manifesto por *role*, e cada
  chamada **percorre todas as camadas do mapa** procurando a sua. São
  9 manifestos × 2 roles = 18 varreduras da lista de camadas por quadro, e
  desenha os 283 objetos do `world_03` inteiros, inclusive os que estão a
  4 000 px fora da tela.
- Nada em nenhum desses caminhos consulta a câmera.

### 1.4 Plano de render

Em ordem de impacto por risco. **Os itens 1 e 2 sozinhos resolvem a maior
parte do problema** e não mudam um pixel do que aparece na tela.

**R1 — Culling de câmera — IMPLEMENTADO (09/08/2026).**
`internal/tilemap/viewport.go` + os passes de `renderer.go`,
`terrain_renderer.go`, `vegetation.go` e `trail_renderer.go`. Medido em
`work/perf/verify_viewport.py`: `world_03` de 23.044 para 1.071 quads em média
(1.530 no pior caso), **22×**; `world_02` 15×; `world_01` 13×. O script prova
que nenhuma célula visível é cortada, calculando a verdade de forma
independente. Ver `doc/tilemap.md`, seção "Só se desenha o que a câmera mostra".
Falta `go build` / `go test` e a sessão em jogo.

Descrição original:
Derivar da `Camera2DState` o retângulo de células visíveis, com uma margem de
1 célula (a rampa de borda do shader lê vizinho) mais a margem que a peça de
manifesto mais alta exige (uma árvore ancorada abaixo da tela ainda desenha
copa dentro dela — medir do manifesto, não chutar). Aplicar em:

- `TerrainRenderer.eachCell` → recebe a janela e itera só ela;
- `drawTileLayer` → mesmo laço restrito;
- `ManifestRenderer.Draw` → testa o retângulo desenhado contra a janela.

Efeito esperado no `world_03`: de 23 044 quads para ~1 400 (127 células × 10
passes de material, antes de qualquer outra otimização) — **fator ~16**.

**R2 — Máscaras de borda pré-computadas (impacto alto, risco baixo).**
`edgeMask`/`cornerMask` dependem só da camada `ground`, que é imutável em
runtime. Calcular uma vez no `Load()` e guardar `[]maskEntry` por material.
O quadro passa a ler um array em vez de fazer 8 buscas lineares por célula.
Elimina ~150 mil `stackRank` por quadro.

**R3 — Índice de células por material (impacto alto, risco baixo).**
Guardar, no `Load()`, a lista de células que cada material pinta (`map[int][]cellIdx`).
As 10 varreduras completas da grade viram 10 iterações sobre listas curtas,
já intersectadas com a janela do R1. No `world_01` o material `stone` visita
hoje 2 700 células para pintar algumas dezenas.

**R4 — IMPLEMENTADO (09/08/2026), com fallback.**
`internal/tilemap/terrain_batch.go` + `assets/shaders/terrain_batch_*.fs`:
máscara de vizinhos vai para textura, índice da célula vai no tint do
`DrawTexturePro`. Um `BeginShaderMode` por MATERIAL: 1.104 draw calls no
`world_03` viram ~10. O caminho por célula continua no código e assume se o
shader não carregar, faltar uniform ou a grade passar de 255 células por lado.
Conferido por `work/perf/verify_terrain_batch.py`. Descrição original:
A troca de shader por célula é o custo estrutural. Duas saídas, nessa ordem de
preferência:

- **Uniforms por vértice em vez de por draw.** Passar `edge`, `corner` e
  `tileRect` como atributos de vértice (via cor/UV secundária de um mesh
  próprio) e desenhar **todas as células de um material em um único
  `DrawMesh`**. 18 844 draw calls viram 9 (uma por material). Exige montar um
  mesh por material no `Load()` — o mesmo dado que o R3 já produz.
- **Alternativa mais barata de implementar:** manter o shader ligado por
  material (um `BeginShaderMode` por material, não por célula) e mover
  `edge`/`corner` para uma textura de máscara do tamanho da grade, amostrada
  pelo fragment shader pela posição de mundo. 9 binds por quadro.

Trade-off: o R4 mexe no pipeline de terreno, que é a parte visualmente mais
delicada do projeto (`doc/tilemap.md` documenta em detalhe por que a pilha e a
rampa são o que são). **Só vale começar depois de R1–R3 medidos**, porque
R1–R3 podem já ter tirado o problema da zona crítica.

**R5 — `manifestOwnsGID` fora do laço de tiles (impacto médio, risco baixo).**
Resolver uma vez no `Load()` um `map[int]bool` de GIDs pertencentes a
manifesto, ou melhor: um `set` de `firstgid` de tileset de manifesto. Hoje
alocamos `filepath.Clean` por tile por quadro.

**R6 — Objetos de manifesto indexados por camada (impacto baixo, risco baixo).**
Resolver o ponteiro da camada de cada manifesto no `Load()` em vez de varrer
`m.Layers` 18 vezes por quadro.

**R7 — `rl.IsKeyPressed(rl.KeyF3)` sai de dentro de `DrawWithCamera`.**
Leitura de input dentro do passe de desenho é acoplamento, não custo — mas
custa uma linha corrigir e evita que o toggle dependa de quem desenha.

---

## 2. Memória e VRAM

### 2.1 Medida (VRAM = largura × altura × 4 bytes, que é o que a GPU guarda)

| Grupo | Sempre carregado | Tamanho |
|---|---|---|
| Atlas de manifesto (9) | sim, em todo mapa | **71,9 MB** |
| Texturas de terreno (10) | sim, em todo mapa | 6,8 MB |
| Toppings do mapa | por mapa | 2–6 MB |
| Fita de trilha | se o mapa tem trilha | 1,0 MB |
| **Total por mapa** | | **~82–86 MB** |

O `world_01` de fato usa: `village_vegetation` (4 MB), `forest_trees` (1,5 MB),
`village_buildings` (6 MB), `village_fence_v2` (16 MB), pilha verde (0,75 MB) e
um topping (2 MB) ≈ **30 MB**. Os outros **~55 MB são desperdício puro** —
muralha de fortaleza, defesas de cerco, props de mata escura e pinheiros que
aquele mapa nunca desenha.

Os três maiores atlas são 2048×2048 = 16 MB cada: `fortress_wall`,
`siege_defenses` e `village_fence_v2`. O `village_fence_v2` chama atenção: 16 MB
de VRAM para 16 peças de cerca.

Sprites (carregados sob demanda, e isso está certo):

| | |
|---|---|
| 5 folhas de personagem | 3,75 MB cada = 18,8 MB |
| 5 `reference.png` | 6,0 MB cada = 30,0 MB |
| Inimigos (orc idle+walk, slime, lobo) | 13,5 MB |

`ShowCharacterSelect` carrega os 5 `reference.png` (30 MB) e descarrega ao
sair — correto. Mas `portraitCache` em `internal/ui/dialogue_box.go` carrega o
mesmo `reference.png` por personagem que fala e **nunca descarrega**: um mapa
com diálogo de cinco personagens deixa 30 MB residentes até o fim da sessão.

### 2.2 Plano de memória

**M1 — IMPLEMENTADO (09/08/2026).** Descrição original:
`NewManifestRenderers()` passa a receber o `*TiledMap` e só instancia o
renderer cujo manifesto tem alguma peça citada nas camadas de objeto daquele
mapa. Mesma ideia para `NewTerrainRenderer`: varrer a camada `ground` uma vez,
descobrir os materiais presentes, e carregar as pilhas que os contêm (a pilha
inteira, não só o material — a camada de baixo é desenhada sob a de cima).
Economia esperada: **~55 MB no `world_01`**.

**M2 — IMPLEMENTADO (09/08/2026).** `internal/tilemap/texture_cache.go`.
Descrição original:
Um `texcache` no pacote `assets` ou `tilemap`: `Acquire(path)` devolve a
textura e incrementa; `Release(path)` decrementa e descarrega no zero. Resolve
dois problemas de uma vez — a troca de mapa deixa de re-decodificar 30–80 MB de
PNG (o *stutter* do portal) e atlas compartilhado entre mapas deixa de existir
em duplicata.

**M3 — Descarregar antes de carregar, com o cache do M2 (impacto médio).**
Com M2 no lugar, `travelTo` pode `Acquire` o destino e depois `Release` a
origem sem pico: o que os dois usam nunca sai da VRAM, e o que só a origem
usava é liberado. Sem M2, inverter a ordem arrisca tela preta — **não inverter
sem o cache**.

**M4 — Comprimir/reduzir os atlas grandes (impacto alto, risco de arte).**
`village_fence_v2` a 2048² para 16 peças de cerca é o candidato óbvio a
reempacotamento. Vale medir a ocupação real de cada atlas antes de mexer; é
trabalho de arte, não de código, e passa pela `create-tiled-assets`.
No Android, considerar formato comprimido de GPU (ETC2/ASTC) — a economia é de
4× a 8× em VRAM e o custo é qualidade e pipeline de build.

**M5 — Retrato de diálogo com teto (impacto baixo, risco baixo).**
`portraitCache` passa a descarregar ao trocar de mapa, ou guarda no máximo o
elenco da cena atual. 30 MB residentes por 200×250 px de retrato na tela é
desproporcional — o retrato podia inclusive ser uma versão reduzida da
`reference.png`, gerada no build.

---

## 3. Simulação (CPU do host)

Com 83 inimigos de guarnição no `world_03`:

| Onde | Custo por quadro |
|---|---|
| `ResolveEnemyOverlap` | O(n²) × `enemyOverlapIterations` ≈ **13 800 pares** |
| `Enemy.Update` (separação) | cada inimigo percorre os 83 ativos ≈ **6 900 pares** |
| `sort.Slice(active)` | ordenação por string a cada quadro |
| `checkProjectileCollisions` | projéteis × inimigos, lista completa |

Nada disso queima uma máquina de desktop hoje, mas cresce ao quadrado e o
mapa 3 já tem 83 monstros. Correções, todas de risco baixo:

**S1 — Grade espacial (spatial hash) para vizinhança.** Uma grade de células do
tamanho do maior raio resolve separação, overlap e colisão de projétil com
consulta local em vez de par a par. É a mesma ideia que o `footprintIndex` já
usa para colisão de props — o padrão existe no projeto.

**S2 — Ordenação estável sem realocar.** Manter o slice `active` reutilizado
entre quadros (`active = active[:0]`) e ordenar por um índice numérico em vez
de comparar strings. Hoje há uma alocação por quadro mais N comparações de
string.

**S3 — Simulação em passo fixo, desacoplada do render.** `UpdateSimulation(dt)`
roda uma vez por quadro renderizado, então a física do host muda com o fps.
Um acumulador a 30 ou 60 Hz fixo torna a simulação determinística e permite
baixar a taxa de simulação sem baixar a de render (ver N2).

---

## 4. Rede

### 4.1 Medida

Tamanho real de um objeto serializado, com IDs do formato que o jogo gera:

| Payload | Bytes JSON |
|---|---|
| `PlayerState` | 231 |
| `EnemyState` | 125 |
| `ProjectileState` | 152 |

Tráfego **por cliente**, a 60 Hz, estado completo em todo tique:

| Cenário | Por tique | Por segundo |
|---|---|---|
| 2 jogadores, 30 inimigos | 5,9 KB | **348 KB/s (2,9 Mbps)** |
| 4 jogadores, 60 inimigos | 10,1 KB | **594 KB/s (4,9 Mbps)** |
| 4 jogadores, 83 inimigos (`world_03`) | 13,0 KB | **763 KB/s (6,2 Mbps)** |

E ainda não é tudo. Três amplificadores somam por cima disso:

**(a) `BroadcastStateUpdate` é chamado duas vezes por caminho.**
`UpdateSimulation` chama `BroadcastFullState` → `BroadcastStateUpdate` a 60 Hz.
**E** o handler de `MsgInput` chama `BroadcastStateUpdate` *a cada input
recebido*, de cada cliente, a 60 Hz cada. Com N clientes o estado de jogador sai
**60 × N** vezes por segundo em vez de 60. Com 4 jogadores são 240 broadcasts/s
de uma lista que mudou em um jogador.

**(b) O log dentro do broadcast.**
`BroadcastStateUpdate` imprime **2 + 2N linhas de log por chamada**. A 240
chamadas/s com 4 jogadores são **~2 400 linhas de log por segundo**, com
`fmt` e I/O síncrono no caminho quente do host. O cliente faz o mesmo em
`handleMessage` para *todo* `state_update` e *todo* `enemy_update`. Isso não é
"um pouco de overhead": é provavelmente o maior custo de CPU do host em
multiplayer, e é a correção mais barata do documento inteiro.

**(c) Uma escrita + flush por mensagem por peer.**
São 4 mensagens por tique (`state_update`, `enemy_update`,
`projectile_update`, `cooldown`), cada uma com `Flush()` próprio. Com
`TCP_NODELAY` (padrão em Go) são ~240 pacotes/s por cliente onde 60 bastariam.

### 4.2 O protocolo em si

Hoje: **JSON sobre TCP, uma conexão, estado completo em todo tique**.

Os problemas estruturais, na ordem em que doem:

1. **Estado completo, sem delta.** Um orc parado no fundo do mapa é
   retransmitido 60 vezes por segundo, íntegro.
2. **JSON.** `"max_health":120` ocupa 17 bytes para um número que cabe em 1.
   Nomes de campo são ~60% do payload. E `encoding/json` aloca em ambos os
   lados a 60 Hz.
3. **TCP para estado de movimento.** Posição é dado que **substitui** o
   anterior: perder um snapshot não importa, o próximo já corrige. Mas o TCP
   garante entrega ordenada, então um pacote perdido **segura todos os
   seguintes** (head-of-line blocking) — num Wi-Fi ruim isso vira congelamento
   e depois teleporte, que é exatamente o pior comportamento possível para
   posição.
4. **Sem interpolação no cliente.** O cliente desenha a última posição recebida
   crua. Qualquer variação no intervalo entre snapshots aparece como tremor.
5. **IDs como string em todo pacote.** `"enemy_1699999999_12"` são ~22 bytes,
   60 vezes por segundo, por inimigo.

### 4.3 Plano de rede

Dividido em duas faixas: o que **não muda o protocolo** (fazer primeiro,
sempre) e o que **muda** (decisão consciente, trade-offs declarados).

#### Faixa A — sem mudar o protocolo

**N1 — Tirar o log do caminho quente (impacto altíssimo, risco nenhum).**
Remover ou colocar atrás de um nível de log os `log.Printf` por jogador/por
inimigo em `BroadcastStateUpdate`, `sendStateToClient` e nos handlers do
cliente. Manter log de evento (join, disconnect, morte), não de tique.
**Fazer isto antes de qualquer medição de rede** — os números atuais estão
contaminados por ele.

**N2 — Taxa de snapshot própria — IMPLEMENTADO (09/08/2026).**
`SnapshotHz = 20` em `internal/network/broadcast_rate.go`. Simulação segue a
60 Hz. Descrição original:
Snapshot a 20 Hz em vez de 60 corta o tráfego a **um terço** (2,9 Mbps → ~1
Mbps) e é padrão da indústria. Depende de N5 (interpolação) para não piorar a
suavidade — os dois andam juntos.

**N3 — IMPLEMENTADO (09/08/2026).** E com ele apareceram DOIS irmãos que
teriam anulado o N2 sozinhos: `UpdatePlayerState`, chamado a cada quadro pelo
laço do jogo para o próprio host, e `UpdatePlayerPosition`. Os três só
atualizam estado agora. Descrição original:
O input só atualiza o estado autoritativo; quem publica é o tique da simulação.
Elimina a amplificação por número de clientes.

**N4 — Delta: só mandar o que mudou (impacto alto, risco médio).**
Manter o último snapshot enviado por peer e emitir apenas entidades cuja
posição/vida mudou além de um limiar, mais a lista explícita de IDs removidos.
Numa guarnição em que a maioria dos monstros está parada no próprio posto,
isso corta a maior parte do tráfego do `world_03`. Regra que **não pode ser
perdida**: hoje "lista vazia" significa "limpe tudo" (`doc/network.md`); com
delta, esse significado tem de virar uma mensagem explícita, ou snapshot
completo periódico (a cada ~1 s) como âncora.

**N5 — IMPLEMENTADO (09/08/2026).** `internal/network/interpolation.go`,
100 ms de atraso. Descrição original:
O cliente guarda os dois últimos snapshots e desenha ~100 ms no passado,
interpolando. É o que permite baixar a taxa (N2) sem o jogador ver diferença —
e melhora a suavidade *mesmo sem* baixar a taxa.

**N6 — Uma escrita por tique (impacto médio, risco baixo).**
Agrupar as 4 mensagens do tique num único envio com um `Flush()` só. Menos
syscalls, menos pacotes, menos overhead de cabeçalho.

**N7 — IMPLEMENTADO (09/08/2026).** `internal/network/wire.go`: identidade
anunciada uma vez, estado a cada tique. `PlayerState` 231 → 165 B,
`EnemyState` 125 → 71 B. Descrição original:
`Color` e `Character` do `PlayerState` são constantes depois do join — mandar
no join e no `state_update` de entrada, não a 60 Hz. `MaxHealth` idem.
`EnemyState.Color` e `Type` são constantes por inimigo — mandar no evento de
spawn. Isso sozinho tira ~40% dos bytes sem tocar em JSON/TCP.

#### Faixa B — mudar o protocolo

Foi pedido explicitamente que, se existir protocolo melhor, ele entre no plano
com os trade-offs declarados. Existe, e o trade-off é real.

**N8 — Serialização binária no lugar de JSON.**

- *Ganho:* 3× a 5× menos bytes e muito menos alocação nos dois lados. Um
  `EnemyState` de 125 B vira ~14 B (id uint32, x/y int16, vida uint8, flags).
- *Custo:* o protocolo deixa de ser legível no `tcpdump` e num `log`, e
  qualquer divergência de versão entre host e cliente vira corrupção silenciosa
  em vez de erro de parse. Exige campo de versão e recusa explícita de conexão
  incompatível.
- *Meio-termo honesto:* CBOR ou MessagePack dá ~2× de ganho com biblioteca
  pronta, mantendo o modelo de dados atual e a possibilidade de dump legível.
  Se a escolha for essa, o passo é mecânico (trocar o codec, manter as structs).

**N9 — UDP para estado, TCP para eventos.**

O desenho: um socket UDP carrega `state_update`, `enemy_update` e
`projectile_update` (dados que **substituem** o anterior — perder é aceitável),
cada datagrama com número de sequência; o cliente descarta o que chega fora de
ordem. O TCP existente continua carregando o que **não pode ser perdido**:
join, `combat_event`, `dialogue`, `reset_stage`, `game_over`, `cooldown`.

- *Ganho:* acaba o head-of-line blocking. Numa rede com 2% de perda, o
  comportamento passa de "congela e teleporta" para "um quadro de posição
  antiga". É a diferença entre jogável e não jogável em Wi-Fi doméstico ruim.
  Um datagrama de estado também dispensa o custo de fluxo/ACK do TCP.
- *Custo (declarado, e não é pequeno):*
  - **NAT.** Sem conexão, o host precisa aprender o endereço UDP de cada
    cliente. Em LAN, que é o único cenário suportado hoje
    (`doc/network.md`: "Discovery is LAN-only"), isso é simples: o cliente
    manda um "hello" UDP após o join TCP e o host responde para a origem.
    Fora da LAN vira problema de furo de NAT, que é projeto próprio.
  - **Firewall.** Já se sabe que o firewall do Windows é um ponto de atrito
    aqui (`doc/network.md` documenta a dependência de TCP 9000 inbound). Uma
    porta UDP a mais é uma exceção de firewall a mais, e o modo de falha —
    "conectou mas não vejo ninguém se mover" — é mais confuso para o jogador
    que "não conectou". Precisa de fallback automático para TCP quando nenhum
    datagrama chegar em ~2 s.
  - **MTU.** Datagrama tem de caber em ~1200 bytes úteis. Com 83 inimigos em
    JSON isso estoura; com N8 (binário) cabe. **N9 depende de N8 na prática**,
    ou de particionar o snapshot por datagrama.
  - **Complexidade.** Dois transportes, duas ordens de chegada, um ponto novo
    onde host e cliente podem discordar sobre o que já aconteceu.
- *Recomendação:* **não é o primeiro passo.** N1+N3 são de graça, e N2+N5+N7
  reduzem o tráfego em ~5× mantendo um transporte só. Se depois disso a
  latência sob perda ainda for o problema — e a única forma de saber é medir
  com perda induzida — aí N8+N9 juntos, nessa ordem, valem o custo.

**N10 — QUIC / ENet / Steam Networking, como alternativa a rolar o próprio.**

- **ENet** é a resposta clássica: UDP com canais, uns confiáveis e outros não,
  exatamente o modelo que N9 descreve, já resolvido e testado há vinte anos.
  Custo: dependência C, o que atravessa o build Android (o projeto já linka
  raylib via cgo, então não é território novo, mas é mais uma peça no NDK).
- **QUIC** (`quic-go`) dá streams independentes sobre UDP, sem
  head-of-line blocking entre streams, com criptografia e Go puro. Custo: o
  handshake TLS pede certificado, o que num host de LAN doméstico é cerimônia
  desproporcional (certificado auto-assinado + confiança explícita), e a
  biblioteca é grande para o que se ganha aqui.
- **Steam Datagram Relay** resolveria NAT e roteamento de uma vez, mas amarra o
  jogo à plataforma e não vale enquanto o escopo for LAN.

Para o escopo atual — LAN, host autoritativo, até ~4 jogadores — **ENet é a
escolha mais defensável** se a Faixa A não bastar. Não é a mais barata; é a que
tem menos armadilha por conta própria.

---

## 4½. Conforto visual — por que o mapa 3 cansa a vista e o mapa 1 não

Relato: andar pelo `world_03` movendo a câmera causa tontura; o `world_01` não.
Medido com `work/perf/measure_comfort.py`.

### O que a medida DESCARTA

A hipótese intuitiva — "o mapa 3 é mais poluído, tem mais coisa na tela" —
**não se sustenta**. Por tela de 1920×1080 (127 células):

| | `world_01` | `world_03` |
|---|---|---|
| Objetos (árvore, casa, prop) por tela | 8,2 | 8,5 |
| Tiles de topping por tela | 12,9 | 12,8 |
| Transições de material por tela | 13,6 | **8,6** |

A densidade de cenário é praticamente idêntica, e o mapa 1 tem *mais* junções
de terreno por tela. **O problema não está na composição do mapa.** Está em
como o mesmo cenário é desenhado.

### O veredito do `world_02`

O experimento B (§ "Experimentos decisivos") foi respondido em jogo:
**`world_01` é o mais confortável, `world_02` é aceitável, `world_03` é o que
causa tontura.** Essa ordem é o dado mais informativo da investigação, porque
**elimina a hipótese que parecia mais forte**.

| | `world_01` | `world_02` | `world_03` |
|---|---|---|---|
| Conforto relatado | agradável | **aceitável** | tontura |
| Área da tela com ruído do shader | 14% | **100%** | 100% |
| Camadas de ruído por célula | 0,23 | 3,23 | 4,49 |
| Luminância média do chão | 131 | **77** | 64 |
| Contraste local (RMS) | 12,8 | **23,6** | 20,3 |
| Draw calls de terreno | 3 323 | 12 702 | **23 044** |
| Inimigos simultâneos | 12 (pico) | 35 (pico) | **83 (permanente)** |

O `world_02` tem **a mesma cobertura de ruído do `world_03`** (100% da tela) e
**contraste local ainda MAIOR** (23,6 contra 20,3), com luminância na mesma
faixa. Se o dither ou a arte escura fossem a causa dominante, o `world_02`
teria de incomodar tanto quanto o 3. Ele não incomoda.

Só duas grandezas crescem junto com o desconforto na ordem 1 < 2 < 3, e as
duas são carga de quadro:

- **draw calls de terreno**: 3 323 → 12 702 → 23 044;
- **inimigos vivos ao mesmo tempo**: 12 → 35 → **83**.

E a segunda separa o 2 do 3 de um jeito que a primeira não separa sozinha: no
`world_02` os 35 inimigos são o **pico transitório** da horda 2, num momento em
que o jogador está lutando parado e não atravessando o mapa. No `world_03` os
**83 nascem no carregamento do mapa e ficam** — `PlaceGarrison` não repõe nem
esvazia, então a carga máxima está presente desde o primeiro segundo,
exatamente enquanto o jogador caminha e rola a câmera. É a única fase em que
"andar pelo mapa" e "carga máxima" acontecem ao mesmo tempo.

**Conclusão: a causa dominante é frame pacing, não a arte nem o dither.** O
ranking abaixo foi reordenado por causa disso — a análise do dither continua
válida e vale corrigir, mas ela é agravante, não a raiz.

### Causa 1 — carga de quadro: 30× mais draw calls e 83 inimigos permanentes

Duas cargas somadas, e o `world_03` é o único mapa que tem as duas ao mesmo
tempo o tempo todo.

**Terreno.** Do §1: 18 844 draw calls de terreno por quadro contra 623 do
`world_01` — porque cada célula com camada é embrulhada em
`Begin/EndShaderMode`, que esvazia o batch.

**Guarnição.** Os 83 inimigos do `world_03` custam, por quadro, no host:

| n inimigos | Pares de separação (`Enemy.Update`) | Pares de overlap (`ResolveEnemyOverlap`) | Sprites + barras |
|---|---|---|---|
| 12 (`world_01`) | 144 | 264 | 24 |
| 35 (`world_02`) | 1 225 | 2 380 | 70 |
| **83 (`world_03`)** | **6 889** | **13 612** | **166** |

Os dois laços são O(n²) e o `world_03` roda os dois no teto desde o carregamento.
Vale registrar que o próprio código já previu isso, em `wave_runs.go`, sobre os
35 concorrentes da horda 2 do `world_02`:

> *"35 simultaneos e o numero a MEDIR, nao um numero fechado […] este e o ponto
> onde o quadro pode cair. Se cair, baixar aqui antes de mexer em qualquer
> outra coisa."*

O `world_03` está permanentemente em **2,4× esse número**.

E o desenho não ajuda: `EntityManager.DrawAll` percorre o **mapa** de inimigos
(ordem aleatória, então os tipos se intercalam e a textura troca a cada
inimigo), **sem culling nenhum** — os 83 são desenhados mesmo os que estão
fora da tela — e intercala `DrawHealthBar` entre um sprite e o próximo, o que
quebra o batch de textura mais uma vez por inimigo.

**O efeito na câmera é o que o jogador sente.** A câmera segue o jogador
diretamente, sem suavização e sem passo fixo
(`internal/game/camera.go`, `p.Update(dir, dt, ...)`), então variação de tempo
de quadro vira variação de deslocamento:

| fps | passo da câmera por quadro (jogador a 200 u/s) |
|---|---|
| 60 estável | 3,3 / 3,3 / 3,3 / 3,3 px |
| 60/30 alternando | 3,3 / **6,7** / 3,3 / **6,7** px |
| 40–50 irregular | 4,4 / 5,3 / 3,8 / 4,9 px |

Judder de rolagem em tela cheia é causa reconhecida de desconforto, e é a única
explicação que reproduz a ordem 1 < 2 < 3 relatada.

### Causa 2 — o dither do shader vira ruído dinâmico ao mover a câmera

`assets/shaders/terrain_blend_{100,330}.fs`, últimas linhas:

```glsl
float n = fract(sin(dot(uv*127.0, vec2(12.9898,78.233)))*43758.5453);
mask = clamp(mask + (n-0.5)*0.16, 0.0, 1.0);
```

É um hash de ruído branco somado ao alpha da camada. Ele existe por um bom
motivo — quebrar o *banding* da rampa de borda. Mas tem dois defeitos, e os
dois só aparecem no mapa 3:

**(a) O hash não tem coerência espacial nenhuma, então re-sorteia a cada
deslocamento subpixel da câmera.** Medido: deslocar `uv` em 0,1% de célula
(0,13 px de tela) já produz um valor completamente descorrelacionado.

```
uv deslocado de 0,13 px:  0.773  0.505  0.076  0.770  0.416  0.592
```

Com a câmera parada o padrão é estático e ninguém vê nada. Com a câmera
andando, **cada pixel da tela sorteia um valor novo a cada quadro**. O chão
ferve. E ferve *incoerentemente*: uma textura em movimento dá sinal de
movimento numa direção; ruído branco dá sinal em todas ao mesmo tempo, que é
exatamente o estímulo que o sistema visual não consegue integrar. Isso casa
com o relato ao pé da letra — incomoda ao mover, não ao parar.

**(b) O dither é aplicado no interior da célula, não só na rampa.** Com
`mask = 1.0` no miolo, `1.0 + (n-0.5)*0.16` ainda modula metade dos pixels em
até 8%. Ou seja: o ruído cobre a superfície inteira de toda célula que recebe
camada, não a faixa de fade que ele deveria dissolver.

E é aqui que os dois mapas se separam de vez:

| | `world_01` | `world_02` | `world_03` |
|---|---|---|---|
| Área da tela com ruído | **14%** | 100% | **100%** |
| Camadas de ruído empilhadas por célula (média) | **0,23** | 3,23 | **4,49** |
| Distribuição | 86% com 0 camadas | — | 40% com 3, 19% com **7** |
| Modulação de luminância medida | **0,1%** | — | **1,2%** |

No `world_01`, **86% do chão não passa pelo shader** — a pilha verde só tem
camada onde há terra ou pedra, e a vila é quase toda grama base. No `world_03`
a pilha da mata cobre o mapa inteiro, com 4,5 camadas em média, **cada uma
sorteando o próprio ruído independentemente**.

1,2% de modulação parece pouco em números absolutos. Não é: o limiar de
detecção humana para ruído *dinâmico* de campo cheio fica na casa de 0,5–1%, e
aqui ele cobre 100% da tela, a 60 Hz, num fundo escuro (onde a pupila está
mais aberta e a sensibilidade a cintilação é maior).

Por que isto é **agravante e não a raiz**: o `world_02` tem a mesma cobertura
de 100% e é aceitável em jogo. O ruído piora o quadro que já está instável —
não produz o desconforto sozinho. Continua valendo corrigir, porque a correção
é barata e não recompila Go.

### Causa 3 — o chão anda em passos inteiros, os sprites em passos suaves

Duas coisas somadas:

- O terreno **não define filtro de textura**, então usa o padrão do raylib, que
  é `POINT` (vizinho mais próximo). Sprites (`ApplySpriteFilter`) e a fita da
  trilha usam `BILINEAR` explicitamente.
- A câmera não é ancorada em pixel: `Camera.Target` recebe `p.Position`, que é
  `float32` (`camera.go`), e `Offset` é `sw/2` — meio pixel em tela de largura
  ímpar.

Resultado: o chão é amostrado por vizinho mais próximo com um deslocamento
fracionário que muda continuamente, então ele **salta de texel em texel**
enquanto árvores, monstros e o jogador deslizam suavemente por cima. As duas
camadas discordam sobre onde o mundo está. Isso vale para os dois mapas, mas a
amplitude do artefato é o contraste local da arte, e a do mapa 3 é 37% maior.

### Causa 4 — a arte é mais escura e mais contrastada (isto é intenção)

| | `world_01` | `world_03` |
|---|---|---|
| Luminância média do chão | 131 | **65** |
| Contraste local (RMS) | 14,6 | **20,0** |
| Frequência espacial alta | 12,4 | 12,9 |

A frequência espacial é praticamente igual — a arte do mapa 3 **não é mais
"suja"** que a do mapa 1. O que ela é: **metade do brilho com 37% mais
contraste**. Isso é o bioma sombrio funcionando como projetado
(`doc/art_style.md`), e é o último item a mexer, não o primeiro: cena escura
com alto contraste amplifica todos os artefatos acima, mas sozinha não produz
ruído dinâmico.

### Causa 4 revisada — a arte do mapa 3 TEM mais informação, e minha primeira medida não viu

A Causa 4 dizia que a frequência espacial da arte "empatava" (12,4 contra
12,9) e concluía que o mapa 3 não é mais "sujo" que o mapa 1. **O número estava
certo e a pergunta estava errada**, por dois motivos:

1. Ela media o gradiente entre pixels **vizinhos** — a frequência mais alta que
   existe na imagem, e justamente onde o olho é **menos** sensível. A curva de
   sensibilidade ao contraste tem pico por volta de 4 ciclos/grau (~12 px de
   período numa tela típica) e cai dos dois lados.
2. Ela olhava **um tile de cada vez**, então não via o período de repetição.

Medindo a **tela composta** (camada `ground` real, texturas reais, mesma janela
de span) e pesando o espectro pela sensibilidade do olho —
`work/perf/measure_information.py`:

| Mapa | Luminância | Contraste/luminância | **Banda média (8–64 px)** | Carga perceptual |
|---|---|---|---|---|
| `world_01` | 129 | 0,111 | **19,9 %** | 1,0× |
| `world_02` | 71 | 0,338 | **27,6 %** | 13,8× |
| `world_03` | 63 | 0,328 | **34,1 %** | 14,1× |
| `world_04` | 43 | 0,418 | 32,0 % | 18,2× |

Duas honestidades sobre esta tabela:

- **A carga total NÃO separa o `world_02` do `world_03`** (13,8 contra 14,1),
  e o 02 é aceitável em jogo. Então ela sozinha não é a resposta.
- **A fração na banda média separa, e na ordem exata do desconforto
  relatado**: 19,9 → 27,6 → 34,1 para agradável → aceitável → tonto.

**O mecanismo é o `span`.** A grama da vila é uma folha espremida numa célula
de 128 px, então 64,5 % da energia dela cai na banda **fina**, que o olho
atenua — ela é "ruidosa" numa frequência descartada. As texturas do bioma
escuro são folhas de 512 px desenhadas 1:1, com estrutura no tamanho de 8 a
64 px: exatamente onde a visão amplifica. Mesmo gradiente por pixel, carga
perceptual completamente diferente.

Isso também explica por que o `span` foi um ganho de nitidez real e um custo de
conforto real ao mesmo tempo: ele foi criado para pôr mais conteúdo único por
área (`doc/tilemap.md`), e conseguiu.

Onde mexer, por material:

| Material | Contraste/lum | Banda média | Carga |
|---|---|---|---|
| `siege gravel` | 0,280 | **48,9 %** | 6 045 |
| `dark flagstone` | 0,219 | **43,5 %** | 2 683 |
| `castle blocks` | 0,298 | 40,1 % | 5 974 |
| `dark grass` | 0,355 | 32,8 % | 9 903 |
| `sparse grass` | 0,361 | 31,4 % | 10 511 |
| `bare soil` | 0,298 | 29,6 % | 7 226 |
| `castle water` | **0,648** | 29,7 % | **22 612** |
| `forest grass` | 0,268 | **12,9 %** | 5 314 |
| `grass` (vila) | 0,096 | 20,4 % | 578 |

`siege gravel` e `dark flagstone` são os que mais concentram energia na banda
sensível — e são o chão do pátio da fortaleza, que é onde a fase 3 se passa.
`forest grass` é a mais calma do bioma inteiro (12,9 %), o que mostra que dá
para ser escura e densa sem ser cansativa.

**Previsão falseável:** o `world_04` tem 32,0 % na banda média e a maior carga
de todos (18,2×, puxada pela água a 0,648 de contraste relativo). Ele deve
incomodar quase como o mapa 3, ou mais. Se não incomodar, esta métrica está
errada e é para descartá-la.

Três alavancas, em ordem de dano à direção de arte:

- **A1 — comprimir o contraste relativo das texturas do bioma.** Transformação
  medida sobre os PNGs, reversível, sem tocar em paleta nem composição: o bioma
  continua escuro e morrendo. A carga cai com o quadrado da redução.
- **A2 — atenuar especificamente a banda de 8–64 px**, preservando o resto.
  Mais cirúrgico, mas é justo a estrutura que faz a pedra ler como pedra.
- **A3 — baixar o span de 4×4 para 2×2.** Reverte uma decisão deliberada e
  documentada (`doc/tilemap.md`: "reduzir para 256 derruba o span para 2×2 e
  joga metade do ganho fora"). Não recomendado.

Nenhuma foi aplicada: é decisão de direção de arte, e passa pela
`create-tiled-assets`.

### Experimentos ainda úteis (nenhum precisa recompilar Go)

**B — RESPONDIDO.** O `world_02` é aceitável, o que aponta frame pacing e não o
dither. Registrado acima em "O veredito do `world_02`".

**A — isolar o dither.** Vale ainda assim, para medir quanto do desconforto
residual é dele. Os shaders são lidos do disco em runtime (`rl.LoadShader`),
então basta editar o arquivo e reabrir o jogo. Nos **dois** arquivos
`assets/shaders/terrain_blend_100.fs` e `terrain_blend_330.fs`, trocar `*0.16`
por `*0.0`:

```glsl
mask = clamp(mask + (n-0.5)*0.0, 0.0, 1.0);
```

Fica com banda visível nas rampas de borda — é teste, não correção.

**C — confirmar a hipótese da guarnição.** Reduzir temporariamente os esquadrões
de `garrisonSquads` (`internal/network/garrisons.go`) para ~1/3 dos efetivos e
andar pelo `world_03`. Se ficar confortável, a carga de entidade é a metade
dominante da Causa 1; se não mudar, é o terreno. Isto **recompila Go**, mas é o
único jeito de separar as duas metades — e a resposta decide se a Fase 2 (§5)
basta ou se a Fase 5 (grade espacial) tem de vir junto.

**D — contador de fps.** Antes de qualquer correção, ver o número. `rl.DrawFPS`
no HUD, ou o item 2 do §6. "Ficou melhor" sem medida não fecha nada.

### Correções propostas

**C1 — Ancorar o hash no texel e restringi-lo à rampa (impacto alto, risco
baixo, não recompila Go).** Nos dois `terrain_blend_*.fs`:

```glsl
// Ancorar o hash na grade de texel: sem isso, 0,13 px de deslocamento de
// camera sorteia um valor novo para o MESMO ponto do mundo, e o chao ferve
// enquanto o jogador anda.
vec2 texel = floor(uv * 128.0);
float n = fract(sin(dot(texel, vec2(12.9898,78.233)))*43758.5453);
// O dither existe para quebrar banda NA RAMPA. No interior (mask = 1) ele so
// espalha ruido pela tela inteira — 100% da area do mapa 3 contra 14% do
// mapa 1. Este peso vale 1 no meio da rampa e 0 nos dois extremos.
float ramp = mask * (1.0 - mask) * 4.0;
mask = clamp(mask + (n-0.5)*0.16*ramp, 0.0, 1.0);
```

O primeiro trecho mata a fervura (o ruído passa a se mover *com* o chão, como
uma textura). O segundo tira o ruído do miolo das células, o que no `world_03`
reduz a área ruidosa de 100% da tela para as faixas de fade. Conferir contra o
`world_01` depois: ele é mapa aprovado e a mudança tem de ser invisível lá
(86% da área dele não passa pelo shader de qualquer forma).

**C2 — Ancorar a câmera em pixel (impacto médio, risco baixo).** Ao fim de
`Camera2DState.Update`, depois do clamp, arredondar `Target` e `Offset` para
inteiro. O chão e os sprites passam a concordar sobre onde o mundo está. O
custo é o jogador andar em passos de 1 px, que a 200 u/s é imperceptível e é o
que jogo 2D faz.

**C3 — Culling de câmera no terreno — IMPLEMENTADO (09/08/2026).**
Era a correção da causa dominante, metade "terreno". Medido: `world_03` de
23.044 para 1.071 quads por quadro (22×). Ver R1 no §1.4.

**C4 — Culling e batching dos inimigos — IMPLEMENTADO (09/08/2026).**
`EntityManager.DrawAll` recebe a janela visível; `entity/enemy_draw_box.go`
define a caixa de cada inimigo a partir da geometria do **quadro** (e não do
`Radius` de combate, que cortaria a cabeça e o espadão do orc na borda). O
mesmo culling foi aplicado ao laço de inimigos remotos do cliente. A outra
metade da Causa 1, e nada disso existia:

- `EntityManager.DrawAll` testa o inimigo contra a janela da câmera antes de
  desenhar (os 83 do `world_03` são todos desenhados hoje, on-screen ou não);
- desenhar **todos os sprites primeiro e todas as barras de vida depois**, em
  vez de intercalar — a barra entre dois sprites quebra o batch de textura uma
  vez por inimigo;
- iterar em ordem **agrupada por tipo** em vez da ordem do `map`, para trocar
  de textura uma vez por tipo em vez de uma vez por inimigo.

**C5 — Grade espacial na simulação (S1 do §3, impacto alto).**
Os 6 889 + 13 612 pares por quadro do `world_03` viram consulta local. O padrão
já existe no projeto (`footprintIndex`), é copiá-lo.

**C6 — Passo fixo de simulação (S3).** Desacopla a velocidade do jogador do
tempo de quadro. Com fps instável, hoje a câmera anda em passos irregulares;
com passo fixo mais interpolação de render, ela anda em passos iguais mesmo se
o fps oscilar. É a rede de segurança: mesmo que o fps caia, a câmera não
tranca.

**C7 — Rever o efetivo da guarnição (decisão de design, não de código).**
83 inimigos permanentes é 2,4× o número que o próprio `wave_runs.go` já
apontava como limite a medir. Se C3–C6 não bastarem, este é o botão — e é o
único item aqui que muda o jogo, então vem depois dos que não mudam.

**C8 — Arte, por último e só se necessário.** Subir a luminância base da pilha
da mata ou baixar o contraste local das duas gramas escuras. O `world_02` já
mostrou que a arte escura **não** é a causa: ele tem contraste local maior que
o do `world_03` e é confortável. Mexer aqui sem necessidade é gastar direção de
arte para consertar um problema de engenharia.

### Ordem

1. ~~**D** — pôr um contador de fps na tela.~~ **FEITO**: `game.DrawPerfHUD`,
   atrás do F3, com fps, tempo médio e **pior quadro** dos últimos 120, mais os
   contadores de `tilemap.FrameStats()`.
2. ~~**C3** — culling de câmera no terreno.~~ **FEITO**, 22× no `world_03`.
3. ~~**C4** — culling e batching dos inimigos.~~ **FEITO**.
4. **C1** — o shader (dois arquivos, reabre o jogo, não recompila Go). Barato,
   e tira o agravante que sobra depois que o quadro estabiliza.
5. **C2** — ancorar a câmera em pixel. Melhora os três mapas.
6. **C5/C6** se o fps ainda não estiver estável e liso.
7. **C7/C8** só depois de tudo isso, e como decisão de design/arte.

---

## 4¾. Medição em jogo (09/08/2026)

Capturas do F3 em 2560×1600, como host.

### Primeira rodada — logo depois do culling

| | `world_01` | com trilha | `world_03` (pátio) | `world_03` (vazio) |
|---|---|---|---|---|
| fps | 60 | 60 | 60 | 60 |
| Quadro médio | 16,7 ms | 16,9 ms | 16,7 ms | 16,7 ms |
| **Pior quadro** | 16,9 ms | **43,4 ms** | **20,4 ms** | 16,9 ms |
| CPU mapa | 1,0 ms | 4,0 ms | 3,7 ms | 1,0 ms |
| CPU entidades | 0,0 ms | 1,0 ms | 1,0 ms | 0,0 ms |
| Resto (GPU + vsync) | 15,6 ms | 11,9 ms | 12,0 ms | 15,6 ms |
| Quads de terreno | 572 | 1 626 | 1 380 | 892 |
| Inimigos vivos / desenhados | 4 / 4 | 5 / 1 | **81 / 2** | 0 / 0 |

Estas capturas ainda não traziam o nome do mapa (a linha existia mas ficava
atrás da barra de título — ver abaixo), e as duas do meio tiveram de ser
atribuídas por dedução. Uma delas foi atribuída ao mapa errado. É a razão de a
etiqueta existir.

### Segunda rodada — depois de preload, cache da trilha, culling do F3 e N1

| | `world_01` | `world_02` | `world_03` | `world_04` |
|---|---|---|---|---|
| fps | 60 | 60 | 60 | 60 |
| Quadro médio | 16,7 ms | 16,7 ms | 16,7 ms | 16,7 ms |
| **Pior quadro** | **16,9 ms** | **17,2 ms** | **17,0 ms** | **17,8 ms** |
| CPU mapa | 1,0 ms | 2,1 ms | 2,6 ms | 1,7 ms |
| CPU entidades | 0,0 ms | 0,0 ms | 0,5 ms | 0,0 ms |
| Resto (GPU + vsync) | 15,6 ms | 14,6 ms | 13,5 ms | 15,0 ms |
| Quads de terreno | 536 | 991 | 1 472 | 834 |
| Inimigos vivos / desenhados | 2 / 1 | 0 / 0 | **81 / 0** | 0 / 0 |

**O pico acabou.** De 43,4 ms para 17,0 ms — todos os quatro mapas dentro de um
milissegundo do alvo de vsync, e o `world_03` com os 81 inimigos da guarnição
simulados enquanto **nenhum** é desenhado.

Com isso, o custo de quadro deixa de ser candidato a causa do desconforto: não
sobrou judder para explicar. O que resta da análise do §4½ é o **dither do
shader** (C1) e a **âncora de pixel da câmera** (C2), que são independentes do
fps — se ainda incomodar, é aí.

E a **grade espacial (S1/C5) saiu do caminho crítico**: o quadro fecha em
16,7 ms com a simulação O(n²) rodando para 81 inimigos.

Três leituras:

**O culling entregou.** 572 / 1 626 / 1 380 quads onde antes eram 3 323 /
12 702 / 23 044, batendo com o previsto por `verify_viewport.py`. A CPU do mapa
caiu para 1–4 ms e o "resto" de 12–15 ms é espera de vsync: **o quadro está
sobrando tempo**.

**A guarnição deixou de ser custo de desenho.** `81 vivos / 2 desenhados` com o
quadro em 16,7 ms responde o experimento C sem precisar tocar em
`garrisonSquads`. E como o quadro fecha mesmo com a simulação O(n²) rodando
para os 81, **a grade espacial (S1/C5) deixou de ser urgente**.

**O que sobrou é pico isolado, não fps sustentado.** 43,4 ms com a câmera
andando é um salto de 8,7 px onde os quadros vizinhos andam 3,3 px — é
precisamente o judder que causa o desconforto, e agora é o único defeito
restante. Três causas encontradas e corrigidas:

1. **Textura carregada dentro do passe de desenho.** `enemyTexture` carrega sob
   demanda, então a primeira aparição de um orc parava o quadro para decodificar
   um PNG de 1232×1206 e subir 5,7 MB para a GPU. Corrigido com
   `entity.PreloadEnemyTextures()` antes do laço.
   **O culling piorou esse caso**, e vale registrar: antes todo inimigo era
   desenhado e a folha subia no primeiro quadro depois do spawn, longe da tela;
   depois do culling ela sobe no quadro em que o monstro fica *visível* — o pior
   momento possível.
2. **`Trail.Path` reamostrava e suavizava a curva inteira a cada quadro**,
   alocando centenas de `rl.Vector2` para algo que não muda enquanto o mapa não
   muda. Resolvido uma vez no `Load` (`TrailRenderer.Prepare`). Os dois piores
   picos vieram de mapas **com trilha**.
3. **`CollisionGrid.DrawDebug` desenhava o mapa inteiro sem culling** — e é o F3
   que liga esse desenho, então o próprio ato de medir custava mais de mil
   retângulos por quadro. O medidor media a si mesmo.

Junto disso, **N1 foi aplicado** (log fora do caminho quente do host e do
cliente), agora por um motivo novo: além de CPU e banda em multiplayer, é o tipo
de alocação a 60 Hz que produz quadro isolado longo.

### Defeito colateral encontrado: a janela é maior que a tela

`rl.InitWindow(0, 0, ...)` faz o raylib usar a resolução do monitor para uma
janela **com decoração**, então a área de cliente é maior que a área útil do
desktop e as primeiras dezenas de pixels ficam atrás da barra de título. Foi
assim que a linha de identificação do mapa sumiu das capturas do F3.

O painel do medidor ganhou margem e está resolvido; **o resto do HUD continua
com a borda de cima cortada**. Corrigir de verdade é escolher entre janela
sem decoração em tamanho de monitor (borderless) ou tamanho de cliente medido
com `rl.GetMonitorHeight` menos as decorações. Não corrigido ainda: muda o modo
de janela do jogo, que é decisão sua.

---

## 4⅞. Cliente mobile — o lag que cresce com o tempo de partida (19/08/2026)

Sintoma relatado: em celular, papel de CLIENTE, o jogo engasga cada vez mais
conforme a partida avança e o aparelho esquenta. O host não relata o mesmo.
Seis causas confirmadas e corrigidas, nesta ordem de impacto. Nenhuma mexe no
pipeline de terreno/shader (R4, fora de escopo desta tarefa).

### 1. Log no caminho quente — CORRIGIDO

`Client.Send` logava o JSON inteiro por mensagem, e o laço do jogo chamava
`SendMessage(inputMsg)` a cada quadro: 60 `log.Printf` por segundo, cada um
formatando e alocando a string do payload inteiro. `handleMessage` também
logava um `Combat event: …` e um `took X damage` por evento de combate —
volume que cresce com o número de inimigos em campo, batendo com o "piora com
o tempo" relatado.

Removidos os dois. Só sobrou log de EVENTO: morte do jogador LOCAL (nova
linha, antes não existia nenhuma), disconnect, GAME OVER, reset de fase,
viagem — a mesma regra que N1 já aplicou no host (doc/network.md).

**Medido** (`go test -bench BenchmarkLogPrintfSendingMessage`, `io.Discard`
como writer — piso de CPU/alocação; o custo real em Android é maior, porque
logcat é uma chamada de IPC e não uma escrita em memória):

| | ns/op | B/op | allocs/op |
|---|---|---|---|
| `log.Printf` por mensagem | 47,9 | 192 | 2 |
| sem log | 0,1 | 0 | 0 |

A 60 Hz (taxa antiga de publicação de input), só esta chamada custava
**~11,5 KB/s de lixo e 120 allocs/s** — antes de contar a formatação em si ou
a I/O do logcat.

### 2. Input do cliente de 60 Hz para ~20 Hz — CORRIGIDO

O host publica a 20 Hz (`SnapshotHz`, `broadcast_rate.go`) mas o cliente
mandava `MsgInput` a cada quadro: 60 marshals JSON, 60 `Flush()` (uma syscall
cada) e 60 despertares de rádio por segundo.

Novo `internal/network/client_input_rate.go`: mesmo desenho acumulador do
host, com uma regra a mais — uma transição de movimento (parar, começar a
andar, mudar de sentido) publica na hora, porque é o vão que a interpolação
de 100 ms (`interpolation.go`) não cobre sozinha. `loop.go` ganhou uma
chamada (`network.SendPlayerInput(payload, dt)`) no lugar do bloco que
montava e enviava a mensagem — a decisão de quando publicar mora inteira no
pacote `network`, não no laço.

**Medido:**

- Tamanho no fio de um `MsgInput` real (com `player_id` no formato do jogo):
  **164 bytes**, incluindo o `\n` delimitador.
- Antes (60 Hz): 164 × 60 = **9,84 KB/s** de uplink + 60 `Flush()`/s.
- Depois (~20 Hz em regime, mais os quadros de transição): 164 × 20 =
  **3,28 KB/s** + 20 `Flush()`/s em regime — **~3× menos tráfego de subida e
  ~3× menos syscalls**, com o mesmo desenho que já provou N2 no sentido
  host → cliente.

Testado em `client_input_rate_test.go`: cadência de ~20 Hz a 60 fps
constante, nenhuma publicação a cada quadro, e a transição de movimento força
envio mesmo fora do tique (sem forçar em mudança só de magnitude na mesma
direção).

### 3. Backlog do socket ao voltar do background — CORRIGIDO

App pausado (tela bloqueada) e retomado: o SO para de entregar leituras ao
processo, mas o host continua publicando a `SnapshotHz`. Ao voltar,
`bufio.Reader` já tem uma fila inteira de mensagens no buffer, prontas sem
nova leitura de rede — decodificar e aplicar cada uma (inclusive estado que
nunca chegou a aparecer na tela) é o pico de CPU que explica o segundo
tranco do sintoma.

Novo `internal/network/client_backlog.go`: `Client.drainStaleSnapshots`
consome, sem bloquear em rede nova, todo o backlog já bufferizado e mantém só
o snapshot mais RECENTE de cada família (`state_update`, `enemy_update`,
`projectile_update`, `cooldown`). Mensagens de EVENTO encontradas no meio do
backlog nunca são descartadas — são tratadas na hora, recursivamente drenando
o que vier depois delas. `client.go` trocou o `json.Decoder` por leitura de
linha (`bufio.Reader.ReadBytes('\n')`), porque todo envio do host já termina
em `\n` (`ClientConn.send`) e só assim dá para consultar `Buffered()` para
saber se há mais dado já em mãos sem tocar a rede.

Testado em `client_backlog_test.go`: um backlog de três `state_update`
coalesce para o último; buffer vazio devolve a mensagem intacta; um tipo que
não é snapshot não consome nada do buffer.

**Não medido em aparelho**: o cenário exato (segundos em background, fila
real de um socket TCP Android) exige um dispositivo físico, que este
ambiente de desenvolvimento não tem. A correção está coberta por teste
determinístico do mecanismo de coalescência; a confirmação de campo (o pico
de CPU sumindo no F3 ao destravar a tela) fica pendente da sua sessão em
jogo.

Cautela junto (item 4 abaixo): o primeiro quadro depois de um hiato longo
também tem `dt` grande, e isso é físico, não de rede — coberto separadamente.

### 4. `dt` travado no primeiro quadro após um hiato — CORRIGIDO

`rl.GetFrameTime()` mede o tempo desde o quadro anterior; depois de um app
pausado por segundos, o primeiro quadro de volta carrega esse hiato inteiro
num `dt` só, e tudo que multiplica por `dt` (física, interpolação) teleporta.

Novo `internal/game/frame_time.go`: `clampFrameDT` trava o teto em 50 ms
(~3 quadros a 60 fps — generoso para não travar visivelmente num engasgo
comum, curto o bastante para nunca simular um hiato de segundos como um
quadro só). `loop.go` chama `dt := clampFrameDT(rl.GetFrameTime())` no lugar
de `rl.GetFrameTime()` cru — uma troca de uma linha; a lógica mora no arquivo
novo.

Não é uma correção de custo, é uma rede de segurança para o item 3: sem ela,
o quadro que processa o backlog também herdaria um `dt` gigante e a física
teleportaria mesmo com a rede já drenada.

### 5. Alocação por quadro no desenho — CORRIGIDO

`InterpolatedPlayers`, `InterpolatedEnemies` (`interpolation.go`) e
`GetAllPlayers` (`globals.go`) alocavam um mapa novo a cada chamada, e são
chamadas várias vezes por quadro (o cliente desenha jogadores remotos, depois
usa a mesma lista para mirar os inimigos; a lógica de portal/clímax/diálogo
também lê `GetAllPlayers` antes do desenho). Com 83 inimigos em campo isso é
pressão de GC constante — que num celular é "esquenta e engasga" ao pé da
letra.

As três agora escrevem num buffer de pacote reaproveitado (`clear()` +
reinserção) em vez de `make(map[...])` por chamada. Seguro porque **todo**
chamador no projeto consome o mapa na hora, na mesma goroutine de desenho, e
nunca o guarda entre um quadro e o seguinte nem entre duas chamadas —
conferido chamador por chamador antes de mexer (`renderer.go`,
`dialogue.go`, `climax_gate.go`, `portal_party.go`, `loop.go`).

**Medido** (`go test ./internal/network/... -bench . -benchmem`, 83
inimigos / 4 jogadores — a guarnição do `world_03` e o teto de jogadores
documentado):

| | antes (naive) | depois (buffer) | ganho |
|---|---|---|---|
| `InterpolatedEnemies` (83 inimigos) | 2892 ns/op, 12376 B/op, 4 allocs/op | 1198 ns/op, 0 B/op, 0 allocs/op | 2,4× mais rápido, zero alocação |
| `GetAllPlayers` (4 jogadores) | 308 ns/op, 1200 B/op, 2 allocs/op | 83 ns/op, 0 B/op, 0 allocs/op | 3,7× mais rápido, zero alocação |

Com `InterpolatedEnemies` chamado uma vez por quadro e `GetAllPlayers`
(direta ou via `InterpolatedPlayers`) cerca de três vezes, a 60 fps isso é
**~936 KB/s e ~600 allocs/s de lixo de GC eliminados** só nesta rota — o
tipo de alocação por quadro que produz pausa de GC intercalada, o mesmo
mecanismo que a Fase 4¾ já flagrou como causa de quadro isolado longo.

Benchmarks ficam em `internal/network/interpolation_bench_test.go` e
`globals_bench_test.go` para remedir depois de qualquer mudança nesta rota.

### 6. FPS alvo por plataforma — MECANISMO PRONTO, padrão inalterado

`Config` ganhou `TargetFPS int32`; `loop.go` chama
`rl.SetTargetFPS(cfg.TargetFPS)` no lugar do `60` fixo. `DefaultConfig` e
`AndroidConfig` continuam devolvendo 60 — **nenhum comportamento mudou**,
só existe agora onde declarar 30 no Android sem `if isAndroid` no laço do
jogo, como pede AGENTS.md.

**Não medido**: comparar 30 contra 60 no Android de verdade (calor, FPS
sustentado, percepção do jogador com host a 20 Hz + interpolação de 100 ms
cobrindo o vão) exige o aparelho físico. Fica para quando houver uma sessão
em jogo para medir os dois casos, como pedido — o padrão em 60 não muda até
essa decisão.

### 7. Corrida de dados em `Client.Send` — CORRIGIDO

`Client.Send` escrevia no `bufio.Writer` sem exclusão mútua. Não é um bug
observado em produção hoje (só o laço do jogo escreve), mas é o mesmo risco
estrutural que o host já resolveu (`ClientConn.writeMu`, `host.go`):
`bufio.Writer` não é seguro para escrita concorrente, e um segundo escritor
futuro (retry, reconexão) corromperia o fluxo de JSON sem aviso, sem chance
de ressincronizar. `Client` ganhou o mesmo `writeMu sync.Mutex` do lado do
host. Correção de correção, não de custo — sem número associado.

### O que ficou de fora

Este documento cobre o cliente mobile (rede + alocação de desenho). Não
mexeu no pipeline de terreno/shader (R4, §1.4) nem em qualquer decisão de
design (efetivo de guarnição, FPS alvo definitivo). `go build ./...` e
`go test ./...` continuam a referência de correção; os testes que já
falhavam antes desta tarefa (conteúdo de mapa, corredor de canhões, contagem
de hordas — nenhum arquivo tocado aqui) seguem falhando pelos mesmos motivos
de antes.

---

## 5. Ordem recomendada

Cada faixa entrega valor sozinha. Não começar a próxima antes de medir a
anterior.

| Fase | Itens | Ganho esperado | Risco |
|---|---|---|---|
| **1** | N1 (log), N3 (input não faz broadcast), R7 | CPU de host despenca em multiplayer; ~4× menos broadcasts | mínimo |
| **2** | ~~R1 (culling)~~ **FEITO** — resta R2 (máscara pré-computada), R3 (índice), R5, R6 | R1 entregou **22×** menos quads no `world_03`; o resto agora é marginal, porque o laço já só visita ~127 células | baixo |
| **3** | M1 (carregar só o usado), M2 (cache), M3, M5 | ~55 MB de VRAM no `world_01`; troca de mapa sem stutter | baixo/médio |
| **4** | N7 (payload magro), N2 (20 Hz), N5 (interpolação), N6 | tráfego de ~2,9 Mbps para ~0,4 Mbps | médio |
| **5** | S1 (grade espacial), S2, S3 (passo fixo) | simulação para de crescer ao quadrado | médio |
| **6** | R4 (instanciar terreno), N4 (delta) | tira o teto restante de draw call e de banda | médio/alto |
| **7** | N8 (binário) + N9 (UDP) ou N10 (ENet) | latência sob perda de pacote | alto |

---

## 6. Como medir, para não otimizar no escuro

O projeto não tem instrumentação de performance hoje. Antes da Fase 2, três
coisas baratas:

1. **Contador de draw call e de quad por quadro**, atrás do F3, desenhado no
   canto. Sem ele, "ficou mais rápido" é impressão.
2. **`rl.GetFPS()` e tempo de quadro em ms** no overlay F3, com o pior quadro
   dos últimos 60 — média esconde travada, e travada é o que o jogador sente.
3. **Contador de bytes enviados/recebidos por segundo** em `Host.broadcast` e
   no `readLoop` do cliente, atrás do mesmo toggle.

E, para a rede, um teste com perda induzida (`clumsy` no Windows, `tc netem` no
Linux) a 2% e 5% — é o único jeito de saber se N9 vale o custo dele.

Reprodução dos números deste doc:

```bash
python3 work/perf/measure_cost.py
```

---

## 7. O que NÃO mudar

Estas coisas parecem otimizáveis e não são:

- **A ordem da pilha de terreno e a rampa de borda** (`doc/tilemap.md`). São
  desenho, não implementação; qualquer otimização de terreno tem de preservar
  o resultado na tela pixel a pixel.
- **Host autoritativo.** Toda a lógica de combate, cooldown e morte depende
  disso. Otimização de rede nunca deve virar "o cliente decide".
- **`footprintIndex` em pixels** (`internal/tilemap/collision_footprints.go`).
  Já é indexado por célula e já custa o tamanho da caixa consultada, não o
  número de props. É o modelo a copiar em S1, não a mudar.
- **Carregar sprite sob demanda** (`internal/entity/sprite_cache.go`). Já está
  certo.
