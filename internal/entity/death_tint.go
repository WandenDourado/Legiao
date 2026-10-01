package entity

// A dead player is drawn from the same sprite sheet as a live one, through a
// grayscale shader. Doing it on the GPU keeps one texture per character (a
// desaturated copy of every sheet would double the sprite memory) and lets a
// corpse keep the exact pose it died in.
//
// If the shader cannot be compiled the tint degrades to nothing: the body is
// still drawn, still immobile, just not grey. That is a cosmetic loss, not a
// broken frame.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/assets"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	deathShader     rl.Shader
	deathShaderTried bool
	deathShaderOK   bool
)

// BeginDeathTint starts the grayscale pass. Every draw until EndDeathTint
// comes out desaturated. Always pair the two, even when the shader failed to
// load: EndDeathTint knows to do nothing in that case.
func BeginDeathTint() {
	if !deathShaderTried {
		deathShaderTried = true
		vs, fs := grayscaleShaderPaths()
		deathShader = rl.LoadShader(assets.Path(vs), assets.Path(fs))
		deathShaderOK = rl.IsShaderValid(deathShader)
		if !deathShaderOK {
			log.Printf("[Entity] shader de morte indisponivel (%s); corpos ficam coloridos", fs)
		}
	}
	if deathShaderOK {
		rl.BeginShaderMode(deathShader)
	}
}

// EndDeathTint closes the grayscale pass.
func EndDeathTint() {
	if deathShaderOK {
		rl.EndShaderMode()
	}
}

// UnloadDeathTint releases the shader. Call once on shutdown.
func UnloadDeathTint() {
	if deathShaderOK {
		rl.UnloadShader(deathShader)
		deathShaderOK = false
	}
	deathShaderTried = false
}
