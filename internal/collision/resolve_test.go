package collision

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// wall is a Solid made of one axis-aligned rectangle. The overlap test is
// written out instead of calling raylib so the package's rules can be checked
// without a window or a loaded map.
type wall struct {
	x, y, width, height float32
}

func (w wall) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	return pos.X+width/2 > w.x && pos.X-width/2 < w.x+w.width &&
		pos.Y+height/2 > w.y && pos.Y-height/2 < w.y+w.height
}

// vertical is a tall wall spanning x in [100, 200].
var vertical = wall{x: 100, y: -1000, width: 100, height: 2000}

const (
	boxW = 40
	boxH = 40
)

func TestResolveTakesTheWholeMoveWhenClear(t *testing.T) {
	got := Resolve(rl.NewVector2(0, 0), rl.NewVector2(10, 5), boxW, boxH, vertical)
	if got != rl.NewVector2(10, 5) {
		t.Fatalf("clear path: got %v, want (10, 5)", got)
	}
}

func TestResolveNilSolidNeverBlocks(t *testing.T) {
	got := Resolve(rl.NewVector2(150, 0), rl.NewVector2(10, 0), boxW, boxH, nil)
	if got != rl.NewVector2(160, 0) {
		t.Fatalf("nil solid: got %v, want (160, 0)", got)
	}
}

func TestResolveSlidesAlongObstacle(t *testing.T) {
	// Diagonal into the wall: X is refused, Y still goes through.
	got := Resolve(rl.NewVector2(70, 0), rl.NewVector2(40, 10), boxW, boxH, vertical)
	if got != rl.NewVector2(70, 10) {
		t.Fatalf("slide: got %v, want (70, 10)", got)
	}
}

func TestResolveStopsWhenBothAxesBlocked(t *testing.T) {
	from := rl.NewVector2(70, 0)
	if got := Resolve(from, rl.NewVector2(40, 0), boxW, boxH, vertical); got != from {
		t.Fatalf("blocked: got %v, want %v", got, from)
	}
}

func TestResolveLetsAnEntityEscapeFromInsideASolid(t *testing.T) {
	// Spawned on top of an obstacle: refusing to move would trap it forever.
	got := Resolve(rl.NewVector2(150, 0), rl.NewVector2(10, 0), boxW, boxH, vertical)
	if got != rl.NewVector2(160, 0) {
		t.Fatalf("escape: got %v, want (160, 0)", got)
	}
}

// north e south sao as duas direcoes de contorno ao longo da face de `vertical`.
var (
	none  = rl.Vector2{}
	north = rl.NewVector2(0, -1)
	south = rl.NewVector2(0, 1)
)

func TestResolveDetourStepsSidewaysWhenStuck(t *testing.T) {
	got, dir := ResolveDetour(rl.NewVector2(70, 0), rl.NewVector2(40, 0), boxW, boxH, vertical, none)
	if got != rl.NewVector2(70, 40) {
		t.Fatalf("detour: got %v, want (70, 40)", got)
	}
	if dir != south {
		t.Fatalf("detour dir: got %v, want %v", dir, south)
	}
}

func TestResolveDetourKeepsTheCommittedDirection(t *testing.T) {
	got, dir := ResolveDetour(rl.NewVector2(70, 0), rl.NewVector2(40, 0), boxW, boxH, vertical, north)
	if got != rl.NewVector2(70, -40) || dir != north {
		t.Fatalf("committed detour: got %v dir %v, want (70, -40) dir %v", got, dir, north)
	}
}

func TestResolveDetourTreatsAUselessSlideAsStuck(t *testing.T) {
	// Pressed flat against the wall while the target pulls it barely off-axis:
	// the plain slide would crawl back toward y=0 and re-block forever.
	got, dir := ResolveDetour(rl.NewVector2(70, 5), rl.NewVector2(40, -0.5), boxW, boxH, vertical, south)
	if dir != south {
		t.Fatalf("useless slide should detour, got dir %v", dir)
	}
	if got.Y <= 5 {
		t.Fatalf("detour should keep going around, got %v", got)
	}
}

