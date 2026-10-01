package entity

import (
	"image/color"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Uma folha 2x1 quadros de 8x8: o quadro 0 tem so a metade esquerda opaca e
// vermelha; o quadro 1 esta vazio.
func testSheet() []color.RGBA {
	px := make([]color.RGBA, 16*8)
	for y := 0; y < 8; y++ {
		for x := 0; x < 4; x++ {
			px[y*16+x] = color.RGBA{200, 10, 10, 255}
		}
	}
	return px
}

func TestDeathMaskKeepsOnlyOpaqueCells(t *testing.T) {
	m := buildDeathSheetMask(testSheet(), 16, 8, 8, 8, 2, 1, 4)
	if m == nil || len(m.Fine) != 2 || len(m.Coarse) != 2 {
		t.Fatalf("mascara com formato errado: %+v", m)
	}
	if got := len(m.Fine[0]); got != 2 {
		t.Fatalf("quadro 0 deveria ter 2 celulas opacas (coluna esquerda), tem %d", got)
	}
	for _, c := range m.Fine[0] {
		if c.U != 0 {
			t.Errorf("celula opaca fora da metade esquerda: %+v", c)
		}
		if c.Col.R != 200 || c.Col.G != 10 {
			t.Errorf("cor media errada: %+v", c.Col)
		}
	}
	if len(m.Fine[1]) != 0 {
		t.Errorf("quadro vazio virou %d graos", len(m.Fine[1]))
	}
	// Grao dobrado: uma celula 8x8 so metade coberta passa do corte de 30%.
	if len(m.Coarse[0]) != 1 {
		t.Errorf("grao dobrado deveria ter 1 celula, tem %d", len(m.Coarse[0]))
	}
}

func testBurst(mirror bool) *ashBurst {
	var cells []deathCell
	for v := 0; v < 64; v += 4 {
		for u := 0; u < 64; u += 4 {
			cells = append(cells, deathCell{U: uint16(u), V: uint16(v), W: 4, H: 4, Col: color.RGBA{100, 100, 100, 255}})
		}
	}
	def := GetEnemyDef(EnemyTypeGarrison)
	ad := EnemyAnimDef{FrameWidth: 64, FrameHeight: 64, Columns: 1}
	pose := DeathPose{Type: EnemyTypeGarrison, CX: 500, CY: 300, Mirror: mirror}
	return newAshBurst(pose, cells, 4, rl.Texture2D{ID: 1}, ad, def)
}

func TestAshBurstReleasesWithinSweepAndDownwindFirst(t *testing.T) {
	b := testBurst(false)
	tn := b.tune
	var first, last *ashFrag
	for i := range b.frags {
		f := &b.frags[i]
		if f.release < tn.Delay-1e-4 || f.release > tn.Delay+tn.Sweep+1e-4 {
			t.Fatalf("grao solta fora da varredura: %v", f.release)
		}
		if first == nil || f.release < first.release {
			first = f
		}
		if last == nil || f.release > last.release {
			last = f
		}
	}
	// O primeiro a soltar esta mais a favor do vento que o ultimo.
	if first.hx*b.dirX+first.hy*b.dirY <= last.hx*b.dirX+last.hy*b.dirY {
		t.Errorf("a frente devia andar contra o vento: primeiro %v,%v ultimo %v,%v (vento %v,%v)",
			first.hx, first.hy, last.hx, last.hy, b.dirX, b.dirY)
	}
}

func TestAshBurstMirrorFlipsHomeAndSource(t *testing.T) {
	a, b := testBurst(false), testBurst(true)
	for i := range a.frags {
		if a.frags[i].hx != -b.frags[i].hx || a.frags[i].hy != b.frags[i].hy {
			t.Fatalf("celula %d nao espelhou: %v,%v x %v,%v", i, a.frags[i].hx, a.frags[i].hy, b.frags[i].hx, b.frags[i].hy)
		}
		if b.frags[i].srcW >= 0 || b.frags[i].srcX != a.frags[i].srcX {
			t.Fatalf("celula %d: espelho e largura negativa na MESMA regiao da folha", i)
		}
	}
}

func TestAshBurstRunsToCompletion(t *testing.T) {
	b := testBurst(false)
	steps := 0
	for !b.done() {
		b.step(1.0 / 60)
		if steps++; steps > 60*10 {
			t.Fatalf("a explosao nao terminou em 10 s (t=%v end=%v puffs=%d)", b.t, b.end, len(b.puffs))
		}
	}
	if b.live != 0 {
		t.Errorf("terminou com %d graos vivos", b.live)
	}
}

// Detectar a morte: desenhado no quadro passado e ausente neste. Trocar de
// mundo NAO e morte.
func TestDeathWatchSpawnsOnlyForVanishedAndNotAcrossWorlds(t *testing.T) {
	now := 100.0
	var spawned []EnemyType
	oldClock, oldSpawn, oldStarted := deathClock, deathSpawn, deathMaskStarted
	deathClock = func() float64 { return now }
	deathSpawn = func(p DeathPose) { spawned = append(spawned, p.Type) }
	deathMaskStarted = true
	defer func() {
		deathClock, deathSpawn, deathMaskStarted = oldClock, oldSpawn, oldStarted
		SetDeathFXEnabled(true)
	}()
	SetDeathFXEnabled(true)

	SyncDeathFXWorld("world_01", 0)
	now += deathFXWorldGrace + 0.1
	alive := map[string]bool{"a": true, "b": true}
	isAlive := func(id string) bool { return alive[id] }

	BeginDeathFXFrame(isAlive)
	watchEnemyPose("a", DeathPose{Type: EnemyTypeBasic})
	watchEnemyPose("b", DeathPose{Type: EnemyTypeFast})

	delete(alive, "a")
	BeginDeathFXFrame(isAlive)
	if len(spawned) != 1 || spawned[0] != EnemyTypeBasic {
		t.Fatalf("esperava so o slime desintegrando, veio %v", spawned)
	}

	// Troca de mapa: o lobo some junto com o mundo, e nao morreu.
	watchEnemyPose("b", DeathPose{Type: EnemyTypeFast})
	SyncDeathFXWorld("world_02", 0)
	delete(alive, "b")
	BeginDeathFXFrame(isAlive)
	if len(spawned) != 1 {
		t.Fatalf("troca de mundo disparou desintegracao: %v", spawned)
	}

	// Desligado: nada e registrado nem disparado.
	now += deathFXWorldGrace + 0.1
	SetDeathFXEnabled(false)
	watchEnemyPose("c", DeathPose{Type: EnemyTypeBasic})
	BeginDeathFXFrame(isAlive)
	if len(spawned) != 1 {
		t.Fatalf("efeito desligado ainda disparou: %v", spawned)
	}
}
