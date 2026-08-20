---
name: create-tiled-assets
description: Create and integrate Tiled map assets (vegetation, buildings, fences, terrain, props) for Legiao. Use when asked to add or fix tilesets, atlas sheets, manifests, map object placement, or collision footprints. Works on hosts with a native image generator (generate + integrate) and on hosts without one (author a generation prompt, then integrate the returned image).
---

# Create Tiled Assets

Produce village assets as atlas sheets, then integrate them through an
**explicit manifest**. Two documents bind this work: `doc/tileset_spec.md`
(module contracts: anchors, required variants, layers) and `doc/art_style.md`
(palette, lighting, perspective, scale). Read both before planning a set.

## Four rules that came from real failures

1. **The manifest is the only source of truth.** Every piece has a named entry
   with its measured pixel rect, anchor, layer role and explicit collision.
   Code never computes a crop from grid arithmetic. Slicing a hand-painted
   atlas by `localID % columns × 128` is what produced floating tree canopies.
2. **Measure, never assume.** Source rects come from `measure_atlas.py` reading
   the real PNG, never from the layout the prompt asked for. Generators return
   pieces at different sizes, offsets and scales than requested — every time.
3. **Assemble it and look at it.** Before integrating, render the set as it will
   appear (`render_scene.py`) and audit it numerically (`audit_joints.py`).
   Every defect in this project's history — converging fence perspective,
   4 px rail steps, a gate that looked open but left a 4 px passage — was
   invisible in the manifest numbers and obvious in a render. Judging a sheet
   by its thumbnail is not a check.
### Defeitos que aparecem sempre, e o reparo de cada um

Os três abaixo vieram em folhas diferentes, geradas em momentos diferentes.
Todos são de **acabamento**, não de desenho — por isso reparo local, nunca
regeração (regra 4 abaixo). Ferramentas em `work/tiled-assets/`.

| Defeito | Como se manifesta | Reparo |
|---|---|---|
| **Realce alto** | A peça vira lanterna sobre o chão. O gerador ilumina cada peça como se ela fosse o assunto. | `dark_biome_fit.shoulder` — curva de ombro que deixa o escuro intacto e comprime só a faixa clara, com a força resolvida por bisseção |
| **Folha escura demais** | O oposto, e acontece: a grama viva voltou mais escura que a grama *doente*. | `dark_biome_fit.lift` — gamma resolvido contra a luminância da cor **média**, que é a métrica da régua |
| **Franja do matte** | Halo rosa nos galhos finos. **Não é magenta puro**: é magenta misturado com a arte em toda proporção, então um teste de "está magenta?" só pega a ponta. | Recompor a folha chapada sobre magenta e deixar `key_matte.py` des-misturar. Ele copia sem tocar folha que já tem alfa — recompor devolve a entrada que ele espera |

**A régua do realce** (`doc/art_style.md` 11.4b): um objeto do bioma sombrio
fica de **+4 a +8 de luminância (p90) acima do chão** em que vai ficar. A média
engana; o que grita na tela é o realce. E a régua **muda quando o chão muda** —
peça aprovada contra chão de teste não está aprovada contra o chão final.

`measure_terrain.py` mede terreno; `dark_biome_fit.highlight` mede peça.

### Caixa de origem sobreposta

O gerador desenha as peças em diagonal e perto umas das outras, então a **caixa**
de uma invade a da vizinha mesmo sem um pixel em comum — e recortar pela caixa
arrasta pedaço do vizinho junto. `validate_manifest.py` pega.

O conserto é **reempacotar**: copiar cada peça mascarada pelos próprios pixels
para um atlas novo, com padding. Ver `build_dark_props_manifest.py`.

Cuidado ao medir: `measure_atlas.py` pode fundir uma fileira inteira num blob
só quando as peças estão próximas. Componente conexo com limiar de alfa separa;
confira que a contagem é **estável** em vários limiares antes de confiar nela.

### Footprint de colisão: sempre dentro da arte

A pergunta que decide o footprint é **quanto da altura da arte é profundidade
de chão**, e ela não tem resposta automática:

