package network

// Metade do cliente do mesmo conserto (item 6): que o handler de
// handleMessage realmente poe a nevoa e os espinhos no ar quando o evento do
// host chega, e os derruba quando o host manda "expire".

import (
	"encoding/json"
	"testing"

	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func resetClientBossReplicationState(t *testing.T) {
	t.Helper()
	ClientSkills = nil
	t.Cleanup(func() { ClientSkills = nil })
}

// TestClientBossFogCastPutsFogInTheAir prova o defeito relatado: sem este
// handler, um cliente via a chefe conjurar e nada acontecia na tela dele — o
// dano chegava pelos eventos de combate de sempre, sem nenhum aviso visual.
func TestClientBossFogCastPutsFogInTheAir(t *testing.T) {
	resetClientBossReplicationState(t)
	c := &Client{}

	payload, err := json.Marshal(BossFogPayload{EventType: "cast", X: 5, Y: 6, W: 200, H: 300})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	c.handleMessage(Message{Type: MsgBossFog, Payload: payload})

	if ClientSkills == nil || !ClientSkills.FogActive() {
		t.Fatal("a nevoa deveria estar no ar depois do evento \"cast\"")
	}
}

// TestClientBossFogExpireDropsFogImmediately prova a outra metade: o cliente
// nao espera a propria contagem (AdvanceFog) concordar, ele obedece o host na
// hora.
func TestClientBossFogExpireDropsFogImmediately(t *testing.T) {
	resetClientBossReplicationState(t)
	c := &Client{}
	ClientSkills = skill.NewManager()
	skill.ActivateDarkFog(ClientSkills, rl.NewRectangle(0, 0, 100, 100))

	payload, err := json.Marshal(BossFogPayload{EventType: "expire"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	c.handleMessage(Message{Type: MsgBossFog, Payload: payload})

	if ClientSkills.FogActive() {
		t.Error("a nevoa deveria ter sumido depois do evento \"expire\"")
	}
}

// TestClientBossThornCastSpawnsEveryPoint prova que a rajada inteira chega
// ao cliente: cada ponto que o host fotografou vira uma marca no chao dele.
func TestClientBossThornCastSpawnsEveryPoint(t *testing.T) {
	resetClientBossReplicationState(t)
	c := &Client{}

	payload, err := json.Marshal(BossThornPayload{Points: []BossThornPoint{
		{X: 111, Y: 222},
		{X: 333, Y: 444},
	}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	c.handleMessage(Message{Type: MsgBossThorn, Payload: payload})

	if ClientSkills == nil || len(ClientSkills.Thorns) != 2 {
		t.Fatalf("esperava 2 espinhos marcados, ClientSkills = %+v", ClientSkills)
	}
}
