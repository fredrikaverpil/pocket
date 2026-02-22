package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golangcilint"
)

// FlagFormatConfig is the flag name for the golangci-lint config file path.
const FlagFormatConfig = "config"

// Format formats Go code using golangci-lint fmt.
var Format = &pk.Task{
	Name:  "go-format",
	Usage: "format Go code",
	Flags: map[string]pk.FlagDef{
		FlagFormatConfig: {Default: "", Usage: "path to golangci-lint config file"},
	},
	Body: pk.Serial(golangcilint.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{"fmt"}

		configPath := pk.GetFlag[string](ctx, FlagFormatConfig)
		if configPath == "" && !golangcilint.HasProjectConfig() {
			configPath = golangcilint.EnsureDefaultConfig()
		}
		if configPath != "" {
			args = append(args, "-c", configPath)
		}

		args = append(args, "./...")
		return pk.Exec(ctx, golangcilint.Name, args...)
	})
}
