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
		previews[i] = characterPreview{
			texture:     rl.LoadTexture(assets.Path(path)),
			isReference: isReference,
		}
	}
	return previews
}

func drawCharacterPreview(preview characterPreview, def entity.CharacterDef, centerX, topY float32) rl.Rectangle {
	source := rl.NewRectangle(0, 0, float32(preview.texture.Width), float32(preview.texture.Height))
	if !preview.isReference {
		source.Width = float32(def.FrameWidth)
		source.Height = float32(def.FrameHeight)
	}

	scale := float32(320) / source.Width
	if heightScale := float32(360) / source.Height; heightScale < scale {
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
