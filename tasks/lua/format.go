package lua

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

var (
	formatFlags  = flag.NewFlagSet("lua-format", flag.ContinueOnError)
	formatConfig = formatFlags.String("config", "", "path to stylua config file")
)

// Format formats Lua files using stylua.
var Format = pk.NewTask("lua-format", "format Lua files", formatFlags,
	pk.Serial(stylua.Install, formatCmd()),
)

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		configPath := *formatConfig
		if configPath == "" {
			configPath = stylua.EnsureDefaultConfig()
		}

		absDir := pk.FromGitRoot(pk.PathFromContext(ctx))

		args := []string{}
		if pk.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		args = append(args, "-f", configPath)
		args = append(args, absDir)

		return pk.Exec(ctx, stylua.Name, args...)
	})
}
