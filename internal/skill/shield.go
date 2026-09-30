package skill

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Shield constants tune the Paladina's Escudo Sagrado skill.
const (
	// ShieldMaxHP is how much damage the shield absorbs before breaking.
	ShieldMaxHP float32 = 50
	// ShieldCooldown is the per-caster cooldown in seconds. Recasting while
	// active regenerates the shield back to ShieldMaxHP (never stacks).
	ShieldCooldown float32 = 6.0
	// ShieldRadius is the visual radius of the energy aura around the caster.
	ShieldRadius float32 = 95
	// shieldBreakTime is how long the shatter animation lasts after breaking.
	shieldBreakTime float32 = 0.45

	// ShieldCastHeal e a fracao da vida MAXIMA que a Paladina recupera ao
	// conjurar o Escudo Sagrado (Gui, 27/08/2026).
	//
	// Muda o que a habilidade E. Antes o escudo so adiava dano, entao lanca-lo
	// com a vida cheia ou com a vida no fim dava exatamente a mesma coisa —
	// e a Paladina nao tinha nenhuma forma propria de se recuperar sem a
	// Sacerdotisa por perto. Com a cura no lancamento, o Q vira a decisao de
	// segurar a linha: 20 de vida a cada 6 segundos de recarga.
	//
	// Fracao e nao valor fixo porque a vida maxima e do PERSONAGEM: se um dia
	// a Paladina tiver mais vida que os outros, a cura acompanha sem ninguem
	// lembrar de reajustar um numero solto.
	ShieldCastHeal float32 = 0.20

	// ShieldBreakDamage e o dano que o escudo cobra AO ESTOURAR, de todo
	// inimigo encostado na Paladina.
	//
	// A regua e o lobo: 80% dos 40 de vida dele (Gui, 27/08/2026). O numero
	// diz o que o estouro pode e o que nao pode — ele nao limpa a matilha
	// sozinho, ele abre espaco e termina quem ja estava ferido. Quebrar o
	// escudo deixa de ser so uma perda.
	//
	// Literal em vez de ler `entity.GetEnemyDef(EnemyTypeFast).Health`: o
	// pacote skill nao conhece entity, e criar essa dependencia por causa de
	// uma constante custa mais do que este comentario. Se a vida do lobo
	// mudar, o teste irmao em host_shield_test.go avisa.
	ShieldBreakDamage float32 = 32
	// ShieldBreakRadius e o alcance do estouro: a propria borda da aura, para
	// o jogador mirar a explosao pelo que ve na tela e nao por um numero
	// invisivel.
	ShieldBreakRadius float32 = ShieldRadius
	// ShieldBreakKnockback e quanto o estouro joga cada inimigo para tras
	// (Gui, 27/08/2026).
	//
	// O empurrao e o que faz o estouro valer mesmo quando nao mata: 32 de dano
	// nao tira um orc de cima da Paladina, 150 px tiram. Ela quebra o escudo
	// cercada e ganha o passo que precisava para relancar — e e por isso que o
	// numero e MAIOR que o raio da aura (95): quem estava dentro do circulo
	// termina fora dele.
	//
	// Nao e maior ainda porque o estouro nao pode virar reposicionamento: uma
	// distancia que jogasse a horda para o outro lado da tela transformaria
	// perder o escudo em ferramenta de controle, e perder o escudo continua
	// sendo perder o escudo.
	ShieldBreakKnockback float32 = 150
)

// Shield is the Paladina's protective energy. It has no duration: it persists,
// following its owner, until it absorbs ShieldMaxHP of damage. All visuals are
// procedural (rings, arcs, orbiting motes) — no sprites.
type Shield struct {
	OwnerID  string
	Position rl.Vector2 // anchor, synced to the owner's position every frame
	HP       float32
	Max      float32
	// Time accumulates for rotation/pulse animation.
	Time float32
	// HitFlash briefly lights the aura up when the shield absorbs damage.
	HitFlash float32
	// Breaking is > 0 while the shatter animation plays (counts down).
	Breaking float32
	// Sparks holds impact/shatter particles.
	Sparks *ParticleEmitter
}

// NewShield creates a full shield anchored at pos.
func NewShield(ownerID string, pos rl.Vector2) *Shield {
	return &Shield{
		OwnerID:  ownerID,
		Position: pos,
		HP:       ShieldMaxHP,
		Max:      ShieldMaxHP,
		Sparks:   NewParticleEmitter(),
	}
}

// Regenerate restores the shield to full strength (recast behavior: the value
// is reset to max, never summed).
func (s *Shield) Regenerate() {
	s.HP = s.Max
	s.Breaking = 0
}

