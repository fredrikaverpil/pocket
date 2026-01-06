// Package govulncheck provides govulncheck tool integration.
package govulncheck

import (
	"context"
	"os/exec"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tool"
)

const name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const version = "v1.1.4"

// Command returns an exec.Cmd for running govulncheck.
// Prefer Run() which auto-prepares the tool.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	return bld.Command(ctx, bld.FromBinDir(name), args...)
}

// Run installs (if needed) and executes govulncheck.
func Run(ctx context.Context, args ...string) error {
	if err := Prepare(ctx); err != nil {
		return err
	}
	return Command(ctx, args...).Run()
}

// Prepare ensures govulncheck is installed.
func Prepare(ctx context.Context) error {
	_, err := tool.GoInstall(ctx, "golang.org/x/vuln/cmd/govulncheck", version)
	return err
}
