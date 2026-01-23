package pk

import (
	"context"
	"testing"
)

func TestShouldSkipGitDiff_NilConfig(t *testing.T) {
	tracker := newExecutionTracker()
	tracker.markDone("test-task", ".")

	// nil config means default behavior (run git diff)
	if shouldSkipGitDiff(nil, tracker) {
		t.Error("expected git diff to run with nil config")
	}
}

func TestShouldSkipGitDiff_DisableByDefault_NoRules(t *testing.T) {
	tracker := newExecutionTracker()
	tracker.markDone("test-task", ".")

	// DisableByDefault with no rules = skip everything.
	cfg := &GitDiffConfig{DisableByDefault: true}
	if !shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to be skipped when DisableByDefault with no rules")
	}
}

func TestShouldSkipGitDiff_DisableByDefault_WithIncludeRule(t *testing.T) {
	lintTask := NewTask("lint", "lint code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("lint", ".")

	// DisableByDefault=true with include rules = opt-in mode.
	// Run git diff if ANY executed task matches a rule.
	cfg := &GitDiffConfig{
		DisableByDefault: true,
		Rules: []GitDiffRule{
			{Task: lintTask}, // include lint
		},
	}

	// lint is in the include list, so git diff should run
	if shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to run when task matches include rule")
	}
}

func TestShouldSkipGitDiff_DisableByDefault_TaskNotIncluded(t *testing.T) {
	lintTask := NewTask("lint", "lint code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("generate", ".") // generate not in include list

	cfg := &GitDiffConfig{
		DisableByDefault: true,
		Rules: []GitDiffRule{
			{Task: lintTask}, // only lint is included
		},
	}

	// generate is NOT in the include list, so skip git diff
	if !shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to be skipped when task not in include rules")
	}
}

func TestShouldSkipGitDiff_NoExecutions(t *testing.T) {
	tracker := newExecutionTracker()
	// No tasks executed

	cfg := &GitDiffConfig{}
	if shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to run with no executions")
	}
}

func TestShouldSkipGitDiff_NoRules(t *testing.T) {
	tracker := newExecutionTracker()
	tracker.markDone("test-task", ".")

	cfg := &GitDiffConfig{} // No rules = run git diff for all
	if shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to run with no rules")
	}
}

func TestShouldSkipGitDiff_TaskMatchesRule(t *testing.T) {
	generateTask := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("generate", ".")

	cfg := &GitDiffConfig{
		Rules: []GitDiffRule{
			{Task: generateTask}, // skip for all paths
		},
	}

	if !shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to be skipped when task matches rule")
	}
}

func TestShouldSkipGitDiff_TaskDoesNotMatchRule(t *testing.T) {
	generateTask := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))
	_ = NewTask("lint", "lint code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("lint", ".")

	cfg := &GitDiffConfig{
		Rules: []GitDiffRule{
			{Task: generateTask}, // skip for generate, not lint
		},
	}

	if shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to run when task doesn't match rule")
	}
}

func TestShouldSkipGitDiff_PathMatchesRule(t *testing.T) {
	generateTask := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("generate", "generated/code")

	cfg := &GitDiffConfig{
		Rules: []GitDiffRule{
			{Task: generateTask, Paths: []string{"generated/.*"}},
		},
	}

	if !shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to be skipped when task+path matches rule")
	}
}

func TestShouldSkipGitDiff_PathDoesNotMatchRule(t *testing.T) {
	generateTask := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("generate", "src/code") // different path

	cfg := &GitDiffConfig{
		Rules: []GitDiffRule{
			{Task: generateTask, Paths: []string{"generated/.*"}},
		},
	}

	if shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to run when path doesn't match rule")
	}
}

