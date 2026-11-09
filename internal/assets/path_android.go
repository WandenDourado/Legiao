//go:build android

package assets

import "strings"

func Path(p string) string {
	return strings.TrimPrefix(p, "assets/")
}
