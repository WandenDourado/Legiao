package entity

import (
	"encoding/json"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// A geometria do orc existe em tres lugares: no build_orc.py que a produziu, no
// orc_manifest.json que ele emite, e copiada na EnemyDef aqui do lado. Copiar e
// a escolha certa - ler JSON em tempo de execucao para saber a altura de um
// quadro seria caro e fragil -, mas copia sem verificacao envelhece.
//
// O jeito de isso quebrar de verdade: alguem roda `build_orc.py --scale 2.0`
// para melhorar a nitidez, a folha dobra de tamanho, e a EnemyDef continua
// dizendo 154x134. O jogo nao falha; ele passa a desenhar um quarto do orc,
// esticado. Este teste transforma isso num erro de teste em vez de uma tarde
// procurando o que aconteceu com o monstro.
type orcManifest struct {
	Variant    string   `json:"variant"`
	Directions []string `json:"directions"`
	Anims      map[string]struct {
		Sheet       string `json:"sheet"`
		Columns     int    `json:"columns"`
		Rows        int    `json:"rows"`
		FrameWidth  int    `json:"frame_width"`
		FrameHeight int    `json:"frame_height"`
		FootLine    int    `json:"foot_line"`
	} `json:"anims"`
}

// repoRoot sobe de internal/entity ate a raiz do repositorio.
func repoRoot() string { return filepath.Join("..", "..") }

func loadOrcManifest(t *testing.T) orcManifest {
	t.Helper()
	path := filepath.Join(repoRoot(), "assets", "sprites", "enemies", "orc", "orc_manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sem orc_manifest.json (%v); rode work/orc-guarnicao/build_orc.py", err)
	}
	var m orcManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("orc_manifest.json ilegivel: %v", err)
	}
	return m
}

// Toda animacao que o manifesto declara tem que existir na EnemyDef com a
// mesma geometria - e vice-versa. As duas direcoes importam: uma folha gerada
// e nao registrada e trabalho invisivel, e uma animacao registrada sem folha e
// um orc que desaparece quando muda de estado.
func TestOrcDefMatchesManifest(t *testing.T) {
	m := loadOrcManifest(t)

	def := GetEnemyDef(EnemyTypeGarrison)
	if def.Type != EnemyTypeGarrison {
		t.Fatal("o orc de guarnicao nao esta registrado; GetEnemyDef caiu no padrao")
	}

	for name, want := range m.Anims {
		anim := EnemyAnim(name)
		if !def.HasAnim(anim) {
			t.Errorf("o manifesto tem a animacao %q e a EnemyDef nao a registra", name)
			continue
		}
		got := def.AnimDef(anim)
		if got.SpritePath != want.Sheet {
			t.Errorf("%s: SpritePath = %q; manifesto diz %q", name, got.SpritePath, want.Sheet)
		}
		if got.FrameWidth != want.FrameWidth || got.FrameHeight != want.FrameHeight {
			t.Errorf("%s: quadro = %dx%d; manifesto diz %dx%d",
				name, got.FrameWidth, got.FrameHeight, want.FrameWidth, want.FrameHeight)
		}
		if got.Columns != want.Columns {
			t.Errorf("%s: Columns = %d; manifesto diz %d", name, got.Columns, want.Columns)
		}
		if got.Rows != want.Rows {
			t.Errorf("%s: Rows = %d; manifesto diz %d", name, got.Rows, want.Rows)
		}
		if got.FootLine != want.FootLine {
			t.Errorf("%s: FootLine = %d; manifesto diz %d", name, got.FootLine, want.FootLine)
		}
	}

	for anim := range def.Anims {
		if _, ok := m.Anims[string(anim)]; !ok {
			t.Errorf("a EnemyDef registra a animacao %q que o manifesto nao tem; "+
				"rode work/orc-guarnicao/build_orc.py --anims %s", anim, anim)
		}
	}
}

// A caminhada tem que durar o tempo que o chao pede, senao o pe desliza.
//
// A passada foi medida no perfil 090 do pacote: o pe varre 63,5 px em relacao
// ao torso entre o passo mais a frente e o mais atras. Isso e arte, nao
// gosto - e o unico numero aqui que nao pode ser afinado no olho, porque
// deslizamento de pe e exatamente o defeito que a pessoa ve e nao sabe nomear.
//
// O teste existe porque FrameTime, Speed e RenderScale sao TRES numeros
// amarrados, cada um num lugar diferente, e mexer em qualquer um sozinho
// quebra a relacao em silencio.
func TestOrcWalkCycleMatchesItsSpeed(t *testing.T) {
	const strideInSource = 63.5 // px, medido em Walk_Armed_Body_090

	def := GetEnemyDef(EnemyTypeGarrison)
	walk := def.AnimDef(AnimWalk)
	if walk.Columns == 0 || walk.FrameTime == 0 {
		t.Fatal("a caminhada do orc nao esta registrada")
	}

	// Um ciclo do pacote sao DOIS passos.
	groundPerCycle := 2 * strideInSource * def.RenderScale
	wantCycle := groundPerCycle / def.Speed
	gotCycle := walk.FrameTime * float32(walk.Columns)

	// 12% de folga: a passada foi medida em pixels, com faixa de tornozelo, e
	// exigir precisao maior seria fingir uma medida que a arte nao tem.
	if ratio := gotCycle / wantCycle; ratio < 0.88 || ratio > 1.12 {
		t.Errorf("ciclo de caminhada de %.2fs; a %0.f px/s numa passada de %.0f px "+
			"na tela ele devia durar %.2fs (FrameTime %.3f). O pe vai %s.",
			gotCycle, def.Speed, groundPerCycle/2, wantCycle, wantCycle/float32(walk.Columns),
			map[bool]string{true: "arrastar para tras", false: "patinar para a frente"}[ratio > 1])
	}
}

