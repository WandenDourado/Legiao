package ui

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// ShowCharacterSelect displays a UI to choose a character. The second return
// value is false when the player backed out instead of confirming, so the
// caller can return to the previous screen without starting a session.
func ShowCharacterSelect() (entity.CharacterType, bool) {
	characters := entity.AllCharacters()
	if len(characters) == 0 {
		return entity.CharMago, false // Fallback
	}

	selectedIndex := 0
	selected := false
	cancelled := false

	previews := loadCharacterPreviews(characters)
	defer func() {
		for _, preview := range previews {
			rl.UnloadTexture(preview.texture)
		}
	}()

	for !selected && !cancelled && !rl.WindowShouldClose() {
		rl.BeginDrawing()
		prevRect, nextRect, backBtn, selectBtn := drawCharacterSelectScreen(characters[selectedIndex], previews[selectedIndex])

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
			// SELECT is tested before BACK: where the two padded hit areas meet
			// in the gap, the primary action wins, so a near-miss never cancels.
			if hit(selectBtn, mouse) {
				selected = true
			} else if hit(backBtn, mouse) {
				cancelled = true
			} else if hit(prevRect, mouse) {
				selectedIndex--
				if selectedIndex < 0 {
					selectedIndex = len(characters) - 1
				}
			} else if hit(nextRect, mouse) {
				selectedIndex++
				if selectedIndex >= len(characters) {
					selectedIndex = 0
				}
			}
		}

		rl.EndDrawing()
	}

	if cancelled || !selected {
		return characters[selectedIndex].Type, false
	}
	return characters[selectedIndex].Type, true
}
