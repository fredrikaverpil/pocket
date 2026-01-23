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

	// Default is ubuntu-latest only, so 2 tasks = 2 entries
	if len(output.Include) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(output.Include))
	}

	for _, entry := range output.Include {
		if entry.OS != "ubuntu-latest" {
			t.Errorf("expected os 'ubuntu-latest', got %q", entry.OS)
		}
		if entry.Shim != "./pok" {
			t.Errorf("expected shim './pok', got %q", entry.Shim)
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

	// Only non-hidden task
	if len(output.Include) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(output.Include))
	}
	if output.Include[0].Task != "lint" {
		t.Errorf("expected task 'lint', got %q", output.Include[0].Task)
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

	// Only non-manual task
	if len(output.Include) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(output.Include))
	}
	if output.Include[0].Task != "lint" {
		t.Errorf("expected task 'lint', got %q", output.Include[0].Task)
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

	if len(cfg.DefaultPlatforms) != 1 {
		t.Errorf("expected 1 default platform, got %d", len(cfg.DefaultPlatforms))
	}
	if cfg.DefaultPlatforms[0] != "ubuntu-latest" {
		t.Errorf("expected default platform 'ubuntu-latest', got %q", cfg.DefaultPlatforms[0])
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
