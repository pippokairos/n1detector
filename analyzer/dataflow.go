package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/callgraph"
)

func findLoops(inspector *inspector.Inspector) map[ast.Node]bool {
	loops := make(map[ast.Node]bool)

	nodeFilter := []ast.Node{
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		loops[node] = true
	})

	return loops
}

func analyzeN1Patterns(
	pass *analysis.Pass,
	dbCalls map[*ast.CallExpr]bool,
	loops map[ast.Node]bool,
	callGraph *callgraph.Graph,
) []Finding {
	var findings []Finding

	// Simple case: Direct DB call inside a loop
	for loopNode := range loops {
		loopFindings := findDBCallsInLoop(pass, loopNode, dbCalls)
		findings = append(findings, loopFindings...)
	}

	// Complex case: Interprocedural analysis
	interproceduralFindings := detectCrossFunction(pass, loops, dbCalls, callGraph)
	findings = append(findings, interproceduralFindings...)

	return findings
}

// findDBCallsInLoop detects direct database calls inside loops
func findDBCallsInLoop(pass *analysis.Pass, loopNode ast.Node, dbCalls map[*ast.CallExpr]bool) []Finding {
	var findings []Finding

	// Use an inspector to look for DB calls inside this loop
	ast.Inspect(loopNode, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if dbCalls[call] {
				findings = append(findings, Finding{
					Pos:       call.Pos(),
					Message:   "DB query inside a loop",
					QueryCall: call,
					LoopNode:  loopNode,
				})
			}
		}
		return true
	})

	return findings
}

// detectCrossFunction handles the more complex case of N+1 queries across functions
func detectCrossFunction(
	pass *analysis.Pass,
	loops map[ast.Node]bool,
	dbCalls map[*ast.CallExpr]bool,
	callGraph *callgraph.Graph,
) []Finding {
	var findings []Finding

	// TODO: implement

	return findings
}
