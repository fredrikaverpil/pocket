// Package treesitter provides tree-sitter related build tasks.
package treesitter

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
)

// parsersKey is the context key for tree-sitter parser names.
type parsersKey struct{}

// WithParser specifies tree-sitter parsers to compile from source.
// Parsers are compiled using the tree-sitter CLI and made available
// to ts_query_ls via its --config flag.
//
// Example:
//
//	pk.WithOptions(
//	    treesitter.Tasks(),
//	    treesitter.WithParser("go"),
//	)
func WithParser(parsers ...string) pk.PathOption {
	return pk.WithContextValue(parsersKey{}, parsers)
}

// parsersFromContext returns the parser names from context.
func parsersFromContext(ctx context.Context) []string {
	if v, ok := ctx.Value(parsersKey{}).([]string); ok {
		return v
	}
	return nil
}

// Tasks returns all tree-sitter tasks composed as a Runnable.
// Tasks run serially to ensure the ts_query_ls install completes
// before any task tries to use it.
func Tasks() pk.Runnable {
	return pk.Serial(QueryFormat, QueryLint)
}