- **Tronco compacto** (`tree_trunk_*`): quase tudo é raiz espalhada no chão, e
  o bloco cobre a arte inteira. A largura sai do perfil de massa (colunas com
  ≥40% da massa máxima), não da caixa delimitadora.
- **Peça alta e estreita** (pedra em pé, tronco morto): altura é altura. Só a
  base bloqueia — a mesma fórmula do tronco viraria uma pedra de 260 px num
  muro de 260 px de profundidade.
- **Peça deitada** (tronco caído, galho grande): a silhueta inteira toca o
  chão, então o footprint acompanha a arte — **sem passar dela**.

Em nenhum caso o footprint transborda o desenho. Ver "Collision playbook".

4. **Repair before regenerating.** Composite, scale, shift and re-cut approved
   art locally; that is free. A regeneration costs the user's budget and can
   regress pieces that were already correct. The complete fence junction set
   and both open gates were built by compositing approved pieces — zero
   additional generations.

## Host modes

**Generator host (Codex or any agent with a native image tool):** run the whole
pipeline — author the prompt, generate, key, measure, manifest, assemble,
audit, integrate, build. Respect the generation budget the user states.

**No-generator host (Claude Cowork/Code):** this environment has no
text-to-image tool. The prompt itself is the deliverable:

1. Write the prompt (template below), say which reference images to attach and
   exactly where to save the result: `assets/tilesets/<set>_source.png`.
2. **STOP.** Never fabricate a placeholder or proceed on an imagined sheet.
3. When the user returns the image, resume at *Key the matte* and finish the
   integration exactly like a generator host would.

## Workflow

1. **Plan.** From `doc/tileset_spec.md`, list every required variant of the set
   and its contract. A kit missing one connection variant cannot be assembled,
   so plan the whole kit before generating anything.
2. **Generate / hand off** per host mode. One visual set = one generation call
   (spec Global Rules): palette, lighting, stroke weight and scale only stay
   consistent within a single call.
3. **Key the matte.** `key_matte.py <set>_source.png --output <set>.png` strips
   the magenta background. A source that already has real alpha passes through
   unchanged. What it does, and why each step exists:

   - *Hard key* removes near-exact matte pixels.
   - *Soft key* handles the feather the generator paints into the matte. From
     `observed = art*(1-k) + matte*k` with `matte = (255,0,255)`, and knowing
     project art always satisfies `min(r,b) ≤ g`, coverage is
     `k ≤ (min(r,b) − g) / 255`. **The divisor is 255**; a smaller one
     overestimates `k`, the un-blend divides green by a near-zero number and
     every piece gets a bright green halo (this happened).
   - *Edge bleed* replaces the colour of semi-transparent pixels with nearby
     solid colour, so the alpha fade carries the piece's own hue.

   What keying **cannot** fix: a wide feather means the outer band of the art
   itself was painted mixed with magenta. Erode that band (`ImageFilter.MinFilter`,
   ~16 px, then a 2 px blur) for area pieces, and never for thin ones — eroding
   grass tufts erases them. Residual biome-wide cast is corrected by matching
   each piece's mean colour to its terrain texture
   (`work/tiled-assets/match_biome_tint.py`), clamped to ±12% so a blue puddle
   or green moss survives.
4. **Measure.** `measure_atlas.py <set>.png --grid 128` reports the real
   alpha-bounds rectangle of every region, whether it fits a cell, and a
   suggested anchor.
5. **Manifest.** Write `assets/<category>_manifest.json` (schema below).
6. **Assemble and audit** (section *Acceptance gates*). Fix locally what can be
   fixed locally; only then consider a correction generation.
7. **Validate.** `validate_manifest.py` and `validate_map.py` must exit 0.
8. **Integrate.** Register the manifest in `manifestSources`
   (`internal/tilemap/vegetation.go`), place named objects in the map's
   objectgroup layer, set collision.
9. **Build & hand over.** `go build ./cmd/desktop` and `go test ./...` must
   pass. Do not run the game. Deliver file links, what changed, and precise
   in-game checks (which objects, where, what F3 should show). The user's F3
   validation is the final gate — never claim visual correctness yourself.

## Generation prompt template

Paste the style block from `doc/art_style.md` §10, then specify:

