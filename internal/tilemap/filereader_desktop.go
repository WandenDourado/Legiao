//go:build !android

package tilemap

import "os"

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
