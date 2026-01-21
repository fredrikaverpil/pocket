// Package golang provides task bundles for Go projects.
package golang

import "github.com/fredrikaverpil/pocket/pk"

// Detect returns a DetectFunc that finds Go modules (directories containing go.mod).
func Detect() pk.DetectFunc {
	return pk.DetectByFile("go.mod")
}

// Tasks returns Go-related tasks with auto-detection for Go modules.
// If no tasks are provided, it defaults to Lint and Test.
//
// Example:
//
//	golang.Tasks() // Runs default tasks (Lint, Test)
//	golang.Tasks(golang.Lint) // Runs only Lint
func Tasks(tasks ...pk.Runnable) pk.Runnable {
	if len(tasks) == 0 {
		tasks = []pk.Runnable{Lint, Test}
	}
	return pk.WithOptions(
		pk.Parallel(tasks...),
		pk.WithDetect(Detect()),
	)
}
