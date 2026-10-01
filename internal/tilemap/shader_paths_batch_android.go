//go:build android

package tilemap

// terrainBatchShaderPaths selects GLSL 100 for Android OpenGL ES 2.0.
func terrainBatchShaderPaths() (string, string) {
	return "assets/shaders/terrain_blend_100.vs", "assets/shaders/terrain_batch_100.fs"
}
