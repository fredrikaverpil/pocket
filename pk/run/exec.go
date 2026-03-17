package run

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
)

// WaitDelay is the time to wait after sending SIGINT before sending SIGKILL.
const WaitDelay = 5 * time.Second

// WarningMarker is implemented by types that can record warnings.
// Used by [Exec] to mark warnings without importing the tracker's package.
type WarningMarker interface {
	MarkWarning()
}

var (
	colorEnvOnce sync.Once
	colorEnvVars []string

	// extraPATHDirs holds additional directories to add to PATH.
	extraPATHDirs   []string
	extraPATHDirsMu sync.Mutex
)

// colorForceEnvVars are the environment variables set to force color output.
var colorForceEnvVars = []string{
	"FORCE_COLOR=1",
	"CLICOLOR_FORCE=1",
	"COLORTERM=truecolor",
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

	if IsTerminal(os.Stdout) {
		colorEnvVars = colorForceEnvVars
	}
}

// DefaultNoticePatterns are the substrings used to detect warning-like output
// from commands when not in verbose mode.
var DefaultNoticePatterns = []string{"warn", "deprecat", "notice", "caution", "error"}

// Exec runs an external command with .pocket/bin prepended to PATH.
func Exec(ctx context.Context, name string, args ...string) error {
	colorEnvOnce.Do(initColorEnv)

	path := PathFromContext(ctx)
	targetDir := path
	if !filepath.IsAbs(path) {
		targetDir = repopath.FromGitRoot(path)
	}
	env := ApplyEnvConfig(os.Environ(), EnvConfigFromContext(ctx))
	env = PrependBinToPath(env)

	resolvedName := LookPathInEnv(name, env)

	cmd := exec.CommandContext(ctx, resolvedName, args...)
	cmd.Dir = targetDir
	cmd.Env = env
	cmd.Env = append(cmd.Env, colorEnvVars...)
	cmd.Stdin = nil
	cmd.WaitDelay = WaitDelay
	setGracefulShutdown(cmd)

	out := outputOrStd(ctx)

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

	patterns := noticePatternsFromContext(ctx)
	if patterns == nil {
		patterns = DefaultNoticePatterns
	}
	if output := buf.String(); ContainsNotice(output, patterns) {
		_, _ = out.Stderr.Write([]byte(output))
		if tracker := trackerFromContext(ctx); tracker != nil {
			if wm, ok := tracker.(WarningMarker); ok {
				wm.MarkWarning()
			}
		}
	}
	return nil
}

// RegisterPATH registers a directory to be added to PATH for all [Exec] calls.
func RegisterPATH(dir string) {
	extraPATHDirsMu.Lock()
	defer extraPATHDirsMu.Unlock()
	if slices.Contains(extraPATHDirs, dir) {
		return
	}
	extraPATHDirs = append(extraPATHDirs, dir)
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

	extraPATHDirsMu.Lock()
	dirs := make([]string, 0, 1+len(extraPATHDirs))
	dirs = append(dirs, binDir)
	dirs = append(dirs, extraPATHDirs...)
	extraPATHDirsMu.Unlock()

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
