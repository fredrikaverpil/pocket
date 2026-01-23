// Package treesitter provides tree-sitter related build tasks.
package treesitter

import (
	"github.com/fredrikaverpil/pocket/pk"
)

// Tasks returns all tree-sitter tasks composed as a Runnable.
// Tasks run serially to ensure the ts_query_ls install completes
// before any task tries to use it.
func Tasks() pk.Runnable {
	return pk.Serial(QueryFormat, QueryLint)
}
