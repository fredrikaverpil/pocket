package pk

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestWithForceRun(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("path-task", "test task", nil, Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}))

	// Create context with tracker.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// Create pathFilter without forceRun.
	pf := WithOptions(task).(*pathFilter)
	pf.resolvedPaths = []string{"."} // Simulate resolved paths.

	// First run should execute.
	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after first run, got %d", got)
	}

	// Second run should be skipped (same task+path).
	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after second run (should skip), got %d", got)
	}

	// Create pathFilter with forceRun.
	pfForce := WithOptions(task, WithForceRun()).(*pathFilter)
	pfForce.resolvedPaths = []string{"."}

	// Should execute despite already having run.
	if err := pfForce.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after run with WithForceRun, got %d", got)
	}
}

func TestPathFilter_MultiplePaths(t *testing.T) {
	var paths []string

	task := NewTask("multi-path-task", "test task", nil, Do(func(ctx context.Context) error {
		paths = append(paths, PathFromContext(ctx))
		return nil
	}))

	// Context WITHOUT tracker - task runs for each path (no dedup).
	ctx := context.Background()

	// Create pathFilter with multiple resolved paths.
	pf := WithOptions(task).(*pathFilter)
	pf.resolvedPaths = []string{"services/api", "services/web", "pkg/utils"}

	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without tracker, should run once per path.
	if len(paths) != 3 {
		t.Errorf("expected 3 paths, got %d: %v", len(paths), paths)
	}

	expected := []string{"services/api", "services/web", "pkg/utils"}
	for i, p := range paths {
		if p != expected[i] {
			t.Errorf("expected path[%d]=%q, got %q", i, expected[i], p)
		}
	}
}

func TestPathFilter_MultiplePathsWithDedup(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("multi-path-dedup-task", "test task", nil, Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}))

	// Context WITH tracker - dedup by (task, path) means each path runs once.
	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	pf := WithOptions(task).(*pathFilter)
	pf.resolvedPaths = []string{"services/api", "services/web", "pkg/utils"}

	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With (task, path) dedup, task runs once per unique path.
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 (once per path), got %d", got)
	}

	// Running again should not add more executions (all paths already done).
	if err := pf.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 3 {
		t.Errorf("expected runCount=3 after second run (all deduplicated), got %d", got)
	}
}

func TestPathFilter_DeduplicationByTaskAndPath(t *testing.T) {
	var runCount atomic.Int32

	task := NewTask("path-dedup-task", "test task", nil, Do(func(_ context.Context) error {
		runCount.Add(1)
		return nil
	}))

	ctx := context.Background()
	tracker := newExecutionTracker()
	ctx = withExecutionTracker(ctx, tracker)

	// First pathFilter runs in services/api.
	pf1 := WithOptions(task).(*pathFilter)
	pf1.resolvedPaths = []string{"services/api"}

	// Second pathFilter runs in a different path - should run (different path).
	pf2 := WithOptions(task).(*pathFilter)
	pf2.resolvedPaths = []string{"services/web"}

	if err := pf1.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 1 {
		t.Errorf("expected runCount=1 after pf1, got %d", got)
	}

	// Same task at different path should run (dedup by task+path).
	if err := pf2.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after pf2 (different path), got %d", got)
	}

	// Running pf1 again should skip (same task+path).
	if err := pf1.run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := runCount.Load(); got != 2 {
		t.Errorf("expected runCount=2 after pf1 again (deduplicated), got %d", got)
	}
}

