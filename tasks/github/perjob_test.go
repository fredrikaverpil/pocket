package github

import (
	"strings"
	"testing"

	"github.com/fredrikaverpil/pocket/pk"
)

func TestDefaultPerJobConfig(t *testing.T) {
	cfg := DefaultPerJobConfig()

	expectedPlatforms := AllPlatforms()
	if len(cfg.DefaultPlatforms) != len(expectedPlatforms) {
		t.Errorf("expected %d default platforms, got %d", len(expectedPlatforms), len(cfg.DefaultPlatforms))
	}
	for i, p := range expectedPlatforms {
		if i < len(cfg.DefaultPlatforms) && cfg.DefaultPlatforms[i] != p {
			t.Errorf("expected platform[%d] = %q, got %q", i, p, cfg.DefaultPlatforms[i])
		}
	}
	if cfg.WindowsShell != "powershell" {
		t.Errorf("expected default WindowsShell 'powershell', got %q", cfg.WindowsShell)
	}
	if cfg.WindowsShim != "ps1" {
		t.Errorf("expected default WindowsShim 'ps1', got %q", cfg.WindowsShim)
	}
}

func TestGetTaskOverride(t *testing.T) {
	overrides := map[string]TaskOverride{
		"py-test:.*":  {Platforms: []string{PlatformUbuntu}},
		"go-.*":       {Platforms: []string{PlatformMacOS}},
		"exact-match": {Platforms: []string{PlatformWindows}},
	}

	tests := []struct {
		taskName      string
		wantMatch     bool
		wantPlatforms []string
	}{
		{"py-test:3.9", true, []string{PlatformUbuntu}},
		{"py-test:3.10", true, []string{PlatformUbuntu}},
		{"py-test:3.11", true, []string{PlatformUbuntu}},
		{"go-lint", true, []string{PlatformMacOS}},
		{"go-test", true, []string{PlatformMacOS}},
		{"go-format", true, []string{PlatformMacOS}},
		{"exact-match", true, []string{PlatformWindows}},
		{"no-match", false, nil},
		{"py-test", false, nil}, // doesn't match "py-test:.*" (requires colon)
	}

	for _, tt := range tests {
		t.Run(tt.taskName, func(t *testing.T) {
			override := getTaskOverride(tt.taskName, overrides)
			gotMatch := len(override.Platforms) > 0
			if gotMatch != tt.wantMatch {
				t.Errorf("getTaskOverride(%q) match=%v, want match=%v", tt.taskName, gotMatch, tt.wantMatch)
			}
			if tt.wantPlatforms != nil && len(override.Platforms) > 0 {
				if override.Platforms[0] != tt.wantPlatforms[0] {
					t.Errorf(
						"getTaskOverride(%q) Platforms=%v, want %v",
						tt.taskName,
						override.Platforms,
						tt.wantPlatforms,
					)
				}
			}
		})
	}
}

func TestGetTaskOverride_InvalidRegexp(t *testing.T) {
	overrides := map[string]TaskOverride{
		"[invalid": {Platforms: []string{PlatformMacOS}},  // invalid regexp
		"valid":    {Platforms: []string{PlatformUbuntu}},
	}

	// Should not panic, just skip invalid patterns
	override := getTaskOverride("valid", overrides)
	if len(override.Platforms) != 1 || override.Platforms[0] != PlatformUbuntu {
		t.Errorf("expected valid pattern to match, got %+v", override)
	}

	// Invalid pattern should be skipped
	_ = getTaskOverride("[invalid", overrides)
	// This might or might not match depending on iteration order, but shouldn't panic
}

func TestShimForPlatform(t *testing.T) {
	tests := []struct {
		platform     string
		windowsShell string
		windowsShim  string
		want         string
	}{
		{PlatformUbuntu, "powershell", "ps1", "./pok"},
		{PlatformMacOS, "powershell", "ps1", "./pok"},
		{PlatformWindows, "powershell", "ps1", ".\\pok.ps1"},
		{PlatformWindows, "powershell", "cmd", ".\\pok.cmd"},
		{"windows-2022", "powershell", "ps1", ".\\pok.ps1"},
		{"windows-2022", "powershell", "cmd", ".\\pok.cmd"},
		{PlatformWindows, "bash", "ps1", "./pok"}, // bash ignores windowsShim
		{PlatformWindows, "bash", "cmd", "./pok"}, // bash ignores windowsShim
		{"ubuntu-22.04", "bash", "ps1", "./pok"},
	}

	for _, tt := range tests {
		t.Run(tt.platform+"_"+tt.windowsShell+"_"+tt.windowsShim, func(t *testing.T) {
			got := shimForPlatform(tt.platform, tt.windowsShell, tt.windowsShim)
			if got != tt.want {
				t.Errorf("shimForPlatform(%q, %q, %q) = %q, want %q",
					tt.platform, tt.windowsShell, tt.windowsShim, got, tt.want)
			}
		})
	}
}

