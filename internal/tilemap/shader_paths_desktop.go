//go:build !android

package tilemap

// terrainShaderPaths selects GLSL 330 for the desktop OpenGL renderer.
func terrainShaderPaths() (string, string) {
	return "assets/shaders/terrain_blend_330.vs", "assets/shaders/terrain_blend_330.fs"
}
