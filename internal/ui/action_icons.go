package ui

// Os icones dos botoes de acao, e por que eles sao textura e nao desenho.
//
// Os tres botoes carregavam TEXTO — "FIRE", "Q", "R". Letra de teclado num
// aparelho que nao tem teclado nao diz nada sobre o que o botao faz, e "FIRE"
// so e legivel para quem le ingles. Os glifos (espada, estrela, estrela com
// anel) dizem "bater", "magia" e "suprema" sem palavra nenhuma.
//
// Sao PNGs BRANCOS com alfa. Brancos porque a cor vem do tingimento no
// desenho: o mesmo arquivo serve ao botao aceso, ao apagado e ao bloqueado
// sem tres versoes do mesmo desenho na pasta.
//
// O cache e por SESSAO e nao por mapa, ao contrario dos retratos
// (portrait_cache.go): sao tres texturas de 256x256, menos de 1 MB somadas, e
// os botoes existem em TODO mapa. Descarregar na viagem so pagaria o
// carregamento de novo do outro lado.

import (
	"github.com/WandenDourado/Legiao/internal/assets"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// ActionIcon names one of the three button glyphs. The value is the asset
// path, so the constant is also the cache key.
type ActionIcon string

const (
	// IconAttack is the crossed sword-and-arrow of the basic attack.
	IconAttack ActionIcon = "assets/ui/icon_attack.png"
	// IconSkill is the four-pointed star with a single orbit (slot Q).
	IconSkill ActionIcon = "assets/ui/icon_skill.png"
	// IconUltimate is the same star ringed with sparkles (slot R).
	IconUltimate ActionIcon = "assets/ui/icon_ultimate.png"
)

var actionIconCache = map[ActionIcon]rl.Texture2D{}

// actionIconTexture loads a glyph once and reports whether it is usable.
//
// A failed load is cached AS the zero texture rather than retried: retrying
// would mean one failed file read per frame per button for the rest of the
// run, and the caller already has a text fallback for exactly this case.
func actionIconTexture(icon ActionIcon) (rl.Texture2D, bool) {
	if tex, ok := actionIconCache[icon]; ok {
		return tex, tex.ID != 0
	}
	tex := rl.LoadTexture(assets.Path(string(icon)))
	actionIconCache[icon] = tex
	return tex, tex.ID != 0
}

// DrawActionIcon paints the glyph centred on the button and scaled so the
// disc drawn inside the PNG lands exactly on the button's circle.
//
// Reports false when the texture is missing, so the caller can fall back to
// its label instead of drawing a button with nothing on it.
func DrawActionIcon(icon ActionIcon, center rl.Vector2, radius float32, tint rl.Color) bool {
	tex, ok := actionIconTexture(icon)
	if !ok {
		return false
	}
	src := rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height))
	dst := rl.NewRectangle(center.X-radius, center.Y-radius, radius*2, radius*2)
	rl.DrawTexturePro(tex, src, dst, rl.NewVector2(0, 0), 0, tint)
	return true
}

// UnloadActionIcons returns the glyphs' VRAM. End of session only — see the
// note at the top on why not per map. GPU call, so it must run on the
// goroutine that holds the OpenGL context (the main one).
func UnloadActionIcons() {
	for icon, tex := range actionIconCache {
		if tex.ID != 0 {
			rl.UnloadTexture(tex)
		}
		delete(actionIconCache, icon)
	}
}
