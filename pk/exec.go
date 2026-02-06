package pk

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// waitDelay is the time to wait after sending SIGINT before sending SIGKILL.
const waitDelay = 5 * time.Second

// Do creates a Runnable that executes a dynamic function.
// Use this when command arguments need to be computed at runtime.
//
//	pk.Do(func(ctx context.Context) error {
//	    return pk.Exec(ctx, "golangci-lint", "run", "--fix", "./...")
//	})
func Do(fn func(ctx context.Context) error) Runnable {
	return &doRunnable{fn: fn}
}

var (
	colorEnvOnce sync.Once
	colorEnvVars []string

	// extraPATHDirs holds additional directories to add to PATH.
	// Used by tools that can't be symlinked (e.g., neovim on Windows).
	extraPATHDirs   []string
	extraPATHDirsMu sync.Mutex
)

// colorForceEnvVars are the environment variables set to force color output.
var colorForceEnvVars = []string{
	"FORCE_COLOR=1",       // Node.js, chalk, many modern tools
	"CLICOLOR_FORCE=1",    // BSD/macOS convention
	"COLORTERM=truecolor", // Indicates color support
	// Git uses its own color config, override via env vars.
	"GIT_CONFIG_COUNT=1",
	"GIT_CONFIG_KEY_0=color.ui",
	"GIT_CONFIG_VALUE_0=always",
}

// initColorEnv detects if stdout is a TTY and prepares env vars to force colors.
func initColorEnv() {
	_, noColor := os.LookupEnv("NO_COLOR")
	if noColor {
		return
	}

	if isTerminal(os.Stdout) {
		colorEnvVars = colorForceEnvVars
	}
}

// isTerminal returns true if the given file is a terminal.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// Exec executes a command with .pocket/bin prepended to PATH.
// The command runs in the directory specified by PathFromContext(ctx).
//
// Environment variables can be customized using ContextWithEnv and ContextWithoutEnv:
//
//	ctx = pk.ContextWithEnv(ctx, "MY_VAR=value")
//	ctx = pk.ContextWithoutEnv(ctx, "UNWANTED_PREFIX")
//	pk.Exec(ctx, "mycmd", "arg1")
//
// If verbose mode is enabled, command output is streamed to context output.
// Otherwise, output is captured and only shown on error.
//
// Commands are terminated gracefully: SIGINT first, then SIGKILL after WaitDelay.
func Exec(ctx context.Context, name string, args ...string) error {
	colorEnvOnce.Do(initColorEnv)

	path := PathFromContext(ctx)
	targetDir := FromGitRoot(path)
	env := applyEnvConfig(os.Environ(), EnvConfigFromContext(ctx))
	env = prependBinToPath(env)

	// Look up the command in our modified PATH, not the current process's PATH.
	// exec.CommandContext uses LookPath with the current PATH, which doesn't
	// include .pocket/bin yet.
	resolvedName := lookPathInEnv(name, env)

	cmd := exec.CommandContext(ctx, resolvedName, args...)
	cmd.Dir = targetDir
	cmd.Env = env
	cmd.Env = append(cmd.Env, colorEnvVars...)
	cmd.Stdin = nil // Prevent subprocess from reading stdin (avoids CI hangs)
	cmd.WaitDelay = waitDelay
	setGracefulShutdown(cmd)

	out := outputFromContext(ctx)

	if Verbose(ctx) {
		cmd.Stdout = out.Stdout
		cmd.Stderr = out.Stderr
		return cmd.Run()
	}

	// Capture output and only show on error (or if warnings detected).
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err != nil {
		// Include output in error for debugging.
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, buf.String())
	}

	// Even on success, show output if it contains warnings/notices.
	if output := buf.String(); containsNotice(output, noticePatternsFromContext(ctx)) {
		_, _ = out.Stderr.Write([]byte(output))
		if tracker := executionTrackerFromContext(ctx); tracker != nil {
			tracker.markWarning()
		}
	}
	return nil
}

// DefaultNoticePatterns are the default substrings used to detect
// warning-like output from commands.
var DefaultNoticePatterns = []string{"warn", "deprecat", "notice", "caution", "error"}

