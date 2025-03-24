package analyzer

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var N1Analyzer = &analysis.Analyzer{
	Name:     "n1query",
	Doc:      "Checks for potential N+1 query patterns in GORM code",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

var gormMethods = map[string]bool{
	"Find":   true,
	"First":  true,
	"Last":   true,
	"Take":   true,
	"Where":  true,
	"Model":  true,
	"Select": true,
}

type Finding struct {
	Pos       token.Pos
	Message   string
	QueryCall *ast.CallExpr
	LoopNode  ast.Node
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Find all GORM query calls
	dbQueryCalls := make(map[*ast.CallExpr]bool)
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		call := node.(*ast.CallExpr)

		if isGormQueryCall(call) {
			dbQueryCalls[call] = true
		}
	})

	callGraph, err := buildCallGraph(pass)
	if err != nil {
		return nil, err
	}

	loops := findLoops(inspector)
	findings := analyzeN1Patterns(pass, dbQueryCalls, loops, callGraph)
	for _, finding := range findings {
		pass.Reportf(finding.Pos, "Potential N+1 query detected: %s", finding.Message)
	}

	return nil, nil
}

func isGormQueryCall(call *ast.CallExpr) bool {
	// Basic pattern matching for GORM calls
	// This can be expanded with more sophisticated selector checking
	if selExpr, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := selExpr.Sel.Name

		return gormMethods[methodName]
	}

	return false
}
