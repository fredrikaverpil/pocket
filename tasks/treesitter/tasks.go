// Package treesitter provides tree-sitter related build tasks.
package treesitter

import (
	"github.com/fredrikaverpil/pocket/pk"
)

// Tasks returns all tree-sitter tasks composed as a Runnable.
func Tasks() pk.Runnable {
	return pk.Parallel(QueryFormat, QueryLint)
}
