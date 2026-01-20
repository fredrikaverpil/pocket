// Package golang provides task bundles for Go projects.
package golang

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns all Go-related tasks as a parallel composition.
// Use with pk.WithOptions and pk.WithIncludePath to target specific directories.
//
//	pk.WithOptions(
//	    golang.Tasks(),
//	    pk.WithIncludePath("services"),
//	)
func Tasks() pk.Runnable {
	return pk.Parallel(Lint)
}
