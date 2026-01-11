// Package lua provides Lua-related build tasks.
package lua

import (
	"context"
	"fmt"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tools/stylua"
)

// TasksOption configures the lua task group.
type TasksOption func(*tasksConfig)

type tasksConfig struct {
	format FormatOptions
}

// WithFormat sets options for the lua-format task.
func WithFormat(opts FormatOptions) TasksOption {
	return func(c *tasksConfig) { c.format = opts }
}

// Tasks returns a Runnable that executes all Lua tasks.
// Runs from repository root since Lua files are typically scattered.
// Use pocket.AutoDetect(lua.Tasks()) to enable path filtering.
//
// Example with options:
//
//	pocket.AutoDetect(lua.Tasks(
//	    lua.WithFormat(lua.FormatOptions{StyluaConfig: ".stylua.toml"}),
//	))
func Tasks(opts ...TasksOption) pocket.Runnable {
	var cfg tasksConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	format := FormatTask().WithOptions(cfg.format)

	return pocket.NewTaskGroup(format).
		DetectBy(func() []string { return []string{"."} })
}

// FormatOptions configures the lua-format task.
type FormatOptions struct {
	StyluaConfig string `usage:"path to stylua config file"`
}

// FormatTask returns a task that formats Lua files using stylua.
// Use WithOptions to set project-level configuration.
func FormatTask() *pocket.Task {
	return pocket.NewTask("lua-format", "format Lua files", formatAction)
}

// formatAction is the action for the lua-format task.
func formatAction(ctx context.Context, tc *pocket.TaskContext) error {
	opts := pocket.GetOptions[FormatOptions](tc)
	configPath := opts.StyluaConfig
	if configPath == "" {
		var err error
		configPath, err = stylua.ConfigPath()
		if err != nil {
			return fmt.Errorf("get stylua config: %w", err)
		}
	}
	return tc.ForEachPath(ctx, func(dir string) error {
		absDir := pocket.FromGitRoot(dir)

		needsFormat, checkOutput, err := formatCheck(ctx, configPath, absDir)
		if err != nil {
			return err
		}
		if !needsFormat {
			tc.Out.Println("No files in need of formatting.")
			return nil
		}

		// Show diff in verbose mode.
		if tc.Verbose && len(checkOutput) > 0 {
			tc.Out.Printf("%s", checkOutput)
		}

		// Now actually format.
		if err := stylua.Run(ctx, "-f", configPath, absDir); err != nil {
			return fmt.Errorf("stylua format failed in %s: %w", dir, err)
		}
		tc.Out.Println("Formatted files.")
		return nil
	})
}

// formatCheck runs stylua --check to see if formatting is needed.
// Returns true if files need formatting, along with the check output.
func formatCheck(ctx context.Context, configPath, dir string) (needsFormat bool, output []byte, err error) {
	cmd, err := stylua.Command(ctx, "--check", "-f", configPath, dir)
	if err != nil {
		return false, nil, fmt.Errorf("prepare stylua: %w", err)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	output, checkErr := cmd.CombinedOutput()
	return checkErr != nil, output, nil
}
