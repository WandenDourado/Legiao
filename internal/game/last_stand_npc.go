package game

// O heroi que aparece quando ninguem no grupo joga com ele.
//
// QUEM aparece depende do mapa (network/last_stand_heroes.go): no mapa 2 e o
// Necromante, no 3 e a Sacerdotisa. Por isso o personagem vem junto da
// posicao, em vez de ser escrito aqui — desenhar o Necromante no mapa da
// Sacerdotisa seria um erro invisivel no codigo e gritante na tela.
//
// Ele e desenhado pelo MESMO caminho de um jogador remoto, e nao por um
// desenho proprio: e a mesma folha de sprite, a mesma escala e a mesma linha
// de contato com o chao, entao ele nao se denuncia como uma peca de outra
// natureza no meio da luta.
//
// Ele tambem nao e um jogador. Nao esta em h.players, entao nao conta no HUD
// de jogadores, nao pesa na checagem de Game Over e nao precisa morrer,
// ressuscitar nem ser sincronizado. Ele existe para ser dono de uma legiao e
// para ser visto; as duas coisas custam o que se ve aqui.

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"
)

// lastStandNPCFrame is the sprite frame the summoned hero is drawn on. They
// are a static figure: they arrive, cast, and are gone, so an idle pose reads
// better than an animation nobody triggers.
const (
	lastStandNPCFrame = 0
	lastStandNPCRow   = 0
)

// drawLastStandNPC draws the summoned hero when one is on the field.
func drawLastStandNPC() {
	pos, char, active := network.LastStandNPC()
	if !active || char == "" {
		return
	}
	def := entity.GetCharacter(char)
	tex := entity.SharedTexture(char)
	entity.DrawRemotePlayer(
		tex, tex.ID != 0, def,
		pos.X, pos.Y,
		lastStandNPCFrame, lastStandNPCRow,
		0,   // sem velocidade: ele nao anda, entao nao espelha
		"",  // sem cor de jogador; a arte dele ja e a identidade
		20,
		false,
	)
}
