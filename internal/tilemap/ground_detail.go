package tilemap

// groundDetailVisible liga a camada de tiles `ground_detail` — flores,
// galhos e pedrinhas espalhados sobre o terreno.
//
// Desligavel porque o celular antigo pediu (28/09/2026), mas com a medida na
// mao: essa camada cobre ~10% da tela nos mapas 1 a 3 e nada nos outros, do
// mesmo atlas, sem draw call propria. Desligar corta uns poucos por cento do
// preenchimento. Entra no preset Baixo porque e de graca, nao porque resolve
// (plans/plan_qualidade_grafica_mobile.md). Lida so na goroutine de desenho.
var groundDetailVisible = true

// SetGroundDetailVisible liga ou desliga a camada. Vale no proximo quadro.
func SetGroundDetailVisible(on bool) { groundDetailVisible = on }
