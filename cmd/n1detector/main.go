package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/pippokairos/n1detector/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	var (
		ignoreFiles = flag.String("ignore", "", "Comma-separated list of files or patterns to ignore")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
	)
	setupConfig(*ignoreFiles, *verbose)
	singlechecker.Main(analyzer.N1Analyzer)
}

func setupConfig(ignoreFiles string, verbose bool) {
	config := &analyzer.Config{
		IgnoreFiles: parseIgnore(ignoreFiles),
		Verbose:     verbose,
	}

	analyzer.SetConfig(config)

	if verbose {
		fmt.Printf("Running with configuration: ignore=%v\n", config.IgnoreFiles)
	}
}

func parseIgnore(ignoreFiles string) []string {
	if ignoreFiles == "" {
		return nil
	}

	return strings.Split(ignoreFiles, ",")
}
