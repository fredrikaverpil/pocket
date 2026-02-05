package markdown

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/prettier"
)

var (
	formatFlags  = flag.NewFlagSet("md-format", flag.ContinueOnError)
	formatCheck  = formatFlags.Bool("check", false, "check only, don't write")
	formatConfig = formatFlags.String("config", "", "path to prettier config file")
)

// Format formats Markdown files using prettier.
var Format = pk.NewTask(pk.TaskConfig{
	Name:  "md-format",
	Usage: "format Markdown files",
	Flags: formatFlags,
	Body:  pk.Serial(prettier.Install, formatCmd()),
})

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		configPath := *formatConfig
		if configPath == "" {
			configPath = prettier.EnsureDefaultConfig()
		}

		args := []string{}
		if *formatCheck {
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
