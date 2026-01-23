// Package tsqueryls provides the ts_query_ls tool for tree-sitter query files.
package tsqueryls

import "github.com/fredrikaverpil/pocket/pk"

// Name is the binary name for ts_query_ls.
const Name = "ts_query_ls"

// GitURL is the repository URL for ts_query_ls.
const GitURL = "https://github.com/ribru17/ts_query_ls"

// Install creates a task that ensures ts_query_ls is available.
// ts_query_ls is used for formatting and linting tree-sitter query (.scm) files.
var Install = pk.NewTask("install:ts_query_ls", "install ts_query_ls", nil,
	pk.InstallCargo(Name, pk.WithCargoGit(GitURL)),
).Hidden().Global()
