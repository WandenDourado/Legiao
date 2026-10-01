package skill

// Como uma magia pergunta ao mapa se ali passa.
//
// PORQUE `collision.Solid` E NAO UMA LISTA DE RETANGULOS. Ate 22/08/2026 as
// magias recebiam `[]rl.Rectangle` com TODO o solido do mapa e chamavam
// `tilemap.IsColliding`, que compara a caixa contra CADA retangulo da lista.
// O world_02 tem 1.132 celulas solidas na camada `collision` mais os apoios
// das 179 pecas de vegetacao: cerca de 1.400 comparacoes POR TESTE.
//
// Para uma bola de fogo (uma por vez) isso era invisivel. Para a Legiao
// Espectral nao: sao TRINTA espectros (LegionCount), cada um testando ate
// quatro vezes por quadro em `moveSpecter`, mais a separacao par a par —
// milhoes de comparacoes por quadro, e o custo CRESCE COM O TAMANHO DO MAPA.
// Foi por isso que a suprema do Necromante rodava lisa no mapa 1 e derrubava
// o jogo para 16 fps depois do climax do mapa 2 (doc/performance.md, 4⁹⁄₁₀).
//
// `CollidesCentered` consulta so as celulas que a caixa toca, mais um indice
// espacial de apoios: uma caixa de 24 px numa celula de 128 px olha uma ou
// quatro celulas, nao mil e quatrocentas. E a MESMA porta que o jogador e o
// monstro ja usavam (`EntityManager.Solid`); as magias e que estavam de fora.

import (
	"github.com/WandenDourado/Legiao/internal/collision"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// blocked is the nil-safe obstacle test for a square box centered on pos.
// A nil Solid means "no map loaded yet", and then nothing blocks — the same
// rule internal/collision uses for players and monsters.
func blocked(solid collision.Solid, pos rl.Vector2, size float32) bool {
	if solid == nil {
		return false
	}
	return solid.CollidesCentered(pos, size, size)
}
