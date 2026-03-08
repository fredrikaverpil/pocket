package github

import (
	"strings"
	"testing"

	"github.com/fredrikaverpil/pocket/pk"
	"gotest.tools/v3/assert"
)

func TestGetTaskOption(t *testing.T) {
	options := map[string]PerPocketTaskJobOption{
		"py-test:.*":  {Platforms: []Platform{Ubuntu}},
		"go-.*":       {Platforms: []Platform{MacOS}},
		"exact-match": {Platforms: []Platform{Windows}},
	}

	tests := []struct {
		name          string
		taskName      string
		wantPlatforms []Platform
	}{
		{name: "regexp match py-test:3.9", taskName: "py-test:3.9", wantPlatforms: []Platform{Ubuntu}},
		{name: "regexp match py-test:3.10", taskName: "py-test:3.10", wantPlatforms: []Platform{Ubuntu}},
		{name: "regexp match py-test:3.11", taskName: "py-test:3.11", wantPlatforms: []Platform{Ubuntu}},
		{name: "regexp match go-lint", taskName: "go-lint", wantPlatforms: []Platform{MacOS}},
		{name: "regexp match go-test", taskName: "go-test", wantPlatforms: []Platform{MacOS}},
		{name: "regexp match go-format", taskName: "go-format", wantPlatforms: []Platform{MacOS}},
		{name: "exact match", taskName: "exact-match", wantPlatforms: []Platform{Windows}},
		{name: "no match", taskName: "no-match", wantPlatforms: nil},
		{name: "py-test without colon", taskName: "py-test", wantPlatforms: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTaskOption(tt.taskName, options)
			assert.DeepEqual(t, tt.wantPlatforms, got.Platforms)
		})
	}
}

func TestGetTaskOption_InvalidRegexp(t *testing.T) {
	options := map[string]PerPocketTaskJobOption{
		"[invalid": {Platforms: []Platform{MacOS}},
		"valid":    {Platforms: []Platform{Ubuntu}},
	}

	// Should not panic, just skip invalid patterns.
	got := getTaskOption("valid", options)
	assert.DeepEqual(t, []Platform{Ubuntu}, got.Platforms)

	// Invalid pattern should be skipped, not panic.
	_ = getTaskOption("[invalid", options)
}

func TestGenerateStaticJobs_Default(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	flags := WorkflowFlags{}
	jobs := GenerateStaticJobs(tasks, flags)

	// 2 tasks x 3 platforms = 6 jobs.
	assert.Equal(t, 6, len(jobs))

	// Verify all jobs have required fields populated.
	for _, job := range jobs {
		assert.Assert(t, job.ID != "", "job ID should not be empty")
		assert.Assert(t, job.Name != "", "job Name should not be empty")
		assert.Assert(t, job.Task != "", "job Task should not be empty")
		assert.Assert(t, job.Platform != "", "job Platform should not be empty")
		assert.Assert(t, job.Shell != "", "job Shell should not be empty")
		assert.Assert(t, job.Shim != "", "job Shim should not be empty")
	}
}

func TestGenerateStaticJobs_JobID(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "go-test", Usage: "test go code"},
	}

	flags := WorkflowFlags{
		Platforms: []Platform{Ubuntu},
	}
	jobs := GenerateStaticJobs(tasks, flags)

	assert.Equal(t, 1, len(jobs))

	got := jobs[0]
	want := StaticJob{
		ID:       "go-test-ubuntu",
		Name:     "go-test (" + Ubuntu + ")",
		Task:     "go-test",
		Platform: Ubuntu,
		Shell:    "bash",
		Shim:     "./pok",
		GitDiff:  false,
	}
	assert.DeepEqual(t, want, got)
}

func TestGenerateStaticJobs_PerPocketTaskJobOptions(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	flags := WorkflowFlags{
		Platforms: AllPlatforms(),
		PerPocketTaskJobOptions: map[string]PerPocketTaskJobOption{
			"lint": {Platforms: []Platform{Ubuntu}},
		},
	}
	jobs := GenerateStaticJobs(tasks, flags)

	// lint: 1 platform, test: 3 platforms = 4 jobs.
	assert.Equal(t, 4, len(jobs))

	// Count lint jobs and verify platform.
	lintCount := 0
	for _, job := range jobs {
		if job.Task == "lint" {
			lintCount++
			assert.Equal(t, Ubuntu, job.Platform)
		}
	}
	assert.Equal(t, 1, lintCount)
}

