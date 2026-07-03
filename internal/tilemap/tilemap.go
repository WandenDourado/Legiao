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
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Width   float32 `json:"width"`
	Height  float32 `json:"height"`
	Visible bool    `json:"visible"`
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
