package markdown

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/repopath"
	"github.com/fredrikaverpil/pocket/tools/prettier"
)

// FormatFlags holds flags for the Format task.
type FormatFlags struct {
	Check  bool   `flag:"check"  usage:"check only, don't write"`
	Config string `flag:"config" usage:"path to prettier config file"`
}

// Format formats Markdown files using prettier.
var Format = &pk.Task{
	Name:  "md-format",
	Usage: "format Markdown files",
	Flags: FormatFlags{},
	Body:  pk.Serial(prettier.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := pk.GetFlags[FormatFlags](ctx)
		configPath := f.Config
		if configPath == "" && !prettier.HasProjectConfig() {
			configPath = prettier.EnsureDefaultConfig()
		}

		args := []string{}
		if f.Check {
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
		pattern := repopath.FromGitRoot("**/*.md")
		args = append(args, pattern)

		return prettier.Exec(ctx, args...)
	})
}
