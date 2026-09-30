package entity

// O passo de simulacao da desintegracao: soltar os graos cuja hora chegou e
// soprar os soltos. So visual — nada aqui toca vida, colisao ou rede.

import (
	"math"
)

func (b *ashBurst) step(dt float32) {
	tn := b.tune
	b.t += dt
	t := b.t
	s := b.pop()
	// Amortecimento por SEGUNDO (exp), nao por chamada: o jogo roda com dt
	// variavel e o ritmo do rastro nao pode depender do fps.
	damp := float32(math.Exp(float64(-tn.Drag * dt)))
	live := 0
	b.minX, b.minY = b.cx-b.frameW, b.cy-b.frameH
	b.maxX, b.maxY = b.cx+b.frameW, b.cy+b.frameH
	for i := range b.frags {
		f := &b.frags[i]
		if !f.free {
			if t < f.release {
				live++
				continue
			}
			f.free = true
			f.x, f.y = b.cx+f.hx*s, b.cy+f.hy*s
			a := deathRand() * 2 * math.Pi
			k := tn.Kick * (0.4 + deathRand()*0.6)
			f.vx = b.dirX*tn.WindAcc*0.12 + float32(math.Cos(float64(a)))*k
			f.vy = b.dirY*tn.WindAcc*0.12 + float32(math.Sin(float64(a)))*k
			f.rot = b.angle
			if deathRand() < tn.PuffP {
				b.puffs = append(b.puffs, ashPuff{x: f.x, y: f.y, vx: f.vx * 0.5, vy: f.vy * 0.5, c: f.ash})
			}
		}
		age := t - f.release
		if age > f.life {
			continue
		}
		live++
		ph := float64(f.seed) * 6.283
		tt := float64(t)
		tx := math.Sin(float64(f.y)*0.021+tt*2.1+ph) + 0.5*math.Sin(float64(f.y)*0.057-tt*3.3+ph*2)
		ty := math.Cos(float64(f.x)*0.019-tt*1.7+ph) + 0.5*math.Cos(float64(f.x)*0.049+tt*2.9+ph*3)
		lift := tn.Lift
		if f.ember {
			lift *= 2.2
		}
		f.vx += (b.dirX*tn.WindAcc + float32(tx)*tn.Turb) * dt
		f.vy += (b.dirY*tn.WindAcc + float32(ty)*tn.Turb - lift) * dt
		f.vx *= damp
		f.vy *= damp
		f.x += f.vx * dt
		f.y += f.vy * dt
		f.rot += f.spin * dt * 57.3
		b.minX, b.minY = min(b.minX, f.x), min(b.minY, f.y)
		b.maxX, b.maxY = max(b.maxX, f.x), max(b.maxY, f.y)
	}
	b.live = live

	kept := b.puffs[:0]
	for _, p := range b.puffs {
		p.age += dt
		if p.age >= tn.PuffLife {
			continue
		}
		p.vx += b.dirX * tn.WindAcc * 0.5 * dt
		p.vy += (b.dirY*tn.WindAcc*0.5 - tn.Lift) * dt
		p.vx *= damp
		p.vy *= damp
		p.x += p.vx * dt
		p.y += p.vy * dt
		b.minX, b.minY = min(b.minX, p.x-tn.PuffR1), min(b.minY, p.y-tn.PuffR1)
		b.maxX, b.maxY = max(b.maxX, p.x+tn.PuffR1), max(b.maxY, p.y+tn.PuffR1)
		kept = append(kept, p)
	}
	b.puffs = kept
}
