package network

// Onde estao as gargulas vivas, para a seta de borda de tela
// (game/sentry_markers.go, 27/09/2026). O desenho e o mesmo nas duas pontas;
// so a FONTE muda: o host tem as criaturas no EntityManager, o cliente tem o
// espelho interpolado — o mesmo que ele acabou de desenhar no quadro.

import (
	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// LivingSentryCenters acrescenta a dst o centro VISIVEL de cada gargula viva
// (o pe mais o HitOffsetY da EnemyDef, a mesma conta de Enemy.HitCenter) e o
// devolve. dst e reaproveitado pelo chamador para nao alocar por quadro.
//
// So gargula ACORDADA (SentriesAwake, 28/09/2026): no mapa 4 as torres estao
// em campo desde a chegada mas caladas ate "As sentinelas despertaram", e uma
// seta apontando para uma torre que ainda nao atira so assustaria o grupo
// antes da hora.
func LivingSentryCenters(dst []rl.Vector2) []rl.Vector2 {
	if !SentriesAwake() {
		return dst
	}
	off := entity.GetEnemyDef(entity.EnemyTypeCastleSentry).HitOffsetY
	if Role == "host" && CurrentHost != nil {
		for _, e := range CurrentHost.EntityManager.GetAllEnemies() {
			if e != nil && e.IsActive && e.Type == entity.EnemyTypeCastleSentry {
				dst = append(dst, e.HitCenter())
			}
		}
		return dst
	}
	for _, e := range InterpolatedEnemies() {
		if e.Type == string(entity.EnemyTypeCastleSentry) {
			dst = append(dst, rl.NewVector2(float32(e.X), float32(e.Y)+off))
		}
	}
	return dst
}
