package analyzer

import (
	"fmt"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func buildCallGraph(pass *analysis.Pass) (*callgraph.Graph, error) {
	config := &packages.Config{
		Mode: packages.NeedDeps |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypes |
			packages.NeedImports |
			packages.NeedName,
		Tests: false,
	}

	// Use "." for now to avoid circular reference issues
	// A more robust solution would determine this from pass.Pkg
	pkgs, err := packages.Load(config, ".")
	if err != nil {
		return nil, fmt.Errorf("error loading packages: %v", err)
	}

	program, packages := ssautil.AllPackages(pkgs, ssa.BuilderMode(0))
	program.Build()

	mainPkgs := make(map[*ssa.Package]bool)
	for _, p := range packages {
		mainPkgs[p] = true
	}

	cg := cha.CallGraph(program)

	return cg, nil
}