- **Canvas**: power-of-two PNG (1024×1024 or 2048×2048), world grid 128×128 px.
- **Background**: solid flat magenta `#FF00FF` filling the canvas — not
  transparency; many generators cannot produce alpha. Magenta is generation
  only, forbidden inside the art.
- **The complete piece list for ONE set**, naming every required variant.
- **Per piece**: intended size and whether it is grid-aligned (padding inside
  its cell) or a free-size object (full pixels inside an allocated region,
  never cut by a 128 px crop).
- **Gaps**: at least 40–60 px of magenta between pieces and from the canvas
  edges; nothing touching, nothing cropped.
- **Ground contact**: every piece's bottom edge is a clean contact line — no
  grass, ground patch or shadow blob painted underneath.
- **No** background scenery, grid lines, text, labels, borders or visual effects.

Clauses that exist because a generation failed without them:

- *Horizontal and vertical pieces are drawn separately, never one rotated.*
  Rotating a horizontal piece rotates its lighting, post depth and ground contact.
- *A run drawn "away from the camera" uses **parallel** bands, never converging
  perspective* — converging rails cannot connect to the next piece.
- *A piece that continues a run exits its edge cut FLAT at full thickness*, at
  the same offset in every piece of the run.
- *A gate is ONE self-contained piece including both posts*; the open variant
  keeps the posts in exactly the closed variant's positions and leaves the
  opening completely clear (state a minimum in pixels).
- *Trunk and canopy are separate pieces* sharing one ground anchor.
- *A ground topping is a few small loose objects, never a filled patch of
  ground.* Asking for "cracked ground", "sand ripples" or "a moss patch"
  returns a disc of differently-coloured terrain that reads as a stain on the
  tile — artificial at every density. Ask for pebbles, blades, twigs, leaves
  and crack **lines**, with the background showing between them, occupying
  40–65% of the cell. A topping element is never taller than the character's
  boot (~40 px against a 190 px character).
- *Crisp edges against the matte.* Explicitly forbid airbrushed, glowing or
  feathered edges: a soft edge mixes magenta into the art itself, and no keying
  recovers it (this cost a full sheet).
- Give **exact measurements from already-approved pieces** in every correction
  prompt (rail heights above the ground line, thicknesses, band width). Vague
  wording produced a junction set 20% oversized and, on the next pass, a
  junction set with one rail instead of two.

## Manifest schema and anchors

```json
{
  "atlas": "assets/tilesets/<set>.png",
  "pieces": {
    "tree_trunk_oak": {
      "source": {"x": 33, "y": 592, "width": 140, "height": 140},
      "anchor": {"x": 70, "y": 120},
      "role": "structures_back",
      "collision": true,
      "collisionFootprint": {"offsetX": -32, "offsetY": -48, "width": 64, "height": 48}
    }
  }
}
```

`role` ∈ `ground_detail` | `structures_back` | `foreground`. Optional
`clipOk: true` marks an intentional cut (the base/roof split of one building)
so the validator does not report clipped art.

Anchor conventions, by piece type:

| Piece type | Anchor | Why |
|---|---|---|
| Standalone prop, tree, bush | bottom-center of the art | Placed by its own position. |
| Building base + roof | shared bottom-center of the whole building | The roof's anchor sits *below* its rect — that is correct, the validator warns and moves on. |
| Connecting run piece (fence, wall) | bottom-**left**, at the piece's ground line | A run is laid out by advancing along the axis; a centered anchor makes every placement a subtraction. |
| Piece whose band continues past the ground line (corner with a band going down) | above the image bottom | The ground line is not the bottom of the art. |

## Collision playbook

**A colisão fica DENTRO da arte.** O que é bloqueado tem que ser o que o
jogador vê; chão visivelmente livre em que ele esbarra é o defeito. O motor
colide contra os **retângulos do manifesto**, em pixels — não há grade, não há
arredondamento, e nada precisa ser empurrado para caber em célula.

