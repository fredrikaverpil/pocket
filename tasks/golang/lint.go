package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// LintOptions configures the go-lint task.
type LintOptions struct {
	Config  string `arg:"config"   usage:"path to golangci-lint config file"`
	SkipFix bool   `arg:"skip-fix" usage:"don't auto-fix issues"`
}

// Lint runs golangci-lint with auto-fix enabled by default.
var Lint = pocket.Task("go-lint", "run golangci-lint", pocket.Serial(
	golangcilint.Install,
	lintCmd(),
)).With(LintOptions{})

func lintCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[LintOptions](ctx)

		args := []string{"run"}
		if pocket.Verbose(ctx) {
			args = append(args, "-v")
		}
		if opts.Config != "" {
			args = append(args, "-c", opts.Config)
		} else if configPath, err := pocket.ConfigPath(ctx, "golangci-lint", golangcilint.Config); err == nil && configPath != "" {
			args = append(args, "-c", configPath)
		}
		if !opts.SkipFix {
			args = append(args, "--fix")
		}
		args = append(args, "./...")

		return pocket.Exec(ctx, golangcilint.Name, args...)
	})
}
