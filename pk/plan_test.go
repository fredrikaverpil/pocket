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
		return &Task{Name: name, Usage: usage, Do: func(_ context.Context) error { return nil }}
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
		task := &Task{
			Name:   "internal",
			Usage:  "internal task",
			Hidden: true,
			Do:     func(_ context.Context) error { return nil },
		}

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
		task := newTask("deploy", "deploy to prod")

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
		return &Task{Name: name, Usage: "usage", Do: func(_ context.Context) error { return nil }}
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

func TestNewPlan_DetectBasedShimGeneration(t *testing.T) {
	allDirs := []string{".", "services", "services/api", "services/web", "pkg"}

	newTask := func(name string) *Task {
		return &Task{Name: name, Usage: "usage", Do: func(_ context.Context) error { return nil }}
	}

	detectServices := func(dirs []string, _ string) []string {
		var result []string
		for _, d := range dirs {
			if d == "services/api" || d == "services/web" {
				result = append(result, d)
			}
		}
		return result
	}

	task := newTask("go-lint")

	cfg := &Config{
		Auto: WithOptions(task, WithDetect(detectServices)),
	}

	plan, err := newPlan(cfg, "/tmp", allDirs)
	if err != nil {
		t.Fatal(err)
	}

	// moduleDirectories should include detected paths.
	dirs := plan.moduleDirectories
	for _, expected := range []string{".", "services/api", "services/web"} {
		if !slices.Contains(dirs, expected) {
			t.Errorf("expected moduleDirectories to contain %q, got %v", expected, dirs)
		}
	}

	// taskRunsInPath should return true for detected paths.
	if !plan.taskRunsInPath("go-lint", "services/api") {
		t.Error("expected go-lint visible from services/api")
	}
	if !plan.taskRunsInPath("go-lint", "services/web") {
		t.Error("expected go-lint visible from services/web")
	}

	// taskRunsInPath should return false for non-detected paths.
	if plan.taskRunsInPath("go-lint", "pkg") {
		t.Error("expected go-lint NOT visible from pkg")
	}
}

func TestNewPlan_BuiltinConflict(t *testing.T) {
	allDirs := []string{"."}

	// Each builtin name should cause an error
	for _, b := range builtins {
		t.Run(b.Name, func(t *testing.T) {
			task := &Task{Name: b.Name, Usage: "conflicting task", Do: func(_ context.Context) error {
				return nil
			}}

			cfg := &Config{Auto: task}
			_, err := newPlan(cfg, "/tmp", allDirs)

			if err == nil {
				t.Errorf("expected error for task named %q, got nil", b.Name)
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
			task := &Task{Name: name, Usage: "valid task", Do: func(_ context.Context) error {
				return nil
			}}

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
		task1 := &Task{Name: "lint", Usage: "first lint", Do: func(_ context.Context) error {
			return nil
		}}
		task2 := &Task{Name: "lint", Usage: "second lint", Do: func(_ context.Context) error {
			return nil
		}}

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
		task := &Task{Name: "lint", Usage: "lint code", Do: func(_ context.Context) error {
			return nil
		}}

		cfg := &Config{Auto: Serial(task, task)}
		_, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Errorf("unexpected error for same task used twice: %v", err)
		}
	})

	t.Run("DifferentSuffixesAreOK", func(t *testing.T) {
		// Same base name with different suffixes is fine
		task := &Task{Name: "py-test", Usage: "test", Do: func(_ context.Context) error {
			return nil
		}}

		cfg := &Config{
			Auto: Serial(
				WithOptions(task, WithNameSuffix("3.9")),
				WithOptions(task, WithNameSuffix("3.10")),
			),
		}
		_, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Errorf("unexpected error for different suffixes: %v", err)
		}
	})
}

func TestNewPlan_InvalidRegexPattern(t *testing.T) {
	allDirs := []string{".", "services", "pkg"}
	task := &Task{Name: "test", Usage: "test", Do: func(_ context.Context) error { return nil }}

	t.Run("InvalidInclude", func(t *testing.T) {
		cfg := &Config{
			Auto: WithOptions(task, WithIncludePath("[invalid(regex")),
		}
		_, err := newPlan(cfg, "/tmp", allDirs)
		if err == nil {
			t.Fatal("expected error for invalid regex pattern")
		}
		if !strings.Contains(err.Error(), "invalid pattern") {
			t.Errorf("expected 'invalid pattern' error, got: %v", err)
		}
	})

	t.Run("InvalidExclude", func(t *testing.T) {
		cfg := &Config{
			Auto: WithOptions(task, WithIncludePath("services"), WithExcludePath("[bad")),
		}
		_, err := newPlan(cfg, "/tmp", allDirs)
		if err == nil {
			t.Fatal("expected error for invalid regex pattern")
		}
		if !strings.Contains(err.Error(), "invalid pattern") {
			t.Errorf("expected 'invalid pattern' error, got: %v", err)
		}
	})
}

