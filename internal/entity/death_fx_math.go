package entity

// Utilitarios da desintegracao: sorteio barato, ruido de valor e interpolacao.
// Tudo roda na goroutine de desenho, entao o gerador nao precisa de trava.

import "math"

var deathRandState uint32 = 0x9E3779B9

// deathRand devolve [0,1). Xorshift32: o efeito sorteia milhares de numeros
// por morte e nao precisa de nada melhor que isso.
func deathRand() float32 {
	x := deathRandState
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	deathRandState = x
	return float32(x>>8) / float32(1<<24)
}

func hash2(x, y int32) float32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return float32(h^(h>>16)) / 4294967296
}

// valueNoise2 e ruido de valor suave em [0,1). E ele que faz a frente de
// desintegracao rachar em manchas em vez de andar em linha reta.
func valueNoise2(x, y float32) float32 {
	xf, yf := float32(math.Floor(float64(x))), float32(math.Floor(float64(y)))
	xi, yi := int32(xf), int32(yf)
	tx, ty := x-xf, y-yf
	u, v := tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	a := lerpF(hash2(xi, yi), hash2(xi+1, yi), u)
	b := lerpF(hash2(xi, yi+1), hash2(xi+1, yi+1), u)
	return lerpF(a, b, v)
}

func lerpF(a, b, t float32) float32 { return a + (b-a)*t }

func clampUnit(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// smoothstep entre a e b.
func smoothF(a, b, v float32) float32 {
	t := clampUnit((v - a) / (b - a))
	return t * t * (3 - 2*t)
}

func lerpU8(a, b uint8, t float32) uint8 {
	return uint8(float32(a) + (float32(b)-float32(a))*t)
}
