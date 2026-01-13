// Package govulncheck provides govulncheck tool integration.
package govulncheck

import (
	"context"

	"github.com/fredrikaverpil/pocket"
)

const name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const version = "v1.1.4"

const pkg = "golang.org/x/vuln/cmd/govulncheck"

// Tool is the govulncheck tool.
//
// Example usage in a task action:
//
//	govulncheck.Tool.Exec(ctx, tc, "./...")
var Tool = pocket.NewTool(name, version, install)

func install(ctx context.Context, tc *pocket.TaskContext) error {
	_, err := pocket.GoInstall(ctx, tc, pkg, version)
	return err
}
