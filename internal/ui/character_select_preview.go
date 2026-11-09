package ui

import (
	"github.com/WandenDourado/Legiao/internal/assets"
	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type characterPreview struct {
	texture     rl.Texture2D
	isReference bool
}

func loadCharacterPreviews(characters []entity.CharacterDef) []characterPreview {
	previews := make([]characterPreview, len(characters))
	for i, character := range characters {
		path := character.SpritePath
		isReference := character.ReferenceImagePath != ""
		if isReference {
			path = character.ReferenceImagePath
		}
		tex := rl.LoadTexture(assets.Path(path))
		// A arte entra com 1536 px de altura e e desenhada com uns 340 no
		// card: uma reducao de ~4,5x. Sem filtro o raylib amostra por vizinho
		// mais proximo e o card serrilha e "chia" ao redimensionar a janela —
		// era esse o cansaco visual, nao a conta de escala, que sempre
		// preservou a proporcao.
		rl.SetTextureFilter(tex, rl.FilterBilinear)
		previews[i] = characterPreview{
			texture:     tex,
			isReference: isReference,
		}
	}
	return previews
}

func drawCharacterPreview(preview characterPreview, def entity.CharacterDef, centerX, topY, maxWidth, maxHeight float32) rl.Rectangle {
	source := rl.NewRectangle(0, 0, float32(preview.texture.Width), float32(preview.texture.Height))
	if !preview.isReference {
		source.Width = float32(def.FrameWidth)
		source.Height = float32(def.FrameHeight)
	}

	scale := maxWidth / source.Width
	if heightScale := maxHeight / source.Height; heightScale < scale {
		scale = heightScale
	}
	destination := rl.NewRectangle(
		centerX-source.Width*scale/2,
		topY,
		source.Width*scale,
		source.Height*scale,
	)
	rl.DrawTexturePro(preview.texture, source, destination, rl.NewVector2(0, 0), 0, rl.White)
	return destination
}