// assertTaskNames verifies that plan.Tasks() produces the expected task names in order.
func assertTaskNames(t *testing.T, got []TaskInfo, want []string) {
	t.Helper()
	gotNames := make([]string, len(got))
	for i, ti := range got {
		gotNames[i] = ti.Name
	}
	if !slices.Equal(gotNames, want) {
		t.Errorf("task names mismatch\ngot:  %v\nwant: %v", gotNames, want)
	}
}

// findTaskInfo returns the TaskInfo with the given name, or nil if not found.
func findTaskInfo(tasks []TaskInfo, name string) *TaskInfo {
	for i := range tasks {
		if tasks[i].Name == name {
			return &tasks[i]
		}
	}
	return nil
}

func TestNewPlan_ComposedConfigs(t *testing.T) {
	noop := func(_ context.Context) error { return nil }

	t.Run("Creosote", func(t *testing.T) {
		// Stub tasks matching the real creosote config structure.
		uvInstall := &Task{Name: "install:uv", Usage: "install uv", Hidden: true, Global: true, Do: noop}
		pyFormat := &Task{
			Name: "py-format", Usage: "format python files", Do: noop,
			Flags: map[string]FlagDef{"python": {Default: "", Usage: "python version"}},
		}
		pyLint := &Task{
			Name: "py-lint", Usage: "lint python files", Do: noop,
			Flags: map[string]FlagDef{"python": {Default: "", Usage: "python version"}},
		}
		pyTypecheck := &Task{
			Name: "py-typecheck", Usage: "type-check python files", Do: noop,
			Flags: map[string]FlagDef{"python": {Default: "", Usage: "python version"}},
		}
		pyTest := &Task{
			Name: "py-test", Usage: "run python tests", Do: noop,
			Flags: map[string]FlagDef{
				"python":   {Default: "", Usage: "python version"},
				"coverage": {Default: false, Usage: "enable coverage"},
			},
		}
		creosoteTask := &Task{Name: "creosote", Usage: "run creosote self-check", Do: noop}
		preCommitCheck := &Task{Name: "pre-commit-check", Usage: "check pre-commit rev format", Do: noop}
		ghWorkflows := &Task{
			Name: "github-workflows", Usage: "bootstrap github workflows", Do: noop,
			Flags: map[string]FlagDef{
				"skip-pocket":           {Default: false, Usage: "exclude pocket workflow"},
				"include-pocket-matrix": {Default: false, Usage: "include pocket-matrix workflow"},
			},
		}

		// Detect function that returns root (simulating pyproject.toml at root).
		detectPyproject := func(_ []string, _ string) []string { return []string{"."} }

		// Compose exactly like the creosote config.
		cfg := &Config{
			Auto: Serial(
				// Python tasks with 3.9 suffix.
				WithOptions(
					Serial(uvInstall, pyFormat, pyLint, Parallel(pyTypecheck, pyTest)),
					WithNameSuffix("3.9"),
					WithFlag(pyFormat, "python", "3.9"),
					WithFlag(pyLint, "python", "3.9"),
					WithFlag(pyTypecheck, "python", "3.9"),
					WithFlag(pyTest, "python", "3.9"),
					WithFlag(pyTest, "coverage", true),
					WithDetect(detectPyproject),
				),

				// Additional Python version tests.
				WithOptions(
					Parallel(
						WithOptions(pyTest, WithNameSuffix("3.10"), WithFlag(pyTest, "python", "3.10")),
						WithOptions(pyTest, WithNameSuffix("3.11"), WithFlag(pyTest, "python", "3.11")),
						WithOptions(pyTest, WithNameSuffix("3.12"), WithFlag(pyTest, "python", "3.12")),
						WithOptions(pyTest, WithNameSuffix("3.13"), WithFlag(pyTest, "python", "3.13")),
					),
					WithDetect(detectPyproject),
				),

				Parallel(
					creosoteTask,
					preCommitCheck,
					WithOptions(
						ghWorkflows,
						WithFlag(ghWorkflows, "skip-pocket", true),
						WithFlag(ghWorkflows, "include-pocket-matrix", true),
					),
				),
			),
		}

		plan, err := newPlan(cfg, "/tmp", []string{"."})
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()

		// Verify task names in order.
		assertTaskNames(t, tasks, []string{
			"install:uv:3.9",
			"py-format:3.9", "py-lint:3.9", "py-typecheck:3.9", "py-test:3.9",
			"py-test:3.10", "py-test:3.11", "py-test:3.12", "py-test:3.13",
			"creosote", "pre-commit-check", "github-workflows",
		})

		// Verify flag propagation on py-format:3.9.
		pyFormat39 := findTaskInfo(tasks, "py-format:3.9")
		if pyFormat39 == nil {
			t.Fatal("expected to find py-format:3.9")
		}
		if pyFormat39.Flags["python"] != "3.9" {
			t.Errorf("py-format:3.9 python flag: got %v, want %q", pyFormat39.Flags["python"], "3.9")
		}

		// Verify coverage flag on py-test:3.9.
		pyTest39 := findTaskInfo(tasks, "py-test:3.9")
		if pyTest39 == nil {
			t.Fatal("expected to find py-test:3.9")
		}
		if pyTest39.Flags["coverage"] != true {
			t.Errorf("py-test:3.9 coverage flag: got %v, want true", pyTest39.Flags["coverage"])
		}
		if pyTest39.Flags["python"] != "3.9" {
			t.Errorf("py-test:3.9 python flag: got %v, want %q", pyTest39.Flags["python"], "3.9")
		}

		// Verify flag on a different python version.
		pyTest312 := findTaskInfo(tasks, "py-test:3.12")
		if pyTest312 == nil {
			t.Fatal("expected to find py-test:3.12")
		}
		if pyTest312.Flags["python"] != "3.12" {
			t.Errorf("py-test:3.12 python flag: got %v, want %q", pyTest312.Flags["python"], "3.12")
		}

		// Verify hidden status on install:uv:3.9.
		uvInstall39 := findTaskInfo(tasks, "install:uv:3.9")
		if uvInstall39 == nil {
			t.Fatal("expected to find install:uv:3.9")
		}
		if !uvInstall39.Hidden {
			t.Error("install:uv:3.9 should be hidden")
		}

		// Verify github-workflows flags.
		ghTask := findTaskInfo(tasks, "github-workflows")
		if ghTask == nil {
			t.Fatal("expected to find github-workflows")
		}
		if ghTask.Flags["skip-pocket"] != true {
			t.Errorf("github-workflows skip-pocket flag: got %v, want true", ghTask.Flags["skip-pocket"])
		}
		if ghTask.Flags["include-pocket-matrix"] != true {
			t.Errorf(
				"github-workflows include-pocket-matrix flag: got %v, want true",
				ghTask.Flags["include-pocket-matrix"],
			)
		}

		// Verify paths: all tasks should run at root since detect returns ".".
		for _, ti := range tasks {
			if !slices.Contains(ti.Paths, ".") {
				t.Errorf("%s: expected paths to contain '.', got %v", ti.Name, ti.Paths)
			}
		}
	})

	t.Run("NeotestGolang", func(t *testing.T) {
		// Stub tasks matching the real neotest-golang config structure.
		mdFormat := &Task{Name: "md-format", Usage: "format markdown", Do: noop}
		luaFormat := &Task{Name: "lua-format", Usage: "format lua", Do: noop}
		docsTask := &Task{Name: "docs", Usage: "generate docs", Do: noop}

		goFix := &Task{Name: "go-fix", Usage: "update code for newer go", Do: noop}
		goFormat := &Task{Name: "go-format", Usage: "format go code", Do: noop}
		goLint := &Task{Name: "go-lint", Usage: "lint go code", Do: noop}
		goTest := &Task{Name: "go-test", Usage: "run go tests", Do: noop}
		goVulncheck := &Task{Name: "go-vulncheck", Usage: "run govulncheck", Do: noop}

		queryFormat := &Task{
			Name: "query-format", Usage: "format tree-sitter queries", Do: noop,
			Flags: map[string]FlagDef{"parsers": {Default: "", Usage: "parser names"}},
		}
		queryLint := &Task{
			Name: "query-lint", Usage: "lint tree-sitter queries", Do: noop,
			Flags: map[string]FlagDef{
				"parsers": {Default: "", Usage: "parser names"},
				"fix":     {Default: false, Usage: "auto-fix"},
			},
		}

		nvimTestStable := &Task{Name: "nvim-test:stable", Usage: "plenary tests (stable)", Do: noop}
		nvimTestNightly := &Task{Name: "nvim-test:nightly", Usage: "plenary tests (nightly)", Do: noop}

		ghWorkflows := &Task{
			Name: "github-workflows", Usage: "bootstrap github workflows", Do: noop,
			Flags: map[string]FlagDef{
				"skip-pocket":           {Default: false, Usage: "exclude pocket workflow"},
				"include-pocket-matrix": {Default: false, Usage: "include pocket-matrix workflow"},
			},
		}

		// Simulate directory structure with go.mod in root and some subdirs.
		allDirs := []string{
			".", "lua", "queries", "tests", "tests/go", "tests/features",
		}

		// Detect function that finds go.mod in root only.
		detectGoMod := func(_ []string, _ string) []string { return []string{"."} }

		// Compose exactly like the neotest-golang config.
		cfg := &Config{
			Auto: Serial(
				Parallel(mdFormat, luaFormat, docsTask),

				WithOptions(
					Serial(goFix, goFormat, goLint, Parallel(goTest, goVulncheck)),
					WithDetect(detectGoMod),
					WithExcludeTask(goTest, "tests/go", "tests/features"),
					WithExcludeTask(goLint, "tests/go", "tests/features"),
				),

				Serial(
					WithOptions(queryFormat, WithFlag(queryFormat, "parsers", "go")),
					WithOptions(queryLint, WithFlag(queryLint, "parsers", "go")),
				),

				Parallel(nvimTestStable, nvimTestNightly),

				WithOptions(
					ghWorkflows,
					WithFlag(ghWorkflows, "skip-pocket", true),
					WithFlag(ghWorkflows, "include-pocket-matrix", true),
				),
			),
		}

		plan, err := newPlan(cfg, "/tmp", allDirs)
		if err != nil {
			t.Fatal(err)
		}

		tasks := plan.Tasks()

		// Verify task names in order.
		assertTaskNames(t, tasks, []string{
			"md-format", "lua-format", "docs",
			"go-fix", "go-format", "go-lint", "go-test", "go-vulncheck",
			"query-format", "query-lint",
			"nvim-test:stable", "nvim-test:nightly",
			"github-workflows",
		})

		// Verify flag propagation on query-format.
		qf := findTaskInfo(tasks, "query-format")
		if qf == nil {
			t.Fatal("expected to find query-format")
		}
		if qf.Flags["parsers"] != "go" {
			t.Errorf("query-format parsers flag: got %v, want %q", qf.Flags["parsers"], "go")
		}

		// Verify flag propagation on query-lint.
		ql := findTaskInfo(tasks, "query-lint")
		if ql == nil {
			t.Fatal("expected to find query-lint")
		}
		if ql.Flags["parsers"] != "go" {
			t.Errorf("query-lint parsers flag: got %v, want %q", ql.Flags["parsers"], "go")
		}

		// Verify path excludes for go-test: should NOT include tests/go or tests/features.
		goTestInfo := plan.pathMappings["go-test"]
		for _, excluded := range []string{"tests/go", "tests/features"} {
			if slices.Contains(goTestInfo.resolvedPaths, excluded) {
				t.Errorf("go-test should be excluded from %q, got paths %v", excluded, goTestInfo.resolvedPaths)
			}
		}

		// Verify path excludes for go-lint: same exclusions.
		goLintInfo := plan.pathMappings["go-lint"]
		for _, excluded := range []string{"tests/go", "tests/features"} {
			if slices.Contains(goLintInfo.resolvedPaths, excluded) {
				t.Errorf("go-lint should be excluded from %q, got paths %v", excluded, goLintInfo.resolvedPaths)
			}
		}

		// Verify go-fix still runs at detected path (no exclusions).
		goFixInfo := plan.pathMappings["go-fix"]
		if !slices.Contains(goFixInfo.resolvedPaths, ".") {
			t.Errorf("go-fix should run at root, got paths %v", goFixInfo.resolvedPaths)
		}

		// Verify github-workflows flags.
		ghTask := findTaskInfo(tasks, "github-workflows")
		if ghTask == nil {
			t.Fatal("expected to find github-workflows")
		}
		if ghTask.Flags["skip-pocket"] != true {
			t.Errorf("github-workflows skip-pocket: got %v, want true", ghTask.Flags["skip-pocket"])
		}
	})
}

func TestPlan_ContextValues(t *testing.T) {
	allDirs := []string{"."}

	// Define a context key for testing
	type versionKey struct{}

	t.Run("ContextValuesAreCapturedInTaskInstance", func(t *testing.T) {
		task := &Task{Name: "py-test", Usage: "test", Do: func(_ context.Context) error {
			return nil
		}}

		cfg := &Config{
			Auto: Serial(
				WithOptions(task, WithNameSuffix("3.9"), WithContextValue(versionKey{}, "3.9")),
				WithOptions(task, WithNameSuffix("3.10"), WithContextValue(versionKey{}, "3.10")),
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

		task := &Task{Name: "test", Usage: "test", Do: func(_ context.Context) error {
			return nil
		}}

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
