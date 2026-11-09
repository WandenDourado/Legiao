# Village Tileset Assembly Specification

This specification is the contract for generating, selecting, and placing
village scenery. A generated image is not usable until every required module
can be mapped to one of the contracts below. The base grid is **128x128 px**.
Coordinates use top-left origin; an anchor is measured in the image rectangle.

## Global Rules

- A ground cell is 128x128 px. Grid-aligned modules must have transparent
  padding inside their cell, never unexplained pixels outside it.
- The anchor of a normal 1x1 ground prop is bottom-center (64, 120). Its
  collision footprint is defined separately and is never inferred from alpha.
  It is measured in pixels against the art and collided in pixels; it must stay
  inside the opaque art (`validate_manifest.py` fails past 8 px).
- Any image wider or taller than one cell is a Tiled object/image asset, not a
  cropped 1x1 atlas tile. Its source rectangle must include its full pixels.
- Render order is ground, ground_detail, structures_back, entities, then
  foreground. Collision belongs only to the explicit collision layer.
- Adjacent modules meet on their declared connection edge. Transparent padding
  is allowed only away from that edge.
- Modules from one visual set (one house kit, one fence kit, or one vegetation
  family) must be generated in the same image-generation call and assembled
  from that sheet whenever possible. Splitting a set across calls is allowed
  only for a documented missing variant; the new call must then generate the
  entire dependent mini-set, not an isolated piece, to preserve palette,
  lighting, stroke weight, and scale.

## Terrain

| Module | Pixels / cells | Anchor | Contract |
|---|---:|---|---|
| Grass, dirt, stone base | 256x256 px source, repeated into 1x1 cells | cell center | Seamless on all four edges. |
| Terrain type cell | 128x128 px / 1x1 | cell center | Encodes grass, dirt, or stone only; it does not contain a painted transition. |

The renderer owns the 8-neighbor blob mask. A dirt/stone cell blends over
grass: cardinal neighbors retain the material at that edge, and missing
diagonals cut the corresponding corner. No fixed artwork may decide a terrain
edge. Ground decorations are separate transparent 1x1 modules and passable.

## Fences

**Current contract: manifest objects (`assets/fences_manifest.json`).** Fence
pieces are free-size drawings placed on the Tiled `fences` object layer, not
1x1 tiles. Anchor is the bottom-LEFT corner of the art at its ground line, so a
run is laid out by advancing x (or y) by the piece width (or height). Pieces
whose vertical band continues below the ground line (`corner_nw`, `corner_ne`,
`tee_s`) have their anchor above the image bottom.

| Piece | Connects |
|---|---|
| `fence_h_start` / `fence_h_middle` / `fence_h_end` | Rails exit left/right at 44–81 px and 115–154 px above the ground line. |
| `fence_v_start` / `fence_v_middle` / `fence_v_end` | 40 px band exits top/bottom; `v_middle` tiles with itself. |
| `fence_corner_nw/ne/sw/se` | One rail branch + one band branch at a right angle. |
| `fence_tee_n` / `fence_tee_s` | Rails straight through + one band branch. |
| `fence_gate_h_closed/open`, `fence_gate_v_closed/open` | Open variants keep both posts in the closed variant's positions and leave ≥130 px of clear opening. |

Pieces are joined by connection point, not by width: `work/tiled-assets/place_fence_lot.py`
measures where each piece's rail/band actually starts and ends and places the
next piece one pixel after it, closing the 5–7 px gap that butt-joining leaves.
Corner and tee pieces are composed from the approved straight pieces, so their
rail heights match the run exactly. Nothing is aligned to a cell centre: that
was compensation for a collision grid that no longer exists.

Fence collision comes from the manifest, one `collisionFootprints` entry per
piece — a horizontal run is a band from the top of the rail down to the base of
the posts, a vertical run is a strip on the post column, a corner is the two of
them (an L), and an open gate is two posts with a walkable gap between them.

Two things this replaced, both defects:

- Painting the `collision` layer instead. It blocked a whole 128 px cell for a
  ~40 px rail, and it was the only reason fences needed a hand-painted layer.
- A band only at the base of the posts. It reads as correct and is not: the
  player's collision box sits at their feet, so with only the base blocked they
  planted a foot between the rails and stood inside the fence.

**Legacy (unused): 1x1 fence tiles.**
All fence modules are 128x128 px / 1x1 and use bottom-center (64, 112).
Rails must reach their declared connection edge; posts are centered on a grid
edge, not duplicated on both sides of every module.

| Variant | Required | Use |
|---|---|---|
| H_start, H_middle, H_end | Yes | Horizontal run: start + middle x N + end. |
| V_start, V_middle, V_end | Yes | Vertical run: start + middle x N + end. |
| corner_NE, corner_NW, corner_SE, corner_SW | Yes | Direction change; no generic straight tile is stacked to form a corner. |
| gate_H, gate_V | As needed | Replaces one middle module and stays passable when open. |

