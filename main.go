package main

import (
	"fmt"
	"os"
	"os/exec"
)

// Temporary main.go that delegates to .pocket/main.go
// This will be replaced by the pok shim generator in Phase 2.
func main() {
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = ".pocket" // Change to .pocket directory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running .pocket: %v\n", err)
		os.Exit(1)
	}
}
