package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// FormatOptions configures the go-format task.
type FormatOptions struct {
	Config string `arg:"config" usage:"path to golangci-lint config file"`
}

// Format formats Go code using golangci-lint fmt.
var Format = pocket.Task("go-format", "format Go code",
	pocket.Serial(golangcilint.Install, formatCmd()),
	pocket.Opts(FormatOptions{}),
)

func formatCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[FormatOptions](ctx)

		args := []string{"fmt"}
		if opts.Config != "" {
			args = append(args, "-c", opts.Config)
		} else if configPath, err := pocket.ConfigPath(ctx, "golangci-lint", golangcilint.Config); err == nil {
			if configPath != "" {
				args = append(args, "-c", configPath)
			}
		}
		args = append(args, "./...")

		return pocket.Exec(ctx, golangcilint.Name, args...)
	})
}
