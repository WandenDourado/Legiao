//go:build !android

package tilemap

// terrainBatchShaderPaths selects GLSL 330 for the desktop OpenGL renderer.
// O vertex shader e o MESMO do terrain_blend: ele so repassa posicao, uv e cor,
// e a cor e justamente o canal que o batch reaproveita para o indice da celula.
func terrainBatchShaderPaths() (string, string) {
	return "assets/shaders/terrain_blend_330.vs", "assets/shaders/terrain_batch_330.fs"
}
