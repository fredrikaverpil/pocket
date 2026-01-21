// Package golang provides task bundles for Go projects.
package golang

import "github.com/fredrikaverpil/pocket/pk"

// Detect returns a DetectFunc that finds Go modules (directories containing go.mod).
func Detect() pk.DetectFunc {
	return pk.DetectByFile("go.mod")
}

// Tasks returns all Go-related tasks with auto-detection for Go modules.
// By default, it detects directories containing go.mod files.
// Use with pk.WithExcludePath to filter out specific directories.
//
//	pk.WithOptions(
//	    golang.Tasks(),
//	    pk.WithExcludePath("vendor"),
//	)
func Tasks() pk.Runnable {
	return pk.WithOptions(
		pk.Parallel(Lint),
		pk.WithDetect(Detect()),
	)
}
