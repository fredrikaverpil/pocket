// Package govulncheck provides govulncheck tool integration.
package govulncheck

import (
	"context"
	"os/exec"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tool"
)

const name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const version = "v1.1.4"

// Command prepares the tool and returns an exec.Cmd for running govulncheck.
func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if err := Prepare(ctx); err != nil {
		return nil, err
	}
	return pocket.Command(ctx, pocket.FromBinDir(pocket.BinaryName(name)), args...), nil
}

// Run installs (if needed) and executes govulncheck.
func Run(ctx context.Context, args ...string) error {
	cmd, err := Command(ctx, args...)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// Prepare ensures govulncheck is installed.
func Prepare(ctx context.Context) error {
	_, err := tool.GoInstall(ctx, "golang.org/x/vuln/cmd/govulncheck", version)
	return err
}