// Tests for static job generation

func TestGenerateStaticJobs_Default(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := DefaultPerJobConfig()
	jobs := GenerateStaticJobs(tasks, cfg)

	// 2 tasks × 3 platforms = 6 jobs
	if len(jobs) != 6 {
		t.Fatalf("expected 6 jobs, got %d", len(jobs))
	}

	// Verify job structure
	for _, job := range jobs {
		if job.ID == "" {
			t.Error("job ID should not be empty")
		}
		if job.Name == "" {
			t.Error("job Name should not be empty")
		}
		if job.Task == "" {
			t.Error("job Task should not be empty")
		}
		if job.Platform == "" {
			t.Error("job Platform should not be empty")
		}
		if job.Shell == "" {
			t.Error("job Shell should not be empty")
		}
		if job.Shim == "" {
			t.Error("job Shim should not be empty")
		}
	}
}

func TestGenerateStaticJobs_JobID(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "go-test", Usage: "test go code"},
	}

	cfg := PerJobConfig{
		DefaultPlatforms: []string{PlatformUbuntu},
	}
	jobs := GenerateStaticJobs(tasks, cfg)

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]
	if job.ID != "go-test-ubuntu" {
		t.Errorf("expected ID 'go-test-ubuntu', got %q", job.ID)
	}
	if job.Name != "go-test ("+PlatformUbuntu+")" {
		t.Errorf("expected Name 'go-test (%s)', got %q", PlatformUbuntu, job.Name)
	}
}

func TestGenerateStaticJobs_TaskOverrides(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := PerJobConfig{
		DefaultPlatforms: AllPlatforms(),
		TaskOverrides: map[string]TaskOverride{
			"lint": {Platforms: []string{PlatformUbuntu}}, // lint only on ubuntu
		},
	}
	jobs := GenerateStaticJobs(tasks, cfg)

	// lint: 1 platform, test: 3 platforms = 4 jobs
	if len(jobs) != 4 {
		t.Fatalf("expected 4 jobs, got %d", len(jobs))
	}

	// Count lint jobs
	lintCount := 0
	for _, job := range jobs {
		if job.Task == "lint" {
			lintCount++
			if job.Platform != PlatformUbuntu {
				t.Errorf("lint should only run on ubuntu-latest, got %q", job.Platform)
			}
		}
	}
	if lintCount != 1 {
		t.Errorf("expected 1 lint job, got %d", lintCount)
	}
}

func TestGenerateStaticJobs_TaskOverridesRegexp(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "py-test:3.9", Usage: "test python 3.9"},
		{Name: "py-test:3.10", Usage: "test python 3.10"},
		{Name: "py-test:3.11", Usage: "test python 3.11"},
		{Name: "go-lint", Usage: "lint go code"},
	}

	cfg := PerJobConfig{
		DefaultPlatforms: AllPlatforms(),
		TaskOverrides: map[string]TaskOverride{
			"py-test:.*": {Platforms: []string{PlatformUbuntu}}, // regexp: match all py-test variants
			"go-lint":    {Platforms: []string{PlatformUbuntu}},
		},
	}
	jobs := GenerateStaticJobs(tasks, cfg)

	// py-test:3.9, py-test:3.10, py-test:3.11: 1 platform each = 3 jobs
	// go-lint: 1 platform = 1 job
	// Total: 4 jobs
	if len(jobs) != 4 {
		t.Fatalf("expected 4 jobs, got %d", len(jobs))
	}

	// Verify py-test tasks have platform override applied
	for _, job := range jobs {
		if strings.HasPrefix(job.Task, "py-test:") {
			if job.Platform != PlatformUbuntu {
				t.Errorf("%s should only run on %s (matched by py-test:.*), got %q", job.Task, PlatformUbuntu, job.Platform)
			}
		} else if job.Task == "go-lint" {
			if job.Platform != PlatformUbuntu {
				t.Errorf("go-lint should only run on ubuntu-latest, got %q", job.Platform)
			}
		}
	}
}

func TestGenerateStaticJobs_ExcludeTasks(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "format", Usage: "format code"},
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := PerJobConfig{
		DefaultPlatforms: []string{PlatformUbuntu},
		ExcludeTasks:     []string{"format"},
	}
	jobs := GenerateStaticJobs(tasks, cfg)

	// 3 tasks - 1 excluded = 2 jobs
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	for _, job := range jobs {
		if job.Task == "format" {
			t.Error("format task should be excluded")
		}
	}
}

