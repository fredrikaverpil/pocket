package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// FormatFlags holds flags for the Format task.
type FormatFlags struct {
	Config string `flag:"config" usage:"path to golangci-lint config file"`
}

// Format formats Go code using golangci-lint fmt.
var Format = &pk.Task{
	Name:  "go-format",
	Usage: "format Go code",
	Flags: FormatFlags{},
	Body:  pk.Serial(golangcilint.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[FormatFlags](ctx)
		args := []string{"fmt"}

		configPath := f.Config
		if configPath == "" && !golangcilint.HasProjectConfig() {
			configPath = golangcilint.EnsureDefaultConfig()
		}
		if configPath != "" {
			args = append(args, "-c", configPath)
		}

		args = append(args, "./...")
		return run.Exec(ctx, golangcilint.Name, args...)
	})
}
