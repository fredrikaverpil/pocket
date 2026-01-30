package github

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fredrikaverpil/pocket/pk"
)

func TestGenerateMatrix_Default(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := DefaultMatrixConfig()
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Default is all three platforms, so 2 tasks × 3 platforms = 6 entries
	if len(output.Include) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(output.Include))
	}

	// Verify all platforms are present for each task
	platforms := make(map[string]int)
	for _, entry := range output.Include {
		platforms[entry.OS]++
	}
	for _, p := range []string{"ubuntu-latest", "macos-latest", "windows-latest"} {
		if platforms[p] != 2 { // 2 tasks per platform
			t.Errorf("expected 2 entries for platform %q, got %d", p, platforms[p])
		}
	}
}

func TestGenerateMatrix_MultiplePlatforms(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "test", Usage: "run tests"},
	}

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
	}
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// 1 task * 3 platforms = 3 entries
	if len(output.Include) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(output.Include))
	}

	// Check that we have all platforms
	platforms := make(map[string]bool)
	for _, entry := range output.Include {
		platforms[entry.OS] = true
	}

	expected := []string{"ubuntu-latest", "macos-latest", "windows-latest"}
	for _, p := range expected {
		if !platforms[p] {
			t.Errorf("expected platform %q in output", p)
		}
	}
}

func TestGenerateMatrix_TaskOverrides(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
		TaskOverrides: map[string]TaskOverride{
			"lint": {Platforms: []string{"ubuntu-latest"}}, // lint only on ubuntu
		},
	}
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// lint: 1 platform, test: 3 platforms = 4 entries
	if len(output.Include) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(output.Include))
	}

	// Count lint entries
	lintCount := 0
	for _, entry := range output.Include {
		if entry.Task == "lint" {
			lintCount++
			if entry.OS != "ubuntu-latest" {
				t.Errorf("lint should only run on ubuntu-latest, got %q", entry.OS)
			}
		}
	}
	if lintCount != 1 {
		t.Errorf("expected 1 lint entry, got %d", lintCount)
	}
}

func TestGenerateMatrix_ExcludeTasks(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "format", Usage: "format code"},
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest"},
		ExcludeTasks:     []string{"format"},
	}
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// 3 tasks - 1 excluded = 2 entries
	if len(output.Include) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(output.Include))
	}

	for _, entry := range output.Include {
		if entry.Task == "format" {
			t.Error("format task should be excluded")
		}
	}
}

func TestGenerateMatrix_HiddenTasksExcluded(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code", Hidden: false},
		{Name: "install:tool", Usage: "install tool", Hidden: true},
	}

	cfg := DefaultMatrixConfig()
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Only non-hidden task × 3 platforms = 3 entries
	if len(output.Include) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(output.Include))
	}
	for _, entry := range output.Include {
		if entry.Task != "lint" {
			t.Errorf("expected task 'lint', got %q", entry.Task)
		}
	}
}

func TestGenerateMatrix_ManualTasksExcluded(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code", Manual: false},
		{Name: "deploy", Usage: "deploy to prod", Manual: true},
	}

	cfg := DefaultMatrixConfig()
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Only non-manual task × 3 platforms = 3 entries
	if len(output.Include) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(output.Include))
	}
	for _, entry := range output.Include {
		if entry.Task != "lint" {
			t.Errorf("expected task 'lint', got %q", entry.Task)
		}
	}
}

func TestGenerateMatrix_WindowsShim(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "test", Usage: "run tests"},
	}

	tests := []struct {
		name         string
		windowsShell string
		windowsShim  string
		wantShim     string
	}{
		{"default", "", "", ".\\pok.ps1"},
		{"powershell_ps1", "powershell", "ps1", ".\\pok.ps1"},
		{"powershell_cmd", "powershell", "cmd", ".\\pok.cmd"},
		{"bash", "bash", "", "./pok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := MatrixConfig{
				DefaultPlatforms: []string{"windows-latest"},
				WindowsShell:     tt.windowsShell,
				WindowsShim:      tt.windowsShim,
			}
			data, err := GenerateMatrix(tasks, cfg)
			if err != nil {
				t.Fatalf("GenerateMatrix() failed: %v", err)
			}

			var output matrixOutput
			if err := json.Unmarshal(data, &output); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if len(output.Include) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(output.Include))
			}
			if output.Include[0].Shim != tt.wantShim {
				t.Errorf("expected shim %q, got %q", tt.wantShim, output.Include[0].Shim)
			}
		})
	}
}

