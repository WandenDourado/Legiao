package game

// Stage skip (F8, desktop only): jumps straight to the next phase, without
// clearing the hordes and without walking to the portal. A development switch
// like F2 and F5 — there is no touch equivalent and none is wanted.
//
// Shift+F8 jumps straight to the LAST campaign map instead of the next one.
// Plain F8 only ever advances one phase at a time, so reaching a phase deep
// in the campaign (o corredor final, mapa 6) meant pressing it five times
// from the village every single playtest. Shift+F8 gets there in one press,
// from anywhere — including a map outside campaignMaps, where plain F8 does
// nothing at all.

import (
	"log"

	"github.com/WandenDourado/Legiao/internal/network"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// campaignMaps is the order the phases are meant to be played in.
//
// F8 walks this list instead of following the current map's `target_map`, e
// isso ja escondeu um defeito de fase inteira: o portal do world_02 apontava
// para o world_01 — quem terminava a mata voltava para a vila — e ninguem via,
// porque quem testava usava F8 e o F8 nao le o portal. Este comentario chegou a
// DESCREVER a divergencia como se fosse um fato da vida.
//
// As duas fontes agora sao amarradas por
// `campaign_portals_test.go`: cada portal da campanha tem de levar a fase
// seguinte desta lista, e a ultima volta ao comeco.
//
// Add a map here when it joins the campaign.
var campaignMaps = []string{
	"assets/maps/world_01.json",
	"assets/maps/world_02.json",
	"assets/maps/world_03.json",
	"assets/maps/world_04.json",
	"assets/maps/world_05.json",
	"assets/maps/world_06.json",
	// A arena da Senhora das Trevas. O corredor do 6 e a aproximacao; aqui a
	// campanha termina, e a fase so acaba quando ela cai.
	"assets/maps/world_07.json",
}

// UpdateStageSkip advances to the next campaign map when F8 is pressed, or
// jumps straight to the last one when Shift+F8 is pressed.
//
// Like the portal, the skip takes the WHOLE PARTY: it announces the
// destination and the arrival is applied to every machine by
// ApplyPendingTravel. A local jump would leave the host simulating a map its
// clients never left — which is exactly what this used to do.
//
// Only the host may skip, for the same reason only the host may restart the
// stage: it is the machine that owns the simulation. On a client the key does
// nothing.
func UpdateStageSkip(cfg Config, current *World) {
	if cfg.FullScreen || current == nil || network.Role == "client" {
		return
	}
	shiftHeld := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	if shiftHeld && rl.IsKeyPressed(rl.KeyF8) {
		jumpToLastCampaignMap(current)
		return
	}
	if !rl.IsKeyPressed(rl.KeyF8) {
		return
	}
	next, ok := nextCampaignMap(current.Path)
	if !ok {
		return
	}
	log.Printf("[F8] pulando fase: %s -> %s", current.Path, next)
	// Empty spawn name: arrive on the destination's own player_spawn. A skip
	// is not coming through a door, so there is no arrival marker to honour.
	network.StartTravel(next, "")
}

// jumpToLastCampaignMap sends the party straight to lastCampaignMap(),
// regardless of where current sits in (or outside) campaignMaps.
//
// UltimatesGrantedOn deriva o conjunto de supremas do INDICE da fase na
// lista, entao chegar direto no mapa 6 por aqui ja libera as quatro supremas
// que ele pede (Necromante, Sacerdotisa, Arqueiro, Mago) — o mesmo resultado
// de andar as cinco fases anteriores, sem precisar andar nenhuma.
func jumpToLastCampaignMap(current *World) {
	last := lastCampaignMap()
	if last == "" || current.Path == last {
		return
	}
	log.Printf("[Shift+F8] pulando direto para a ultima fase: %s -> %s", current.Path, last)
	network.StartTravel(last, "")
}

// lastCampaignMap is the final phase of the campaign, or "" if campaignMaps
// is somehow empty — a guard, not an expected case.
func lastCampaignMap() string {
	if len(campaignMaps) == 0 {
		return ""
	}
	return campaignMaps[len(campaignMaps)-1]
}

// A liberacao das supremas mora em progression.go e le esta mesma lista.

// nextCampaignMap is the map that follows path, or false when there is none.
// Both misses are worth a log line rather than silence, because a key that
// does nothing looks broken: the player is on the last phase, or on a map that
// is not part of the campaign at all (a terrain validation map, say).
func nextCampaignMap(path string) (string, bool) {
	for i, m := range campaignMaps {
		if m != path {
			continue
		}
		if i+1 >= len(campaignMaps) {
			log.Printf("[F8] %s e a ultima fase da campanha; nao ha para onde pular", path)
			return "", false
		}
		return campaignMaps[i+1], true
	}
	log.Printf("[F8] %s nao esta em campaignMaps; F8 nao sabe qual seria a proxima fase", path)
	return "", false
}
