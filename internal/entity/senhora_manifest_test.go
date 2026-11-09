package entity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A geometria da Senhora das Trevas existe em dois lugares: no manifesto que o
// montar_folhas.py emitiu e copiada na EnemyDef. Copiar e a escolha certa - ler
// JSON em tempo de desenho seria caro e fragil - mas copia sem verificacao
// envelhece.
//
// O jeito de isso quebrar de verdade: alguem remonta as folhas com uma margem
// diferente, os quadros mudam de tamanho, e a EnemyDef continua dizendo
// 610x590. O jogo nao falha; ele passa a desenhar um pedaco da chefe, esticado,
// e o FootLine erra a linha do chao em alguns pixels por animacao - que le como
// a criatura pulando de altura ao trocar de estado.
type senhoraManifest struct {
	Mode        string  `json:"mode"`
	RenderScale float32 `json:"render_scale"`
	Anims       map[string]struct {
		Sheet       string  `json:"sheet"`
		FrameWidth  int     `json:"frame_width"`
		FrameHeight int     `json:"frame_height"`
		Columns     int     `json:"columns"`
		FrameTime   float32 `json:"frame_time_seconds"`
		Loop        bool    `json:"loop"`
		PlayOrder   []int   `json:"play_order"`
		FootLine    int     `json:"foot_line"`
	} `json:"anims"`
}

func loadSenhoraManifest(t *testing.T) senhoraManifest {
	t.Helper()
	path := filepath.Join(repoRoot(), "assets", "sprites", "enemies",
		"senhora_das_trevas", "senhora_das_trevas_manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sem senhora_das_trevas_manifest.json (%v); rode "+
			"work/enemy-sprites/senhora-das-trevas/montar_folhas.py", err)
	}
	var m senhoraManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifesto ilegivel: %v", err)
	}
	return m
}

func TestSenhoraDefMatchesManifest(t *testing.T) {
	m := loadSenhoraManifest(t)
	def := GetEnemyDef(EnemyTypeDarkLady)

	if string(def.Mode) != m.Mode {
		t.Errorf("Mode = %q, manifesto diz %q", def.Mode, m.Mode)
	}
	if def.RenderScale != m.RenderScale {
		t.Errorf("RenderScale = %v, manifesto diz %v", def.RenderScale, m.RenderScale)
	}
	if len(def.Anims) != len(m.Anims) {
		t.Fatalf("EnemyDef tem %d animacoes, manifesto tem %d", len(def.Anims), len(m.Anims))
	}

	for nome, esperado := range m.Anims {
		anim := EnemyAnim(nome)
		ad, ok := def.Anims[anim]
		if !ok {
			t.Errorf("manifesto tem %q e a EnemyDef nao", nome)
			continue
		}
		if ad.FrameWidth != esperado.FrameWidth || ad.FrameHeight != esperado.FrameHeight {
			t.Errorf("%s: quadro %dx%d, manifesto diz %dx%d", nome,
				ad.FrameWidth, ad.FrameHeight, esperado.FrameWidth, esperado.FrameHeight)
		}
		if ad.Columns != esperado.Columns {
			t.Errorf("%s: Columns = %d, manifesto diz %d", nome, ad.Columns, esperado.Columns)
		}
		if ad.FrameTime != esperado.FrameTime {
			t.Errorf("%s: FrameTime = %v, manifesto diz %v", nome, ad.FrameTime, esperado.FrameTime)
		}
		if ad.FootLine != esperado.FootLine {
			t.Errorf("%s: FootLine = %d, manifesto diz %d", nome, ad.FootLine, esperado.FootLine)
		}
		if ad.Looping() != esperado.Loop {
			t.Errorf("%s: Looping = %v, manifesto diz loop = %v", nome, ad.Looping(), esperado.Loop)
		}
		if len(ad.PlayOrder) != len(esperado.PlayOrder) {
			t.Errorf("%s: PlayOrder tem %d passos, manifesto tem %d", nome,
				len(ad.PlayOrder), len(esperado.PlayOrder))
			continue
		}
		for i := range ad.PlayOrder {
			if ad.PlayOrder[i] != esperado.PlayOrder[i] {
				t.Errorf("%s: PlayOrder[%d] = %d, manifesto diz %d", nome, i,
					ad.PlayOrder[i], esperado.PlayOrder[i])
			}
		}
	}
}

// Toda coluna citada por um PlayOrder tem que existir na folha. Um indice fora
// da faixa nao estoura - Column normaliza -, ele desenha o quadro errado em
// silencio, que e pior.
func TestSenhoraPlayOrderDentroDaFolha(t *testing.T) {
	def := GetEnemyDef(EnemyTypeDarkLady)
	for nome, ad := range def.Anims {
		for i, col := range ad.PlayOrder {
			if col < 0 || col >= ad.Columns {
				t.Errorf("%s: PlayOrder[%d] = %d, fora das %d colunas", nome, i, col, ad.Columns)
			}
		}
		if ad.Steps() <= 0 {
			t.Errorf("%s: Steps() = %d", nome, ad.Steps())
		}
	}
}

// O golpe e a conjuracao TEM que ser one-shot, e a espera TEM que ser laco.
//
// Nao e preciosismo de teste: se o attack_windup deixasse de ser laco ele
// terminaria sozinho e a janela de desvio sumiria; se o attack_strike virasse
// laco a chefe ficaria golpeando para sempre e nunca voltaria para idle.
func TestSenhoraCiclosCorretos(t *testing.T) {
	def := GetEnemyDef(EnemyTypeDarkLady)
	esperado := map[EnemyAnim]bool{ // true = toca uma vez
		AnimIdle:          false,
		AnimIdleScan:      true,
		AnimCastLoop:      false,
		AnimCastRelease:   true,
		AnimAttackWindup:  false,
		AnimAttackStrike:  true,
	}
	for anim, oneShot := range esperado {
		ad, ok := def.Anims[anim]
		if !ok {
			t.Errorf("falta a animacao %q", anim)
			continue
		}
		if ad.OneShot != oneShot {
			t.Errorf("%s: OneShot = %v, esperado %v", anim, ad.OneShot, oneShot)
		}
	}
	if !def.IsBoss() {
		t.Error("IsBoss() = false; a maquina de estados de chefe nao vai rodar")
	}
}