Isto já foi o contrário, e a inversão custou caro. A regra antiga mandava o
footprint **transbordar** a arte, porque `CollisionGrid` quantizava tudo em
células de 128 px e um footprint menor que meia célula sumia. O resultado foi
casa bloqueando 34 px de grama de cada lado e 64 px na frente da porta, cerca
bloqueando célula cheia para um corrimão de 40 px, e footprint escrito como
"arte + 10 px" em vez de medido. `validate_manifest.py` agora **falha** com
mais de 8 px fora da arte opaca.

**Objeto sólido (casa, pedregulho): footprint = a massa da arte.** Meça os
pixels opacos da peça que encosta no chão e use essa caixa. A casa é a `_base`
recortada na linha do beiral: o bloco vai do topo da parede até o pé dela, sem
folga. Beiral, copa e hera flutuam acima do chão e nunca colidem.

**Trecho fino (cerca, muro): footprint próprio, não célula pintada.** Trecho
horizontal: faixa do **topo do corrimão até a base dos postes**, do comprimento
da peça. Uma faixa só na base parece certa e não é — o jogador planta o pé
entre os corrimãos, porque a caixa dele fica nos pés e o desenho dela é a
altura inteira. Trecho vertical: tira estreita na coluna do poste, cobrindo a
peça inteira. Não alinhe nada a centro de célula — isso era compensação de
quantização e não existe mais.

**Peça que não cabe num retângulo usa `collisionFootprints` (plural).** Um
canto de cerca é um L e um portão aberto é dois postes com vão no meio;
descrever qualquer um dos dois como uma caixa só bloqueia o miolo vazio do
canto ou fecha o portão.

**Altura de arte nem sempre é profundidade de chão — decida peça a peça.** Uma
pedra em pé de 260 px é 260 px *para cima*: bloquear a altura dela vira um muro
de 260 px de fundo. Um tronco compacto é o contrário — a arte é raiz espalhada
no chão, então o bloco cobre a arte inteira. Nenhuma regra única acerta os
dois; `doc/tilemap.md` tem a tabela por tipo de peça.

**Largura de tronco sai do perfil de massa, não da caixa delimitadora.** As
colunas com ao menos 40% da massa máxima. A caixa pegaria ponta de raiz
translúcida e bloquearia grama.

**Footprint menor que a arte é normal e desejável** quando a arte é sombra,
saia de arbusto ou ponta de galho — nada disso é parede.

## Placing connecting runs

Never butt-join by width. A piece's bounding box starts 5–7 px before its rail
does (the stone footing is wider than the post), so `x += width` leaves a hole
at every joint. Measure, per piece, the first and last column where the rail
actually reaches, and place the next piece so its rail resumes one pixel later.
`work/tiled-assets/place_fence_lot.py` is the working reference implementation.

**Build junction pieces from the approved straight pieces.** A corner composed
of `h_start` + the vertical band inherits the run's rail heights by
construction. Junctions drawn independently came back 3–7 px off and stepped at
every corner.

When cutting a band to composite: verify the crop contains no post cap — a
chevron inside the source shows up repeated along every band. Stretch the clean
segment to length rather than tiling it; tiling leaves a visible seam.

## Acceptance gates

Run in order; each is cheap and catches a class the next one cannot.

```bash
python skills/create-tiled-assets/scripts/validate_manifest.py assets/<cat>_manifest.json
python skills/create-tiled-assets/scripts/validate_map.py assets/maps/world_01.json --manifest assets/<cat>_manifest.json --layer <layer>
python skills/create-tiled-assets/scripts/audit_joints.py assets/maps/world_01.json --manifest assets/<cat>_manifest.json --layer <layer>
python skills/create-tiled-assets/scripts/render_scene.py assets/maps/world_01.json --manifest assets/<cat>_manifest.json --layer <layer> --collision --grass --scale 0.4 --out /tmp/scene.png
```

Then **look at `/tmp/scene.png`** and check, numerically where possible:

| Check | Threshold |
|---|---|
| Joint holes in a run | ≤ 3 px (`audit_joints` reports anything larger) |
| Deliberate opening (gate) | ≥ 96 px clear entre os retângulos dos postes; a caixa do jogador é 40 px |
| Rail/band alignment between piece types | identical to the approved straight piece, ±1 px |
| Colisão além da arte opaca | 0 px é o alvo, 8 px é o limite duro (`validate_manifest.py`) |
| Colisão de um trecho fino | faixa de ~48 px na linha do chão, vão do portão livre |
| Scale against the world | door 130–170 px, character ~186 px on screen |

