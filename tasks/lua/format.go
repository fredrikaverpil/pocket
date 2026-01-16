package lua

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// FormatOptions configures the lua-format task.
type FormatOptions struct {
	StyluaConfig string `arg:"stylua-config" usage:"path to stylua config file"`
}

// Format formats Lua files using stylua.
var Format = pocket.Func("lua-format", "format Lua files", pocket.Serial(
	stylua.Install,
	format,
)).With(FormatOptions{})

func format(ctx context.Context) error {
	opts := pocket.Options[FormatOptions](ctx)
	configPath := opts.StyluaConfig
	if configPath == "" {
		var err error
		configPath, err = pocket.ConfigPath(ctx, "stylua", stylua.Config)
		if err != nil {
			return fmt.Errorf("get stylua config: %w", err)
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
}
