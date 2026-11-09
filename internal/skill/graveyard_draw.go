package skill

// Procedural visuals for the Necromante's Cemitério: rotting darkened ground,
// skeletal hands clawing upward, drifting grave mist and rising soul wisps.
//
// O DESENHO E DIVIDIDO EM DOIS PASSES, E A DIVISAO E DE PROPOSITO.
//
// O Cemitério e chao amaldicoado. Desenhado inteiro depois dos corpos — que e
// onde todo o bloco de magias roda (game/renderer.go) — o retangulo escuro
// passava por cima de quem estivesse dentro dele: o mapa poluia e a leitura
// invertia, porque o personagem parecia estar DEBAIXO da magia em vez de pisar
// nela. Relato dos testadores em 20/09/2026, e o mesmo defeito que o
// pentagrama do Mago teve em 10/09/2026.
//
// A correcao la foi tirar o selo da fila das magias. Aqui nao da para tirar o
// efeito inteiro, porque ele nao e todo chao:
//
//	DrawGround   — solo apodrecido, rim, manchas, miasma verde e o buraco sob
//	               cada mao. Vai ANTES dos corpos, junto com o selo do Mago.
//	DrawOverlay  — os ossos das maos e as almas subindo. Continuam DEPOIS dos
//	               corpos, porque estao acima do chao: e a mao agarrando a
//	               perna de quem esta dentro que faz a magia comunicar o que
//	               ela faz.
//
// Mover tudo para antes dos corpos seria uma linha so, e custaria exatamente
// isso: as maos passariam por tras dos personagens.
//
// As duas metades moram em graveyard_draw_ground.go e
// graveyard_draw_overlay.go. Este arquivo guarda a paleta, a animacao
// cosmetica e o portao que as duas compartilham.

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	graveRot    = rl.NewColor(22, 14, 26, 255)    // rotten earth
	graveRotRim = rl.NewColor(52, 32, 66, 255)    // bruised purple rim
	graveGreen  = rl.NewColor(110, 220, 140, 255) // sickly necro-green
	gravePurple = rl.NewColor(150, 60, 220, 255)
	graveBone   = rl.NewColor(215, 205, 182, 255)
)

// advanceVisual ages the zone and animates its cosmetic particles. Used by
// both the host step and the client advance.
func (g *Graveyard) advanceVisual(dt float32) {
	g.Age += dt
	if g.IsCursing() {
		// Rising soul wisps: alternating sickly green and violet motes.
		g.soulAccum += dt
		for g.soulAccum > 0.08 {
			g.soulAccum -= 0.08
			u := float32(rl.GetRandomValue(0, int32(GraveyardLength)))
			v := float32(rl.GetRandomValue(-int32(GraveyardWidth/2), int32(GraveyardWidth/2)))
			c := graveGreen
			if rl.GetRandomValue(0, 1) == 0 {
				c = gravePurple
			}
			vel := rl.NewVector2(float32(rl.GetRandomValue(-12, 12)), -30-float32(rl.GetRandomValue(0, 35)))
			g.Souls.Emit(g.point(u, v), vel, 1.1, 4, c)
		}
	}
	g.Souls.Update(dt)
}

// visible reporta se ha alguma coisa a desenhar, e e o mesmo portao nos dois
// passes: um cemiterio totalmente desvanecido ainda pode ter almas no ar.
func (g *Graveyard) visible() bool {
	return g.FadeAlpha() > 0 || len(g.Souls.particles) > 0
}