func TestResolveDetourStaysCommittedWhileStillBlocked(t *testing.T) {
	// A slide that still makes ground is normally taken, but not while a
	// detour is in progress: giving up the way around at the first usable
	// slide is what leaves an entity hovering next to a big obstacle forever.
	from := rl.NewVector2(70, 0)
	got, dir := ResolveDetour(from, rl.NewVector2(40, 10), boxW, boxH, vertical, south)
	if dir != south {
		t.Fatalf("commitment dropped, dir %v", dir)
	}
	if got == rl.NewVector2(70, 10) {
		t.Fatalf("took the plain slide instead of continuing around: %v", got)
	}
}

func TestResolveDetourClearsTheDirectionOnceThePathOpens(t *testing.T) {
	got, dir := ResolveDetour(rl.NewVector2(0, 0), rl.NewVector2(10, 0), boxW, boxH, vertical, south)
	if got != rl.NewVector2(10, 0) || dir != none {
		t.Fatalf("open path: got %v dir %v, want (10, 0) dir zero", got, dir)
	}
}

// barricade e uma linha de ponta a ponta com UM vao, que e a forma de toda
// barricada do mapa 3 e de toda cerca do mapa 1.
type barricade struct {
	y, height    float32
	gapX0, gapX1 float32
}

func (b barricade) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	if pos.Y+height/2 <= b.y || pos.Y-height/2 >= b.y+b.height {
		return false
	}
	return !(pos.X-width/2 > b.gapX0 && pos.X+width/2 < b.gapX1)
}

// TestDetourWalksTowardTheOpeningAndNotAwayFromIt e o teste do defeito
// relatado: "quando existe uma cerca ou barricada os monstros ficam batendo de
// frente em vez de dar a volta".
//
// A causa nao era falta de contorno — o contorno existia e comecava. Era o LADO:
// ele saia de uma rotacao fixa (+90 graus do rumo), entao a criatura que ja
// tinha deslizado ate a beira do vao virava e caminhava para o lado oposto.
// Medido na versao antiga: 27 segundos andando para longe de uma abertura que
// estava a 60 px.
func TestDetourWalksTowardTheOpeningAndNotAwayFromIt(t *testing.T) {
	// Vao a OESTE de quem esta contornando. O sinal fixo mandava para leste.
	wall := barricade{y: 0, height: 64, gapX0: 400, gapX1: 900}
	from := rl.NewVector2(2000, 86)
	toward := rl.NewVector2(-3, -20) // rumo ao alvo, do outro lado da linha

	_, dir := ResolveDetour(from, toward, boxW, boxH, wall, none)
	if dir.X >= 0 {
		t.Fatalf("contornou para %v; o vao esta a oeste, entao X tem de ser negativo", dir)
	}
}

func TestDetourFindsTheNEARESTOpening(t *testing.T) {
	// Duas aberturas, uma de cada lado, e a de leste esta mais perto. A
	// varredura alterna os lados justamente para isto: ela nao pode preferir um
	// lado, tem de preferir o mais proximo.
	wall := twoGaps{y: 0, height: 64, west: [2]float32{0, 300}, east: [2]float32{1200, 1500}}
	from := rl.NewVector2(1000, 86)
	_, dir := ResolveDetour(from, rl.NewVector2(0, -20), boxW, boxH, wall, none)
	if dir.X <= 0 {
		t.Fatalf("contornou para %v; a abertura mais proxima esta a leste", dir)
	}
}

type twoGaps struct {
	y, height  float32
	west, east [2]float32
}

func (g twoGaps) CollidesCentered(pos rl.Vector2, width, height float32) bool {
	if pos.Y+height/2 <= g.y || pos.Y-height/2 >= g.y+g.height {
		return false
	}
	for _, gap := range [2][2]float32{g.west, g.east} {
		if pos.X-width/2 > gap[0] && pos.X+width/2 < gap[1] {
			return false
		}
	}
	return true
}
