package entity

// Uma morte em andamento: os fragmentos de UM monstro, do golpe final ate a
// ultima cinza sumir. Espelho de `class AshBurst` do preview aprovado.

import (
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DeathPose e o quadro exato que estava na tela quando o monstro morreu.
type DeathPose struct {
	Type   EnemyType
	Anim   EnemyAnim
	Frame  int     // coluna JA resolvida (EnemyAnimDef.Column)
	Row    int     // linha JA validada; 0 fora do modo direcional
	Mirror bool    // modo direcional espelhado
	Angle  float32 // graus; so o modo radial gira
	CX, CY float32 // centro do quadro desenhado, em mundo
}

type ashFrag struct {
	hx, hy        float32 // centro "em casa", relativo ao centro do quadro, sem o pop
	srcX, srcY    float32 // canto da celula na folha
	srcW, srcH    float32 // negativo em srcW = espelhado (convencao do raylib)
	w, h          float32 // tamanho em mundo
	x, y, vx, vy  float32
	rot, spin     float32
	release, life float32
	seed          float32
	base, ash     color.RGBA
	ember, free   bool
}

type ashPuff struct {
	x, y, vx, vy, age float32
	c                 color.RGBA
}

type ashBurst struct {
	tex        rl.Texture2D
	frameSrc   rl.Rectangle // o quadro inteiro, para o clarao
	frameW     float32      // tamanho do quadro em mundo
	frameH     float32
	angle      float32 // rotacao do desenho (radial)
	cx, cy     float32
	dirX, dirY float32
	tune       deathTuning
	boss       bool
	cellWorld  float32
	t, end     float32
	frags      []ashFrag
	puffs      []ashPuff
	live       int
	minX, minY float32 // caixa da poeira viva, para o culling
	maxX, maxY float32
}

// newAshBurst monta a explosao a partir da pose. Devolve nil quando nao ha
// textura ou celulas (mascara ainda nao pronta, folha ausente).
func newAshBurst(pose DeathPose, cells []deathCell, cellSrc int, tex rl.Texture2D, ad EnemyAnimDef, def EnemyDef) *ashBurst {
	if len(cells) == 0 || tex.ID == 0 {
		return nil
	}
	scale := def.RenderScale
	if scale <= 0 {
		scale = 1
	}
	tune := deathTuneFor(def)
	fw, fh := float32(ad.FrameWidth), float32(ad.FrameHeight)
	b := &ashBurst{
		tex:      tex,
		frameSrc: rl.NewRectangle(float32(pose.Frame)*fw, float32(pose.Row)*fh, fw, fh),
		frameW:   fw * scale, frameH: fh * scale,
		angle: pose.Angle, cx: pose.CX, cy: pose.CY,
		tune: tune, boss: def.IsBoss(),
		cellWorld: float32(cellSrc) * scale,
	}
	if pose.Mirror {
		b.frameSrc.Width = -fw
	}
	wind := float64(deathWindBaseDeg+(deathRand()*2-1)*deathWindJitterDeg) * math.Pi / 180
	b.dirX, b.dirY = float32(math.Cos(wind)), float32(math.Sin(wind))

	rad := float64(pose.Angle) * math.Pi / 180
	cs, sn := float32(math.Cos(rad)), float32(math.Sin(rad))
	b.frags = make([]ashFrag, len(cells))
	pmin, pmax := float32(math.MaxFloat32), float32(-math.MaxFloat32)
	for i, c := range cells {
		lx := (float32(c.U) + float32(c.W)/2 - fw/2) * scale
		ly := (float32(c.V) + float32(c.H)/2 - fh/2) * scale
		if pose.Mirror {
			lx = -lx
		}
		hx, hy := lx*cs-ly*sn, lx*sn+ly*cs
		f := &b.frags[i]
		f.hx, f.hy = hx, hy
		f.srcX = b.frameSrc.X + float32(c.U)
		f.srcY = b.frameSrc.Y + float32(c.V)
		f.srcW, f.srcH = float32(c.W), float32(c.H)
		if pose.Mirror {
			// Largura negativa e como o raylib espelha uma regiao — a mesma
			// convencao de drawEnemyDirectional. A celula continua lendo os
			// PROPRIOS pixels; so o lugar dela (lx acima) e o sentido mudam.
			f.srcW = -f.srcW
		}
		f.w, f.h = float32(c.W)*scale, float32(c.H)*scale
		p := hx*b.dirX + hy*b.dirY
		f.release = p // provisorio: projecao no vento
		pmin, pmax = min(pmin, p), max(pmax, p)
		f.spin = (deathRand() - 0.5) * 8
		f.life = tune.LifeMin + (tune.LifeMax-tune.LifeMin)*deathRand()
		f.seed = deathRand()
		f.ember = deathRand() < tune.EmberFrac
		lum := 0.30*float32(c.Col.R) + 0.59*float32(c.Col.G) + 0.11*float32(c.Col.B)
		g := 78 + lum*0.5 // claro o bastante para a cinza ler no chao escuro
		f.base = color.RGBA{
			uint8(float32(c.Col.R) * deathCharTint[0]),
			uint8(float32(c.Col.G) * deathCharTint[1]),
			uint8(float32(c.Col.B) * deathCharTint[2]), 255}
		f.ash = color.RGBA{uint8(min(g+8, 255)), uint8(min(g+3, 255)), uint8(g), 255}
	}
	// O lado a favor do vento solta primeiro: a frente anda CONTRA o vento e
	// a poeira sai para longe do que ainda e corpo, sem passar por cima dele.
	span := max(pmax-pmin, 1)
	ns := float32(cellSrc) * tune.NoiseScale
	nOff := deathRand() * 100
	end := tune.Delay
	for i, c := range cells {
		f := &b.frags[i]
		p01 := (pmax - f.release) / span
		n := valueNoise2(float32(c.U)/ns+nOff, float32(c.V)/ns+nOff*0.7)
		f.release = tune.Delay + tune.Sweep*((1-tune.Noise)*p01+tune.Noise*n)
		end = max(end, f.release+f.life)
	}
	b.end = end + 0.05
	b.live = len(b.frags)
	return b
}

// pop e o inchaco do golpe: 1 -> 1+Pop -> 1 em PopT segundos.
func (b *ashBurst) pop() float32 {
	q := clampUnit(b.t / b.tune.PopT)
	return 1 + b.tune.Pop*float32(math.Sin(math.Pi*float64(q)))
}

func (b *ashBurst) done() bool { return b.t >= b.end && len(b.puffs) == 0 }

// bodyGone diz se o corpo ja cedeu inteiro (resta so poeira).
func (b *ashBurst) bodyGone() bool {
	return b.t >= b.tune.Delay+b.tune.Sweep+0.05
}
