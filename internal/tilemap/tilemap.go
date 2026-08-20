package tilemap

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/WandenDourado/Legiao/internal/assets"
)

type TiledMap struct {
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	TileWidth  int       `json:"tilewidth"`
	TileHeight int       `json:"tileheight"`
	Layers     []Layer   `json:"layers"`
	Tilesets   []Tileset `json:"tilesets"`
}

type Layer struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Visible bool     `json:"visible"`
	Opacity float64  `json:"opacity"`
	ID      int      `json:"id"`
	Width   int      `json:"width"`
	Height  int      `json:"height"`
	Data    []int    `json:"data,omitempty"`
	Objects []Object `json:"objects,omitempty"`
}

type Object struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Width  float32 `json:"width"`
	Height float32 `json:"height"`
	// Polyline is set on Tiled polyline objects. Its points are relative to
	// the object's X/Y.
	Polyline   []Point    `json:"polyline,omitempty"`
	Visible    bool       `json:"visible"`
	Properties []Property `json:"properties,omitempty"`
}

// Point is a vertex of a Tiled polyline or polygon.
type Point struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// Property is a Tiled custom property. Value is left untyped because Tiled
// writes strings, numbers and bools into the same field.
type Property struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// StringProperty returns the named custom property as a string, or "" when the
// object does not carry it.
func (o Object) StringProperty(name string) string {
	for _, p := range o.Properties {
		if p.Name != name {
			continue
		}
		if s, ok := p.Value.(string); ok {
			return s
		}
		return ""
	}
	return ""
}

// FloatProperty returns the named custom property as a number, or fallback
// when the object does not carry it. Tiled writes every number as a JSON
// number, which decodes into float64.
func (o Object) FloatProperty(name string, fallback float32) float32 {
	for _, p := range o.Properties {
		if p.Name != name {
			continue
		}
		if n, ok := p.Value.(float64); ok {
			return float32(n)
		}
		return fallback
	}
	return fallback
}

// IntProperty returns the named custom property as an int, or 0 when the
// object does not carry it. Tiled writes `int` properties as JSON numbers, so
// this is FloatProperty with the truncation made explicit at the call site
// instead of at every reader.
func (o Object) IntProperty(name string) int {
	return int(o.FloatProperty(name, 0))
}

// BoolProperty returns the named custom property as a bool. Absent is false,
// which is the right default for a flag: a zone that does not say it gates
// anything does not gate anything.
func (o Object) BoolProperty(name string) bool {
	for _, p := range o.Properties {
		if p.Name != name {
			continue
		}
		if b, ok := p.Value.(bool); ok {
			return b
		}
		return false
	}
	return false
}

type Tileset struct {
	FirstGID    int    `json:"firstgid"`
	Source      string `json:"source"`
	Image       string `json:"image,omitempty"`
	TileWidth   int    `json:"tilewidth,omitempty"`
	TileHeight  int    `json:"tileheight,omitempty"`
	Columns     int    `json:"columns,omitempty"`
	ImageWidth  int    `json:"imagewidth,omitempty"`
	ImageHeight int    `json:"imageheight,omitempty"`
	ImagePath   string `json:"-"`
}

func LoadTiledMap(path string) (*TiledMap, error) {
	data, err := readFile(assets.Path(path))
	if err != nil {
		return nil, fmt.Errorf("failed to load map %s: %w", path, err)
	}

	var m TiledMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	mapDir := filepath.Dir(path)
	for i := range m.Tilesets {
		ts := &m.Tilesets[i]
		if ts.Source != "" {
			tsPath := filepath.Clean(filepath.Join(mapDir, ts.Source))
			tsxDir := filepath.Dir(tsPath)
			if err := loadTSX(tsPath, ts); err != nil {
				return nil, err
			}
			if ts.Image != "" {
				ts.ImagePath = filepath.Clean(filepath.Join(tsxDir, ts.Image))
			}
		} else if ts.Image != "" {
			ts.ImagePath = filepath.Clean(filepath.Join(mapDir, ts.Image))
		}
	}

	return &m, nil
}

func (m *TiledMap) TilesetForGID(gid int) (Tileset, bool) {
	if gid == 0 {
		return Tileset{}, false
	}
	var best Tileset
	found := false
	for _, ts := range m.Tilesets {
		if ts.FirstGID <= gid {
			if !found || ts.FirstGID > best.FirstGID {
				best = ts
				found = true
			}
		}
	}
	return best, found
}
