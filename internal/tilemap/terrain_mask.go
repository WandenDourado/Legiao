package tilemap

// Terrain is painted in layers, not side by side. A cell receives its own
// material AND every material below it in the same stack, so the top layer
// dissolves into the one already on screen instead of ending at a line.
//
// That is what removes the seam at a dirt/stone joint. With every cell drawn
// as a self-contained quad over grass, both sides retracted from the shared
// edge and the grass underneath showed through the gap. Layered, the stone
// simply dissolves into the dirt that is already there, and the outer border
// of a paved area reads grass -> trodden earth -> stone.
//
// terrainStacks below is that ordering, and there is one stack per ground:
// materials in different stacks never paint on each other, so a dirt path
// cannot drag a dark grass underlay along with it.

// terrainStack is one ground and the layers put on it, bottom to top.
type terrainStack struct {
	materials []int
	// edgeWidth is how much of a cell this stack's layers take to fade out.
	// A road or a paved yard has a made edge and keeps the short ramp; a biome
	// does not end at a line, so its layers fade over half the cell and the
	// border stops reading as a staircase of whole cells.
	edgeWidth float32
}

// The green ground is what happened to it: grass, the dirt walking wore into
// it, the stone building put on it. The dark biome is a different ground and
// not a mark left on this one, and it is a SEQUENCE of the same forest floor
// dying, IN THAT ORDER: healthy grass, then dense sick grass, then thinning
// grass, then bare earth. Walking north reads as one wood giving out, each
// stage fading into the next, because neighbouring stages are neighbours in
// the stack.
//
// The order is the design and not an implementation detail. Bare earth used to
// sit at the bottom, which put THINNING grass next to healthy grass at the
// border: the wood appeared to lose its undergrowth before it darkened, which
// is backwards. Dense dark grass belongs against the healthy grass; the ground
// only opens up afterwards.
//
// terrainForestPath sits BELOW the grass and not above it. A path is where the
// grass is gone, so the grass fading at its border dissolves INTO the path,
// which is what a trodden edge looks like. Above the grass it would have had
// every biome stage painted underneath it and printed a dark ring around
// itself; in the green stack it faded against the VILLAGE grass base and
// printed a bright green halo instead.
//
// terrainForestGrass sits at the bottom of that stack and not in the green one,
// and that placement is the whole point. Material from different stacks never
// paints over each other, so while the light side of a transition map used the
// VILLAGE grass, the border was a single hard step whatever shape it was cut
// in. In the same stack the engine ramps it by itself: a dark grass cell now
// draws healthy grass, then bare soil, then thinning grass under it, each layer
// fading at the borders it does not share.
//
// The dark stack does not stop at dead earth: siege gravel and dark flagstone
// continue it upward, because what happens to that ground next is that someone
// built a fortress on it. Built ground in a stack of its own would have met the
// dead earth at a single hard step; here it fades into it, which is the picture
// map 3 wants — pavement the dead ground is taking back at its edge.
var terrainStacks = []terrainStack{
	{materials: []int{terrainGrass, terrainDirt, terrainStone}, edgeWidth: 0.34},
	{materials: []int{terrainForestPath, terrainForestGrass, terrainDarkGrass,
		terrainSparseGrass, terrainBareSoil, terrainSiegeGravel,
		terrainDarkFlagstone}, edgeWidth: 0.50},
	{materials: []int{terrainCastleStone, terrainCastleBlocks, terrainCastleWater, terrainCastleCarpet}, edgeWidth: 0.01},
}

// stackRank locates a material: which stack it belongs to and how high it sits
// in it. ok is false for a material no stack declares, which is never painted.
func stackRank(material int) (stack, rank int, ok bool) {
	for s, entry := range terrainStacks {
		for r, m := range entry.materials {
			if m == material {
				return s, r, true
			}
		}
	}
	return 0, 0, false
}

// edgeWidthFor is how far into the cell the material's border fade reaches.
func edgeWidthFor(material int) float32 {
	if stack, _, ok := stackRank(material); ok {
		return terrainStacks[stack].edgeWidth
	}
	return terrainStacks[0].edgeWidth
}

// paintedWith reports whether a cell of the given kind receives a layer of
// material: the same stack, at or below the cell's own height in it.
func paintedWith(kind, material int) bool {
	kindStack, kindRank, kindOK := stackRank(kind)
	materialStack, materialRank, materialOK := stackRank(material)
	if !kindOK || !materialOK || kindStack != materialStack {
		return false
	}
	return kindRank >= materialRank
}

// linked reports whether the neighbour at (x+dx, y+dy) also carries material,
// in which case the layer does not fade at that border.
//
// Off-map neighbours count as linked. There is no ground out there to dissolve
// into, so fading at the world border only exposed the grass base as a rim —
// wrong for a road running off the edge, and very wrong for a map that is one
// biome from corner to corner.
func linked(layer Layer, x, y, dx, dy, material int) float32 {
	nx, ny := x+dx, y+dy
	if nx < 0 || ny < 0 || nx >= layer.Width || ny >= layer.Height {
		return 1
	}
	if paintedWith(layer.Data[ny*layer.Width+nx], material) {
		return 1
	}
	return 0
}

// edgeMask is the 4-neighbour mask (north, east, south, west) the shader uses
// to fade the layer out at its borders.
func edgeMask(layer Layer, x, y, material int) []float32 {
	return []float32{
		linked(layer, x, y, 0, -1, material),
		linked(layer, x, y, 1, 0, material),
		linked(layer, x, y, 0, 1, material),
		linked(layer, x, y, -1, 0, material),
	}
}

// cornerMask completes the 8-neighbour blob mask for diagonal terrain joins.
func cornerMask(layer Layer, x, y, material int) []float32 {
	return []float32{
		linked(layer, x, y, -1, -1, material),
		linked(layer, x, y, 1, -1, material),
		linked(layer, x, y, 1, 1, material),
		linked(layer, x, y, -1, 1, material),
	}
}
