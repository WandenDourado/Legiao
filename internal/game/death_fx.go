package game

// A ligacao da desintegracao dos monstros (entity/death_fx.go) com o quadro:
// quem esta vivo, em que mundo, e quando desenhar. A logica mora em entity;
// aqui so se responde "quem e a lista autoritativa neste papel de rede".

import (
	"github.com/WandenDourado/Legiao/internal/entity"
	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// beginEnemyDeathFX roda uma vez por quadro, ANTES de desenhar os inimigos:
// dispara a desintegracao de quem foi desenhado no quadro anterior e morreu
// desde entao.
//
// A lista de vivos e a COMPLETA, nunca a do culling: um monstro que so saiu
// da tela nao morreu. No host e o EntityManager; no cliente e o snapshot, que
// o host manda inteiro (host.go, BroadcastStateUpdate).
func beginEnemyDeathFX(w *World) {
	if !entity.DeathFXEnabled() {
		return
	}
	// Trocar de mapa ou reiniciar a fase esvazia a lista de uma vez, e nada
	// daquilo morreu.
	entity.SyncDeathFXWorld(w.Path, network.StageGeneration())
	if network.Role == "host" && network.CurrentHost != nil {
		entity.BeginDeathFXFrame(network.CurrentHost.EntityManager.EnemyAlive)
		return
	}
	alive := network.InterpolatedEnemies()
	entity.BeginDeathFXFrame(func(id string) bool {
		_, ok := alive[id]
		return ok
	})
}

// drawEnemyDeathFX avanca e desenha as mortes em curso. Dentro do bloco de
// camera, logo depois dos corpos: a poeira passa por cima de quem ainda esta
// vivo, e por baixo das magias e das barras.
func drawEnemyDeathFX(view rl.Rectangle) {
	entity.UpdateDeathFX(rl.GetFrameTime())
	entity.DrawDeathFX(view)
}
