//go:build android

package tilemap

// terrainShaderPaths selects GLSL 100 for Android OpenGL ES 2.0.
func terrainShaderPaths() (string, string) {
	return "assets/shaders/terrain_blend_100.vs", "assets/shaders/terrain_blend_100.fs"
}
