package entity

// O desenho da desintegracao, em tres passes que cobrem TODAS as mortes em
// curso de uma vez (e nao tres passes por morte): corpo texturizado, poeira
// lisa e o passe aditivo. Agrupar por passe e o que mantem a troca de textura
// e de modo de mistura em tres por quadro, com uma ou trinta mortes na tela.

import (
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func (b *ashBurst) visible(view rl.Rectangle) bool {
	return b.maxX >= view.X && b.minX <= view.X+view.Width &&
		b.maxY >= view.Y && b.minY <= view.Y+view.Height
}

// drawBody: o que ainda e corpo — cada celula e um pedaco da propria folha,
// escurecendo para carvao, mais escura perto da frente.
func (b *ashBurst) drawBody() {
	tn, t, s := b.tune, b.t, b.pop()
	charT := smoothF(tn.Flash*0.5, tn.Flash+tn.Char, t)
	for i := range b.frags {
		f := &b.frags[i]
		if f.free {
			continue
		}
		near := smoothF(f.release-tn.Edge*3, f.release, t)
		lvl := float32(math.Round(float64((charT*0.65+near*0.35)*(deathTintLevels-1)))) / (deathTintLevels - 1)
		tint := color.RGBA{
			uint8(255 * lerpF(1, deathCharTint[0], lvl)),
			uint8(255 * lerpF(1, deathCharTint[1], lvl)),
			uint8(255 * lerpF(1, deathCharTint[2], lvl)), 255}
		// +0,35 px: sobreposicao minima para a malha de celulas nao mostrar
		// costura quando a camera cai entre dois pixels.
		w, h := f.w*s+0.35, f.h*s+0.35
		dst := rl.NewRectangle(b.cx+f.hx*s, b.cy+f.hy*s, w, h)
		rl.DrawTexturePro(b.tex, rl.NewRectangle(f.srcX, f.srcY, f.srcW, f.srcH), dst,
			rl.NewVector2(w/2, h/2), b.angle, tint)
	}
}

// drawDust: tufos de fumaca e os graos soltos, cor lisa passando de carvao
// para cinza.
func (b *ashBurst) drawDust() {
	tn, t := b.tune, b.t
	for _, p := range b.puffs {
		q := p.age / tn.PuffLife
		r := lerpF(tn.PuffR0, tn.PuffR1, float32(math.Sqrt(float64(q))))
		a := tn.PuffA * float32(math.Sin(math.Pi*float64(q)))
		inner := rl.Fade(p.c, a)
		outer := rl.Fade(p.c, 0) // para Blank, nunca para preto (halo escuro)
		rl.DrawCircleGradient(int32(p.x), int32(p.y), r, inner, outer)
	}
	for i := range b.frags {
		f := &b.frags[i]
		if !f.free || f.ember {
			continue
		}
		age := t - f.release
		q := age / f.life
		if q > 1 {
			continue
		}
		toAsh := smoothF(0, 0.45, q)
		hot := 1 - smoothF(0, tn.Hot, age)
		col := color.RGBA{
			lerpU8(lerpU8(f.base.R, f.ash.R, toAsh), deathEmber.R, hot),
			lerpU8(lerpU8(f.base.G, f.ash.G, toAsh), deathEmber.G, hot),
			lerpU8(lerpU8(f.base.B, f.ash.B, toAsh), deathEmber.B, hot), 255}
		alpha := 0.92 * (1 - smoothF(0.3, 1, q))
		size := b.cellWorld * lerpF(0.85, 0.2, float32(math.Sqrt(float64(q))))
		rl.DrawRectanglePro(rl.NewRectangle(f.x, f.y, size, size),
			rl.NewVector2(size/2, size/2), f.rot, rl.Fade(col, alpha))
	}
}

// drawGlow roda DENTRO de BeginBlendMode(BlendAdditive): clarao do golpe,
// frente em brasa e faiscas.
func (b *ashBurst) drawGlow() {
	tn, t, s := b.tune, b.t, b.pop()
	if t < tn.Flash {
		// O proprio quadro somado a si mesmo e o clarao — sem textura branca
		// extra nem shader.
		w, h := b.frameW*s, b.frameH*s
		rl.DrawTexturePro(b.tex, b.frameSrc, rl.NewRectangle(b.cx, b.cy, w, h),
			rl.NewVector2(w/2, h/2), b.angle, rl.Fade(rl.White, 0.85*(1-t/tn.Flash)))
	}
	for i := range b.frags {
		f := &b.frags[i]
		var x, y, size, alpha float32
		col := deathEmber
		switch {
		case !f.free:
			lead := f.release - t
			if lead > tn.Edge {
				continue
			}
			e := 1 - lead/tn.Edge
			alpha, size = tn.EdgeA*e*e, b.cellWorld*s
			x, y = b.cx+f.hx*s, b.cy+f.hy*s
		case f.ember:
			q := (t - f.release) / f.life
			if q > 1 {
				continue
			}
			alpha = 1 - smoothF(0.2, 1, q)
			size = b.cellWorld * lerpF(0.6, 0.2, q)
			x, y, col = f.x, f.y, deathSpark
		default:
			continue
		}
		rl.DrawRectanglePro(rl.NewRectangle(x, y, size, size),
			rl.NewVector2(size/2, size/2), 0, rl.Fade(col, alpha))
	}
}
