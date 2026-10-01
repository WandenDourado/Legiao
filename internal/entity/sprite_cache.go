package entity

import (
	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Character sprite sheets are shared by every player who picked the same
// character, so they are loaded once here instead of once per renderer. Dead
// players reuse the very same sheet — the grey comes from a shader, not from a
// second copy of the texture (see death_tint.go).

var sharedSheets = map[CharacterType]rl.Texture2D{}

// SharedTexture returns the cached sprite sheet for a character, loading it on
// first use. A failed load is cached as the zero texture so it is not retried
// every frame; callers must check the returned ID before drawing.
func SharedTexture(ct CharacterType) rl.Texture2D {
	if tex, ok := sharedSheets[ct]; ok {
		return tex
	}
	def := GetCharacter(ct)
	tex := rl.LoadTexture(assets.Path(def.SpritePath))
	if tex.ID != 0 {
		ApplySpriteFilter(tex)
	}
	sharedSheets[ct] = tex
	return tex
}

// UnloadSharedTextures releases every cached sheet. Call once on shutdown.
func UnloadSharedTextures() {
	for ct, tex := range sharedSheets {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
		delete(sharedSheets, ct)
	}
}
