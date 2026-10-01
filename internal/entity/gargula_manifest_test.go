package entity

import (
	"encoding/json"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

// A geometria da gargula existe em tres lugares, pelo mesmo motivo que a do
// orc: no build_gargula.py que a produziu, no gargula_manifest.json que ele
// emite, e copiada na EnemyDef. Copiar e a escolha certa - ler JSON em tempo de
// desenho seria caro e fragil - mas copia sem verificacao envelhece.
//
// O jeito de isso quebrar de verdade: alguem mexe em FRAME_W/FRAME_H ou em
// EXTENT no build para a asa caber melhor, a folha muda de tamanho, e a
// EnemyDef continua dizendo 448x256. O jogo nao falha; ele passa a desenhar um
// pedaco da gargula, esticado.
type gargulaManifest struct {
	Image       string `json:"image"`
	Mode        string `json:"mode"`
	FrameWidth  int    `json:"frame_width"`
	FrameHeight int    `json:"frame_height"`
	Frames      int    `json:"frames"`
	FootLine    int    `json:"foot_line"`
	Animations  map[string]struct {
		Frames    int     `json:"frames"`
		FrameTime float32 `json:"frame_time_seconds"`
	} `json:"animations"`
}

func loadGargulaManifest(t *testing.T) gargulaManifest {
	t.Helper()
	path := filepath.Join(repoRoot(), "assets", "sprites", "enemies", "gargula", "gargula_manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sem gargula_manifest.json (%v); rode work/enemy-sprites/gargula/build_gargula.py", err)
	}
	var m gargulaManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("gargula_manifest.json ilegivel: %v", err)
	}
	return m
}

// A EnemyDef da sentinela tem que dizer exatamente o que o manifesto diz.
func TestGargulaDefMatchesManifest(t *testing.T) {
	m := loadGargulaManifest(t)
	def := GetEnemyDef(EnemyTypeCastleSentry)

	if def.Mode != EnemyModeFixed {
		t.Errorf("modo %q; a folha e de vista fixa e nao pode girar nem escolher linha", def.Mode)
	}
	idle := def.AnimDef(AnimIdle)
	if idle.FrameWidth != m.FrameWidth || idle.FrameHeight != m.FrameHeight {
		t.Errorf("quadro %dx%d; manifesto diz %dx%d",
			idle.FrameWidth, idle.FrameHeight, m.FrameWidth, m.FrameHeight)
	}
	if idle.Columns != m.Frames {
		t.Errorf("Columns = %d; manifesto tem %d quadros", idle.Columns, m.Frames)
	}
	if idle.FootLine != m.FootLine {
		t.Errorf("FootLine = %d; manifesto diz %d", idle.FootLine, m.FootLine)
	}
	if anim, ok := m.Animations["idle"]; ok && idle.FrameTime != anim.FrameTime {
		t.Errorf("FrameTime = %.3f; manifesto diz %.3f", idle.FrameTime, anim.FrameTime)
	}
}

// E o manifesto tem que concordar com o PNG, senao os dois podem mentir juntos.
//
// A conta e `largura = FrameWidth * quadros` e `altura = FrameHeight` - UMA
// linha. Uma folha de vista fixa nao tem linha por direcao: e essa a diferenca
// que separa este teste do equivalente do orc.
func TestGargulaSheetMatchesManifest(t *testing.T) {
	m := loadGargulaManifest(t)
	path := filepath.Join(repoRoot(), "assets", "sprites", "enemies", "gargula", m.Image)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer file.Close()
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatalf("PNG ilegivel: %v", err)
	}

	wantW := m.FrameWidth * m.Frames
	if cfg.Width != wantW || cfg.Height != m.FrameHeight {
		t.Errorf("folha %dx%d; manifesto pede %d quadros de %dx%d = %dx%d",
			cfg.Width, cfg.Height, m.Frames, m.FrameWidth, m.FrameHeight, wantW, m.FrameHeight)
	}
	if m.FootLine > m.FrameHeight {
		t.Errorf("FootLine %d cai fora do quadro de altura %d", m.FootLine, m.FrameHeight)
	}
}

// A sentinela nao anda, e a folha depende disso: com Speed > 0 o EnemyDef
// mandaria a criatura sair do pedestal, e a arte - desenhada de um angulo so -
// a seguiria de costas para onde ela fosse.
func TestGargulaNaoAnda(t *testing.T) {
	def := GetEnemyDef(EnemyTypeCastleSentry)
	if def.Speed != 0 {
		t.Errorf("Speed = %.0f; a sentinela e um posto fixo e a folha so tem uma vista", def.Speed)
	}
	if def.TurnRate != 0 {
		t.Errorf("TurnRate = %.0f; vista fixa nao gira", def.TurnRate)
	}
}