Acceptance is the whole set assembled, not a piece in isolation: the fence's
horizontal row passed on its own and the kit was still unusable.

## Local repair catalog (free, always try first)

| Symptom | Repair |
|---|---|
| Junction/corner missing or misaligned | Composite it: approved straight piece + band from the approved run piece, band drawn behind the post. |
| Open gate delivered as loose leaf + detached post | Composite into one piece, placing each part at the closed gate's own post positions. |
| A returned part is 20–26% oversized | Uniform resize to the approved piece's height before compositing (single LANCZOS resample). |
| Band/rail seam repeats along a run | Re-cut the source segment away from caps and stretch instead of tiling. |
| Piece rect clips faint pixels | Grow the rect 1 px; the validator flags it as a warning. |
| Green halo around every keyed piece | The soft-key divisor is wrong — it must be 255 (see step 3). |
| Mauve ring that survives keying | The art's outer band is contaminated: erode ~16 px on area pieces; thin pieces get a strong biome tint match instead. |
| Piece reads as a different biome | Two different repairs, pick by piece type. **Area patch**: match its mean colour to the terrain texture, clamped to ±12%, preserving luminance (`match_biome_tint.py`). **Small discrete elements** (pebbles, twigs): matching the mean would tint grey pebbles orange — instead rotate the tile's mean HUE into the biome family and cap saturation (`fix_topping_hue.py`). Measure first: the dirt family sits near 38–40°, warm grey stone near 45°, and generator output that reads "rosy" measures 0–20°. |
| Collision too large / too small | Remeça o footprint contra os pixels opacos da peça; nunca redesenhe a arte. |
| Wrong-looking placement in game | F3 overlay first (anchor vs rect vs cells) — it is a manifest bug far more often than an art bug. |

Regenerate only when the art itself is defective. Then: regenerate the **whole
dependent mini-set** in one call conditioned on the approved sheets, state
explicitly which pieces are already approved and must not change, and paste the
measured numbers of the approved pieces into the prompt.

## Audit trail

- `assets/tilesets/<set>_source.png`, `<set>b_source.png`, `<set>c_source.png` —
  every generated sheet, never overwritten; the runtime atlas is `<set>.png`.
- `work/tiled-assets/prompt-*.md` — the exact prompt of each generation, with
  the correction passes numbered.
- `work/tiled-assets/build_*_atlas.py`, `place_*.py` — the composition and
  placement scripts, so the atlas can be rebuilt from the sources.

## Scripts

| Script | Purpose |
|---|---|
| `key_matte.py` | Remove the `#FF00FF` matte and despill edges; passes real-alpha sources through. |
| `measure_atlas.py` | Alpha-bounds scan: real rect, occupied cells, grid-aligned flag, suggested anchor. |
| `validate_manifest.py` | Rects inside the atlas, no overlaps, no clipped art, valid roles, e **todo footprint dentro da arte opaca** (máx. 8 px fora). |
| `validate_map.py` | Every named object resolves to a manifest piece; reports pieces never placed. |
| `audit_joints.py` | Renders placed runs and measures joint holes and deliberate openings. |
| `render_scene.py` | Renders map objects (optionally with the collision overlay) so the assembly can be inspected. |
| `scene_utils.py` | Shared loading/rendering helpers for the two scripts above. |

## Category quick reference

| Category | Placement | Collision |
|---|---|---|
| Terrain | 1×1 tile layer; renderer owns blob blending | none |
| Ground detail / toppings | 1×1 tiles on `ground_detail`, padding inside the cell, scattered per biome | none (passable) |
| Buildings | manifest objects, base + roof split at the eave line | footprint = a massa opaca da `_base` |
| Fences / walls | manifest objects placed by connection point | `collisionFootprints` por peça (L no canto, vão no portão) |
| Vegetation | manifest objects, trunk + canopy sharing an anchor | só o tronco, medido no barril |
| Rocks / props | manifest objects when oversized, tiles when 1×1 | footprint only when declared |
