package ui

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ShowCharacterSelect displays a UI to choose a character and returns the chosen character type.
func ShowCharacterSelect() entity.CharacterType {
	characters := entity.AllCharacters()
	if len(characters) == 0 {
		return entity.CharWizard // Fallback
	}

	selectedIndex := 0
	selected := false

	previews := loadCharacterPreviews(characters)
	defer func() {
		for _, preview := range previews {
			rl.UnloadTexture(preview.texture)
		}
	}()

	for !selected && !rl.WindowShouldClose() {
		rl.BeginDrawing()
		rl.ClearBackground(rl.RayWhite)

		sw := float32(rl.GetScreenWidth())
		sh := float32(rl.GetScreenHeight())

		title := "Select Your Character"
		titleWidth := rl.MeasureText(title, 30)
		rl.DrawText(title, int32(sw/2)-titleWidth/2, 50, 30, rl.Black)

		charDef := characters[selectedIndex]
		destRect := drawCharacterPreview(previews[selectedIndex], charDef, sw/2, 110)

		nameWidth := rl.MeasureText(charDef.Name, 24)
		rl.DrawText(charDef.Name, int32(sw/2)-nameWidth/2, int32(destRect.Y+destRect.Height+20), 24, rl.DarkGray)

		// Navigation buttons
		previewMiddle := destRect.Y + destRect.Height/2
		prevRect := rl.NewRectangle(sw/2-200, previewMiddle-25, 50, 50)
		nextRect := rl.NewRectangle(sw/2+150, previewMiddle-25, 50, 50)

		rl.DrawRectangleRec(prevRect, rl.LightGray)
		rl.DrawRectangleRec(nextRect, rl.LightGray)
		rl.DrawText("<", int32(prevRect.X+15), int32(prevRect.Y+15), 20, rl.Black)
		rl.DrawText(">", int32(nextRect.X+15), int32(nextRect.Y+15), 20, rl.Black)

		// Select button
		selectBtn := rl.NewRectangle(sw/2-100, sh-100, 200, 50)
		rl.DrawRectangleRec(selectBtn, rl.DarkGray)

		selectText := "SELECT"
		selectTextWidth := rl.MeasureText(selectText, 20)
		rl.DrawText(selectText, int32(selectBtn.X)+int32(selectBtn.Width/2)-selectTextWidth/2, int32(selectBtn.Y+15), 20, rl.White)

		// Handle Input
		if rl.IsKeyPressed(rl.KeyLeft) {
			selectedIndex--
			if selectedIndex < 0 {
				selectedIndex = len(characters) - 1
			}
		}
		if rl.IsKeyPressed(rl.KeyRight) {
			selectedIndex++
			if selectedIndex >= len(characters) {
				selectedIndex = 0
			}
		}
		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeySpace) {
			selected = true
		}

		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			mouse := rl.GetMousePosition()
			if rl.CheckCollisionPointRec(mouse, prevRect) {
				selectedIndex--
				if selectedIndex < 0 {
					selectedIndex = len(characters) - 1
				}
			} else if rl.CheckCollisionPointRec(mouse, nextRect) {
				selectedIndex++
				if selectedIndex >= len(characters) {
					selectedIndex = 0
				}
			} else if rl.CheckCollisionPointRec(mouse, selectBtn) {
				selected = true
			}
		}

		rl.EndDrawing()
	}

	return characters[selectedIndex].Type
}
