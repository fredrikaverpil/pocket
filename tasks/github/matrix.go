package github

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
)

// MatrixConfig configures GitHub Actions matrix generation.
type MatrixConfig struct {
	// DefaultPlatforms for all tasks. Default: ["ubuntu-latest", "macos-latest", "windows-latest"]
	DefaultPlatforms []string

	// TaskOverrides provides per-task platform configuration.
	// Keys are treated as regular expressions and matched against task names.
	// Example: "py-test:.*" matches "py-test:3.9", "py-test:3.10", etc.
	TaskOverrides map[string]TaskOverride

	// ExcludeTasks removes tasks from the matrix entirely.
	ExcludeTasks []string

	// WindowsShell determines which shell to use on Windows.
	// Options: "powershell" (pwsh), "bash" (Git Bash)
	// Default: "powershell"
	WindowsShell string

	// WindowsShim determines which shim to use on Windows when WindowsShell is "powershell".
	// Options: "ps1" (pok.ps1), "cmd" (pok.cmd)
	// Default: "ps1"
	// Ignored when WindowsShell is "bash" (always uses ./pok).
	WindowsShim string

	// DisableGitDiff prevents -g from being added in CI.
	// Default (false): matrix outputs gitDiff: true for all tasks.
	DisableGitDiff bool
}

// TaskOverride configures a single task in the matrix.
type TaskOverride struct {
	// Platforms overrides DefaultPlatforms for this task.
	// Empty means use DefaultPlatforms.
	Platforms []string
}

// DefaultMatrixConfig returns sensible defaults.
func DefaultMatrixConfig() MatrixConfig {
	return MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
		WindowsShell:     "powershell",
		WindowsShim:      "ps1",
	}
}

// MatrixConfigKey is the context key for MatrixConfig.
// Use with pk.WithContextValue to configure static workflow generation:
//
//	pk.WithContextValue(github.MatrixConfigKey{}, github.MatrixConfig{...})
type MatrixConfigKey struct{}

// matrixConfigFromContext retrieves MatrixConfig from context.
// Returns DefaultMatrixConfig() if not set.
func matrixConfigFromContext(ctx context.Context) MatrixConfig {
	if cfg, ok := ctx.Value(MatrixConfigKey{}).(MatrixConfig); ok {
		return cfg
	}
	return DefaultMatrixConfig()
}

// StaticJob represents a single job in the static workflow.
type StaticJob struct {
	ID       string // GHA job identifier, e.g., "go-test-ubuntu"
	Name     string // Display name, e.g., "go-test (ubuntu-latest)"
	Task     string // Task name, e.g., "go-test"
	Platform string // Runner platform, e.g., "ubuntu-latest"
	Shell    string // Shell to use, "bash" or "pwsh"
	Shim     string // Shim command, "./pok" or ".\\pok.ps1"
	GitDiff  bool   // Whether to check for uncommitted changes
}

// GenerateStaticJobs creates static job definitions from tasks.
func GenerateStaticJobs(tasks []pk.TaskInfo, cfg MatrixConfig) []StaticJob {
	if cfg.DefaultPlatforms == nil {
		cfg.DefaultPlatforms = []string{"ubuntu-latest", "macos-latest", "windows-latest"}
	}
	if cfg.WindowsShell == "" {
		cfg.WindowsShell = "powershell"
	}
	if cfg.WindowsShim == "" {
		cfg.WindowsShim = "ps1"
	}

	excludeSet := make(map[string]bool)
	for _, name := range cfg.ExcludeTasks {
		excludeSet[name] = true
	}

	var jobs []StaticJob
	for _, task := range tasks {
		// Skip hidden, manual, and excluded tasks
		if task.Hidden || task.Manual || excludeSet[task.Name] {
			continue
		}

		// Get override for this task (if any) using regexp matching
		override := getTaskOverride(task.Name, cfg.TaskOverrides)

		// Determine platforms for this task
		platforms := cfg.DefaultPlatforms
		if len(override.Platforms) > 0 {
			platforms = override.Platforms
		}

		// Create job for each platform
		for _, platform := range platforms {
			jobs = append(jobs, StaticJob{
				ID:       jobID(task.Name, platform),
				Name:     task.Name + " (" + platform + ")",
				Task:     task.Name,
				Platform: platform,
				Shell:    shellForPlatform(platform, cfg.WindowsShell),
				Shim:     shimForPlatform(platform, cfg.WindowsShell, cfg.WindowsShim),
				GitDiff:  !cfg.DisableGitDiff,
			})
		}
	}

	return jobs
}

