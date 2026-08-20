package ui

import (
	"fmt"
	"strings"

	"github.com/WandenDourado/Legiao/internal/assets"
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Dialogue box layout, all proportional to the screen so the box reads the
// same on a phone and on a desktop monitor.
const (
	dialogueSideMargin   = 0.06 // of screen width, each side
	dialogueBottomMargin = 0.05 // of screen height
	dialogueHeightRatio  = 0.26 // of screen height
	dialoguePadding      = 18.0
	dialogueMinHeight    = 160.0
)

// placeholderPortraits lists characters whose reference art is still borrowed
// from somebody else. Drawing it would put another character's face on their
// line, which is worse than no portrait, so they speak unillustrated.
// Delete the entry as soon as the character has its own reference.png.
//
// Currently empty: every registered character owns its art. Installing a
// character with borrowed art means adding it here; installing its real art
// means removing it again, or the portrait stays silently blank.
var placeholderPortraits = map[entity.CharacterType]bool{}

// portraitCache keeps one texture per speaker. Loaded on first use and kept
// for the session: a scene flips between two or three speakers repeatedly, and
// reloading a 1k texture per line would stutter the box open.
var portraitCache = map[entity.CharacterType]rl.Texture2D{}

// DrawDialogueBox draws the line the host is currently narrating. It draws
// nothing when no dialogue is running, so the caller can call it every frame.
//
// It is display only on every machine, host included: what appears here is
// whatever network.GetDialogueState() holds.
func DrawDialogueBox(screenWidth, screenHeight float32) {
	d := network.GetDialogueState()
	if !d.Active || d.Text == "" {
		return
	}

	box := dialogueRect(screenWidth, screenHeight)

	// Dim the whole scene so the eye goes to the box and the frozen battle
	// behind it reads as paused rather than as a bug.
	rl.DrawRectangle(0, 0, int32(screenWidth), int32(screenHeight), rl.Fade(rl.Black, 0.35))

	rl.DrawRectangleRec(box, rl.Fade(rl.Black, 0.85))
	rl.DrawRectangleLinesEx(box, 2, rl.Fade(rl.RayWhite, 0.45))

	textX := box.X + dialoguePadding
	textWidth := box.Width - dialoguePadding*2

	if tex, ok := portraitTexture(d.Portrait); ok {
		frame := drawDialoguePortrait(tex, box)
		textX = frame.X + frame.Width + dialoguePadding
		textWidth = box.X + box.Width - dialoguePadding - textX
	}

	nameSize := dialogueFontSize(screenHeight, 0.030)
	bodySize := dialogueFontSize(screenHeight, 0.026)
	hintSize := dialogueFontSize(screenHeight, 0.020)

	y := box.Y + dialoguePadding
	if d.Speaker != "" {
		rl.DrawText(d.Speaker, int32(textX), int32(y), nameSize, rl.Gold)
		y += float32(nameSize) + 8
	}

	for _, line := range wrapText(d.Text, bodySize, int32(textWidth)) {
		// Stop before spilling out of the box: a line that would be drawn
		// under the border is not readable anyway.
		if y+float32(bodySize) > box.Y+box.Height-dialoguePadding-float32(hintSize) {
			break
		}
		rl.DrawText(line, int32(textX), int32(y), bodySize, rl.RayWhite)
		y += float32(bodySize) + 6
	}

	drawDialogueHint(d, box, hintSize)
}

// dialogueRect is the single source of the box geometry: the drawing and the
// touch hit-area both read it, so tapping always matches what is on screen.
func dialogueRect(screenWidth, screenHeight float32) rl.Rectangle {
	height := screenHeight * dialogueHeightRatio
	if height < dialogueMinHeight {
		height = dialogueMinHeight
	}
	margin := screenWidth * dialogueSideMargin
	return rl.NewRectangle(
		margin,
		screenHeight-height-screenHeight*dialogueBottomMargin,
		screenWidth-margin*2,
		height,
	)
}

// DialogueBoxRect exposes the box area so the input layer can treat a tap
// inside it as "next line" without duplicating the layout maths.
func DialogueBoxRect(screenWidth, screenHeight float32) rl.Rectangle {
	return dialogueRect(screenWidth, screenHeight)
}

// drawDialoguePortrait fits the speaker's reference art into a framed panel on
// the left of the box and returns the panel it used.
func drawDialoguePortrait(tex rl.Texture2D, box rl.Rectangle) rl.Rectangle {
	frame := rl.NewRectangle(
		box.X+dialoguePadding,
		box.Y+dialoguePadding,
		(box.Height-dialoguePadding*2)*0.8,
		box.Height-dialoguePadding*2,
	)
	rl.DrawRectangleRec(frame, rl.Fade(rl.Black, 0.5))

	source := rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height))
	// Fit the whole figure: references are framed differently per character
	// (some full body, some square), so cropping to a head would land on the
	// chest for one of them.
	scale := frame.Width / source.Width
	if h := frame.Height / source.Height; h < scale {
		scale = h
	}
	drawn := rl.NewRectangle(
		frame.X+(frame.Width-source.Width*scale)/2,
		frame.Y+(frame.Height-source.Height*scale),
		source.Width*scale,
		source.Height*scale,
	)
	rl.DrawTexturePro(tex, source, drawn, rl.NewVector2(0, 0), 0, rl.White)
	rl.DrawRectangleLinesEx(frame, 1, rl.Fade(rl.RayWhite, 0.25))
	return frame
}

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
		portraitCache[ct] = rl.Texture2D{}
		return rl.Texture2D{}, false
	}
	tex := rl.LoadTexture(assets.Path(def.ReferenceImagePath))
	portraitCache[ct] = tex
	return tex, tex.ID != 0
}

// drawDialogueHint tells each machine what it can do: the host advances, a
// client waits. Without it a client reads a frozen screen with no explanation.
func drawDialogueHint(d network.DialogueState, box rl.Rectangle, size int32) {
	hint := "Enter ou toque para continuar"
	if network.Role == "client" {
		hint = "aguardando o narrador..."
	}
	if d.Total > 0 {
		hint = fmt.Sprintf("%s   %d/%d", hint, d.Index, d.Total)
	}
	width := rl.MeasureText(hint, size)
	rl.DrawText(hint,
		int32(box.X+box.Width-dialoguePadding)-width,
		int32(box.Y+box.Height-dialoguePadding-float32(size)),
		size, rl.Fade(rl.LightGray, 0.8))
}

// dialogueFontSize scales a font with the screen and keeps it inside a range
// that stays legible on a phone without turning into a headline on a monitor.
func dialogueFontSize(screenHeight, ratio float32) int32 {
	size := screenHeight * ratio
	if size < 16 {
		size = 16
	}
	if size > 34 {
		size = 34
	}
	return int32(size)
}

// wrapText breaks a paragraph into lines that fit maxWidth. A word longer than
// the whole line is left to overflow rather than split mid-word: that only
// happens with a malformed script, and a broken word is harder to spot than a
// line running long.
func wrapText(text string, fontSize int32, maxWidth int32) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, 4)
	current := words[0]
	for _, w := range words[1:] {
		candidate := current + " " + w
		if rl.MeasureText(candidate, fontSize) > maxWidth {
			lines = append(lines, current)
			current = w
			continue
		}
		current = candidate
	}
	return append(lines, current)
}
