//go:build !android

package ui

// No computador o jogador digita no teclado fisico; o teclado desenhado so
// ocuparia a tela.
const useOnScreenKeyboard = false
