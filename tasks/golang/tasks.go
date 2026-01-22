// Package golang provides task bundles for Go projects.
package golang

import "github.com/fredrikaverpil/pocket/pk"

// Detect returns a DetectFunc that finds Go modules (directories containing go.mod).
func Detect() pk.DetectFunc {
	return pk.DetectByFile("go.mod")
}

// Tasks returns Go-related tasks composed together.
// If no tasks are provided, it defaults to running Fix, Format, Lint,
// then Test and Vulncheck in parallel.
//
// Mutating tasks (Fix, Format, Lint) are always run sequentially.
// Non-mutating tasks (Test, Vulncheck) are run in parallel.
//
// Use with pk.WithDetect to specify where tasks should run:
//
//	pk.WithOptions(
//	    golang.Tasks(),
//	    pk.WithDetect(golang.Detect()),
//	)
func Tasks(tasks ...pk.Runnable) pk.Runnable {
	if len(tasks) == 0 {
		return pk.Serial(
			Fix,
			Format,
			Lint,
			pk.Parallel(Test, Vulncheck),
		)
	}

	return pk.Serial(tasks...)
}
