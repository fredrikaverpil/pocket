package engine

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

	"github.com/fredrikaverpil/pocket/pk/repopath"
	"golang.org/x/term"
)

// WaitDelay is the time to wait after sending SIGINT before sending SIGKILL.
const WaitDelay = 5 * time.Second

var (
	ColorEnvOnce sync.Once
	ColorEnvVars []string

	// ExtraPATHDirs holds additional directories to add to PATH.
	ExtraPATHDirs   []string
	ExtraPATHDirsMu sync.Mutex
)

// ColorForceEnvVars are the environment variables set to force color output.
var ColorForceEnvVars = []string{
	"FORCE_COLOR=1",
	"CLICOLOR_FORCE=1",
	"COLORTERM=truecolor",
	"GIT_CONFIG_COUNT=1",
	"GIT_CONFIG_KEY_0=color.ui",
	"GIT_CONFIG_VALUE_0=always",
}

// InitColorEnv detects if stdout is a TTY and prepares env vars to force colors.
func InitColorEnv() {
	_, noColor := os.LookupEnv("NO_COLOR")
	if noColor {
		return
	}

	if IsTerminal(os.Stdout) {
		ColorEnvVars = ColorForceEnvVars
	}
}

// IsTerminal returns true if the given file is a terminal.
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// DefaultNoticePatterns are the substrings used to detect warning-like output
// from commands when not in verbose mode.
var DefaultNoticePatterns = []string{"warn", "deprecat", "notice", "caution", "error"}

// Exec runs an external command with .pocket/bin prepended to PATH.
func Exec(ctx context.Context, name string, args ...string) error {
	ColorEnvOnce.Do(InitColorEnv)

	path := PathFromContext(ctx)
	targetDir := path
	if !filepath.IsAbs(path) {
		targetDir = repopath.FromGitRoot(path)
	}
	env := ApplyEnvConfig(os.Environ(), EnvConfigFromContext(ctx))
	env = PrependBinToPath(env)

	resolvedName := LookPathInEnv(name, env)

	//nolint:gosec // Task runner executes user-configured commands by design.
	cmd := exec.CommandContext(ctx, resolvedName, args...)
	cmd.Dir = targetDir
	cmd.Env = env
	cmd.Env = append(cmd.Env, ColorEnvVars...)
	cmd.Stdin = nil
	cmd.WaitDelay = WaitDelay
	SetGracefulShutdown(cmd)

	out := outputFromContext(ctx)

	if Verbose(ctx) {
		cmd.Stdout = out.Stdout
		cmd.Stderr = out.Stderr
		return cmd.Run()
	}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, buf.String())
	}

	patterns := NoticePatternsFromContext(ctx)
	if patterns == nil {
		patterns = DefaultNoticePatterns
	}
	if output := buf.String(); ContainsNotice(output, patterns) {
		_, _ = out.Stderr.Write([]byte(output))
		if tracker := TrackerFromContext(ctx); tracker != nil {
			if wm, ok := tracker.(WarningMarker); ok {
				wm.MarkWarning()
			}
		}
	}
	return nil
}

// RegisterPATH registers a directory to be added to PATH for all Exec calls.
func RegisterPATH(dir string) {
	ExtraPATHDirsMu.Lock()
	defer ExtraPATHDirsMu.Unlock()
	if slices.Contains(ExtraPATHDirs, dir) {
		return
	}
	ExtraPATHDirs = append(ExtraPATHDirs, dir)
}

// ContainsNotice returns true if the output contains any of the given patterns.
func ContainsNotice(output string, patterns []string) bool {
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

// PrependBinToPath adds .pocket/bin and registered directories to the front of PATH.
func PrependBinToPath(environ []string) []string {
	binDir := repopath.FromBinDir()

	ExtraPATHDirsMu.Lock()
	dirs := make([]string, 0, 1+len(ExtraPATHDirs))
	dirs = append(dirs, binDir)
	dirs = append(dirs, ExtraPATHDirs...)
	ExtraPATHDirsMu.Unlock()

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

// ApplyEnvConfig applies environment variable overrides from the config.
func ApplyEnvConfig(environ []string, cfg EnvConfig) []string {
	if len(cfg.Filter) == 0 && len(cfg.Set) == 0 {
		return environ
	}

	result := make([]string, 0, len(environ))
	for _, e := range environ {
		key, _, _ := strings.Cut(e, "=")

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

		if _, willReplace := cfg.Set[key]; willReplace {
			continue
		}

		result = append(result, e)
	}

	for key, value := range cfg.Set {
		result = append(result, key+"="+value)
	}

	return result
}

// LookPathInEnv looks up a command in the PATH from the given environment.
func LookPathInEnv(name string, env []string) string {
	if strings.ContainsRune(name, filepath.Separator) {
		return name
	}

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

	for dir := range strings.SplitSeq(pathEnv, string(filepath.ListSeparator)) {
		path := filepath.Join(dir, name)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			return path
		}
		if runtime.GOOS == "windows" {
			exePath := path + ".exe"
			if fi, err := os.Stat(exePath); err == nil && !fi.IsDir() {
				return exePath
			}
		}
	}

	return name
}
