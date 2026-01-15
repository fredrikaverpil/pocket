package python

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/ruff"
)

// LintOptions configures the py-lint task.
type LintOptions struct {
	RuffConfig string `arg:"ruff-config" usage:"path to ruff config file"`
}

// Lint lints Python files using ruff check.
var Lint = pocket.Func("py-lint", "lint Python files", pocket.Serial(
	ruff.Install,
	lint,
)).With(LintOptions{})

func lint(ctx context.Context) error {
	opts := pocket.Options[LintOptions](ctx)
	configPath := opts.RuffConfig
	if configPath == "" {
		var err error
		configPath, err = pocket.ConfigPath("ruff", ruff.Config)
		if err != nil {
			return fmt.Errorf("get ruff config: %w", err)
		}
	}

	args := []string{"check"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	args = append(args, pocket.Path(ctx))

	return pocket.Exec(ctx, ruff.Name, args...)
}
