package skill

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Sanctuary constants tune the Priestess (Sacerdotisa) Sanctuary skill.
const (
	// SanctuaryRadius is the world-space radius of the healing area.
	SanctuaryRadius float32 = 200
	// SanctuaryDuration is how long the sanctuary exists before fading out.
	SanctuaryDuration float32 = 7.0
	// SanctuaryFade is how long the graceful dissipation takes at the end.
	SanctuaryFade float32 = 1.5
	// SanctuaryHealPerSec is the continuous HP restored to allies inside the area.
	SanctuaryHealPerSec float32 = 12
	// SanctuaryCooldown is the time the caster must wait between casts.
	SanctuaryCooldown float32 = 14.0

	// SanctuarySpeedBonus e quanto a Sacerdotisa anda mais rapido enquanto o
	// Santuario dura (Gui, 31/08/2026).
	//
	// Entra junto com a aura movel e faz parte da mesma decisao: o Santuario
	// deixou de ser um lugar no chao para ser um estado DELA. Se a cura anda
	// com ela, quem decide onde a cura acontece e o passo dela — e um passo
	// 25% mais rapido e o que transforma "a magia me segue" em "eu levo a
	// magia", que e a diferenca entre reposicionar o grupo e correr atras
	// dele. O bonus vale so para a conjuradora: aliados ganham a cura, nao a
	// mobilidade, senao a habilidade vira ferramenta de deslocamento do time.
	SanctuarySpeedBonus float32 = 0.25

	// SanctuaryGroundDrop e quanto o centro visual do chao desce em relacao
	// ao centro da sprite. A sprite tem ~220 px de altura ancorados no meio,
	// entao os pes ficam bem abaixo do ponto que o jogo chama de "posicao": a
	// aura desenhada no centro exato pareceria presa na cintura dela.
	//
	// So o DESENHO desce. A cura mede a partir da posicao real (Contains), que
	// e a mesma que o host usa para todo mundo.
	SanctuaryGroundDrop float32 = 56

	// SanctuaryVisualRadius e o alcance visual INTEIRO, e e o raio que o
	// culling usa (doc/skill_performance.md, Regra 2): o anel pulsante e os
	// raios saem alem da borda de cura, entao cullar pelo raio de cura
	// cortaria a magia enquanto ainda havia luz dela na tela.
	SanctuaryVisualRadius float32 = SanctuaryRadius * 1.25
)

// Sanctuary is the Sacerdotisa's mobile holy aura. It has no sprite: the
// consecrated ground, the rotating light rings and the floating sacred motes
// are all drawn with raylib primitives.
//
// AURA MOVEL, NAO ZONA NO CHAO (Gui, 31/08/2026). Ate aqui o Santuario era um
// lugar: a Sacerdotisa o largava a frente dela e quem quisesse a cura ia ate
// o circulo. Isso punha a classe curandeira em conflito com a propria magia —
// para continuar curando ela tinha de ficar parada num ponto que a luta ja
// tinha deixado para tras. Agora a aura e ancorada NELA todo quadro, entao
// curar deixou de custar posicao.
//
// Por isso a colecao passou a ser indexada por dono (sanctuary_manager.go):
// uma Sacerdotisa tem UMA aura, e relancar renova em vez de empilhar.
type Sanctuary struct {
	ID       string
	OwnerID  string
	Position rl.Vector2
	// Age is the elapsed time since the sanctuary was created.
	Age float32
	// Dead becomes true once the sanctuary has fully dissipated.
	Dead bool
	// HealTick accumulates dt so healing is applied in steady increments.
	HealAccum float32
}

// NewSanctuary creates a sanctuary anchored at center (world-space).
func NewSanctuary(ownerID string, center rl.Vector2) *Sanctuary {
	return &Sanctuary{
		ID:       ownerID,
		OwnerID:  ownerID,
		Position: center,
		Age:      0,
		Dead:     false,
	}
}

// Refresh restarts the aura's life (recast behavior: renova, nunca empilha).
func (s *Sanctuary) Refresh(center rl.Vector2) {
	s.Position = center
	s.Age = 0
	s.HealAccum = 0
	s.Dead = false
}

// LifeRatio returns 0..1 progress through the sanctuary's active lifespan
// (excluding fade). Useful for ramps/pulses that should peak mid-life.
func (s *Sanctuary) LifeRatio() float32 {
	if SanctuaryDuration <= 0 {
		return 1
	}
	r := s.Age / SanctuaryDuration
	if r > 1 {
		return 1
	}
	return r
}

// FadeAlpha returns the current global opacity multiplier for the sanctuary.
// It is 1 while active, then linearly ramps 1..0 during the fade window.
func (s *Sanctuary) FadeAlpha() float32 {
	if s.Age <= SanctuaryDuration {
		return 1
	}
	over := s.Age - SanctuaryDuration
	a := 1 - over/SanctuaryFade
	if a < 0 {
		return 0
	}
	return a
}

// IsHealing reports whether the sanctuary is still in its active window —
// a janela em que ela cura E em que o bonus de velocidade vale. O fade final
// e so desenho: a habilidade ja acabou quando ele comeca.
func (s *Sanctuary) IsHealing() bool {
	return s.Age <= SanctuaryDuration
}

// GroundCenter e onde a aura e DESENHADA: o mesmo ponto, deslocado para os
// pes de quem a carrega. Ver SanctuaryGroundDrop.
func (s *Sanctuary) GroundCenter() rl.Vector2 {
	return rl.NewVector2(s.Position.X, s.Position.Y+SanctuaryGroundDrop)
}

// Contains reports whether world point p lies inside the sanctuary's radius.
func (s *Sanctuary) Contains(p rl.Vector2) bool {
	return rl.Vector2Distance(s.Position, p) <= SanctuaryRadius
}
