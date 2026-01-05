// Package govulncheck provides govulncheck tool integration.
package govulncheck

import (
	"context"
	"os/exec"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tool"
	"github.com/goyek/goyek/v3"
)

const name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const version = "v1.1.4"

// Prepare is a goyek task that installs govulncheck.
// Hidden from task list (no Usage field).
var Prepare = goyek.Define(goyek.Task{
	Name: "govulncheck:prepare",
	Action: func(a *goyek.A) {
		if _, err := tool.GoInstall(a.Context(), "golang.org/x/vuln/cmd/govulncheck", version); err != nil {
			a.Fatal(err)
		}
	},
})

// Command returns an exec.Cmd for running govulncheck.
// Call Prepare first or use as a goyek Deps.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	return bld.Command(ctx, bld.FromBinDir(name), args...)
}

// Run executes govulncheck with the given arguments.
// Call Prepare first or use as a goyek Deps.
func Run(ctx context.Context, args ...string) error {
	return Command(ctx, args...).Run()
}
