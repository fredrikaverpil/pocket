package markdown

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/prettier"
)

// Flag names for the Format task.
const (
	FlagCheck  = "check"
	FlagConfig = "config"
)

// Format formats Markdown files using prettier.
var Format = &pk.Task{
	Name:  "md-format",
	Usage: "format Markdown files",
	Flags: map[string]pk.FlagDef{
		FlagCheck:  {Default: false, Usage: "check only, don't write"},
		FlagConfig: {Default: "", Usage: "path to prettier config file"},
	},
	Body: pk.Serial(prettier.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		configPath := pk.GetFlag[string](ctx, FlagConfig)
		if configPath == "" && !prettier.HasProjectConfig() {
			configPath = prettier.EnsureDefaultConfig()
		}

		args := []string{}
		if pk.GetFlag[bool](ctx, FlagCheck) {
			args = append(args, "--check")
		} else {
			args = append(args, "--write")
		}

		if configPath != "" {
			args = append(args, "--config", configPath)
		}

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
