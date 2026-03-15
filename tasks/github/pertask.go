package github

import (
	"regexp"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
)

// Platform represents a GitHub Actions runner platform.
type Platform = string

// Common GitHub Actions runner platforms.
const (
	Ubuntu  Platform = "ubuntu-latest"
	MacOS   Platform = "macos-latest"
	Windows Platform = "windows-latest"
)

// AllPlatforms returns the three standard GitHub Actions runner platforms.
func AllPlatforms() []Platform {
	return []Platform{Ubuntu, MacOS, Windows}
}

// PerPocketTaskJobOption configures a single task in the per-task matrix.
// Keys in WorkflowFlags.PerPocketTaskJobOptions are treated as regular
// expressions and matched against task names.
type PerPocketTaskJobOption struct {
	// Platforms overrides the default platforms for this task.
	// Empty means use the default platforms from WorkflowFlags.Platforms.
	Platforms []Platform

	// Exclude removes this task from the matrix entirely.
	Exclude bool

	// GitDiff overrides the default git-diff setting for this task.
	// nil means use the default from WorkflowFlags.GitDiff.
	GitDiff *bool

	// CommitsCheck overrides the default commits-check setting for this task.
	// nil means use the default from WorkflowFlags.CommitsCheck.
	CommitsCheck *bool
}

// StaticJob represents a single job in the static workflow.
type StaticJob struct {
	ID           string   // GHA job identifier, e.g., "go-test-ubuntu".
	Name         string   // Display name, e.g., "go-test (ubuntu-latest)".
	Task         string   // Task name, e.g., "go-test".
	Platform     Platform // Runner platform, e.g., "ubuntu-latest".
	Shell        string   // Shell to use, "bash" or "pwsh".
	Shim         string   // Shim command, "./pok" or ".\\pok.ps1".
	GitDiff      bool     // Whether to check for uncommitted changes.
	CommitsCheck bool     // Whether to validate conventional commits.
}

// GenerateStaticJobs creates static job definitions from tasks and workflow flags.
func GenerateStaticJobs(tasks []pk.TaskInfo, flags WorkflowFlags) []StaticJob {
	platforms := flags.Platforms
	if len(platforms) == 0 {
		platforms = AllPlatforms()
	}

	var jobs []StaticJob
	for _, task := range tasks {
		// Skip hidden and manual tasks.
		if task.Hidden || task.Manual {
			continue
		}

		// Get option for this task (if any) using regexp matching.
		option := getTaskOption(task.Name, flags.PerPocketTaskJobOptions)

		// Skip excluded tasks.
		if option.Exclude {
			continue
		}

		// Determine platforms for this task.
		taskPlatforms := platforms
		if len(option.Platforms) > 0 {
			taskPlatforms = option.Platforms
		}

		// Determine git-diff for this task.
		gitDiff := boolVal(flags.GitDiff)
		if option.GitDiff != nil {
			gitDiff = *option.GitDiff
		}

		// Determine commits-check for this task.
		commitsCheck := boolVal(flags.CommitsCheck)
		if option.CommitsCheck != nil {
			commitsCheck = *option.CommitsCheck
		}

		// Create job for each platform.
		for _, platform := range taskPlatforms {
			jobs = append(jobs, StaticJob{
				ID:           jobID(task.Name, platform),
				Name:         task.Name + " (" + platform + ")",
				Task:         task.Name,
				Platform:     platform,
				Shell:        shellForPlatform(platform),
				Shim:         shimForPlatform(platform),
				GitDiff:      gitDiff,
				CommitsCheck: commitsCheck,
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
	// Sanitize task name: replace colons and dots with dashes.
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
	short := strings.TrimSuffix(platform, "-latest")
	short = strings.ReplaceAll(short, ".", "-")
	return short
}

// getTaskOption finds the PerPocketTaskJobOption for a task name by matching
// against the patterns in the options map. Patterns are regular expressions.
func getTaskOption(taskName string, options map[string]PerPocketTaskJobOption) PerPocketTaskJobOption {
	for pattern, option := range options {
		re, err := regexp.Compile("^" + pattern + "$")
		if err != nil {
			// Invalid pattern, skip.
			continue
		}
		if re.MatchString(taskName) {
			return option
		}
	}
	return PerPocketTaskJobOption{}
}

// shellForPlatform returns the appropriate shell for the platform.
func shellForPlatform(platform string) string {
	if strings.Contains(platform, "windows") {
		return "pwsh"
	}
	return "bash"
}

// shimForPlatform returns the appropriate shim command for the platform.
func shimForPlatform(platform string) string {
	if strings.Contains(platform, "windows") {
		return ".\\pok.ps1"
	}
	return "./pok"
}

// boolVal safely dereferences a *bool, returning false for nil.
func boolVal(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}
