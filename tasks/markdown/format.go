package markdown

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/repopath"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/rumdl"
)

// FormatFlags holds flags for the Format task.
type FormatFlags struct {
	Check  bool   `flag:"check"  usage:"check only, don't write"`
	Config string `flag:"config" usage:"path to rumdl config file"`
}

// Format formats Markdown files using rumdl.
var Format = &pk.Task{
	Name:  "md-format",
	Usage: "format Markdown files",
	Flags: FormatFlags{},
	Body:  pk.Serial(rumdl.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[FormatFlags](ctx)

		args := []string{"fmt"}
		if f.Check {
			args = append(args, "--check")
		}
		if run.Verbose(ctx) {
			args = append(args, "--verbose")
		}

		// Fall back to the bundled config when the project has no rumdl
		// config of its own.
		configPath := f.Config
		if configPath == "" && !rumdl.HasProjectConfig() {
			configPath = rumdl.EnsureDefaultConfig()
		}
		if configPath != "" {
			args = append(args, "--config", configPath)
		}

		// Keep rumdl's cache inside .pocket to avoid polluting the repo.
		args = append(args, "--cache-dir", repopath.FromToolsDir("rumdl", "cache"))

		// Scan the whole repo from the git root; rumdl respects .gitignore.
		args = append(args, repopath.FromGitRoot("."))

		return run.Exec(ctx, rumdl.Name, args...)
	})
}
