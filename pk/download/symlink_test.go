package download

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	t.Run("copies content and sets permissions", func(t *testing.T) {
		// Arrange
		srcDir := t.TempDir()
		src := filepath.Join(srcDir, "source")
		if err := os.WriteFile(src, []byte("hello world"), 0o644); err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(t.TempDir(), "dest")

		// Act
		err := CopyFile(src, dst)
		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("read dest: %v", err)
		}
		if string(got) != "hello world" {
			t.Errorf("content: got %q, want %q", string(got), "hello world")
		}
		info, err := os.Stat(dst)
		if err != nil {
			t.Fatalf("stat dest: %v", err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("permissions: got %o, want %o", info.Mode().Perm(), 0o755)
		}
	})

	t.Run("returns error for missing source", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "nonexistent")
		dst := filepath.Join(t.TempDir(), "dest")

		// Act
		err := CopyFile(src, dst)

		// Assert
		if err == nil {
			t.Fatal("expected error for missing source, got nil")
		}
	})
}
