package network

import (
	"sync"
	"time"

	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/skill"
)

// MOVIMENTO E PREVISTO NA MAQUINA DO JOGADOR, entao o bonus de velocidade
// tambem tem de ser (Gui, 31/08/2026).
//
// O jogador local anda em game/loop.go, no proprio quadro dele, e so publica
// onde parou — nem o host reescreve esse passo. Um multiplicador que so
// existisse no host nao mudaria nada do que a Sacerdotisa sente no teclado.
//
// E o bonus nao precisa de mensagem nova para chegar aqui: a aura ja e
// replicada no lancamento e envelhece com o mesmo relogio nos dois lados,
// entao cada maquina DERIVA o multiplicador do estado que ja tem
// (doc/skill_performance.md, Regra 6 — nao transmita o que da para derivar).
// O preco e o de sempre neste jogo: o cliente comeca a correr no quadro em que
// a mensagem de lancamento chega, nao no quadro em que apertou a tecla.

// LocalSkills devolve o Manager que ESTA maquina usa para as magias: o do host
// quando ela hospeda, o espelho do cliente quando ela e cliente.
func LocalSkills() *skill.Manager {
	if Role == "host" && CurrentHost != nil {
		return CurrentHost.Skills
	}
	return ClientSkills
}

// MoveSpeedMultiplier devolve o multiplicador de velocidade de playerID
// somando todo efeito que mexe no passo dele: a aura do Santuario e o folego
// do renascer (ArmLocalRespawnBoost). Este e o ponto onde o proximo entra sem
// que game/loop.go precise saber que ele existe.
func MoveSpeedMultiplier(playerID string) float32 {
	m := LocalSkills()
	if m == nil || playerID == "" {
		return 1
	}
	mult := m.SanctuaryMoveMultiplier(playerID)
	if playerID == LocalPlayerID && localRespawnBoostActive() {
		mult *= entity.RespawnGraceSpeed
	}
	return mult
}

// O FOLEGO DO RENASCER, metade da velocidade (playtest de 27/09/2026,
// plans/plan_playtest_27_09.md §4). A imunidade e do host
// (host_respawn_grace.go); o passo, como o do Santuario, tem de ser de quem
// move o corpo. E ele tambem nao precisa de mensagem: a borda morto -> vivo
// ja chega a toda maquina (game.SyncLocalPlayer), e e ali que ele e armado.
//
// Prazo em relogio de parede e nao em segundos a envelhecer: assim ninguem
// precisa lembrar de chamar um tick, e game/loop.go continua so orquestrando.
var (
	respawnBoostMu    sync.Mutex
	respawnBoostUntil time.Time
)

// ArmLocalRespawnBoost liga o passo extra do jogador local por
// entity.RespawnGrace segundos. Chamado na borda em que ele levanta.
func ArmLocalRespawnBoost() {
	respawnBoostMu.Lock()
	respawnBoostUntil = time.Now().Add(time.Duration(entity.RespawnGrace * float32(time.Second)))
	respawnBoostMu.Unlock()
}

// ClearLocalRespawnBoost desliga o passo extra. Chamado pelo reinicio de
// fase (RequestLocalReset), para uma tentativa nova nao comecar embalada.
func ClearLocalRespawnBoost() {
	respawnBoostMu.Lock()
	respawnBoostUntil = time.Time{}
	respawnBoostMu.Unlock()
}

func localRespawnBoostActive() bool {
	respawnBoostMu.Lock()
	defer respawnBoostMu.Unlock()
	return time.Now().Before(respawnBoostUntil)
}