// jobID creates a valid GHA job identifier from task and platform.
// Colons and dots are replaced with dashes, platform is shortened.
// Examples:
//   - "go-test" + "ubuntu-latest" -> "go-test-ubuntu"
//   - "py-test:3.9" + "macos-latest" -> "py-test-3-9-macos"
//   - "lint" + "ubuntu-22.04" -> "lint-ubuntu-22-04"
func jobID(task, platform string) string {
	// Sanitize task name: replace colons and dots with dashes
	sanitized := strings.NewReplacer(":", "-", ".", "-").Replace(task)
	short := platformShort(platform)
	return sanitized + "-" + short
}

// platformShort extracts the short name from a platform string.
// Examples:
//   - "ubuntu-latest" -> "ubuntu"
//   - "macos-latest" -> "macos"
//   - "windows-latest" -> "windows"
//   - "ubuntu-22.04" -> "ubuntu-22-04"
func platformShort(platform string) string {
	// Remove "-latest" suffix if present
	short := strings.TrimSuffix(platform, "-latest")
	// Replace dots with dashes for version numbers
	short = strings.ReplaceAll(short, ".", "-")
	return short
}

// getTaskOverride finds the TaskOverride for a task name by matching against
// the patterns in TaskOverrides. Patterns are regular expressions.
func getTaskOverride(taskName string, overrides map[string]TaskOverride) TaskOverride {
	for pattern, override := range overrides {
		re, err := regexp.Compile("^" + pattern + "$")
		if err != nil {
			// Invalid pattern, skip
			continue
		}
		if re.MatchString(taskName) {
			return override
		}
	}
	return TaskOverride{}
}

// shellForPlatform returns the appropriate shell for the platform.
func shellForPlatform(platform, windowsShell string) string {
	if strings.Contains(platform, "windows") {
		switch windowsShell {
		case "bash":
			return "bash"
		default:
			return "pwsh"
		}
	}
	return "bash"
}

// shimForPlatform returns the appropriate shim command for the platform.
func shimForPlatform(platform, windowsShell, windowsShim string) string {
	if strings.Contains(platform, "windows") {
		switch windowsShell {
		case "bash":
			return "./pok"
		default:
			// powershell - use windowsShim to determine ps1 or cmd
			switch windowsShim {
			case "cmd":
				return ".\\pok.cmd"
			default:
				return ".\\pok.ps1"
			}
		}
	}
	return "./pok"
}

// Deprecated types and functions for dynamic matrix generation.
// These are kept for backwards compatibility but should not be used.
// Use GenerateStaticJobs instead for static workflow generation.

// matrixEntry is a single entry in the GHA matrix.
// Deprecated: Use StaticJob and GenerateStaticJobs instead.
type matrixEntry struct {
	Task    string `json:"task"`
	OS      string `json:"os"`
	Shell   string `json:"shell"`
	Shim    string `json:"shim"`
	GitDiff bool   `json:"gitDiff"`
}

// matrixOutput is the JSON structure for fromJson().
// Deprecated: Use StaticJob and GenerateStaticJobs instead.
type matrixOutput struct {
	Include []matrixEntry `json:"include"`
}

// GenerateMatrix creates the GitHub Actions matrix JSON from tasks.
// Deprecated: Use GenerateStaticJobs instead for static workflow generation.
func GenerateMatrix(tasks []pk.TaskInfo, cfg MatrixConfig) ([]byte, error) {
	if cfg.DefaultPlatforms == nil {
		cfg.DefaultPlatforms = []string{"ubuntu-latest", "macos-latest", "windows-latest"}
	}
	if cfg.WindowsShell == "" {
		cfg.WindowsShell = "powershell"
	}
	if cfg.WindowsShim == "" {
		cfg.WindowsShim = "ps1"
	}

	excludeSet := make(map[string]bool)
	for _, name := range cfg.ExcludeTasks {
		excludeSet[name] = true
	}

	entries := make([]matrixEntry, 0)
	for _, task := range tasks {
		// Skip hidden, manual, and excluded tasks
		if task.Hidden || task.Manual || excludeSet[task.Name] {
			continue
		}

		// Get override for this task (if any) using regexp matching
		override := getTaskOverride(task.Name, cfg.TaskOverrides)

		// Determine platforms for this task
		platforms := cfg.DefaultPlatforms
		if len(override.Platforms) > 0 {
			platforms = override.Platforms
		}

		// Create entry for each platform
		for _, platform := range platforms {
			entries = append(entries, matrixEntry{
				Task:    task.Name,
				OS:      platform,
				Shell:   shellForPlatform(platform, cfg.WindowsShell),
				Shim:    shimForPlatform(platform, cfg.WindowsShell, cfg.WindowsShim),
				GitDiff: !cfg.DisableGitDiff,
			})
		}
	}

	return json.Marshal(matrixOutput{Include: entries})
}
