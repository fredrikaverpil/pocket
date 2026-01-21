package pk

import (
	"context"
	"slices"
	"testing"
)

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
}
