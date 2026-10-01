package input

import "math"

// Curva de resposta do joystick de movimento (Android).
//
// Por que existe: Player.Update faz Velocity = dir * Speed, entao o
// comprimento de dir E a velocidade. No teclado dir sempre tem comprimento 1
// (a diagonal e normalizada em ProcessInput). No joystick, GetJoystickDelta
// devolve o deslocamento do polegar LINEAR em 0..1, e 1 so acontece arrastando
// o raio maximo inteiro a partir de onde o dedo pousou (o raio ainda cresce
// com JoystickRadiusScale). Na pratica o polegar fica em 40-70% do raio, e o
// jogador do celular andava a 40-70% da velocidade do jogador do desktop.
//
// A curva tira a zona morta e alcanca a velocidade cheia bem antes da borda:
// a partir de JoystickFullSpeedAt o celular anda exatamente como o teclado.
// Entre as duas ainda da para andar devagar de proposito.
const (
	// JoystickDeadZone: abaixo disto (fracao do raio maximo) o personagem
	// nao anda — tremida do polegar parado nao vira deriva.
	JoystickDeadZone float32 = 0.15
	// JoystickFullSpeedAt: a partir disto o comprimento e 1, o mesmo do
	// teclado. Fica ABAIXO de entity.SprintThreshold (0,70) de proposito:
	// no desktop a corrida (Shift) e so animacao e nao muda a velocidade, e
	// aqui tambem — velocidade cheia com passo de andar, depois o passo de
	// correr.
	JoystickFullSpeedAt float32 = 0.55
)

// JoystickMoveResponse converte o deslocamento cru do joystick (componentes
// em [-1, 1], comprimento <= 1, como GetJoystickDelta devolve) no vetor de
// movimento. A direcao e preservada; so o comprimento passa pela curva.
func JoystickMoveResponse(x, y float32) (float32, float32) {
	m := float32(math.Sqrt(float64(x*x + y*y)))
	if m <= JoystickDeadZone {
		return 0, 0
	}
	s := (m - JoystickDeadZone) / (JoystickFullSpeedAt - JoystickDeadZone)
	if s > 1 {
		s = 1
	}
	return x / m * s, y / m * s
}
