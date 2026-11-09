//go:build android

package tilemap

import "github.com/WandenDourado/Legiao/internal/assets"

// readFile reads a map, tileset or vegetation file through the Android Asset
// Manager. The implementation lives in internal/assets so the project has
// exactly one asset reader.
// Callers pass an already resolved path: readFile(assets.Path(p)).
func readFile(path string) ([]byte, error) {
	return assets.ReadFile(path)
}
