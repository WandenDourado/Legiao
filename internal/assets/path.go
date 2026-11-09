//go:build !android

// Package assets provides platform-aware path resolution for game assets.
// Always use assets.Path() when loading any file with raylib functions.
// On desktop, the path is returned as-is (relative to the executable).
// On Android, the "assets/" prefix is stripped because the Android Asset
// Manager already roots at the assets/ folder inside the APK.
package assets

func Path(p string) string {
	return p
}
