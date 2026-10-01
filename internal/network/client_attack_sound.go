package network

// O SOM DO ATAQUE BASICO NO CLIENTE.
//
// As magias tem um evento de rede proprio, entao o cliente sabe a hora exata
// em que uma foi lancada e o som sai de dentro do `internal/skill`, no mesmo
// codigo que host e cliente executam. O ataque basico dos quatro personagens
// de projetil nao tem: o projetil simplesmente APARECE no instantaneo
// (ProjectileUpdatePayload) e depois some dele.
//
// Entao o "alguem atacou" e deduzido aqui — um id de projetil que nao estava
// no instantaneo anterior e um disparo novo. Continua sem custar rede: o
// instantaneo ja vinha assim, e a alternativa (uma mensagem por golpe) seria
// a mais cara de todas, porque ataque basico e o que mais acontece na partida.
//
// A Paladina nao passa por aqui: o golpe dela ja tem evento proprio
// (MsgMelee), e o som sai de skill.NewSwordSweep, nos dois lados de uma vez.

import (
	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// announceNewProjectiles toca o ataque de cada projetil que apareceu agora.
// prev e o instantaneo anterior; next, o que acabou de chegar.
//
// prev nil significa "primeiro instantaneo desta sessao" (inicio, troca de
// mapa, reconexao): tudo que esta no ar seria novidade, e o grupo inteiro
// disparando de uma vez nao e o que aconteceu — e o que o cliente perdeu
// enquanto estava fora. Nesse caso nao soa nada.
func announceNewProjectiles(prev map[string]ProjectileState, next []ProjectileState) {
	if prev == nil {
		return
	}
	for _, ps := range next {
		if _, seen := prev[ps.ProjectileID]; seen {
			continue
		}
		playAttackSoundFor(ps.OwnerID, rl.NewVector2(float32(ps.X), float32(ps.Y)))
	}
}

// playAttackSoundFor descobre a classe do dono no instantaneo de jogadores —
// que o cliente ja mantem para desenhar — e deixa o personagem escolher o
// proprio som. Sem dono conhecido nao ha classe, e sem classe nao ha som.
func playAttackSoundFor(ownerID string, pos rl.Vector2) {
	RemotePlayersMutex.Lock()
	p, ok := RemotePlayers[ownerID]
	RemotePlayersMutex.Unlock()
	if !ok {
		return
	}
	entity.PlayAttackSound(entity.CharacterType(p.Character), pos)
}
