package world

// Bounds holds the playable world dimensions.
// Calculated once at startup based on screen size.
type Bounds struct {
	Width  float32
	Height float32
}

// NewBounds returns a Bounds set to mapScale times the screen dimensions.
func NewBounds(screenWidth, screenHeight float32, mapScale float32) Bounds {
	return Bounds{
		Width:  screenWidth * mapScale,
		Height: screenHeight * mapScale,
	}
}

// NewBoundsFromMap returns a Bounds derived from a TiledMap's grid and tile dimensions.
// mapWidth and mapHeight are in tiles; tileWidth and tileHeight are in pixels.
func NewBoundsFromMap(mapWidth, mapHeight, tileWidth, tileHeight int) Bounds {
	return Bounds{
		Width:  float32(mapWidth * tileWidth),
		Height: float32(mapHeight * tileHeight),
	}
}
