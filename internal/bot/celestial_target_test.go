package bot

// As DUAS cargas das Flechas Celestiais nao podem ir para a mesma torre.
//
// A recarga da suprema so arma depois da segunda carga (ability.Charged), entao
// entre um disparo e o outro `View.UltimateReady` continua verdadeiro. Sem
// memoria, o bot reavaliava "qual e a gargula mais perto" no quadro seguinte,
// achava a MESMA — a flecha ainda estava no ar — e gastava a segunda carga
// nela. Ver arqueiro.go.

import (
	"testing"

	"github.com/WandenDourado/Legiao/internal/entity"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func archerView(foes []Foe) View {
	return View{
		Self:          Ally{Char: entity.CharArqueiro, Pos: rl.NewVector2(0, 0), Health: 100, MaxHealth: 100},
		UltimateReady: true,
		UltimateRange: 4800,
		Foes:          foes,
		Dt:            1.0 / 60.0,
	}
}

// twoSentries: duas torres bem dentro do alcance e LONGE UMA DA OUTRA, para
// que nao caiam na mesma linha de tiro (secondSentryAligned) — o caso deste
// teste e "dois alvos, duas flechas", nao "uma flecha, duas torres".
func twoSentries() []Foe {
	return []Foe{
		{ID: "oeste", Pos: rl.NewVector2(600, 0), HitCentre: rl.NewVector2(600, 0), IsSentry: true},
		{ID: "leste", Pos: rl.NewVector2(0, 900), HitCentre: rl.NewVector2(0, 900), IsSentry: true},
	}
}

func TestTheSecondArrowGoesToTheOtherSentry(t *testing.T) {
	b := &arqueiroBrain{}
	foes := twoSentries()

	first := b.Think(archerView(foes))
	if first.Skill == nil || first.Skill.SkillID != "celestial_arrows" {
		t.Fatalf("o primeiro quadro nao lancou a suprema: %+v", first.Skill)
	}

	second := b.Think(archerView(foes))
	if second.Skill == nil || second.Skill.SkillID != "celestial_arrows" {
		t.Fatalf("o segundo quadro nao lancou a segunda carga: %+v", second.Skill)
	}
	if second.Skill.Aim == first.Skill.Aim {
		t.Errorf("as duas cargas foram para o mesmo alvo (%v)", first.Skill.Aim)
	}
}

func TestASingleSentryDoesNotEatBothCharges(t *testing.T) {
	b := &arqueiroBrain{}
	foes := []Foe{
		{ID: "unica", Pos: rl.NewVector2(600, 0), HitCentre: rl.NewVector2(600, 0), IsSentry: true},
	}

	if first := b.Think(archerView(foes)); first.Skill == nil {
		t.Fatal("o primeiro quadro nao lancou a suprema")
	}
	// Uma flecha ja basta (40 de dano perfurante contra 40 de vida): a segunda
	// carga fica guardada para a proxima torre em vez de ser jogada fora.
	if second := b.Think(archerView(foes)); second.Skill != nil {
		t.Errorf("gastou a segunda carga na mesma torre: %+v", second.Skill)
	}
}

func TestTheArcherTriesAgainAfterTheArrowHadTimeToMiss(t *testing.T) {
	b := &arqueiroBrain{}
	foes := []Foe{
		{ID: "unica", Pos: rl.NewVector2(600, 0), HitCentre: rl.NewVector2(600, 0), IsSentry: true},
	}

	if first := b.Think(archerView(foes)); first.Skill == nil {
		t.Fatal("o primeiro quadro nao lancou a suprema")
	}
	// Passado o voo maximo de uma flecha (celestialMemory), um alvo AINDA DE PE
	// quer dizer que ela errou — e ai ele pode voltar a mirar nele.
	stale := archerView(foes)
	stale.Dt = celestialMemory + 0.1
	if again := b.Think(stale); again.Skill == nil {
		t.Error("a torre sobreviveu a flecha e o bot nunca mais tentou")
	}
}

func TestASentryThatDiedLeavesTheMemory(t *testing.T) {
	b := &arqueiroBrain{}
	b.noteLoosed("morta")
	// O host tira o inimigo morto do EntityManager, entao ele some de v.Foes.
	// Sem esta poda a entrada ficaria no mapa ate o fim da fase.
	b.forgetLoosed(archerView(nil))
	if b.alreadyLoosed("morta") {
		t.Error("a memoria guardou uma torre que ja saiu de campo")
	}
}
