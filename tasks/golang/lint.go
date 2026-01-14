package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// LintOptions configures the go-lint task.
type LintOptions struct {
	Config string `arg:"config" usage:"path to golangci-lint config file"`
	Fix    bool   `arg:"fix"    usage:"auto-fix issues"`
}

// Lint runs golangci-lint.
var Lint = pocket.Func("go-lint", "run golangci-lint", lint).
	With(LintOptions{})

func lint(ctx context.Context) error {
	pocket.Serial(ctx, golangcilint.Install)

	opts := pocket.Options[LintOptions](ctx)

	args := []string{"run"}
	if opts.Config != "" {
		args = append(args, "-c", opts.Config)
	} else if configPath, err := pocket.ConfigPath("golangci-lint", golangcilint.Config); err == nil && configPath != "" {
		args = append(args, "-c", configPath)
	}
	if opts.Fix {
		args = append(args, "--fix")
	}
	args = append(args, "./...")

	return pocket.Exec(ctx, golangcilint.Name, args...)
}
