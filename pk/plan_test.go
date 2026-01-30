package pk

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestPlan_Tasks(t *testing.T) {
	allDirs := []string{".", "services", "services/api", "pkg"}

	newTask := func(name, usage string) *Task {
		return NewTask(name, usage, nil, Do(func(_ context.Context) error { return nil }))
	}

	t.Run("BasicTasks", func(t *testing.T) {
		task1 := newTask("lint", "lint code")
		task2 := newTask("test", "run tests")

		cfg := &Config{
			Auto: Parallel(task1, task2),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()
		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}

		// Find lint task
		var lint *TaskInfo
		for i := range tasks {
			if tasks[i].Name == "lint" {
				lint = &tasks[i]
				break
			}
		}
		if lint == nil {
			t.Fatal("expected to find lint task")
		}
		if lint.Usage != "lint code" {
			t.Errorf("expected usage 'lint code', got %q", lint.Usage)
		}
		if lint.Hidden {
			t.Error("expected hidden=false")
		}
		if lint.Manual {
			t.Error("expected manual=false")
		}
		// Without path filtering, tasks run at root
		if !slices.Contains(lint.Paths, ".") {
			t.Errorf("expected paths to contain '.', got %v", lint.Paths)
		}
	})

	t.Run("HiddenTask", func(t *testing.T) {
		task := newTask("internal", "internal task").Hidden()

		cfg := &Config{
			Auto: task,
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if !tasks[0].Hidden {
			t.Error("expected hidden=true")
		}
	})

	t.Run("ManualTask", func(t *testing.T) {
		task := newTask("deploy", "deploy to prod").Manual()

		cfg := &Config{
			Manual: []Runnable{task},
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if !tasks[0].Manual {
			t.Error("expected manual=true")
		}
	})

	t.Run("ManualTaskWithoutManualMethod", func(t *testing.T) {
		// Task in Config.Manual should be marked manual even without .Manual()
		task := newTask("setup", "one-time setup")

		cfg := &Config{
			Manual: []Runnable{task},
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if !tasks[0].Manual {
			t.Error("task in Config.Manual should be marked manual=true")
		}
	})

	t.Run("ManualTaskInAutoTree", func(t *testing.T) {
		// Task with .Manual() in Auto tree should also be marked manual
		task := newTask("optional", "optional step").Manual()

		cfg := &Config{
			Auto: task,
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if !tasks[0].Manual {
			t.Error("task with .Manual() in Auto should be marked manual=true")
		}
	})

	t.Run("WithPathFiltering", func(t *testing.T) {
		task := newTask("lint", "lint code")

		cfg := &Config{
			Auto: WithOptions(task, WithIncludePath("services")),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}

		// Should have resolved paths from path filtering
		if !slices.Contains(tasks[0].Paths, "services") {
			t.Errorf("expected paths to contain 'services', got %v", tasks[0].Paths)
		}
		if !slices.Contains(tasks[0].Paths, "services/api") {
			t.Errorf("expected paths to contain 'services/api', got %v", tasks[0].Paths)
		}
	})

	t.Run("NilPlan", func(t *testing.T) {
		var plan *Plan
		tasks := plan.Tasks()
		if tasks != nil {
			t.Errorf("expected nil, got %v", tasks)
		}
	})
}

func TestNewPlan_NestedFilters(t *testing.T) {
	allDirs := []string{
		".",
		"services",
		"services/api",
		"services/web",
		"pkg",
		"pkg/utils",
		"vendor",
		"vendor/dep",
	}

	// Helper to create a task
	newTask := func(name string) *Task {
		return NewTask(name, "usage", nil, Do(func(_ context.Context) error { return nil }))
	}

	t.Run("WithOptionsDefaultsToRoot", func(t *testing.T) {
		task := newTask("flag-only-task")

		// WithOptions with only flags (no path options) should default to root
		cfg := &Config{
			Auto: WithOptions(task), // No path options at all
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		info := plan.pathMappings["flag-only-task"]

		// Should only run at root, not in all directories
		if len(info.resolvedPaths) != 1 {
			t.Errorf("expected 1 path (root), got %d: %v", len(info.resolvedPaths), info.resolvedPaths)
		}
		if info.resolvedPaths[0] != "." {
			t.Errorf("expected path '.', got %q", info.resolvedPaths[0])
		}
	})

	t.Run("IntersectionOfInclusions", func(t *testing.T) {
		task := newTask("test-task")

		// Outer includes services, inner includes services/api and pkg
		// Intersection should be services/api
		cfg := &Config{
			Auto: WithOptions(
				WithOptions(task, WithIncludePath("services/api"), WithIncludePath("pkg")),
				WithIncludePath("services"),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		info := plan.pathMappings["test-task"]

		containsPkg := slices.Contains(info.resolvedPaths, "pkg")
		if containsPkg {
			t.Errorf("expected pkg to be excluded by outer filter, but it was present: %v", info.resolvedPaths)
		}
	})

	t.Run("WithSkipTask", func(t *testing.T) {
		task1 := newTask("task1")
		task2 := newTask("task2")

		cfg := &Config{
			Auto: WithOptions(
				Parallel(task1, task2),
				WithSkipTask(task1),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		if findTaskByName(plan, "task1") != nil {
			t.Error("expected task1 to be skipped")
		}
		if findTaskByName(plan, "task2") == nil {
			t.Error("expected task2 to NOT be skipped")
		}
	})

	t.Run("TaskSpecificExclude", func(t *testing.T) {
		task1 := newTask("task1")
		task2 := newTask("task2")

		cfg := &Config{
			Auto: WithOptions(
				Parallel(task1, task2),
				WithExcludeTask(task1, "pkg"),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		info1 := plan.pathMappings["task1"]
		info2 := plan.pathMappings["task2"]

		if slices.Contains(info1.resolvedPaths, "pkg") {
			t.Error("expected task1 to exclude pkg")
		}
		if !slices.Contains(info2.resolvedPaths, "pkg") {
			t.Error("expected task2 to include pkg")
		}
	})

	t.Run("RegexExclude", func(t *testing.T) {
		task := newTask("test-task")

		cfg := &Config{
			Auto: WithOptions(
				task,
				WithExcludePath("services/.*"),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		info := plan.pathMappings["test-task"]

		for _, p := range info.resolvedPaths {
			if slices.Contains([]string{"services/api", "services/web"}, p) {
				t.Errorf("expected %s to be excluded by regex services/.*", p)
			}
		}
		if !slices.Contains(info.resolvedPaths, "pkg") {
			t.Error("expected pkg to be included")
		}
	})

	t.Run("ComplexCombination", func(t *testing.T) {
		taskLint := newTask("go-lint")
		taskTest := newTask("go-test")
		taskExtra := newTask("extra")

		// Plan:
		// Outer: Exclude vendor/.* and skip "extra"
		// Inner: Include services/.* and pkg/.*, but exclude "go-test" in pkg/.*
		cfg := &Config{
			Auto: WithOptions(
				WithOptions(
					Parallel(taskLint, taskTest, taskExtra),
					WithIncludePath("^services", "^pkg"),
					WithExcludeTask(taskTest, "^pkg"),
				),
				WithExcludePath("^vendor"),
				WithSkipTask(taskExtra),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		// Verify "extra" is gone
		if findTaskByName(plan, "extra") != nil {
			t.Error("expected 'extra' task to be skipped globally")
		}

		// Verify "go-lint" paths: should be [services/api, services/web, pkg, pkg/utils]
		// and NOT contain anything from vendor/
		infoLint := plan.pathMappings["go-lint"]
		expectedLint := []string{"services/api", "services/web", "pkg", "pkg/utils"}
		for _, p := range expectedLint {
			if !slices.Contains(infoLint.resolvedPaths, p) {
				t.Errorf("expected go-lint to run in %s", p)
			}
		}
		if slices.Contains(infoLint.resolvedPaths, "vendor/dep") {
			t.Error("expected go-lint to NOT run in vendor/dep")
		}

		// Verify "go-test" paths: should be [services/api, services/web]
		// because it was excluded from pkg/.*
		infoTest := plan.pathMappings["go-test"]
		expectedTest := []string{"services/api", "services/web"}
		for _, p := range expectedTest {
			if !slices.Contains(infoTest.resolvedPaths, p) {
				t.Errorf("expected go-test to run in %s", p)
			}
		}
		if slices.Contains(infoTest.resolvedPaths, "pkg") || slices.Contains(infoTest.resolvedPaths, "pkg/utils") {
			t.Error("expected go-test to be excluded from pkg/.*")
		}
	})

	t.Run("ExcludesRemoveAllDetectedPaths", func(t *testing.T) {
		task := newTask("go-lint")

		// Detection finds services/api and services/web, but excludes remove them all.
		// This should return an error indicating a configuration problem.
		detectServices := func(dirs []string, _ string) []string {
			var result []string
			for _, d := range dirs {
				if d == "services/api" || d == "services/web" {
					result = append(result, d)
				}
			}
			return result
		}

		cfg := &Config{
			Auto: WithOptions(
				task,
				WithDetect(detectServices),
				WithExcludePath("services/api", "services/web"),
			),
		}

		_, err := newPlan(cfg, "/tmp", allDirs)
		if err == nil {
			t.Fatal("expected error when excludes remove all detected paths")
		}

		// Verify the error message is helpful
		errMsg := err.Error()
		if !strings.Contains(errMsg, "go-lint") {
			t.Errorf("expected error to mention task name 'go-lint', got: %v", err)
		}
		if !strings.Contains(errMsg, "excludes removed all") {
			t.Errorf("expected error to mention 'excludes removed all', got: %v", err)
		}
	})

	t.Run("TaskSpecificExcludeRemovesAllPaths", func(t *testing.T) {
		taskLint := newTask("go-lint")
		taskTest := newTask("go-test")

		// Detection finds services/api and services/web.
		// WithExcludeTask removes all paths for go-test only.
		// This should NOT error - it's intentional to skip go-test in those paths.
		detectServices := func(dirs []string, _ string) []string {
			var result []string
			for _, d := range dirs {
				if d == "services/api" || d == "services/web" {
					result = append(result, d)
				}
			}
			return result
		}

		cfg := &Config{
			Auto: WithOptions(
				Parallel(taskLint, taskTest),
				WithDetect(detectServices),
				WithExcludeTask(taskTest, "services/api", "services/web"),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// go-lint should run in both paths
		infoLint := plan.pathMappings["go-lint"]
		if len(infoLint.resolvedPaths) != 2 {
			t.Errorf(
				"expected go-lint to have 2 paths, got %d: %v",
				len(infoLint.resolvedPaths),
				infoLint.resolvedPaths,
			)
		}

		// go-test should have zero paths (excluded from all detected paths)
		infoTest := plan.pathMappings["go-test"]
		if len(infoTest.resolvedPaths) != 0 {
			t.Errorf(
				"expected go-test to have 0 paths, got %d: %v",
				len(infoTest.resolvedPaths),
				infoTest.resolvedPaths,
			)
		}
	})
}

func TestNewPlan_BuiltinConflict(t *testing.T) {
	allDirs := []string{"."}

	// Each builtin name should cause an error
	for _, b := range builtins {
		t.Run(b.Name(), func(t *testing.T) {
			task := NewTask(b.Name(), "conflicting task", nil, Do(func(_ context.Context) error {
				return nil
			}))

			cfg := &Config{Auto: task}
			_, err := newPlan(cfg, "/tmp", allDirs)

			if err == nil {
				t.Errorf("expected error for task named %q, got nil", b.Name())
			} else if !strings.Contains(err.Error(), "conflicts with builtin") {
				t.Errorf("expected 'conflicts with builtin' error, got: %v", err)
			}
		})
	}
}

func TestNewPlan_NoBuiltinConflict(t *testing.T) {
	allDirs := []string{"."}

	// These should NOT conflict (old builtin names are now available)
	validNames := []string{"build", "lint", "test", "clean", "generate", "update"}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			task := NewTask(name, "valid task", nil, Do(func(_ context.Context) error {
				return nil
			}))

			cfg := &Config{Auto: task}
			_, err := newPlan(cfg, "/tmp", allDirs)
			if err != nil {
				t.Errorf("unexpected error for task named %q: %v", name, err)
			}
		})
	}
}

func TestNewPlan_DuplicateTaskName(t *testing.T) {
	allDirs := []string{"."}

	t.Run("SameNameDifferentTasks", func(t *testing.T) {
		task1 := NewTask("lint", "first lint", nil, Do(func(_ context.Context) error {
			return nil
		}))
		task2 := NewTask("lint", "second lint", nil, Do(func(_ context.Context) error {
			return nil
		}))

		cfg := &Config{Auto: Serial(task1, task2)}
		_, err := newPlan(cfg, "/tmp", allDirs)

		if err == nil {
			t.Error("expected error for duplicate task name, got nil")
		} else if !strings.Contains(err.Error(), "duplicate task name") {
			t.Errorf("expected 'duplicate task name' error, got: %v", err)
		}
	})

	t.Run("SameTaskTwiceIsOK", func(t *testing.T) {
		// Same task instance used twice is fine (it's deduplicated in collection)
		task := NewTask("lint", "lint code", nil, Do(func(_ context.Context) error {
			return nil
		}))

		cfg := &Config{Auto: Serial(task, task)}
		_, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Errorf("unexpected error for same task used twice: %v", err)
		}
	})

	t.Run("DifferentSuffixesAreOK", func(t *testing.T) {
		// Same base name with different suffixes is fine
		task := NewTask("py-test", "test", nil, Do(func(_ context.Context) error {
			return nil
		}))

		cfg := &Config{
			Auto: Serial(
				WithOptions(task, WithName("3.9")),
				WithOptions(task, WithName("3.10")),
			),
		}
		_, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Errorf("unexpected error for different suffixes: %v", err)
		}
	})
}

func TestPlan_ContextValues(t *testing.T) {
	allDirs := []string{"."}

	// Define a context key for testing
	type versionKey struct{}

	t.Run("ContextValuesAreCapturedInTaskInstance", func(t *testing.T) {
		task := NewTask("py-test", "test", nil, Do(func(_ context.Context) error {
			return nil
		}))

		cfg := &Config{
			Auto: Serial(
				WithOptions(task, WithName("3.9"), WithContextValue(versionKey{}, "3.9")),
				WithOptions(task, WithName("3.10"), WithContextValue(versionKey{}, "3.10")),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		// Find the 3.9 instance and verify it has the correct context value
		instance39 := findTaskByName(plan, "py-test:3.9")
		if instance39 == nil {
			t.Fatal("expected to find py-test:3.9")
		}
		if len(instance39.contextValues) != 1 {
			t.Errorf("expected 1 context value, got %d", len(instance39.contextValues))
		} else if instance39.contextValues[0].value != "3.9" {
			t.Errorf("expected context value '3.9', got %v", instance39.contextValues[0].value)
		}

		// Find the 3.10 instance and verify it has the correct context value
		instance310 := findTaskByName(plan, "py-test:3.10")
		if instance310 == nil {
			t.Fatal("expected to find py-test:3.10")
		}
		if len(instance310.contextValues) != 1 {
			t.Errorf("expected 1 context value, got %d", len(instance310.contextValues))
		} else if instance310.contextValues[0].value != "3.10" {
			t.Errorf("expected context value '3.10', got %v", instance310.contextValues[0].value)
		}
	})

	t.Run("NestedContextValuesAccumulate", func(t *testing.T) {
		type otherKey struct{}

		task := NewTask("test", "test", nil, Do(func(_ context.Context) error {
			return nil
		}))

		cfg := &Config{
			Auto: WithOptions(
				WithOptions(task, WithContextValue(otherKey{}, "inner")),
				WithContextValue(versionKey{}, "outer"),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		instance := findTaskByName(plan, "test")
		if instance == nil {
			t.Fatal("expected to find test")
		}

		// Should have both context values (outer first, then inner)
		if len(instance.contextValues) != 2 {
			t.Errorf("expected 2 context values, got %d", len(instance.contextValues))
		}
	})
}

func TestNewPlan_ExplicitPath(t *testing.T) {
	// allDirs does NOT include ".tests/stable" — explicit path bypasses resolution.
	allDirs := []string{".", "services", "pkg"}

	newTask := func(name string) *Task {
		return NewTask(name, "usage", nil, Do(func(_ context.Context) error { return nil }))
	}

	t.Run("ReturnsExplicitPathEvenWhenNotInAllDirs", func(t *testing.T) {
		task := newTask("nvim-test")

		cfg := &Config{
			Auto: WithOptions(task, WithExplicitPath(".tests/stable")),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		info := plan.pathMappings["nvim-test"]
		if len(info.resolvedPaths) != 1 {
			t.Fatalf("expected 1 resolved path, got %d: %v", len(info.resolvedPaths), info.resolvedPaths)
		}
		if info.resolvedPaths[0] != ".tests/stable" {
			t.Errorf("expected resolved path '.tests/stable', got %q", info.resolvedPaths[0])
		}
	})

	t.Run("ExplicitPathOverridesIncludes", func(t *testing.T) {
		task := newTask("nvim-test-override")

		// Both explicit path and include are set; explicit should win.
		cfg := &Config{
			Auto: WithOptions(task,
				WithExplicitPath(".tests/nightly"),
				WithIncludePath("services"),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		info := plan.pathMappings["nvim-test-override"]
		if len(info.resolvedPaths) != 1 {
			t.Fatalf("expected 1 resolved path, got %d: %v", len(info.resolvedPaths), info.resolvedPaths)
		}
		if info.resolvedPaths[0] != ".tests/nightly" {
			t.Errorf("expected resolved path '.tests/nightly', got %q", info.resolvedPaths[0])
		}
	})
}
