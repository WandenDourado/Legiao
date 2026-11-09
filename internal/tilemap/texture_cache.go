package tilemap

// Uma textura na VRAM por arquivo, viva enquanto alguem precisar dela.
//
// Atravessar um portal descarregava TUDO e recarregava TUDO, inclusive os
// atlas que o mapa de origem e o de destino usam iguais: 30 a 80 MB de PNG
// decodificados de novo, no meio de uma transicao que o jogador esta olhando.
// E como travelTo carrega o destino ANTES de descarregar a origem, os dois
// ficavam na placa ao mesmo tempo.
//
// O cache resolve os dois de uma vez. Acquire devolve a textura ja carregada e
// soma uma referencia; Release subtrai e so descarrega no zero. O que os dois
// mapas compartilham nunca chega a sair da VRAM, e nao existe duplicata.
//
// Sem mutex: carregar textura e chamada de GPU, que so pode acontecer na
// goroutine que detem o contexto OpenGL — a principal. Um mutex aqui daria a
// impressao de que outra goroutine poderia chamar, e ela nao pode.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type cachedTexture struct {
	texture  rl.Texture2D
	refCount int
}

var textureCache = map[string]*cachedTexture{}

// AcquireTexture carrega a textura (ou reaproveita a que ja esta na VRAM) e
// registra mais um interessado.
//
// path e o caminho do repositorio, como em todo lugar; assets.Path() e
// aplicado aqui dentro, uma vez, pelo mesmo motivo de sempre (o Android le de
// dentro do APK).
//
// Uma textura que nao carrega e cacheada mesmo assim, com ID zero: sem isso um
// arquivo faltando seria uma tentativa de leitura por quadro.
func AcquireTexture(path string) rl.Texture2D {
	if entry, ok := textureCache[path]; ok {
		entry.refCount++
		return entry.texture
	}
	texture := rl.LoadTexture(assets.Path(path))
	if !rl.IsTextureValid(texture) {
		log.Printf("[Tilemap] textura nao carregou: %s", path)
	}
	textureCache[path] = &cachedTexture{texture: texture, refCount: 1}
	return texture
}

// ReleaseTexture desiste de uma referencia e descarrega quando ninguem mais
// quer a textura.
func ReleaseTexture(path string) {
	entry, ok := textureCache[path]
	if !ok {
		return
	}
	entry.refCount--
	if entry.refCount > 0 {
		return
	}
	if rl.IsTextureValid(entry.texture) {
		rl.UnloadTexture(entry.texture)
	}
	delete(textureCache, path)
}

// TextureCacheSize e quantas texturas distintas estao na VRAM. Existe para o
// medidor do F3 e para teste: um numero que so cresce entre travessias de
// portal e vazamento de referencia.
func TextureCacheSize() int { return len(textureCache) }
