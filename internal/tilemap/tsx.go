package tilemap

import (
	"encoding/xml"
	"fmt"

	"github.com/WandenDourado/Legiao/internal/assets"
)

type tsxTileset struct {
	XMLName    xml.Name `xml:"tileset"`
	Name       string   `xml:"name,attr"`
	TileWidth  int      `xml:"tilewidth,attr"`
	TileHeight int      `xml:"tileheight,attr"`
	TileCount  int      `xml:"tilecount,attr"`
	Columns    int      `xml:"columns,attr"`
	Image      tsxImage `xml:"image"`
}

type tsxImage struct {
	Source string `xml:"source,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
}

func loadTSX(path string, ts *Tileset) error {
	data, err := readFile(assets.Path(path))
	if err != nil {
		return fmt.Errorf("failed to load tsx file %s: %w", path, err)
	}

	var tsx tsxTileset
	if err := xml.Unmarshal(data, &tsx); err != nil {
		return err
	}

	ts.Image = tsx.Image.Source
	if ts.TileWidth == 0 {
		ts.TileWidth = tsx.TileWidth
	}
	if ts.TileHeight == 0 {
		ts.TileHeight = tsx.TileHeight
	}
	if ts.Columns == 0 {
		ts.Columns = tsx.Columns
	}
	if ts.ImageWidth == 0 {
		ts.ImageWidth = tsx.Image.Width
	}
	if ts.ImageHeight == 0 {
		ts.ImageHeight = tsx.Image.Height
	}

	return nil
}