func TestGenerateStaticJobs_PerPocketTaskJobOptionsRegexp(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "py-test:3.9", Usage: "test python 3.9"},
		{Name: "py-test:3.10", Usage: "test python 3.10"},
		{Name: "py-test:3.11", Usage: "test python 3.11"},
		{Name: "go-lint", Usage: "lint go code"},
	}

	flags := WorkflowFlags{
		Platforms: AllPlatforms(),
		PerPocketTaskJobOptions: map[string]PerPocketTaskJobOption{
			"py-test:.*": {Platforms: []Platform{Ubuntu}},
			"go-lint":    {Platforms: []Platform{Ubuntu}},
		},
	}
	jobs := GenerateStaticJobs(tasks, flags)

	// py-test:3.9, py-test:3.10, py-test:3.11: 1 platform each = 3 jobs.
	// go-lint: 1 platform = 1 job.
	// Total: 4 jobs.
	assert.Equal(t, 4, len(jobs))

	for _, job := range jobs {
		if strings.HasPrefix(job.Task, "py-test:") {
			assert.Equal(t, Ubuntu, job.Platform,
				"%s should only run on %s (matched by py-test:.*)", job.Task, Ubuntu)
		} else if job.Task == "go-lint" {
			assert.Equal(t, Ubuntu, job.Platform,
				"go-lint should only run on %s", Ubuntu)
		}
	}
}

func TestGenerateStaticJobs_ExcludeTasks(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "format", Usage: "format code"},
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	flags := WorkflowFlags{
		Platforms: []Platform{Ubuntu},
		PerPocketTaskJobOptions: map[string]PerPocketTaskJobOption{
			"format": {Exclude: true},
		},
	}
	jobs := GenerateStaticJobs(tasks, flags)

	// 3 tasks - 1 excluded = 2 jobs.
	assert.Equal(t, 2, len(jobs))

	for _, job := range jobs {
		assert.Assert(t, job.Task != "format", "format task should be excluded")
	}
}

func TestGenerateStaticJobs_HiddenTasksExcluded(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code", Hidden: false},
		{Name: "install:tool", Usage: "install tool", Hidden: true},
	}

	flags := WorkflowFlags{
		Platforms: []Platform{Ubuntu},
	}
	jobs := GenerateStaticJobs(tasks, flags)

	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "lint", jobs[0].Task)
}

func TestGenerateStaticJobs_ManualTasksExcluded(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code", Manual: false},
		{Name: "deploy", Usage: "deploy to prod", Manual: true},
	}

	flags := WorkflowFlags{
		Platforms: []Platform{Ubuntu},
	}
	jobs := GenerateStaticJobs(tasks, flags)

	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "lint", jobs[0].Task)
}

func TestGenerateStaticJobs_GitDiff(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
	}

	t.Run("enabled", func(t *testing.T) {
		flags := WorkflowFlags{
			Platforms: []Platform{Ubuntu},
			GitDiff:   true,
		}
		jobs := GenerateStaticJobs(tasks, flags)
		assert.Assert(t, jobs[0].GitDiff, "expected GitDiff=true")
	})

	t.Run("disabled", func(t *testing.T) {
		flags := WorkflowFlags{
			Platforms: []Platform{Ubuntu},
			GitDiff:   false,
		}
		jobs := GenerateStaticJobs(tasks, flags)
		assert.Assert(t, !jobs[0].GitDiff, "expected GitDiff=false")
	})
}

