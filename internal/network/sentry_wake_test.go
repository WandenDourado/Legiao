package network

// A porta de despertar das gargulas: elas so atiram depois de o grupo alcancar
// o degrau de territorio que a fase declara. Ver sentry_wake.go.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/tilemap"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// castleZones e uma fatia de territorio parecida com a do mapa 4: degraus 1 a
// 3 em bandas horizontais, o grupo subindo do 1 para o 3.
func castleZones() []tilemap.Zone {
	return []tilemap.Zone{
		{Name: "vestibulo", Tier: 1, Rect: rl.NewRectangle(0, 2000, 1000, 1000)},
		{Name: "corredor", Tier: 2, Rect: rl.NewRectangle(0, 1000, 1000, 1000)},
		{Name: "saguao", Tier: 3, Rect: rl.NewRectangle(0, 0, 1000, 1000)},
	}
}

func castleHost(players map[string]*PlayerState) *Host {
	return &Host{
		stageMap:   "assets/maps/world_04.json",
		stageZones: castleZones(),
		players:    players,
	}
}

func TestSentriesHoldFireUntilTheDeclaredTier(t *testing.T) {
	h := castleHost(map[string]*PlayerState{
		// No vestibulo (degrau 1): longe do degrau 3 que a fase declara.
		"p1": {PlayerID: "p1", X: 500, Y: 2500},
	})
	if h.sentriesMayFire() {
		t.Error("as gargulas atiraram com o grupo ainda no vestibulo")
	}
}

func TestSentriesWakeWhenThePartyReachesTheTier(t *testing.T) {
	h := castleHost(map[string]*PlayerState{
		"p1": {PlayerID: "p1", X: 500, Y: 500}, // saguao, degrau 3
	})
	// Um grupo de um: esse um E a maioria.
	if !h.sentriesMayFire() {
		t.Fatal("o grupo alcancou o saguao e as gargulas continuaram caladas")
	}

	// E NAO VOLTAM A DORMIR: recuar nao pode ser uma forma de desligar a fase.
	h.players["p1"].X, h.players["p1"].Y = 500, 2500
	if !h.sentriesMayFire() {
		t.Error("o grupo recuou e as gargulas voltaram a dormir")
	}
}

// UM BATEDOR NAO ACORDA AS TORRES. Era o relato do Gui (27/08/2026): um bot
// avanca lutando ate 700 px a frente da festa, cruzava a boca do saguao e as
// gargulas abriam fogo sobre quem ainda subia o corredor.
func TestOneScoutAheadDoesNotWakeTheSentries(t *testing.T) {
	h := castleHost(map[string]*PlayerState{
		"batedor": {PlayerID: "batedor", X: 500, Y: 500}, // saguao, degrau 3
		"p2":      {PlayerID: "p2", X: 500, Y: 1500},     // corredor
		"p3":      {PlayerID: "p3", X: 500, Y: 1500},     // corredor
		"p4":      {PlayerID: "p4", X: 500, Y: 2500},     // vestibulo
	})
	if h.sentriesMayFire() {
		t.Error("um batedor sozinho no saguao acordou as gargulas")
	}
}

// E UM RETARDATARIO NAO AS CALA. O contrapeso da regra da maioria: com
// "todos", um bot preso la atras deixaria a fase 4 sem fogo de longe, e sem
// ele a emboscada do saguao nao derruba o grupo e o climax nao fecha.
func TestOneStragglerDoesNotKeepTheSentriesAsleep(t *testing.T) {
	h := castleHost(map[string]*PlayerState{
		"p1":        {PlayerID: "p1", X: 500, Y: 500},         // saguao
		"p2":        {PlayerID: "p2", X: 600, Y: 700},         // saguao
		"p3":        {PlayerID: "p3", X: 700, Y: 900},         // saguao
		"retardado": {PlayerID: "retardado", X: 500, Y: 2500}, // vestibulo
	})
	if !h.sentriesMayFire() {
		t.Error("tres de quatro no saguao e as gargulas continuaram caladas")
	}
}

func TestADeadBodyUpAheadDoesNotWakeTheSentries(t *testing.T) {
	h := castleHost(map[string]*PlayerState{
		"vivo":  {PlayerID: "vivo", X: 500, Y: 2500},
		"morto": {PlayerID: "morto", X: 500, Y: 500, IsDead: true},
	})
	if h.sentriesMayFire() {
		t.Error("um corpo caido la na frente acordou as torres sozinho")
	}
}

func TestMapsWithoutAWakeTierFireFromTheFirstFrame(t *testing.T) {
	// O mapa 5 poe as gargulas em campo POR HORDA: a fase ja escolheu o
	// momento delas, e uma segunda porta em cima disso seria uma torre que
	// nasce e nao atira.
	h := &Host{
		stageMap:   "assets/maps/world_05.json",
		stageZones: castleZones(),
		players:    map[string]*PlayerState{"p1": {PlayerID: "p1", X: 500, Y: 2500}},
	}
	if !h.sentriesMayFire() {
		t.Error("um mapa sem degrau declarado nao pode segurar o fogo")
	}
}

// Um degrau declarado que o mapa nao tem seria uma porta que nunca abre — a
// fase inteira sem fogo de longe, em silencio. Prefira falhar aberto.
func TestADeclaredTierWithNoTerritoriesFiresAnyway(t *testing.T) {
	h := &Host{
		stageMap: "assets/maps/world_04.json",
		players:  map[string]*PlayerState{"p1": {PlayerID: "p1", X: 500, Y: 2500}},
	}
	if !h.sentriesMayFire() {
		t.Error("mapa sem territorio nenhum deixou as gargulas caladas para sempre")
	}
}
