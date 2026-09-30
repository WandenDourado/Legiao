package network

// Item 6 (doc/plan_correcoes_mobile_e_rede.md): a nevoa e os espinhoes da
// chefe (map 7) so existiam no host. Estes testes cobrem a metade do host do
// conserto — que os eventos certos saem, com o conteudo certo — irmas dos
// testes de broadcastSentryOrb/broadcastCannonBall que ja existiam.

import (
	"encoding/json"
	"testing"

	"github.com/WandenDourado/Legiao/internal/skill"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// newBroadcastTestHost monta um Host minimo, so com o que broadcastBossFog e
// broadcastBossThorn (e o que os chama) precisam: peers para o broadcast
// escrever, e um Manager de skills para a nevoa viver.
func newBroadcastTestHost(t *testing.T) (*Host, *ClientConn, <-chan Message) {
	t.Helper()
	c, received := newRecordingClientConn(t)
	h := &Host{
		peers:   map[string]*ClientConn{"peer": c},
		players: map[string]*PlayerState{},
		Skills:  skill.NewManager(),
	}
	return h, c, received
}

func firstOfType(t *testing.T, c *ClientConn, received <-chan Message, want MessageType) (Message, bool) {
	t.Helper()
	c.conn.Close()
	for msg := range received {
		if msg.Type == want {
			return msg, true
		}
	}
	return Message{}, false
}

// TestBroadcastBossFogCastCarriesBounds prova que o "cast" leva os limites
// que o cliente precisa para criar a MESMA nevoa que o host acabou de
// conjurar (skill.ActivateDarkFog do lado de la, no handler de client.go).
func TestBroadcastBossFogCastCarriesBounds(t *testing.T) {
	h, c, received := newBroadcastTestHost(t)
	bounds := rl.NewRectangle(10, 20, 3000, 1600)

	h.broadcastBossFog("cast", bounds)

	msg, found := firstOfType(t, c, received, MsgBossFog)
	if !found {
		t.Fatal("MsgBossFog nao foi transmitido")
	}
	var bf BossFogPayload
	if err := json.Unmarshal(msg.Payload, &bf); err != nil {
		t.Fatalf("payload nao decodifica: %v", err)
	}
	if bf.EventType != "cast" || bf.X != 10 || bf.Y != 20 || bf.W != 3000 || bf.H != 1600 {
		t.Errorf("payload = %+v, esperava cast com os limites da arena", bf)
	}
}

// TestResolveFogBroadcastsExpireWhenTTLRunsOut prova a outra metade: quando
// AdvanceFog derruba a nevoa por tempo (e nao por F5/portal, que ja limpam
// direto via ClearBossEffects), resolveFog avisa os clientes.
func TestResolveFogBroadcastsExpireWhenTTLRunsOut(t *testing.T) {
	h, c, received := newBroadcastTestHost(t)
	skill.ActivateDarkFog(h.Skills, rl.NewRectangle(0, 0, 100, 100))

	// dt maior que DarkFogDuration: a nevoa tem que acabar neste quadro.
	h.resolveFog(skill.DarkFogDuration + 1)

	msg, found := firstOfType(t, c, received, MsgBossFog)
	if !found {
		t.Fatal("MsgBossFog (expire) nao foi transmitido quando a nevoa esgotou o TTL")
	}
	var bf BossFogPayload
	if err := json.Unmarshal(msg.Payload, &bf); err != nil {
		t.Fatalf("payload nao decodifica: %v", err)
	}
	if bf.EventType != "expire" {
		t.Errorf("EventType = %q, esperava \"expire\"", bf.EventType)
	}
	if h.Skills.FogActive() {
		t.Error("a nevoa deveria ter sido derrubada no host tambem")
	}
}

// TestResolveFogStaysQuietWhileFogIsStillUp garante que resolveFog nao avisa
// nada enquanto a nevoa continua no ar — um evento por quadro entulharia a
// rede do mesmo jeito que a nevoa ja evita para o dano (ver applyBossDamage,
// `silent`).
func TestResolveFogStaysQuietWhileFogIsStillUp(t *testing.T) {
	h, c, received := newBroadcastTestHost(t)
	skill.ActivateDarkFog(h.Skills, rl.NewRectangle(0, 0, 100, 100))

	h.resolveFog(0.1)

	if _, found := firstOfType(t, c, received, MsgBossFog); found {
		t.Error("nenhum MsgBossFog deveria ter saido com a nevoa ainda no ar")
	}
}

// TestBroadcastBossThornCarriesEveryPoint prova que a rajada inteira viaja
// numa unica mensagem: a foto que castThorns tira do host tem que chegar
// inteira, ou a rajada nasceria espalhada por varios quadros de rede no
// cliente em vez de ler como um golpe so.
func TestBroadcastBossThornCarriesEveryPoint(t *testing.T) {
	h, c, received := newBroadcastTestHost(t)
	centers := []rl.Vector2{
		rl.NewVector2(100, 200),
		rl.NewVector2(300, 400),
		rl.NewVector2(500, 600),
	}

	h.broadcastBossThorn(centers)

	msg, found := firstOfType(t, c, received, MsgBossThorn)
	if !found {
		t.Fatal("MsgBossThorn nao foi transmitido")
	}
	var bt BossThornPayload
	if err := json.Unmarshal(msg.Payload, &bt); err != nil {
		t.Fatalf("payload nao decodifica: %v", err)
	}
	if len(bt.Points) != len(centers) {
		t.Fatalf("payload trouxe %d pontos, esperava %d", len(bt.Points), len(centers))
	}
	for i, c := range centers {
		if bt.Points[i].X != int(c.X) || bt.Points[i].Y != int(c.Y) {
			t.Errorf("ponto %d = %+v, esperava (%.0f,%.0f)", i, bt.Points[i], c.X, c.Y)
		}
	}
}
