//go:build android

package entity

// grayscaleShaderPaths selects GLSL 100 for Android OpenGL ES 2.0.
func grayscaleShaderPaths() (string, string) {
	return "assets/shaders/grayscale_100.vs", "assets/shaders/grayscale_100.fs"
}
