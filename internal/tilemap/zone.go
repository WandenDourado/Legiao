package tilemap

// Zonas: retangulos nomeados que dizem ONDE uma regra vale.
//
// O mapa 3 e uma travessia de territorio, e uma regra de territorio precisa de
// uma area. Marcador de ponto nao serve: `enemy_post_*` diz onde a guarnicao
// nasce, mas nao ate onde ela persegue, e a fortaleza nao e um ponto — e o
// lugar em que o grupo inteiro precisa estar para o climax armar.
//
// Como o portal, tudo vive no arquivo do mapa: `tier` e `climax_gate` sao
// propriedades do Tiled, nao uma tabela em codigo. Zona nova e um retangulo no
// editor e nenhuma recompilacao.

import rl "github.com/gen2brain/raylib-go/raylib"

// zoneLayerName is the object layer that holds named rectangles.
const zoneLayerName = "zones"

// Zone is one named rectangle of the map.
type Zone struct {
	Name string
	Rect rl.Rectangle
	// Tier is the difficulty step of a territory, 1 and up from the entrance
	// to the objective. Zero for zones that are not territories.
	Tier int
	// ClimaxGate marks the area every living player must stand in for the
	// stage's climax to arm.
	ClimaxGate bool
	// GuardRadius overrides how far a post inside this territory notices a
	// trespasser, in world pixels. Zero means "use the default".
	//
	// E do TERRITORIO e nao uma constante porque a escala e do mapa. A faixa
	// do mapa 3 tem 7680 px de largura; a do saguao do mapa 4 tem 3328 e o
	// grupo a atravessa pelo lado CURTO. O mesmo raio se comporta de um jeito
	// numa e de outro na outra.
	GuardRadius int
	// ChaseSlack is how far outside this territory a chase ALREADY UNDER WAY
	// may continue, in world pixels. Zero keeps the old behaviour: crossing the
	// sector line drops the target on the spot.
	//
	// O mapa 3 quer zero — atravessar a barricada quebra a perseguicao, e essa
	// e a regra da fase. Um corredor aberto nao tem barricada nenhuma: la o
	// jogador sai da faixa so por estar andando, e o guarda o larga no meio da
	// briga, que foi exatamente o relato.
	ChaseSlack int
	// CorridorCheckpoint marca o trecho que o grupo precisa ter alcancado
	// (vivo OU morto — a posicao de um corpo conta) para o ultimo suspiro do
	// mapa poder armar. E o mapa 6: nao ha corrida de hordas ali, so os dois
	// canhoes disparando o corredor inteiro, entao `partyIsFalling` nao tem o
	// `WaveState.Total > 0` de sempre para saber que a luta comecou. Este
	// retangulo faz esse papel: sem ele a cena tocaria no primeiro quadro em
	// que alguem nascesse com pouca vida, mesmo parado no portal de entrada.
	CorridorCheckpoint bool
	// ArenaLock marks an arena that seals its entrance after a player crosses
	// into it. The same zone supplies the northern exit-gate area that opens
	// when the map's final wave is cleared.
	ArenaLock bool
}

// Zones returns every named rectangle in the map's zone layer.
//
// A zone with zero width or height is skipped: it is a point somebody dropped
// in the wrong layer, and an empty rectangle contains nothing, so it would
// silently never trigger.
func Zones(m *TiledMap) []Zone {
	var zones []Zone
	for _, layer := range m.Layers {
		if layer.Type != "objectgroup" || layer.Name != zoneLayerName {
			continue
		}
		for _, object := range layer.Objects {
			if object.Width <= 0 || object.Height <= 0 {
				continue
			}
			zones = append(zones, Zone{
				Name:               object.Name,
				Rect:               rl.NewRectangle(object.X, object.Y, object.Width, object.Height),
				Tier:               object.IntProperty("tier"),
				ClimaxGate:         object.BoolProperty("climax_gate"),
				GuardRadius:        object.IntProperty("guard_radius"),
				ChaseSlack:         object.IntProperty("chase_slack"),
				CorridorCheckpoint: object.BoolProperty("corridor_checkpoint"),
				ArenaLock:          object.BoolProperty("arena_lock"),
			})
		}
	}
	return zones
}

// ArenaLockZone returns the arena that seals its entrance after the party
// enters it. A map has at most one such arena.
func ArenaLockZone(zones []Zone) (Zone, bool) {
	for _, z := range zones {
		if z.ArenaLock {
			return z, true
		}
	}
	return Zone{}, false
}

// Contains reports whether a world point is inside the zone.
func (z Zone) Contains(p rl.Vector2) bool {
	return p.X >= z.Rect.X && p.X < z.Rect.X+z.Rect.Width &&
		p.Y >= z.Rect.Y && p.Y < z.Rect.Y+z.Rect.Height
}

// ZoneAt returns the territory containing a point, and whether there is one.
//
// Territories do not overlap by construction (they are horizontal bands), so
// the first hit is the answer. A point outside every territory — the wall, the
// world edge — has none, and the caller decides what that means.
func ZoneAt(zones []Zone, p rl.Vector2) (Zone, bool) {
	for _, z := range zones {
		if z.Tier > 0 && z.Contains(p) {
			return z, true
		}
	}
	return Zone{}, false
}

// ClimaxGateZone returns the area that gates the climax, if the map has one.
func ClimaxGateZone(zones []Zone) (Zone, bool) {
	for _, z := range zones {
		if z.ClimaxGate {
			return z, true
		}
	}
	return Zone{}, false
}

// CorridorCheckpointZone returns the area a gauntlet map declares as "reached
// far enough", if it has one. See the CorridorCheckpoint field.
func CorridorCheckpointZone(zones []Zone) (Zone, bool) {
	for _, z := range zones {
		if z.CorridorCheckpoint {
			return z, true
		}
	}
	return Zone{}, false
}
