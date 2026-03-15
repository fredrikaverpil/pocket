package lua

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/repopath"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// FormatFlags holds flags for the Format task.
type FormatFlags struct {
	Config string `flag:"config" usage:"path to stylua config file"`
}

// Format formats Lua files using stylua.
var Format = &pk.Task{
	Name:  "lua-format",
	Usage: "format Lua files",
	Flags: FormatFlags{},
	Body:  pk.Serial(stylua.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := pk.GetFlags[FormatFlags](ctx)
		configPath := f.Config
		if configPath == "" && !stylua.HasProjectConfig() {
			configPath = stylua.EnsureDefaultConfig()
		}

		absDir := repopath.FromGitRoot(pk.PathFromContext(ctx))

		args := []string{}
		if pk.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if configPath != "" {
			args = append(args, "-f", configPath)
		}
		args = append(args, absDir)

		return pk.Exec(ctx, stylua.Name, args...)
	})
}
