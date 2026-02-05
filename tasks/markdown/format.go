package markdown

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/prettier"
)

// Format formats Markdown files using prettier.
var Format = &pk.Task{
	Name:  "md-format",
	Usage: "format Markdown files",
	Flags: map[string]pk.FlagDef{
		"check":  {Default: false, Usage: "check only, don't write"},
		"config": {Default: "", Usage: "path to prettier config file"},
	},
	Body: pk.Serial(prettier.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		configPath := pk.GetFlag[string](ctx, "config")
		if configPath == "" {
			configPath = prettier.EnsureDefaultConfig()
		}

		args := []string{}
		if pk.GetFlag[bool](ctx, "check") {
			args = append(args, "--check")
		} else {
			args = append(args, "--write")
		}

		args = append(args, "--config", configPath)

		// Add ignore file if available.
		if ignorePath, err := prettier.EnsureIgnoreFile(); err == nil {
			args = append(args, "--ignore-path", ignorePath)
		}

		// Use absolute path pattern since prettier runs from install directory.
		pattern := pk.FromGitRoot("**/*.md")
		args = append(args, pattern)

		return prettier.Exec(ctx, args...)
	})
}