func TestGenerateMatrix_ShimForPlatform(t *testing.T) {
	tests := []struct {
		platform     string
		windowsShell string
		windowsShim  string
		want         string
	}{
		{"ubuntu-latest", "powershell", "ps1", "./pok"},
		{"macos-latest", "powershell", "ps1", "./pok"},
		{"windows-latest", "powershell", "ps1", ".\\pok.ps1"},
		{"windows-latest", "powershell", "cmd", ".\\pok.cmd"},
		{"windows-2022", "powershell", "ps1", ".\\pok.ps1"},
		{"windows-2022", "powershell", "cmd", ".\\pok.cmd"},
		{"windows-latest", "bash", "ps1", "./pok"}, // bash ignores windowsShim
		{"windows-latest", "bash", "cmd", "./pok"}, // bash ignores windowsShim
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

func TestGenerateMatrix_EmptyTasks(t *testing.T) {
	cfg := DefaultMatrixConfig()
	data, err := GenerateMatrix(nil, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(output.Include) != 0 {
		t.Errorf("expected 0 entries, got %d", len(output.Include))
	}

	// Verify the JSON structure is correct for GHA
	expected := `{"include":[]}`
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestDefaultMatrixConfig(t *testing.T) {
	cfg := DefaultMatrixConfig()

	expectedPlatforms := []string{"ubuntu-latest", "macos-latest", "windows-latest"}
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

func TestGenerateMatrix_TaskOverridesRegexp(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "py-test:3.9", Usage: "test python 3.9"},
		{Name: "py-test:3.10", Usage: "test python 3.10"},
		{Name: "py-test:3.11", Usage: "test python 3.11"},
		{Name: "go-lint", Usage: "lint go code"},
	}

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
		TaskOverrides: map[string]TaskOverride{
			"py-test:.*": {Platforms: []string{"ubuntu-latest"}}, // regexp: match all py-test variants
			"go-lint":    {Platforms: []string{"ubuntu-latest"}},
		},
	}
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// py-test:3.9, py-test:3.10, py-test:3.11: 1 platform each = 3 entries
	// go-lint: 1 platform = 1 entry
	// Total: 4 entries
	if len(output.Include) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(output.Include))
	}

	// Verify py-test tasks have platform override applied
	for _, entry := range output.Include {
		if strings.HasPrefix(entry.Task, "py-test:") {
			if entry.OS != "ubuntu-latest" {
				t.Errorf("%s should only run on ubuntu-latest (matched by py-test:.*), got %q", entry.Task, entry.OS)
			}
		} else if entry.Task == "go-lint" {
			if entry.OS != "ubuntu-latest" {
				t.Errorf("go-lint should only run on ubuntu-latest, got %q", entry.OS)
			}
		}
	}
}

