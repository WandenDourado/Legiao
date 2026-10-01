//go:build !android

package assets

import "os"

// ReadFile reads a bundled asset. On desktop the assets tree sits next to the
// executable, so this is a plain file read.
//
// Callers resolve the path first, exactly like they do for rl.Load*:
// assets.ReadFile(assets.Path("assets/dialogues/world_01.json")).
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