// E o manifesto tem que concordar com o PNG. Sem isto, os dois poderiam mentir
// juntos: regerar a folha e esquecer de reescrever o JSON deixaria os dois
// coerentes entre si e errados sobre o arquivo.
func TestOrcSheetMatchesManifest(t *testing.T) {
	m := loadOrcManifest(t)
	for name, anim := range m.Anims {
		path := filepath.Join(repoRoot(), filepath.FromSlash(anim.Sheet))
		file, err := os.Open(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		cfg, _, err := image.DecodeConfig(file)
		file.Close()
		if err != nil {
			t.Errorf("%s: PNG ilegivel: %v", name, err)
			continue
		}

		wantW := anim.FrameWidth * anim.Columns
		wantH := anim.FrameHeight * anim.Rows
		if cfg.Width != wantW || cfg.Height != wantH {
			t.Errorf("%s: folha %dx%d; manifesto pede %d col x %d lin de %dx%d = %dx%d",
				name, cfg.Width, cfg.Height,
				anim.Columns, anim.Rows, anim.FrameWidth, anim.FrameHeight, wantW, wantH)
		}
		if anim.FootLine > anim.FrameHeight {
			t.Errorf("%s: FootLine %d cai fora do quadro de altura %d",
				name, anim.FootLine, anim.FrameHeight)
		}
	}
}

// A dobra do espelho so e exata se a folha guardar exatamente as linhas que a
// matematica espera. Um manifesto com outro numero de direcoes e um manifesto
// para outro renderizador.
func TestOrcManifestDirectionsMatchFold(t *testing.T) {
	m := loadOrcManifest(t)
	if len(m.Directions) != EnemyDirectionRows {
		t.Errorf("manifesto guarda %d direcoes; a dobra de %d facings pede %d linhas",
			len(m.Directions), enemyDirections, EnemyDirectionRows)
	}
}

// O modo direcional e o radial nao podem se misturar. Um sheet direcional que
// declarasse TurnRate seria girado; um radial que declarasse FootLine seria
// deslocado para cima. Nenhum dos dois falha de forma visivel no codigo.
func TestEnemyModesDoNotCrossContaminate(t *testing.T) {
	for enemyType, def := range enemyRegistry {
		for _, anim := range []EnemyAnim{AnimIdle, AnimWalk} {
			ad := def.AnimDef(anim)
			switch def.Mode {
			case EnemyModeDirectional:
				if def.TurnRate != 0 {
					t.Errorf("%s e direcional mas declara TurnRate %.0f; arte direcional nao gira",
						enemyType, def.TurnRate)
				}
				if ad.Rows <= 0 {
					t.Errorf("%s/%s e direcional mas nao declara Rows", enemyType, anim)
				}
			case EnemyModeRadial:
				if ad.FootLine != 0 {
					t.Errorf("%s/%s e radial mas declara FootLine %d; vista de cima nao tem solas",
						enemyType, anim, ad.FootLine)
				}
			case EnemyModeFixed:
				// Os dois campos que o modo fixo existe para nao ter. TurnRate
				// giraria uma arte de camera unica; Rows faria o desenho
				// procurar uma linha de direcao que a folha nao guarda.
				if def.TurnRate != 0 {
					t.Errorf("%s e de vista fixa mas declara TurnRate %.0f; arte de camera unica nao gira",
						enemyType, def.TurnRate)
				}
				if ad.Rows != 0 {
					t.Errorf("%s/%s e de vista fixa mas declara Rows %d; nao ha direcao para escolher",
						enemyType, anim, ad.Rows)
				}
			}
		}
	}
}

// Um inimigo de folha unica tem que continuar respondendo a mesma geometria
// para qualquer animacao que lhe perguntem. E esta a garantia de que o slime e
// o lobo atravessaram a mudanca sem nada mudar para eles.
func TestSingleSheetEnemiesIgnoreAnimations(t *testing.T) {
	for _, enemyType := range []EnemyType{EnemyTypeBasic, EnemyTypeFast} {
		def := GetEnemyDef(enemyType)
		if len(def.Anims) != 0 {
			t.Errorf("%s passou a declarar Anims; este teste precisa ser repensado", enemyType)
			continue
		}
		idle := def.AnimDef(AnimIdle)
		walk := def.AnimDef(AnimWalk)
		if !reflect.DeepEqual(idle, walk) {
			t.Errorf("%s devolve geometrias diferentes para idle e walk: %+v vs %+v",
				enemyType, idle, walk)
		}
		if idle.SpritePath != def.SpritePath || idle.Columns != def.Columns {
			t.Errorf("%s: AnimDef nao devolveu os campos planos da def", enemyType)
		}
	}
}

// E a maquina de estados nao pode escolher um estado que a arte nao tem.
func TestEnemyAnimNeverLeavesIdleWithoutAWalkSheet(t *testing.T) {
	fast := rl.NewVector2(999, 999)
	for enemyType, def := range enemyRegistry {
		got := enemyAnimFor(def, fast)
		if def.HasAnim(AnimWalk) {
			if got != AnimWalk {
				t.Errorf("%s tem folha de caminhada mas em movimento escolheu %q", enemyType, got)
			}
			if still := enemyAnimFor(def, rl.Vector2{}); still != AnimIdle {
				t.Errorf("%s parado escolheu %q", enemyType, still)
			}
			continue
		}
		if got != AnimIdle {
			t.Errorf("%s nao tem folha de caminhada mas escolheu %q", enemyType, got)
		}
	}
}