func TestGenerateStaticJobs_HiddenTasksExcluded(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code", Hidden: false},
		{Name: "install:tool", Usage: "install tool", Hidden: true},
	}

	cfg := PerJobConfig{
		DefaultPlatforms: []string{PlatformUbuntu},
	}
	jobs := GenerateStaticJobs(tasks, cfg)

	// Only non-hidden task = 1 job
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Task != "lint" {
		t.Errorf("expected task 'lint', got %q", jobs[0].Task)
	}
}

func TestGenerateStaticJobs_ManualTasksExcluded(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code", Manual: false},
		{Name: "deploy", Usage: "deploy to prod", Manual: true},
	}

	cfg := PerJobConfig{
		DefaultPlatforms: []string{PlatformUbuntu},
	}
	jobs := GenerateStaticJobs(tasks, cfg)

	// Only non-manual task = 1 job
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Task != "lint" {
		t.Errorf("expected task 'lint', got %q", jobs[0].Task)
	}
}

func TestGenerateStaticJobs_GitDiff(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
	}

	// Default: gitDiff enabled
	cfg := DefaultPerJobConfig()
	cfg.DefaultPlatforms = []string{PlatformUbuntu}
	jobs := GenerateStaticJobs(tasks, cfg)

	if !jobs[0].GitDiff {
		t.Error("expected GitDiff=true by default")
	}

	// Disabled
	cfg.DisableGitDiff = true
	jobs = GenerateStaticJobs(tasks, cfg)

	if jobs[0].GitDiff {
		t.Error("expected GitDiff=false when DisableGitDiff=true")
	}
}

func TestGenerateStaticJobs_WindowsShell(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "test", Usage: "run tests"},
	}

	tests := []struct {
		name         string
		windowsShell string
		windowsShim  string
		wantShell    string
		wantShim     string
	}{
		{"default", "", "", "pwsh", ".\\pok.ps1"},
		{"powershell_ps1", "powershell", "ps1", "pwsh", ".\\pok.ps1"},
		{"powershell_cmd", "powershell", "cmd", "pwsh", ".\\pok.cmd"},
		{"bash", "bash", "", "bash", "./pok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := PerJobConfig{
				DefaultPlatforms: []string{PlatformWindows},
				WindowsShell:     tt.windowsShell,
				WindowsShim:      tt.windowsShim,
			}
			jobs := GenerateStaticJobs(tasks, cfg)

			if len(jobs) != 1 {
				t.Fatalf("expected 1 job, got %d", len(jobs))
			}
			if jobs[0].Shell != tt.wantShell {
				t.Errorf("expected Shell %q, got %q", tt.wantShell, jobs[0].Shell)
			}
			if jobs[0].Shim != tt.wantShim {
				t.Errorf("expected Shim %q, got %q", tt.wantShim, jobs[0].Shim)
			}
		})
	}
}

func TestGenerateStaticJobs_Empty(t *testing.T) {
	cfg := DefaultPerJobConfig()
	jobs := GenerateStaticJobs(nil, cfg)

	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestJobID(t *testing.T) {
	tests := []struct {
		task     string
		platform string
		want     string
	}{
		{"go-test", PlatformUbuntu, "go-test-ubuntu"},
		{"go-test", PlatformMacOS, "go-test-macos"},
		{"go-test", PlatformWindows, "go-test-windows"},
		{"py-test:3.9", PlatformUbuntu, "py-test-3-9-ubuntu"},
		{"py-test:3.10", PlatformMacOS, "py-test-3-10-macos"},
		{"lint", "ubuntu-22.04", "lint-ubuntu-22-04"},
		{"test.unit", PlatformUbuntu, "test-unit-ubuntu"},
	}

	for _, tt := range tests {
		t.Run(tt.task+"_"+tt.platform, func(t *testing.T) {
			got := jobID(tt.task, tt.platform)
			if got != tt.want {
				t.Errorf("jobID(%q, %q) = %q, want %q", tt.task, tt.platform, got, tt.want)
			}
		})
	}
}

func TestPlatformShort(t *testing.T) {
	tests := []struct {
		platform string
		want     string
	}{
		{PlatformUbuntu, "ubuntu"},
		{PlatformMacOS, "macos"},
		{PlatformWindows, "windows"},
		{"ubuntu-22.04", "ubuntu-22-04"},
		{"macos-13", "macos-13"},
		{"windows-2022", "windows-2022"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			got := platformShort(tt.platform)
			if got != tt.want {
				t.Errorf("platformShort(%q) = %q, want %q", tt.platform, got, tt.want)
			}
		})
	}
}