Horizontal and vertical pieces require separately drawn artwork. Rotating a
horizontal fence would rotate the lighting, post depth, rails, and ground
contact incorrectly. A line has no gaps: every endpoint is either capped by an
end module or connected to a corner/gate.

## Houses

**Current contract: whole-house manifest objects.** A house is generated as one
complete building inside an allocated multi-cell region (typically 3x3 to 5x4
cells) and integrated as manifest pieces, never assembled tile by tile:

| Piece | Source rect | Anchor | Layer / collision |
|---|---|---|---|
| `house_<variant>_base` | walls, door, windows — from the eave line down | bottom-center of the full building | structures_back; explicit collision footprint covering the wall base. |
| `house_<variant>_roof` | roof, chimney, gable — from the eave line up | same world anchor as the base (anchor lies below this rect) | foreground; no collision. |

The two rects are a horizontal cut of the same building image at the eave
line, so they never overlap and reassemble exactly at the shared anchor —
the trunk/canopy pattern.

The collision footprint is the `_base`'s own opaque mass — wall top to wall
foot, no margin. Roof overhang, chimney and ivy are drawn above ground level
and never collide. A character blocked at the wall is drawn in front of the
base; a character north of the house is covered by the roof.

This used to be an `N = ceil(artWidth / 128)` cell window with the building
centred in it, because `CollisionGrid` quantised footprints into 128 px cells.
Collision is pixel rectangles now, so the window is gone — and it has to be:
the leftover it produced blocked 34 px of grass on each side of the small house
and 64 px in front of its own door.

**Legacy (paused): modular facade kit.** The tile-by-tile contract below is
retained for a future modular kit but is not used for integration today.

House facade modules are 128x128 px / 1x1 with bottom-center (64, 128).
They share a baseline: wall, window, door, and foundation must meet at the
same Y coordinate.

| Row / component | Required modules | Sequence rule |
|---|---|---|
| Foundation | left, middle, right | One continuous row, only if not embedded in the wall module. |
| Wall | left corner, plain, window, door, right corner | Door is centered or intentionally offset in the facade recipe; windows are symmetric unless a named exception says otherwise. |
| Eave | left, middle, right | Covers the top wall edge continuously. |
| Roof slope | left, middle, right | A straight roof run uses left + middle x N + right. |
| Gable / fronton | left, center, right or dedicated 1x2 piece | Used only at a roof end designed for a gable; never mixed with an unrelated slope tile. |
| Chimney | dedicated module | Anchored to a supported roof cell, never free-floating. |

A house is assembled from an explicit recipe, not a generic fixed array. The
recipe names every row and establishes whether the wall includes its
foundation. A module that already paints a stone base forbids an additional
foundation row below it.

## Trees, Bushes, and Tall Vegetation

| Module | Pixels / cells | Anchor | Layer / collision |
|---|---:|---|---|
| Tree trunk/base | 128x128 / 1x1 | (64, 112) | structures_back; footprint = the trunk's mass (columns with ≥40% of peak alpha mass) over the art's full height — a trunk piece is root flare on the ground, so its height IS ground depth. |
| Tree canopy | 256x256 / 2x2 | bottom-center (128, 224) aligned to trunk | foreground; no collision. |
| Bush, large shrub | 128x128 or 256x128 / 1x1 or 2x1 | bottom-center | structures_back unless explicitly foreground; collision only if declared. |
| Flowers / low grass | 128x128 / 1x1 | bottom-center | ground_detail; passable. |

The canopy and trunk are independent images/modules. A character can be
blocked by the trunk while its sprite is drawn before the canopy. A bush that
extends beyond a cell must use an allocated 2x1/2x2 source rectangle or an
image object; it must never be cut by the 128x128 source crop.

## Ground Toppings

Toppings are 128x128 / 1x1 passable tiles on `ground_detail`, used to break the
repetition of the terrain textures. Each one is **a few small loose objects**
(pebbles, blades, twigs, leaves, a crack line) occupying 40–65% of the cell
with the terrain visible between them — never a filled patch of differently
coloured ground, which reads as a stain on the tile. No element is taller than
about 40 px against a 190 px character. Palette stays inside the biome's own
family; contrast stays low.

## Rocks and Other Props

Small rocks are 128x128 / 1x1, anchored at (64, 116), in structures_back, and
solid only when marked in collision. Larger rocks, ruins, and wells use
256x128 or 256x256 object sources with a declared bottom-center anchor and a
smaller explicit collision footprint. For a tall narrow prop (standing stone,
dead trunk) only the base blocks: there the art's height is height, not ground
depth, and blocking it would turn a 260 px stone into a 260 px deep wall.

## Pre-Integration Checklist

1. Record each sheet cell/object against a named variant above.
2. Reject a sheet that lacks a required connection variant.
3. Test a minimal fence line, corner, gate, house facade, and tree before
   placing them across the village.
4. Verify source rectangles contain every opaque pixel; inspect alpha bounds
   for overhanging vegetation.
5. Add collision only after visual placement and anchors are verified.