// Absorb applies dmg to the shield and returns the damage that leaks through
// to the owner plus whether this hit broke the shield.
func (s *Shield) Absorb(dmg float32) (leftover float32, broken bool) {
	if s.HP <= 0 {
		return dmg, false
	}
	s.HitFlash = 0.25
	if dmg < s.HP {
		s.HP -= dmg
		s.emitImpactSparks(8)
		return 0, false
	}
	leftover = dmg - s.HP
	s.HP = 0
	s.Breaking = shieldBreakTime
	s.emitShatter()
	return leftover, true
}

// Broken reports whether the shield has no strength left.
func (s *Shield) Broken() bool { return s.HP <= 0 }

// Finished reports whether the shield (including its shatter animation) is
// done and can be removed from the manager.
func (s *Shield) Finished() bool {
	return s.HP <= 0 && s.Breaking <= 0
}

// Update advances animation timers and particles.
func (s *Shield) Update(dt float32) {
	s.Time += dt
	if s.HitFlash > 0 {
		s.HitFlash -= dt
	}
	if s.HP <= 0 && s.Breaking > 0 {
		s.Breaking -= dt
	}
	s.Sparks.Update(dt)
}

// emitImpactSparks bursts a few golden motes when a hit is absorbed.
func (s *Shield) emitImpactSparks(n int) {
	s.Sparks.Burst(s.Position, n, 120, 260, 0.35, 10, rl.NewColor(255, 226, 120, 255))
}

// emitShatter bursts the aura outward when the shield breaks.
func (s *Shield) emitShatter() {
	s.Sparks.Burst(s.Position, 34, 180, 420, 0.5, 14, rl.NewColor(255, 236, 150, 255))
	s.Sparks.Burst(s.Position, 18, 90, 220, 0.45, 10, rl.NewColor(170, 210, 255, 255))
}

// Draw renders the holy energy surrounding the owner: a soft golden glow, a
// double rotating rune-arc ring, and orbiting light motes. Opacity scales with
// remaining shield strength so the protection visibly weakens.
func (s *Shield) Draw() {
	rl.BeginBlendMode(rl.BlendAdditive)
	defer rl.EndBlendMode()

	// Shatter phase: expanding fading ring + sparks only.
	if s.HP <= 0 {
		if s.Breaking > 0 {
			prog := 1 - clamp01(s.Breaking/shieldBreakTime)
			r := ShieldRadius * (1 + prog*0.7)
			a := uint8(200 * (1 - prog))
			rl.DrawRing(s.Position, r-4, r+4, 0, 360, 48, rl.NewColor(255, 230, 140, a))
		}
		s.drawSparks()
		return
	}

	ratio := clamp01(s.HP / s.Max)
	// Base intensity: never fully dim while active; flashes on absorbed hits.
	intensity := 0.35 + 0.65*ratio
	if s.HitFlash > 0 {
		intensity += 0.6 * (s.HitFlash / 0.25)
	}
	pulse := float32(math.Sin(float64(s.Time)*3.2))*0.06 + 1

	gold := rl.NewColor(255, 214, 96, 255)
	white := rl.NewColor(255, 250, 220, 255)

	// Soft inner glow hugging the character.
	rl.DrawCircleGradient(int32(s.Position.X), int32(s.Position.Y),
		ShieldRadius*0.9*pulse,
		rl.Fade(gold, 0.20*intensity), rl.Blank)

	// Outer boundary ring (the "bubble" edge).
	r := ShieldRadius * pulse
	rl.DrawRing(s.Position, r-2.5, r+2.5, 0, 360, 56, rl.Fade(gold, 0.55*intensity))

	// Two counter-rotating rune arcs give the energy a living, sacred motion.
	spin := s.Time * 70
	for i := 0; i < 3; i++ {
		start := spin + float32(i)*120
		rl.DrawRing(s.Position, r-7, r-3, start, start+52, 24, rl.Fade(white, 0.7*intensity))
	}
	spin2 := -s.Time * 45
	for i := 0; i < 3; i++ {
		start := spin2 + float32(i)*120 + 60
		rl.DrawRing(s.Position, r+4, r+7, start, start+38, 24, rl.Fade(gold, 0.5*intensity))
	}

	// Orbiting light motes rising around the owner.
	for i := 0; i < 4; i++ {
		ang := float64(s.Time)*1.6 + float64(i)*math.Pi/2
		ox := float32(math.Cos(ang)) * r * 0.92
		oy := float32(math.Sin(ang)) * r * 0.92 * 0.45 // squashed = orbit feel
		bob := float32(math.Sin(float64(s.Time)*2.4+float64(i))) * 10
		p := rl.NewVector2(s.Position.X+ox, s.Position.Y+oy+bob)
		rl.DrawCircleGradient(int32(p.X), int32(p.Y), 9, rl.Fade(white, 0.8*intensity), rl.Blank)
	}

	s.drawSparks()
}

// drawSparks renders impact/shatter particles (already inside additive blend).
func (s *Shield) drawSparks() {
	for _, p := range s.Sparks.particles {
		drawParticle(p)
	}
}
