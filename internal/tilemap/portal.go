package tilemap

import rl "github.com/gen2brain/raylib-go/raylib"

// portalLayerName is the object layer that holds map-to-map transitions.
const portalLayerName = "portals"

// Portal is a rectangle that sends whoever walks into it to another map.
//
// Everything a portal needs lives in the map file: the destination is a Tiled
// custom property, not a table in code, so a new link is one object in Tiled
// and no rebuild.
type Portal struct {
	Name string
	Rect rl.Rectangle
	// TargetMap is the destination map path, relative to the repo root
	// (for example "assets/maps/world_02.json").
	TargetMap string
	// TargetSpawn is the name of the spawn object to arrive at in the
	// destination map. Empty means the destination's player_spawn.
	TargetSpawn string
}

// Portals returns every usable portal in the map's portal layer. Objects
// without a target_map property are skipped: a portal that leads nowhere is a
// mapping mistake, not a portal.
func Portals(m *TiledMap) []Portal {
	var portals []Portal
	for _, layer := range m.Layers {
		if layer.Type != "objectgroup" || layer.Name != portalLayerName {
			continue
		}
		for _, object := range layer.Objects {
			target := object.StringProperty("target_map")
			if target == "" {
				continue
			}
			portals = append(portals, Portal{
				Name:        object.Name,
				Rect:        rl.NewRectangle(object.X, object.Y, object.Width, object.Height),
				TargetMap:   target,
				TargetSpawn: object.StringProperty("target_spawn"),
			})
		}
	}
	return portals
}

// Center is the middle of the portal rectangle in world coordinates.
func (p Portal) Center() rl.Vector2 {
	return rl.NewVector2(p.Rect.X+p.Rect.Width/2, p.Rect.Y+p.Rect.Height/2)
}

// Contains reports whether a centered box overlaps the portal.
func (p Portal) Contains(center rl.Vector2, width, height float32) bool {
	left, right := center.X-width/2, center.X+width/2
	top, bottom := center.Y-height/2, center.Y+height/2
	return left < p.Rect.X+p.Rect.Width &&
		right > p.Rect.X &&
		top < p.Rect.Y+p.Rect.Height &&
		bottom > p.Rect.Y
}
