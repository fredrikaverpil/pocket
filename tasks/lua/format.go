package lua

import (
	"context"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// FormatOptions configures the lua-format task.
type FormatOptions struct {
	StyluaConfig string `arg:"stylua-config" usage:"path to stylua config file"`
}

// Format formats Lua files using stylua.
var Format = pocket.Task("lua-format", "format Lua files", pocket.Serial(
	stylua.Install,
	formatCmd(),
)).With(FormatOptions{})

func formatCmd() pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		opts := pocket.Options[FormatOptions](ctx)
		configPath := opts.StyluaConfig
		if configPath == "" {
			var err error
			configPath, err = pocket.ConfigPath(ctx, "stylua", stylua.Config)
			if err != nil {
				configPath = "" // ignore error, proceed without config
			}
		}

		absDir := pocket.FromGitRoot(pocket.Path(ctx))

		args := []string{}
		if pocket.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if configPath != "" {
			args = append(args, "-f", configPath)
		}
		args = append(args, absDir)

		return pocket.Exec(ctx, stylua.Name, args...)
	})
}
