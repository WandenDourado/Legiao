//go:build android

package ui

// No Android o teclado do sistema nao abre (settings_textfield.go): o teclado
// desenhado pelo jogo e o unico jeito de digitar o nome da sala.
const useOnScreenKeyboard = true