func TestGetTaskOverride(t *testing.T) {
	overrides := map[string]TaskOverride{
		"py-test:.*":  {Platforms: []string{"ubuntu-latest"}},
		"go-.*":       {Platforms: []string{"macos-latest"}},
		"exact-match": {Platforms: []string{"windows-latest"}},
	}

	tests := []struct {
		taskName      string
		wantMatch     bool
		wantPlatforms []string
	}{
		{"py-test:3.9", true, []string{"ubuntu-latest"}},
		{"py-test:3.10", true, []string{"ubuntu-latest"}},
		{"py-test:3.11", true, []string{"ubuntu-latest"}},
		{"go-lint", true, []string{"macos-latest"}},
		{"go-test", true, []string{"macos-latest"}},
		{"go-format", true, []string{"macos-latest"}},
		{"exact-match", true, []string{"windows-latest"}},
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
		"[invalid": {Platforms: []string{"macos-latest"}}, // invalid regexp
		"valid":    {Platforms: []string{"ubuntu-latest"}},
	}

	// Should not panic, just skip invalid patterns
	override := getTaskOverride("valid", overrides)
	if len(override.Platforms) != 1 || override.Platforms[0] != "ubuntu-latest" {
		t.Errorf("expected valid pattern to match, got %+v", override)
	}

	// Invalid pattern should be skipped
	_ = getTaskOverride("[invalid", overrides)
	// This might or might not match depending on iteration order, but shouldn't panic
}

func TestGenerateMatrix_GitDiff_Default(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := DefaultMatrixConfig()

	// Default config = gitDiff enabled for all tasks
	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for _, entry := range output.Include {
		if !entry.GitDiff {
			t.Errorf("expected gitDiff=true for %s, got false", entry.Task)
		}
	}
}

func TestGenerateMatrix_GitDiff_Disabled(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest"},
		DisableGitDiff:   true,
	}

	data, err := GenerateMatrix(tasks, cfg)
	if err != nil {
		t.Fatalf("GenerateMatrix() failed: %v", err)
	}

	var output matrixOutput
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for _, entry := range output.Include {
		if entry.GitDiff {
			t.Errorf("expected gitDiff=false for %s when DisableGitDiff=true, got true", entry.Task)
		}
	}
}

// Tests for static job generation

func TestGenerateStaticJobs_Default(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := DefaultMatrixConfig()
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

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest"},
	}
	jobs := GenerateStaticJobs(tasks, cfg)

	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	job := jobs[0]
	if job.ID != "go-test-ubuntu" {
		t.Errorf("expected ID 'go-test-ubuntu', got %q", job.ID)
	}
	if job.Name != "go-test (ubuntu-latest)" {
		t.Errorf("expected Name 'go-test (ubuntu-latest)', got %q", job.Name)
	}
}

func TestGenerateStaticJobs_TaskOverrides(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
		TaskOverrides: map[string]TaskOverride{
			"lint": {Platforms: []string{"ubuntu-latest"}}, // lint only on ubuntu
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
			if job.Platform != "ubuntu-latest" {
				t.Errorf("lint should only run on ubuntu-latest, got %q", job.Platform)
			}
		}
	}
	if lintCount != 1 {
		t.Errorf("expected 1 lint job, got %d", lintCount)
	}
}

func TestGenerateStaticJobs_ExcludeTasks(t *testing.T) {
	tasks := []pk.TaskInfo{
		{Name: "format", Usage: "format code"},
		{Name: "lint", Usage: "lint code"},
		{Name: "test", Usage: "run tests"},
	}

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest"},
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

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest"},
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

	cfg := MatrixConfig{
		DefaultPlatforms: []string{"ubuntu-latest"},
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
	cfg := DefaultMatrixConfig()
	cfg.DefaultPlatforms = []string{"ubuntu-latest"}
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
			cfg := MatrixConfig{
				DefaultPlatforms: []string{"windows-latest"},
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
	cfg := DefaultMatrixConfig()
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
		{"go-test", "ubuntu-latest", "go-test-ubuntu"},
		{"go-test", "macos-latest", "go-test-macos"},
		{"go-test", "windows-latest", "go-test-windows"},
		{"py-test:3.9", "ubuntu-latest", "py-test-3-9-ubuntu"},
		{"py-test:3.10", "macos-latest", "py-test-3-10-macos"},
		{"lint", "ubuntu-22.04", "lint-ubuntu-22-04"},
		{"test.unit", "ubuntu-latest", "test-unit-ubuntu"},
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
		{"ubuntu-latest", "ubuntu"},
		{"macos-latest", "macos"},
		{"windows-latest", "windows"},
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
