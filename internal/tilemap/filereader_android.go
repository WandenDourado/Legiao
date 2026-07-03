//go:build android

package tilemap

import (
	"bytes"
	"io"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func readFile(path string) ([]byte, error) {
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
