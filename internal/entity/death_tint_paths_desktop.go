//go:build !android

package entity

// grayscaleShaderPaths selects GLSL 330 for the desktop OpenGL renderer.
func grayscaleShaderPaths() (string, string) {
	return "assets/shaders/grayscale_330.vs", "assets/shaders/grayscale_330.fs"
}
