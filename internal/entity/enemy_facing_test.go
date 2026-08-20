package entity

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// O orc tem que encarar o alvo mesmo quando o movimento vai para outro lado.
//
// Este teste existe porque a regra e facil de desfazer sem querer: a linha que
// a garante e uma atribuicao no meio de moveTowardTarget, e o comentario que
// estava ali antes ensinava o CONTRARIO ("Velocity is realigned ... so the
// sprite faces where the enemy is really going"), que continua sendo a regra
// certa para o lobo. Quem mexer nessa funcao vai ler as duas coisas.
func TestDirectionalEnemyFacesTargetNotVelocity(t *testing.T) {
	e := NewEnemy(EnemyTypeGarrison, 0, 0)

	// Alvo a leste. Empurrao de separacao forte para o NORTE, como um orc
	// espremido entre outros dois.
	target := rl.NewVector2(400, 0)
	separation := rl.NewVector2(0, -6)
	e.moveTowardTarget(target, 0.016, separation, MoveEnv{})

	if e.Facing.X <= 0.99 {
		t.Errorf("Facing = %+v; devia apontar para o leste, direto ao alvo", e.Facing)
	}
	// E a velocidade tem mesmo de estar torta, senao o teste nao provou nada.
	if e.Velocity.Y >= 0 {
		t.Fatalf("a separacao nao entortou a velocidade (%+v); o teste virou tautologia", e.Velocity)
	}

	row, mirror, ok := enemyRowForHeading(e.Facing.X, e.Facing.Y)
	if !ok {
		t.Fatal("Facing nao resolveu uma direcao")
	}
	wantRow, wantMirror, _ := enemyRowForIndex(4) // leste
	if row != wantRow || mirror != wantMirror {
		t.Errorf("olhando para a linha %d espelho %v; alvo a leste pede linha %d espelho %v",
			row, mirror, wantRow, wantMirror)
	}
}

// Perder o alvo nao vira de costas. Zerar Facing junto com Velocity faria o
// monstro girar para o norte no quadro em que o ultimo jogador morresse.
func TestLosingTheTargetKeepsTheFacing(t *testing.T) {
	e := NewEnemy(EnemyTypeGarrison, 0, 0)
	e.moveTowardTarget(rl.NewVector2(0, 400), 0.016, rl.Vector2{}, MoveEnv{})
	before := e.Facing

	e.Update(0.016, nil, nil, MoveEnv{})

	if e.Facing != before {
		t.Errorf("Facing mudou de %+v para %+v ao perder o alvo", before, e.Facing)
	}
	if e.Velocity != (rl.Vector2{}) {
		t.Errorf("Velocity = %+v; sem alvo o monstro tem de parar", e.Velocity)
	}
	if e.Anim != AnimIdle {
		t.Errorf("Anim = %q; sem alvo o monstro tem de voltar a parado", e.Anim)
	}
}

// O inimigo radial NAO muda: ele continua encarando para onde de fato anda.
// Um quadrupede que corre rente a uma cerca vira o corpo junto, e foi assim que
// o lobo foi desenhado.
func TestRadialEnemyStillFacesItsVelocity(t *testing.T) {
	e := NewEnemy(EnemyTypeFast, 0, 0)
	e.moveTowardTarget(rl.NewVector2(400, 0), 0.016, rl.NewVector2(0, -6), MoveEnv{})
	e.updateAnimation(0.016)

	wantAngle, ok := radialAngleFor(e.Velocity.X, e.Velocity.Y)
	if !ok {
		t.Fatal("a velocidade do lobo nao resolveu um angulo")
	}
	if e.targetAngle != wantAngle {
		t.Errorf("o lobo mirou %v; a velocidade dele pede %v", e.targetAngle, wantAngle)
	}
}

// A regra do cliente tem de ser a mesma do host, senao o mesmo orc anda de
// costas na tela de quem nao e o host.
func TestNearestFacingTargetPicksTheClosest(t *testing.T) {
	from := rl.NewVector2(100, 100)
	got := NearestFacingTarget(from, []rl.Vector2{
		rl.NewVector2(500, 100),
		rl.NewVector2(140, 100),
		rl.NewVector2(-300, 100),
	})
	if !got.OK {
		t.Fatal("com candidatos na lista o alvo tem de existir")
	}
	if got.Position.X != 140 {
		t.Errorf("escolheu %+v; o mais proximo e (140,100)", got.Position)
	}
}

// Lista vazia nao pode virar o ponto (0,0): sem este booleano, todo monstro sem
// jogador vivo por perto encararia o canto do mapa.
func TestNearestFacingTargetReportsWhenThereIsNoOne(t *testing.T) {
	if got := NearestFacingTarget(rl.NewVector2(10, 10), nil); got.OK {
		t.Errorf("sem candidatos o alvo devia ser invalido; veio %+v", got)
	}
}