// noticePatternsKey is the context key for custom notice patterns.
type noticePatternsKey struct{}

// noticePatternsFromContext returns the notice patterns from context.
// Returns DefaultNoticePatterns if not set.
func noticePatternsFromContext(ctx context.Context) []string {
	if patterns, ok := ctx.Value(noticePatternsKey{}).([]string); ok {
		return patterns
	}
	return DefaultNoticePatterns
}

// containsNotice returns true if the output contains any of the given patterns.
func containsNotice(output string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	lower := strings.ToLower(output)
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// RegisterPATH registers a directory to be added to PATH for all Exec calls.
// Use this for tools that can't be symlinked (e.g., neovim on Windows needs its runtime files).
func RegisterPATH(dir string) {
	extraPATHDirsMu.Lock()
	defer extraPATHDirsMu.Unlock()
	// Avoid duplicates.
	if slices.Contains(extraPATHDirs, dir) {
		return
	}
	extraPATHDirs = append(extraPATHDirs, dir)
}

// prependBinToPath adds .pocket/bin and registered directories to the front of PATH.
func prependBinToPath(environ []string) []string {
	binDir := FromBinDir()

	// Build list of directories to prepend: binDir first, then extra dirs.
	extraPATHDirsMu.Lock()
	dirs := make([]string, 0, 1+len(extraPATHDirs))
	dirs = append(dirs, binDir)
	dirs = append(dirs, extraPATHDirs...)
	extraPATHDirsMu.Unlock()

	// Build the prefix string.
	prefix := strings.Join(dirs, string(filepath.ListSeparator))

	result := make([]string, 0, len(environ))
	for _, env := range environ {
		if path, found := strings.CutPrefix(env, "PATH="); found {
			result = append(result, fmt.Sprintf("PATH=%s%c%s", prefix, filepath.ListSeparator, path))
		} else {
			result = append(result, env)
		}
	}
	return result
}

// applyEnvConfig applies environment variable overrides from the config.
// It filters out variables matching filter prefixes, then applies set overrides.
func applyEnvConfig(environ []string, cfg EnvConfig) []string {
	if len(cfg.Filter) == 0 && len(cfg.Set) == 0 {
		return environ
	}

	result := make([]string, 0, len(environ))
	for _, e := range environ {
		key, _, _ := strings.Cut(e, "=")

		// Skip if key matches any filter prefix
		filtered := false
		for _, prefix := range cfg.Filter {
			if strings.HasPrefix(key, prefix) {
				filtered = true
				break
			}
		}
		if filtered {
			continue
		}

		// Skip if key will be replaced by set
		if _, willReplace := cfg.Set[key]; willReplace {
			continue
		}

		result = append(result, e)
	}

	// Append set overrides
	for key, value := range cfg.Set {
		result = append(result, key+"="+value)
	}

	return result
}

// lookPathInEnv looks up a command in the PATH from the given environment.
// If the command contains a path separator, it is returned as-is.
// If the command is not found, the original name is returned (letting exec fail with a clear error).
func lookPathInEnv(name string, env []string) string {
	// If name contains a path separator, use it directly.
	if strings.ContainsRune(name, filepath.Separator) {
		return name
	}

	// Extract PATH from env.
	var pathEnv string
	for _, e := range env {
		if path, found := strings.CutPrefix(e, "PATH="); found {
			pathEnv = path
			break
		}
	}

	if pathEnv == "" {
		return name
	}

	// Search each directory in PATH.
	for dir := range strings.SplitSeq(pathEnv, string(filepath.ListSeparator)) {
		path := filepath.Join(dir, name)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			return path
		}
		// On Windows, binaries have .exe extension.
		if runtime.GOOS == Windows {
			exePath := path + ".exe"
			if fi, err := os.Stat(exePath); err == nil && !fi.IsDir() {
				return exePath
			}
		}
	}

	return name
}

// doRunnable wraps a function as a Runnable.
type doRunnable struct {
	fn func(ctx context.Context) error
}

func (d *doRunnable) run(ctx context.Context) error {
	return d.fn(ctx)
}
