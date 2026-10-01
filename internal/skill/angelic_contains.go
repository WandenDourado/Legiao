package skill

import rl "github.com/gen2brain/raylib-go/raylib"

// AngelicContains reporta se um PONTO esta dentro de alguma Area Angelical
// ativa, de qualquer conjurador.
//
// Existe separada de `HasAngelic` porque as duas respondem perguntas
// diferentes, e confundi-las quebra a nevoa da chefe: `HasAngelic(id)` diz "a
// Sacerdotisa `id` tem altar de pe", e o que a nevoa precisa saber e "ESTE
// jogador esta em cima de um altar". Quem se salva e quem esta dentro, nao
// quem conjurou.
func (m *Manager) AngelicContains(p rl.Vector2) bool {
	m.ultMutex.RLock()
	defer m.ultMutex.RUnlock()
	for _, a := range m.Angelics {
		if a == nil {
			continue
		}
		if rl.Vector2Distance(a.Position, p) <= AngelicRadius {
			return true
		}
	}
	return false
}
