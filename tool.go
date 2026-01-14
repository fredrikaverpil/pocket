package pocket

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Tool is a Task that installs a binary and provides methods to run it.
//
// Tool embeds *Task, so it satisfies Runnable automatically. The embedded
// task is hidden from CLI and handles installation with deduplication.
//
// Example:
//
//	var Tool = pocket.NewTool("golangci-lint", version, install).
//	    WithConfig(pocket.ToolConfig{
//	        UserFiles:   []string{".golangci.yml"},
//	        DefaultFile: "golangci.yml",
//	        DefaultData: defaultConfig,
//	    })
//
//	func install(ctx context.Context, tc *pocket.TaskContext) error {
//	    return pocket.Download(ctx, tc, url, opts)
//	}
type Tool struct {
	*Task           // embedded - Tool IS-A Task
	binary  string  // binary name (e.g., "golangci-lint")
	version string
	config  *ToolConfig
}

// ToolConfig describes how to find or create a tool's configuration file.
type ToolConfig struct {
	// UserFiles are filenames to search for in the repo root.
	// Checked in order; first match wins.
	UserFiles []string
	// DefaultFile is the filename for the bundled default config,
	// written to .pocket/tools/<name>/ if no user config exists.
	DefaultFile string
	// DefaultData is the bundled default configuration content.
	DefaultData []byte
}

// NewTool creates a tool definition.
//
// The install function has the same signature as task actions, giving it
// full access to TaskContext for output and running other tools.
//
// Example:
//
//	// renovate: datasource=github-releases depName=golangci/golangci-lint
//	const version = "2.7.1"
//
//	var Tool = pocket.NewTool("golangci-lint", version, install)
//
//	func install(ctx context.Context, tc *pocket.TaskContext) error {
//	    // ... installation logic
//	}
func NewTool(name, version string, install TaskAction) *Tool {
	if name == "" {
		panic("pocket.NewTool: name is required")
	}
	if version == "" {
		panic("pocket.NewTool: version is required")
	}
	if install == nil {
		panic("pocket.NewTool: install function is required")
	}
	// Create hidden install task. Version in name ensures unique dedup key.
	taskName := "install:" + name + "@" + version
	return &Tool{
		Task:    NewTask(taskName, "install "+name, install).AsHidden(),
		binary:  name,
		version: version,
	}
}

// WithConfig returns a new Tool with configuration file handling.
func (t *Tool) WithConfig(cfg ToolConfig) *Tool {
	return &Tool{
		Task:    t.Task,
		binary:  t.binary,
		version: t.version,
		config:  &cfg,
	}
}

// BinaryName returns the tool's binary name (without .exe extension).
func (t *Tool) BinaryName() string {
	return t.binary
}

// Version returns the tool's version string.
func (t *Tool) Version() string {
	return t.version
}

// ConfigPath returns the path to the tool's config file.
// It checks for user config files in the repo root first,
// then falls back to writing the bundled default config.
//
// Returns empty string and no error if the tool has no config.
func (t *Tool) ConfigPath() (string, error) {
	if t.config == nil {
		return "", nil
	}

	// Check for user config in repo root.
	for _, configName := range t.config.UserFiles {
		repoConfig := FromGitRoot(configName)
		if _, err := os.Stat(repoConfig); err == nil {
			return repoConfig, nil
		}
	}

	// Write bundled config to .pocket/tools/<name>/<default-file>.
	configDir := FromToolsDir(t.binary)
	configPath := filepath.Join(configDir, t.config.DefaultFile)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return "", fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(configPath, t.config.DefaultData, 0o644); err != nil {
			return "", fmt.Errorf("write default config: %w", err)
		}
	}

	return configPath, nil
}

// binaryPath returns the full path to the tool's binary in .pocket/bin/.
func (t *Tool) binaryPath() string {
	return FromBinDir(BinaryName(t.binary))
}

// Run is promoted from embedded *Task - installs the tool if needed.
// Deduplication is automatic: the same tool is only installed once per execution.

// Tasks overrides the embedded Task.Tasks to return nil.
// This hides tools from CLI task listings.
func (t *Tool) Tasks() []*Task {
	return nil
}

// Exec runs the tool binary with the given arguments.
// Installs the tool first if needed.
//
// Example:
//
//	func lintAction(ctx context.Context, tc *pocket.TaskContext) error {
//	    return golangcilint.Tool.Exec(ctx, tc, "run", "./...")
//	}
func (t *Tool) Exec(ctx context.Context, tc *TaskContext, args ...string) error {
	cmd, err := t.Command(ctx, tc, args...)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// Command returns an exec.Cmd for the tool binary.
// Installs the tool first if needed.
//
// Use this when you need to customize the command (e.g., set Dir) before running.
//
// Example:
//
//	cmd, err := golangcilint.Tool.Command(ctx, tc, "run", "./...")
//	if err != nil {
//	    return err
//	}
//	cmd.Dir = pocket.FromGitRoot(dir)
//	return cmd.Run()
func (t *Tool) Command(ctx context.Context, tc *TaskContext, args ...string) (*exec.Cmd, error) {
	if err := t.Run(ctx, tc.Execution()); err != nil {
		return nil, fmt.Errorf("install %s: %w", t.binary, err)
	}
	return tc.Command(ctx, t.binaryPath(), args...), nil
}
