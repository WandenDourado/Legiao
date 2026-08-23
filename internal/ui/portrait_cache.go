package ui

// Os retratos de dialogo, e quando eles saem da VRAM.
//
// Um retrato e o `reference.png` do personagem: 1536x1024, ~6 MB na placa
// depois de decodificado. O cache existe porque uma cena alterna entre dois ou
// tres oradores dezenas de vezes, e recarregar a folha por FALA abriria a
// caixa com engasgo.
//
// O que faltava era o outro lado: nada nunca descarregava. Cada fase apresenta
// oradores novos, e o elenco da fase 1 continuava residente na fase 6 — em uma
// campanha de sete mapas isso e VRAM que so sobe, nunca desce, e e exatamente
// o "o jogo vai ficando lento conforme avanca nas fases" que a sessao de teste
// de dois jogadores encontrou (doc/performance.md, M5).
//
// A regra agora: o cache vive por MAPA, nao por sessao. `travelTo` descarrega
// ao trocar de fase, do mesmo jeito que `World.Unload` descarrega as texturas
// do mapa que ficou para tras. Quem volta a falar depois disso recarrega, e
// isso custa um carregamento por orador por fase — o preco certo por nao
// carregar o elenco inteiro da campanha.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/assets"
	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// portraitCache keeps one texture per speaker for as long as the current map
// lasts. See UnloadPortraits for why it is not for as long as the session.
var portraitCache = map[entity.CharacterType]rl.Texture2D{}

// portraitTexture resolves a portrait key to the speaker's reference art.
// It reports false for narration (empty key), for an unknown character, and
// for a character still using somebody else's art.
func portraitTexture(key string) (rl.Texture2D, bool) {
	if key == "" {
		return rl.Texture2D{}, false
	}
	ct := entity.CharacterType(key)
	if placeholderPortraits[ct] || !entity.IsRegistered(ct) {
		return rl.Texture2D{}, false
	}
	if tex, ok := portraitCache[ct]; ok {
		return tex, tex.ID != 0
	}
	def := entity.GetCharacter(ct)
	if def.ReferenceImagePath == "" {
		// Cacheado como textura zero para nao tentar ler um arquivo ausente a
		// cada quadro da fala.
		portraitCache[ct] = rl.Texture2D{}
		return rl.Texture2D{}, false
	}
	tex := rl.LoadTexture(assets.Path(def.ReferenceImagePath))
	portraitCache[ct] = tex
	return tex, tex.ID != 0
}

// PreloadPortraits sobe de uma vez os retratos que o mapa pode pedir.
//
// Chamada no carregamento do mapa (game/dialogue.go, syncMap), com as chaves
// que `dialogue.File.PortraitKeys` extraiu do roteiro daquele mapa. Ver o
// comentario daquela funcao para por que o custo tem de ser pago aqui.
//
// Uma chave desconhecida, sem arte propria ou sem `reference.png` e ignorada
// em silencio: `portraitTexture` ja trata os tres casos, e ela e a mesma porta
// que o desenho usa — precarregar por um caminho diferente do de desenhar
// seria precarregar a coisa errada.
func PreloadPortraits(keys []string) {
	loaded := 0
	for _, key := range keys {
		if _, ok := portraitTexture(key); ok {
			loaded++
		}
	}
	if len(keys) > 0 {
		log.Printf("[UI] retratos precarregados: %d de %d chaves", loaded, len(keys))
	}
}

// UnloadPortraits devolve a VRAM de todos os retratos carregados.
//
// Chamada na troca de mapa (game/world_travel.go) e no fim da sessao. E
// chamada de GPU, entao so pode acontecer na goroutine que detem o contexto
// OpenGL — a principal —, que e onde as duas ficam.
func UnloadPortraits() {
	for ct, tex := range portraitCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
		delete(portraitCache, ct)
	}
}

// PortraitCacheSize e quantos retratos estao na VRAM agora. Existe para o
// medidor do F3: um numero que so cresce entre fases e o vazamento voltando.
func PortraitCacheSize() int { return len(portraitCache) }
