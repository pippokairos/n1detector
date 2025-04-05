package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/ssa"
)

var N1Analyzer = &analysis.Analyzer{
	Name: "n1query",
	Doc:  "Checks for potential N+1 query patterns in GORM code (including nested calls)",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
		buildssa.Analyzer,
	},
}

// Keep your GORM method identification
var gormMethods = map[string]bool{
	"Find":        true,
	"First":       true,
	"Last":        true,
	"Take":        true,
	"Where":       true,
	"Model":       true,
	"Select":      true,
	"Updates":     true,
	"Update":      true,
	"Association": true,
}

type Finding struct {
	Pos        token.Pos
	Message    string
	LoopNode   ast.Node
	CallInLoop *ast.CallExpr
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	ssaData := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	prog := ssaData.Pkg.Prog
	callGraph := cha.CallGraph(prog)

	dbQueryFunctions := findDBQueryFunctions(pass, ssaData)
	if GetConfig().Verbose {
		fmt.Printf("Found %d potential DB query functions:\n", len(dbQueryFunctions))
		for fn := range dbQueryFunctions {
			fmt.Printf("  - %s\n", fn.String())
		}
	}

	loops := findLoops(inspector)
	if GetConfig().Verbose {
		fmt.Printf("Found %d loops:\n", len(loops))
		for node := range loops {
			fmt.Printf("  - Loop at %s\n", pass.Fset.Position(node.Pos()))
		}
	}

	for loopNode := range loops {
		if GetConfig().Verbose {
			fmt.Printf("Inspecting loop at: %s\n", pass.Fset.Position(loopNode.Pos()))
		}

		reportedLinesInLoop := make(map[int]bool)
		ast.Inspect(loopNode, func(n ast.Node) bool {
			// Don't descend into nested loops when inspecting calls for the current one
			if n != loopNode {
				switch n.(type) {
				case *ast.ForStmt, *ast.RangeStmt:
					if GetConfig().Verbose {
						fmt.Printf("   -> Skipping nested loop at %s\n", pass.Fset.Position(n.Pos()))
					}
					return false
				}
			}

			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			reportPos := callExpr.Lparen
			line := pass.Fset.Position(reportPos).Line

			if reportedLinesInLoop[line] {
				if GetConfig().Verbose {
					fmt.Printf("   -> Already reported for line %d in this loop, skipping call at %s\n", line, pass.Fset.Position(reportPos))
				}
				return true
			}

			if isGormQueryCall(pass, callExpr) {
				if GetConfig().Verbose {
					fmt.Printf("  🚨 Direct DB call found inside loop at: %s (Line %d)\n", pass.Fset.Position(reportPos), line)
				}
				pass.Reportf(reportPos, "Potential N+1 query detected: DB query called directly inside a loop")
				reportedLinesInLoop[line] = true
				return true
			}

			// This check only runs if the line hasn't been reported yet and
			// the current callExpr is not a direct GORM call.
			if GetConfig().Verbose {
				fmt.Printf("   -> Checking CallExpr at %s (Line %d) for indirect N+1\n", pass.Fset.Position(reportPos), line)
			}

			calleeFn := findCalledSSAFunction(pass, ssaData, callExpr)
			if calleeFn == nil {
				if GetConfig().Verbose {
					fmt.Printf("     -> Could not resolve SSA function for call at %s\n", pass.Fset.Position(reportPos))
				}
				return true
			}

			if GetConfig().Verbose {
				fmt.Printf("     -> Call targets SSA function: %s\n", calleeFn.String())
			}

			if reachesDBQuery(callGraph, calleeFn, dbQueryFunctions) {
				if GetConfig().Verbose {
					fmt.Printf("  🚨 Indirect DB call via function %s called inside loop at: %s (Line %d)\n", calleeFn.Name(), pass.Fset.Position(reportPos), line)
				}
				pass.Reportf(reportPos, "Potential N+1 query detected: call to %s may lead to DB query inside loop", shortFuncName(calleeFn))
				reportedLinesInLoop[line] = true
			} else {
				if GetConfig().Verbose {
					fmt.Printf("     -> Call to %s does not seem to reach a DB query function\n", calleeFn.Name())
				}
			}

			// Continue inspection for siblings and children
			return true
		})
	}

	return nil, nil
}

// isGormQueryCall checks if a CallExpr is a known GORM query method call.
// It also uses type information for better accuracy.
func isGormQueryCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	selExpr, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false // Not a method call like obj.Method()
	}

	// A more robust check would involve tracking the type of selExpr.X accurately.
	// For now, we assume if the method name matches, it's likely GORM.
	methodName := selExpr.Sel.Name
	_, isGormMethod := gormMethods[methodName]
	if !isGormMethod {
		return false
	}

	return true
}

