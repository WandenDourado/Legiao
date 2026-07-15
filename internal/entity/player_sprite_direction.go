package entity

import rl "github.com/gen2brain/raylib-go/raylib"

func walkRowForDirection(dir rl.Vector2) int {
	absX := abs32(dir.X)
	absY := abs32(dir.Y)

	if dir.Y < 0 && absX > absY*0.5 {
		return RowWalkUpLeft
	}
	if dir.Y > 0 && absX > absY*0.5 {
		return RowWalkDownLeft
	}
	if dir.Y < 0 {
		return RowWalkUp
	}
	if dir.Y > 0 {
		return RowWalkDown
	}
	if absX > 0 {
		return RowWalkLeft
	}
	return RowWalkDown
}

func validWalkFrame(frame int) int {
	if frame < 0 || frame >= WizardColumns {
		return 0
	}
	return frame
}

func validWalkRow(row int) int {
	if row < 0 || row >= WizardRows {
		return RowWalkDown
	}
	return row
}

func shouldMirrorWalkRow(row int, velX float32) bool {
	return velX > 0 && (row == RowWalkLeft || row == RowWalkDownLeft || row == RowWalkUpLeft)
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
