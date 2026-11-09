//go:build !android

package apprestart

// Supported: no PC a resolucao de desenho e sempre a do monitor, entao nada
// pede reinicio.
const Supported = false

// Restart nao faz nada no desktop.
func Restart() error { return ErrUnsupported }