// findLoops identifies loop statements in the code.
func findLoops(insp *inspector.Inspector) map[ast.Node]bool {
	loops := make(map[ast.Node]bool)
	nodeFilter := []ast.Node{
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
	}
	insp.Preorder(nodeFilter, func(node ast.Node) {
		loops[node] = true
	})
	return loops
}

// findDBQueryFunctions identifies all SSA functions that contain a query call.
func findDBQueryFunctions(pass *analysis.Pass, ssaData *buildssa.SSA) map[*ssa.Function]bool {
	dbFuncs := make(map[*ssa.Function]bool)

	for _, fn := range ssaData.SrcFuncs {
		if fn == nil {
			continue
		}

		if GetConfig().Verbose {
			fmt.Printf("    Scanning function %s for DB calls\n", fn.String())
		}

		foundInFunc := false
		for _, b := range fn.Blocks {
			for _, instr := range b.Instrs {
				// Check if the instruction corresponds to an AST GORM call
				if call, ok := instr.(ssa.CallInstruction); ok {
					pos := call.Pos()
					if pos == token.NoPos {
						continue // Cannot map instruction without position
					}

					astCall := findAstCallExprAt(pass, pos)
					if astCall != nil {
						if isGormQueryCall(pass, astCall) {
							if GetConfig().Verbose {
								fmt.Printf("      -> Found GORM call (%s) via instruction %v in %s\n", astCall.Fun, instr, fn.String())
							}
							dbFuncs[fn] = true
							foundInFunc = true
							break
						}
					}

					// TODO: Handle dynamic calls (call.Common().IsInvoke())
				}
			}
			if foundInFunc {
				break
			}
		}
	}
	return dbFuncs
}

// findAstCallExprAt finds the *ast.CallExpr whose '(' is at the given position.
func findAstCallExprAt(pass *analysis.Pass, pos token.Pos) *ast.CallExpr {
	for _, file := range pass.Files {
		var found *ast.CallExpr
		ast.Inspect(file, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if call.Lparen == pos {
					found = call
					return false
				}
			}
			if n != nil && !(n.Pos() <= pos && pos < n.End()) {
				return false
			}
			return true
		})
		if found != nil {
			return found
		}
	}
	return nil
}

// findCalledSSAFunction tries to resolve an AST CallExpr to the *ssa.Function being called.
func findCalledSSAFunction(pass *analysis.Pass, ssaData *buildssa.SSA, call *ast.CallExpr) *ssa.Function {
	var obj types.Object
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj = pass.TypesInfo.Uses[fun]
	case *ast.SelectorExpr:
		obj = pass.TypesInfo.Uses[fun.Sel]
	default:
		// Function literals, conversions, ...
		return nil
	}

	if funObj, ok := obj.(*types.Func); ok {
		// Find the SSA function corresponding to this types.Func
		return ssaData.Pkg.Prog.FuncValue(funObj)
	}

	return nil
}

// reachesDBQuery performs a search on the call graph starting from 'startFn'.
// It returns true if any path leads to a function in the 'dbQueryFunctions' map.
func reachesDBQuery(callGraph *callgraph.Graph, startFn *ssa.Function, dbQueryFunctions map[*ssa.Function]bool) bool {
	if startFn == nil {
		return false
	}

	if dbQueryFunctions[startFn] {
		return true
	}

	// Traverse the call graph
	queue := []*ssa.Function{startFn}
	visited := map[*ssa.Function]bool{startFn: true}
	for len(queue) > 0 {
		currentFn := queue[0]
		queue = queue[1:]

		node := callGraph.Nodes[currentFn]
		if node == nil {
			continue // Function might not be in the call graph (e.g., not called, stdlib)
		}

		for _, edge := range node.Out { // Look at functions called by currentFn
			callee := edge.Callee.Func
			if callee != nil && !visited[callee] {
				if dbQueryFunctions[callee] {
					if GetConfig().Verbose {
						fmt.Printf("      -> Path found: %s -> ... -> %s (DB Query Func)\n", startFn.Name(), callee.Name())
					}
					return true // Found a path to a DB query function
				}
				visited[callee] = true
				queue = append(queue, callee)
			}
		}
	}

	return false
}

func shortFuncName(fn *ssa.Function) string {
	if fn == nil {
		return "<unknown>"
	}

	if fn.Parent() != nil {
		return fmt.Sprintf("%s.%s", fn.Parent().RelString(nil), fn.Name())
	}

	if fn.Pkg != nil {
		return fmt.Sprintf("%s.%s", fn.Pkg.Pkg.Name(), fn.Name())
	}

	return fn.Name()
}
