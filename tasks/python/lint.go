package python

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/ruff"
)

// LintOptions configures the py-lint task.
type LintOptions struct {
	PythonVersion string `arg:"python"      usage:"Python version (for target-version inference)"`
	RuffConfig    string `arg:"ruff-config" usage:"path to ruff config file"`
	SkipFix       bool   `arg:"skip-fix"    usage:"don't auto-fix issues"`
}

// Lint lints Python files using ruff check with auto-fix enabled by default.
var Lint = pocket.Task("py-lint", "lint Python files",
	pocket.Serial(ruff.Install, lintCmd()),
	pocket.Opts(LintOptions{}),
)

func lintCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[LintOptions](ctx)
		configPath := opts.RuffConfig
		if configPath == "" {
			configPath, _ = pocket.ConfigPath(ctx, "ruff", ruff.Config)
		}

		args := []string{"check"}
		if pocket.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if !opts.SkipFix {
			args = append(args, "--fix")
		}
		if configPath != "" {
			args = append(args, "--config", configPath)
		}
		if opts.PythonVersion != "" {
			args = append(args, "--target-version", pythonVersionToRuff(opts.PythonVersion))
		}
		args = append(args, pocket.Path(ctx))

		return pocket.Exec(ctx, ruff.Name, args...)
	})
}
