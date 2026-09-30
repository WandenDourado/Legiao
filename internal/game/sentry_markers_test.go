package game

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// A seta das gargulas so existe se DrawFrame a chamar. Em 28/09/2026 a
// chamada sumiu numa edicao paralela de renderer.go, o linker descartou o
// indicador inteiro e o jogo compilou sem reclamar — o Gui so descobriu
// jogando. Este teste le o proprio renderer.go e exige a chamada.
func TestRendererDrawsSentryMarkers(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "renderer.go", nil, 0)
	if err != nil {
		t.Fatalf("nao consegui ler renderer.go: %v", err)
	}
	found := false
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "DrawFrame" {
			return true
		}
		ast.Inspect(fn.Body, func(m ast.Node) bool {
			if call, ok := m.(*ast.CallExpr); ok {
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "drawSentryMarkers" {
					found = true
				}
			}
			return true
		})
		return false
	})
	if !found {
		t.Fatal("DrawFrame nao chama drawSentryMarkers: as setas das gargulas sumiram da tela")
	}
}
