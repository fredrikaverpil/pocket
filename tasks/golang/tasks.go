// Package golang provides task bundles for Go projects.
package golang

import "github.com/fredrikaverpil/pocket/pk"

// Detect returns a DetectFunc that finds Go modules (directories containing go.mod).
func Detect() pk.DetectFunc {
	return pk.DetectByFile("go.mod")
}

// Tasks returns Go-related tasks with auto-detection for Go modules.
// If no tasks are provided, it defaults to running Fix, Format, Lint,
// then Test and Vulncheck in parallel.
//
// Mutating tasks (Fix, Format, Lint) are always run sequentially.
// Non-mutating tasks (Test, Vulncheck) are run in parallel.
func Tasks(tasks ...pk.Runnable) pk.Runnable {
	if len(tasks) == 0 {
		return pk.WithOptions(
			pk.Serial(
				Fix,
				Format,
				Lint,
				pk.Parallel(Test, Vulncheck),
			),
			pk.WithDetect(Detect()),
		)
	}

	// If tasks are provided, we still want to separate mutating from non-mutating
	// if we can identify them. For now, we'll just run them in the order provided
	// but wrapped in the detection.
	// TODO: better heuristics for parallelization of user-provided tasks.
	return pk.WithOptions(
		pk.Serial(tasks...),
		pk.WithDetect(Detect()),
	)
}
