package lua

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// FlagConfig is the flag name for the stylua config file path.
const FlagConfig = "config"

// Format formats Lua files using stylua.
var Format = &pk.Task{
	Name:  "lua-format",
	Usage: "format Lua files",
	Flags: map[string]pk.FlagDef{
		FlagConfig: {Default: "", Usage: "path to stylua config file"},
	},
	Body: pk.Serial(stylua.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		configPath := pk.GetFlag[string](ctx, FlagConfig)
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
