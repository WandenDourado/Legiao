package entity

import rl "github.com/gen2brain/raylib-go/raylib"

// WalkRowFor is walkRowForDirection exported for callers outside the
// package that animate a character without a *Player — a bot's PlayerState
// has no Player to call updateAnimation on.
func WalkRowFor(dir rl.Vector2) int {
	return walkRowForDirection(dir)
}

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

func validWalkFrame(frame, columns int) int {
	if frame < 0 || frame >= columns {
		return 0
	}
	return frame
}

func validWalkRow(row, rows int) int {
	if row < 0 || row >= rows {
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

