package game

// Trava de dt para o primeiro quadro depois de um hiato grande.
//
// rl.GetFrameTime() mede o tempo desde o quadro anterior. Um app Android
// pausado (tela bloqueada) e retomado depois de segundos entrega um dt do
// tamanho do hiato inteiro no primeiro quadro seguinte: fisica, cooldown e
// interpolacao que multiplicam por dt teleportariam tudo de uma vez.
//
// maxFrameDT e generoso o bastante para absorver alguns quadros perdidos sem
// travar visivelmente (a 60 fps um quadro custa ~16,7 ms; 50 ms cobre cerca
// de tres), mas curto o bastante para nunca deixar o jogo simular um hiato de
// segundos como se fosse um unico quadro.
const maxFrameDT float32 = 0.05

// clampFrameDT limita dt ao teto acima. Chamado uma vez por quadro no laco
// principal; nao muda o comportamento em jogo normal, onde dt nunca chega
// perto do teto.
func clampFrameDT(dt float32) float32 {
	if dt > maxFrameDT {
		return maxFrameDT
	}
	return dt
}
