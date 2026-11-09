//go:build android

package main

import rl "github.com/gen2brain/raylib-go/raylib"

func init() {
	rl.SetCallbackFunc(main)
}