func TestWithCleanPath(t *testing.T) {
	t.Run("RemovesAndRecreatesDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a directory with a file inside.
		testPath := "sub/dir"
		absPath := filepath.Join(tmpDir, testPath)
		if err := os.MkdirAll(absPath, 0o755); err != nil {
			t.Fatal(err)
		}
		markerFile := filepath.Join(absPath, "marker.txt")
		if err := os.WriteFile(markerFile, []byte("should be deleted"), 0o644); err != nil {
			t.Fatal(err)
		}

		task := NewTask("clean-test", "test", nil, Do(func(ctx context.Context) error {
			// Verify marker file is gone.
			if _, err := os.Stat(markerFile); !os.IsNotExist(err) {
				t.Error("expected marker file to be removed by cleanPath")
			}
			// Verify directory exists (recreated).
			if _, err := os.Stat(absPath); err != nil {
				t.Errorf("expected directory to exist after cleanPath: %v", err)
			}
			return nil
		}))

		ctx := context.Background()
		// Override git root so FromGitRoot resolves to our tmpDir.
		origFindGitRoot := findGitRootFunc
		findGitRootFunc = func() string { return tmpDir }
		defer func() { findGitRootFunc = origFindGitRoot }()

		pf := WithOptions(task, WithCleanPath()).(*pathFilter)
		pf.resolvedPaths = []string{testPath}

		if err := pf.run(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("SkipsRoot", func(t *testing.T) {
		task := NewTask("clean-root-test", "test", nil, Do(func(_ context.Context) error {
			return nil
		}))

		ctx := context.Background()

		pf := WithOptions(task, WithCleanPath()).(*pathFilter)
		pf.resolvedPaths = []string{"."}

		// Should not error — cleaning root is a no-op.
		if err := pf.run(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWithExplicitPath(t *testing.T) {
	t.Run("CreatesDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		testPath := "explicit/new/dir"
		absPath := filepath.Join(tmpDir, testPath)

		task := NewTask("explicit-test", "test", nil, Do(func(ctx context.Context) error {
			// Verify directory was created.
			if _, err := os.Stat(absPath); err != nil {
				t.Errorf("expected directory to be created: %v", err)
			}
			return nil
		}))

		ctx := context.Background()
		origFindGitRoot := findGitRootFunc
		findGitRootFunc = func() string { return tmpDir }
		defer func() { findGitRootFunc = origFindGitRoot }()

		pf := WithOptions(task, WithExplicitPath(testPath)).(*pathFilter)
		pf.resolvedPaths = []string{testPath}

		if err := pf.run(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("CombinedWithCleanPath", func(t *testing.T) {
		tmpDir := t.TempDir()

		testPath := "combo/dir"
		absPath := filepath.Join(tmpDir, testPath)

		// Pre-create directory with a marker file.
		if err := os.MkdirAll(absPath, 0o755); err != nil {
			t.Fatal(err)
		}
		markerFile := filepath.Join(absPath, "old-artifact.txt")
		if err := os.WriteFile(markerFile, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}

		task := NewTask("combo-test", "test", nil, Do(func(ctx context.Context) error {
			// Marker should be gone (cleaned).
			if _, err := os.Stat(markerFile); !os.IsNotExist(err) {
				t.Error("expected marker file to be removed")
			}
			// Directory should exist (recreated).
			if _, err := os.Stat(absPath); err != nil {
				t.Errorf("expected directory to exist: %v", err)
			}
			return nil
		}))

		ctx := context.Background()
		origFindGitRoot := findGitRootFunc
		findGitRootFunc = func() string { return tmpDir }
		defer func() { findGitRootFunc = origFindGitRoot }()

		pf := WithOptions(task, WithExplicitPath(testPath), WithCleanPath()).(*pathFilter)
		pf.resolvedPaths = []string{testPath}

		if err := pf.run(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDetectByFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create subdirectories
	dirs := []string{
		".",
		"moduleA",
		"moduleB",
		"nomodule",
		"nested/moduleC",
	}
	for _, d := range dirs {
		if d != "." {
			err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Create go.mod files in some directories
	goModDirs := []string{".", "moduleA", "nested/moduleC"}
	for _, d := range goModDirs {
		err := os.WriteFile(filepath.Join(tmpDir, d, "go.mod"), []byte("module test"), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Test DetectByFile
	detect := DetectByFile("go.mod")
	result := detect(dirs, tmpDir)

	// Should find ., moduleA, and nested/moduleC
	if len(result) != 3 {
		t.Errorf("expected 3 directories, got %d: %v", len(result), result)
	}

	expected := map[string]bool{".": true, "moduleA": true, "nested/moduleC": true}
	for _, r := range result {
		if !expected[r] {
			t.Errorf("unexpected directory in result: %s", r)
		}
	}
}

func TestDetectByFile_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories
	dirs := []string{".", "cargoDir", "npmDir", "both"}
	for _, d := range dirs {
		if d != "." {
			err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	// Create marker files
	if err := os.WriteFile(filepath.Join(tmpDir, "cargoDir", "Cargo.toml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "npmDir", "package.json"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "both", "Cargo.toml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "both", "package.json"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test detecting both Cargo.toml and package.json
	detect := DetectByFile("Cargo.toml", "package.json")
	result := detect(dirs, tmpDir)

	if len(result) != 3 {
		t.Errorf("expected 3 directories, got %d: %v", len(result), result)
	}

	expected := map[string]bool{"cargoDir": true, "npmDir": true, "both": true}
	for _, r := range result {
		if !expected[r] {
			t.Errorf("unexpected directory in result: %s", r)
		}
	}
}