func TestShouldSkipGitDiff_MixedExecutions(t *testing.T) {
	generateTask := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))
	_ = NewTask("lint", "lint code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("generate", ".")
	tracker.markDone("lint", ".") // lint is NOT in rules list

	cfg := &GitDiffConfig{
		Rules: []GitDiffRule{
			{Task: generateTask},
		},
	}

	// Should NOT skip because lint ran and is not in rules list
	if shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to run when some executions don't match rules")
	}
}

func TestShouldSkipGitDiff_AllExecutionsMatchRules(t *testing.T) {
	generateTask := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))
	formatTask := NewTask("format", "format code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	tracker := newExecutionTracker()
	tracker.markDone("generate", ".")
	tracker.markDone("format", ".")

	cfg := &GitDiffConfig{
		Rules: []GitDiffRule{
			{Task: generateTask},
			{Task: formatTask},
		},
	}

	// Should skip because all executed tasks are in rules list
	if !shouldSkipGitDiff(cfg, tracker) {
		t.Error("expected git diff to be skipped when all executions match rules")
	}
}

func TestShouldSkipGitDiff_NilTracker(t *testing.T) {
	cfg := &GitDiffConfig{}

	// nil tracker means we can't determine what ran, so run git diff
	if shouldSkipGitDiff(cfg, nil) {
		t.Error("expected git diff to run with nil tracker")
	}
}

func TestMatchesRule_TaskMatchAllPaths(t *testing.T) {
	task := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	rules := []GitDiffRule{
		{Task: task}, // no paths = all paths
	}

	tests := []struct {
		taskName string
		path     string
		want     bool
	}{
		{"generate", ".", true},
		{"generate", "foo/bar", true},
		{"generate", "any/path", true},
		{"lint", ".", false}, // different task
	}

	for _, tt := range tests {
		exec := executedTaskPath{TaskName: tt.taskName, Path: tt.path}
		got := matchesRule(exec, rules)
		if got != tt.want {
			t.Errorf("matchesRule(%s:%s) = %v, want %v", tt.taskName, tt.path, got, tt.want)
		}
	}
}

func TestMatchesRule_TaskWithPaths(t *testing.T) {
	task := NewTask("generate", "generate code", nil, Do(func(ctx context.Context) error {
		return nil
	}))

	rules := []GitDiffRule{
		{Task: task, Paths: []string{"generated/.*", "proto/.*"}},
	}

	tests := []struct {
		taskName string
		path     string
		want     bool
	}{
		{"generate", "generated/code", true},
		{"generate", "proto/api", true},
		{"generate", "src/code", false},   // path doesn't match
		{"generate", ".", false},          // root doesn't match
		{"lint", "generated/code", false}, // different task
	}

	for _, tt := range tests {
		exec := executedTaskPath{TaskName: tt.taskName, Path: tt.path}
		got := matchesRule(exec, rules)
		if got != tt.want {
			t.Errorf("matchesRule(%s:%s) = %v, want %v", tt.taskName, tt.path, got, tt.want)
		}
	}
}

func TestMatchesRule_NilTask(t *testing.T) {
	rules := []GitDiffRule{
		{Task: nil}, // nil task should be skipped
	}

	exec := executedTaskPath{TaskName: "anything", Path: "."}
	if matchesRule(exec, rules) {
		t.Error("expected nil task rule to not match")
	}
}

func TestExecutionTracker_Executed(t *testing.T) {
	tracker := newExecutionTracker()
	tracker.markDone("task1", ".")
	tracker.markDone("task2", "foo")
	tracker.markDone("task1", "bar") // same task, different path

	executed := tracker.executed()
	if len(executed) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(executed))
	}

	// Check that all expected executions are present
	expected := map[string]bool{
		"task1:.":   true,
		"task2:foo": true,
		"task1:bar": true,
	}

	for _, exec := range executed {
		key := exec.TaskName + ":" + exec.Path
		if !expected[key] {
			t.Errorf("unexpected execution: %s", key)
		}
		delete(expected, key)
	}

	if len(expected) > 0 {
		t.Errorf("missing executions: %v", expected)
	}
}
