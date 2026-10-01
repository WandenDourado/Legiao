//go:build android

package assets

import (
	"bytes"
	"io"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ReadFile reads a bundled asset through the Android Asset Manager, which is
// the only way to reach a file inside the APK: there is no filesystem path for
// it. Callers resolve the path with Path() first, which strips the "assets/"
// prefix the manager is already rooted at.
func ReadFile(path string) ([]byte, error) {
	asset, err := rl.OpenAsset(path)
	if err != nil {
		return nil, err
	}
	defer asset.Close()

	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for {
		n, err := asset.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}
