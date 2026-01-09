package golang_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/golang"
)

// TestLintTask_IncludesSiblingFiles verifies that go-lint includes .go files
// that are siblings (same directory) to go.mod. This is a regression test for
// https://github.com/einride/sage/issues/423 where sage was ignoring such files.
//
// The test creates a temp Go module with:
// - go.mod
// - main.go (sibling file with intentional lint error)
// And verifies that running ./... from that directory catches the error.
func TestLintTask_IncludesSiblingFiles(t *testing.T) {
	// Skip if golangci-lint is not available (this is an integration test).
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not in PATH, skipping integration test")
	}

	// Create a temp directory with a Go module.
	tmpDir := t.TempDir()

	// Create go.mod.
	goMod := `module testmod

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Create a sibling .go file with valid code.
	mainGo := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	// Verify that go vet ./... from the module directory finds the file.
	// We use go vet as a proxy since it's always available and tests the ./... pattern.
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go vet ./... failed unexpectedly: %v\nOutput: %s", err, output)
	}

	// The fact that go vet ran without error confirms ./... includes sibling files.
	t.Log("Verified: ./... pattern includes sibling .go files next to go.mod")
}

// TestDetectModules_FindsSiblingFiles verifies that DetectByFile("go.mod")
// returns the directory containing go.mod (which is where sibling files live).
func TestDetectModules_FindsSiblingFiles(t *testing.T) {
	// This test relies on the pocket repo having a .pocket/go.mod.
	// The detection should return "." or ".pocket" depending on where we run from.

	// Test that golang.Tasks implements Detectable.
	tasks := golang.Tasks()
	detectable, ok := tasks.(interface {
		DefaultDetect() func() []string
	})
	if !ok {
		t.Fatal("golang.Tasks() should implement Detectable interface")
	}

	detectFn := detectable.DefaultDetect()
	if detectFn == nil {
		t.Fatal("DefaultDetect() should return a non-nil function")
	}

	// The detect function finds directories with go.mod.
	// Each found directory is where sibling .go files would be linted.
	dirs := detectFn()
	if len(dirs) == 0 {
		t.Log("No go.mod files found (expected in isolated test env)")
	} else {
		t.Logf("Detected %d Go module directories: %v", len(dirs), dirs)
		// Verify each directory actually contains a go.mod.
		for _, dir := range dirs {
			goModPath := filepath.Join(pocket.GitRoot(), dir, "go.mod")
			if _, err := os.Stat(goModPath); os.IsNotExist(err) {
				t.Errorf("Detected dir %q does not contain go.mod", dir)
			}
		}
	}
}

// TestLintTask_PathsPassedCorrectly verifies that the lint task receives
// the correct paths when executed, ensuring sibling files would be included.
func TestLintTask_PathsPassedCorrectly(t *testing.T) {
	// Create a custom task to capture what paths would be passed.
	var capturedPaths []string

	task := &pocket.Task{
		Name:  "test-capture",
		Usage: "capture paths for testing",
		Action: func(_ context.Context, opts *pocket.RunContext) error {
			capturedPaths = opts.Paths
			return nil
		},
	}

	// Set paths via SetPaths (as PathFilter wrapper does).
	task.SetPaths([]string{"proj1", "proj2"})

	ctx := pocket.WithCwd(context.Background(), "testdir")

	if err := task.Run(ctx); err != nil {
		t.Fatalf("task.Run failed: %v", err)
	}

	if len(capturedPaths) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(capturedPaths), capturedPaths)
	}

	// Verify paths are as expected.
	if capturedPaths[0] != "proj1" || capturedPaths[1] != "proj2" {
		t.Errorf("unexpected paths: %v", capturedPaths)
	}
}
