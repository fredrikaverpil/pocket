package github

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/fredrikaverpil/pocket"
)

// MatrixConfig configures GitHub Actions matrix generation.
type MatrixConfig struct {
	// DefaultPlatforms for all tasks. Default: ["ubuntu-latest"]
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
}

// TaskOverride configures a single task in the matrix.
type TaskOverride struct {
	// Platforms overrides DefaultPlatforms for this task.
	// Empty means use DefaultPlatforms.
	Platforms []string

	// SkipGitDiff disables the git-diff check after this task.
	// Useful for tasks that intentionally modify files (e.g., code generators).
	SkipGitDiff bool
}

// DefaultMatrixConfig returns sensible defaults.
func DefaultMatrixConfig() MatrixConfig {
	return MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest"},
		WindowsShell:     "powershell",
		WindowsShim:      "ps1",
	}
}

// matrixEntry is a single entry in the GHA matrix.
type matrixEntry struct {
	Task    string `json:"task"`
	OS      string `json:"os"`
	Shell   string `json:"shell"`
	Shim    string `json:"shim"`
	GitDiff bool   `json:"gitDiff"` // whether to run git-diff after this task
}

// matrixOutput is the JSON structure for fromJson().
type matrixOutput struct {
	Include []matrixEntry `json:"include"`
}

// GenerateMatrix creates the GitHub Actions matrix JSON from tasks.
func GenerateMatrix(tasks []pocket.TaskInfo, cfg MatrixConfig) ([]byte, error) {
	if cfg.DefaultPlatforms == nil {
		cfg.DefaultPlatforms = []string{"ubuntu-latest"}
	}
	if cfg.WindowsShell == "" {
		cfg.WindowsShell = "cmd"
	}

	excludeSet := make(map[string]bool)
	for _, name := range cfg.ExcludeTasks {
		excludeSet[name] = true
	}

	entries := make([]matrixEntry, 0)
	for _, task := range tasks {
		// Skip hidden and excluded tasks
		if task.Hidden || excludeSet[task.Name] {
			continue
		}

		// Get override for this task (if any) using regexp matching
		override := getTaskOverride(task.Name, cfg.TaskOverrides)

		// Determine platforms for this task
		platforms := cfg.DefaultPlatforms
		if len(override.Platforms) > 0 {
			platforms = override.Platforms
		}

		// Determine if git-diff should run (default: true, unless overridden)
		gitDiff := !override.SkipGitDiff

		// Create entry for each platform
		for _, platform := range platforms {
			entries = append(entries, matrixEntry{
				Task:    task.Name,
				OS:      platform,
				Shell:   shellForPlatform(platform, cfg.WindowsShell),
				Shim:    shimForPlatform(platform, cfg.WindowsShell, cfg.WindowsShim),
				GitDiff: gitDiff,
			})
		}
	}

	return json.Marshal(matrixOutput{Include: entries})
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

// MatrixTask creates the gha-matrix task.
// Users pass their AutoRun and MatrixConfig to generate the matrix.
//
// Example usage in .pocket/config.go:
//
//	var autoRun = pocket.Parallel(
//	    pocket.RunIn(golang.Tasks(), pocket.Detect(golang.Detect())),
//	)
//
//	var Config = pocket.Config{
//	    AutoRun: autoRun,
//	    ManualRun: []pocket.Runnable{
//	        github.MatrixTask(autoRun, github.MatrixConfig{
//	            DefaultPlatforms: []string{"ubuntu-latest", "macos-latest"},
//	            TaskOverrides: map[string]github.TaskOverride{
//	                "go-lint": {Platforms: []string{"ubuntu-latest"}},
//	            },
//	        }),
//	    },
//	}
func MatrixTask(autoRun pocket.Runnable, cfg MatrixConfig) *pocket.TaskDef {
	return pocket.Task("gha-matrix", "output GitHub Actions matrix JSON",
		matrixCmd(autoRun, cfg),
		pocket.AsSilent(),
	)
}

func matrixCmd(autoRun pocket.Runnable, cfg MatrixConfig) pocket.Runnable {
	return pocket.Do(func(ctx context.Context) error {
		tasks, err := pocket.CollectTasks(autoRun)
		if err != nil {
			return err
		}
		data, err := GenerateMatrix(tasks, cfg)
		if err != nil {
			return err
		}
		pocket.Printf(ctx, "%s\n", data)
		return nil
	})
}
