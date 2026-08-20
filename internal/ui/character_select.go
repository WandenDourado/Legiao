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

		// Navigation buttons (sized relative to screen so the drawn button
		// matches the clickable area on any display size).
		previewMiddle := destRect.Y + destRect.Height/2
		navSize := max(sh*0.07, 52)
		prevRect := rl.NewRectangle(sw*0.5-navSize*3.2, previewMiddle-navSize/2, navSize, navSize)
		nextRect := rl.NewRectangle(sw*0.5+navSize*2.2, previewMiddle-navSize/2, navSize, navSize)

		rl.DrawRectangleRec(prevRect, rl.LightGray)
		rl.DrawRectangleRec(nextRect, rl.LightGray)
		navFont := int32(max(sh*0.04, 24))
		rl.DrawText("<", int32(prevRect.X+(prevRect.Width-float32(rl.MeasureText("<", navFont)))/2),
			int32(prevRect.Y+(prevRect.Height-float32(navFont))/2), navFont, rl.Black)
		rl.DrawText(">", int32(nextRect.X+(nextRect.Width-float32(rl.MeasureText(">", navFont)))/2),
			int32(nextRect.Y+(nextRect.Height-float32(navFont))/2), navFont, rl.Black)

		// Bottom row: BACK and SELECT share it. Both are sized as a fraction of
		// the screen width (never a fixed pixel width) so they cannot overlap
		// on a narrow phone, where a fixed 200px button would run off the row.
		btnH := max(sh*0.08, 50)
		btnY := sh - btnH*1.4
		rowW := sw * 0.8
		gap := sw * 0.08
		backW := (rowW - gap) * 0.4
		selectW := (rowW - gap) * 0.6
		backBtn := rl.NewRectangle((sw-rowW)/2, btnY, backW, btnH)
		selectBtn := rl.NewRectangle(backBtn.X+backW+gap, btnY, selectW, btnH)

		rl.DrawRectangleRec(backBtn, rl.LightGray)
		rl.DrawRectangleRec(selectBtn, rl.DarkGray)

		btnFont := int32(max(sh*0.028, 20))
		backText := "BACK"
		backTextWidth := rl.MeasureText(backText, btnFont)
		rl.DrawText(backText, int32(backBtn.X+(backBtn.Width-float32(backTextWidth))/2),
			int32(backBtn.Y+(backBtn.Height-float32(btnFont))/2), btnFont, rl.Black)

		selectText := "SELECT"
		selectTextWidth := rl.MeasureText(selectText, btnFont)
		rl.DrawText(selectText, int32(selectBtn.X+(selectBtn.Width-float32(selectTextWidth))/2),
			int32(selectBtn.Y+(selectBtn.Height-float32(btnFont))/2), btnFont, rl.White)

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