func TestGenerateStaticJobs_GitDiffPerTaskOverride(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	gitDiffTrue := true
	gitDiffFalse := false

	t.Run("override_to_false", func(t *testing.T) {
		flags := WorkflowFlags{
			Platforms: []Platform{Ubuntu},
			GitDiff:   true,
			PerPocketTaskJobOptions: map[string]PerPocketTaskJobOption{
				"lint": {GitDiff: &gitDiffFalse},
			},
		}
		jobs := GenerateStaticJobs(tasks, flags)
		for _, job := range jobs {
			if job.Task == "lint" {
				assert.Assert(t, !job.GitDiff, "lint should have GitDiff=false (overridden)")
			} else {
				assert.Assert(t, job.GitDiff, "test should have GitDiff=true (default)")
			}
		}
	})

	t.Run("override_to_true", func(t *testing.T) {
		flags := WorkflowFlags{
			Platforms: []Platform{Ubuntu},
			GitDiff:   false,
			PerPocketTaskJobOptions: map[string]PerPocketTaskJobOption{
				"lint": {GitDiff: &gitDiffTrue},
			},
		}
		jobs := GenerateStaticJobs(tasks, flags)
		for _, job := range jobs {
			if job.Task == "lint" {
				assert.Assert(t, job.GitDiff, "lint should have GitDiff=true (overridden)")
			} else {
				assert.Assert(t, !job.GitDiff, "test should have GitDiff=false (default)")
			}
		}
	})
}

func TestGenerateStaticJobs_WindowsPlatform(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "test", Usage: "run tests"},
	}

	flags := WorkflowFlags{
		Platforms: []Platform{Windows},
	}
	jobs := GenerateStaticJobs(tasks, flags)

	assert.Equal(t, 1, len(jobs))
	assert.Equal(t, "pwsh", jobs[0].Shell)
	assert.Equal(t, ".\\pok.ps1", jobs[0].Shim)
}

func TestGenerateStaticJobs_Empty(t *testing.T) {
	flags := WorkflowFlags{}
	jobs := GenerateStaticJobs(nil, flags)

	assert.Equal(t, 0, len(jobs))
}

func TestJobID(t *testing.T) {
	tests := []struct {
		name     string
		task     string
		platform string
		want     string
	}{
		{name: "ubuntu", task: "go-test", platform: Ubuntu, want: "go-test-ubuntu"},
		{name: "macos", task: "go-test", platform: MacOS, want: "go-test-macos"},
		{name: "windows", task: "go-test", platform: Windows, want: "go-test-windows"},
		{name: "colon in task", task: "py-test:3.9", platform: Ubuntu, want: "py-test-3-9-ubuntu"},
		{name: "colon and version", task: "py-test:3.10", platform: MacOS, want: "py-test-3-10-macos"},
		{name: "custom platform", task: "lint", platform: "ubuntu-22.04", want: "lint-ubuntu-22-04"},
		{name: "dot in task", task: "test.unit", platform: Ubuntu, want: "test-unit-ubuntu"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jobID(tt.task, tt.platform)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlatformShort(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		want     string
	}{
		{name: "ubuntu-latest", platform: Ubuntu, want: "ubuntu"},
		{name: "macos-latest", platform: MacOS, want: "macos"},
		{name: "windows-latest", platform: Windows, want: "windows"},
		{name: "ubuntu-22.04", platform: "ubuntu-22.04", want: "ubuntu-22-04"},
		{name: "macos-13", platform: "macos-13", want: "macos-13"},
		{name: "windows-2022", platform: "windows-2022", want: "windows-2022"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platformShort(tt.platform)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShimForPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		want     string
	}{
		{name: "ubuntu", platform: Ubuntu, want: "./pok"},
		{name: "macos", platform: MacOS, want: "./pok"},
		{name: "windows", platform: Windows, want: ".\\pok.ps1"},
		{name: "windows-2022", platform: "windows-2022", want: ".\\pok.ps1"},
		{name: "ubuntu-22.04", platform: "ubuntu-22.04", want: "./pok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shimForPlatform(tt.platform)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShellForPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		want     string
	}{
		{name: "ubuntu", platform: Ubuntu, want: "bash"},
		{name: "macos", platform: MacOS, want: "bash"},
		{name: "windows", platform: Windows, want: "pwsh"},
		{name: "windows-2022", platform: "windows-2022", want: "pwsh"},
		{name: "ubuntu-22.04", platform: "ubuntu-22.04", want: "bash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellForPlatform(tt.platform)
			assert.Equal(t, tt.want, got)
		})
	}
}
