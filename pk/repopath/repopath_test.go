package repopath

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGitRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "services", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, nested)
	SetGitRootFunc(nil)

	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("evaluate root symlinks: %v", err)
	}

	got, err := GitRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected git root %q, got %q", want, got)
	}
}

func TestGitRoot_NotFound(t *testing.T) {
	chdir(t, t.TempDir())
	SetGitRootFunc(nil)

	got, err := GitRoot()
	if !errors.Is(err, ErrGitRootNotFound) {
		t.Fatalf("expected ErrGitRootNotFound, got %v", err)
	}
	if got != "" {
		t.Errorf("expected empty root, got %q", got)
	}
}

func TestFromGitRoot_NotFoundFallback(t *testing.T) {
	chdir(t, t.TempDir())
	SetGitRootFunc(nil)

	if got := FromGitRoot("pkg"); got != "pkg" {
		t.Errorf("expected fallback path %q, got %q", "pkg", got)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
		SetGitRootFunc(nil)
	})
}
