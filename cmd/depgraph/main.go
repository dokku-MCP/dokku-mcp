package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// 1. Get all packages in the current module
	out, err := exec.Command("go", "list", "./...").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list packages: %v\n", err)
		os.Exit(1)
	}
	pkgs := strings.Fields(string(out))

	// 2. Build a map of package -> imports
	deps := make(map[string][]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("go", "list", "-f", "{{ join .Imports \"\\n\" }}", pkg).Output()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list imports for %s: %v\n", pkg, err)
			continue
		}
		imports := strings.Fields(string(out))
		// Only keep imports that are in our module
		for _, imp := range imports {
			for _, p := range pkgs {
				if imp == p {
					deps[pkg] = append(deps[pkg], imp)
				}
			}
		}
	}

	// 3. Output DOT graph
	fmt.Println("digraph G {")
	for pkg, imports := range deps {
		for _, imp := range imports {
			fmt.Printf("  \"%s\" -> \"%s\";\n", pkg, imp)
		}
	}
	fmt.Println("}")
}
