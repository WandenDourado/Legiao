package game

import (
	"path/filepath"
	"testing"

	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// footprintOnlyRects reports tm's manifest collision footprints — props like
// a gate, a statue, siege wreckage — WITHOUT the hand-painted `collision`
// tile layer, by building a CollisionGrid off a copy of tm with that layer
// dropped (addManifestFootprints does not read it, so this is exactly the
// footprint set, fully enabled, regardless of what any other grid later
// disables). Only footprints are what UpdatePortalExitGate's mechanism
// (CollisionGrid.SetFootprintsEnabledOverlapping) can ever open; the painted
// layer is the room's own walls, which a portal rectangle can legitimately
// border without blocking it — there only needs to be walkable room left
// inside the rectangle. Conflating the two would fail a map whose portal
// simply sits against a wall, which is not this defect and not this fix's
// job.
func footprintOnlyRects(tm *tilemap.TiledMap) []rl.Rectangle {
	clone := *tm
	layers := make([]tilemap.Layer, 0, len(tm.Layers))
	for _, l := range tm.Layers {
		if l.Name == "collision" && l.Type == "tilelayer" {
			continue
		}
		layers = append(layers, l)
	}
	clone.Layers = layers
	return tilemap.NewCollisionGrid(&clone).Rects()
}

func rectIn(rect rl.Rectangle, set []rl.Rectangle) bool {
	for _, r := range set {
		if r == rect {
			return true
		}
	}
	return false
}

// TestPortalExitGateClearsEveryCampaignMap is the regression this defect
// earns. It does not go through TryLoadWorld/World on purpose — that pulls in
// the renderer, which needs a live GPU context this test suite does not have
// — and instead exercises exactly the operation UpdatePortalExitGate performs
// (CollisionGrid.SetFootprintsEnabledOverlapping) against the same files
// every map already has on disk. t.Chdir to the repo root because that is
// where every manifest and map path in this codebase is written relative to
// — exactly like the running game.
//
// For every map in the campaign, once every portal's overlapping footprint is
// disabled — what UpdatePortalExitGate does the instant PortalsUnlocked()
// turns true — no manifest collision footprint may still overlap that
// portal's rectangle. world_06's gates_gate_04 sat inside portal_para_o_chefe
// from the day the piece was placed, blocking 217 of its 512px width; nothing
// caught it until it was reported as "the corridor is too tight to squeeze
// the group through". This test would have caught it the day the piece was
// placed, and catches the next map that repeats the mistake.
func TestPortalExitGateClearsEveryCampaignMap(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	for _, mapPath := range campaignMaps {
		tm, err := tilemap.LoadTiledMap(mapPath)
		if err != nil {
			t.Errorf("%s nao carregou: %v", mapPath, err)
			continue
		}
		portals := tilemap.Portals(tm)
		if len(portals) == 0 {
			continue // sem portal, nada a proteger
		}
		footprints := footprintOnlyRects(tm)

		grid := tilemap.NewCollisionGrid(tm)
		for _, portal := range portals {
			grid.SetFootprintsEnabledOverlapping(portal.Rect, false)
		}

		for _, portal := range portals {
			for _, solid := range grid.Rects() {
				if !rectIn(solid, footprints) {
					continue // parede pintada do mapa: nao e trabalho deste conserto
				}
				if rl.CheckCollisionRecs(solid, portal.Rect) {
					t.Errorf("%s: portal %q (%v) ainda tem um apoio de colisao habilitado sobrepondo (%v) depois de UpdatePortalExitGate abrir a saida",
						mapPath, portal.Name, portal.Rect, solid)
				}
			}
		}
	}
}

// TestWorld06GateSitsInsideThePortalBeforeTheFix proves the regression test
// above is actually exercising something: on the raw map data, before any
// footprint is disabled, world_06's gate does overlap its portal. If this
// stops failing on its own (someone removed the piece, or moved it), the
// test above stops proving anything either — update both together.
func TestWorld06GateSitsInsideThePortalBeforeTheFix(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))

	tm, err := tilemap.LoadTiledMap("assets/maps/world_06.json")
	if err != nil {
		t.Fatalf("world_06 nao carregou: %v", err)
	}
	portals := tilemap.Portals(tm)
	if len(portals) == 0 {
		t.Fatal("world_06 deveria ter um portal (portal_para_o_chefe)")
	}
	footprints := footprintOnlyRects(tm)

	overlapping := false
	for _, portal := range portals {
		for _, fp := range footprints {
			if rl.CheckCollisionRecs(fp, portal.Rect) {
				overlapping = true
			}
		}
	}
	if !overlapping {
		t.Skip("world_06 nao tem mais um apoio de colisao dentro do portal antes do conserto rodar; o mapa mudou — TestPortalExitGateClearsEveryCampaignMap sozinho ja cobre o caso")
	}
}
