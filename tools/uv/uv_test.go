package uv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveStaleVenv(t *testing.T) {
	t.Run("no venv directory", func(t *testing.T) {
		err := removeStaleVenv(context.Background(), filepath.Join(t.TempDir(), "nonexistent"))
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("valid venv is kept", func(t *testing.T) {
		venvDir := t.TempDir()
		// Point home at a directory that exists.
		homeDir := t.TempDir()
		writePyvenvCfg(t, venvDir, "home = "+homeDir+"\n")

		if err := removeStaleVenv(context.Background(), venvDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(venvDir); err != nil {
			t.Fatal("venv directory should still exist")
		}
	})

	t.Run("stale venv is removed", func(t *testing.T) {
		venvDir := t.TempDir()
		// Point home at a directory that does not exist.
		writePyvenvCfg(t, venvDir, "home = /nonexistent/python/path\n")

		if err := removeStaleVenv(context.Background(), venvDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(venvDir); !os.IsNotExist(err) {
			t.Fatal("stale venv directory should have been removed")
		}
	})

	t.Run("no home key is ignored", func(t *testing.T) {
		venvDir := t.TempDir()
		writePyvenvCfg(t, venvDir, "include-system-site-packages = false\nversion = 3.9.25\n")

		if err := removeStaleVenv(context.Background(), venvDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(venvDir); err != nil {
			t.Fatal("venv directory should still exist when no home key is present")
		}
	})
}

func writePyvenvCfg(t *testing.T, venvDir, content string) {
	t.Helper()
	cfgPath := filepath.Join(venvDir, "pyvenv.cfg")
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write pyvenv.cfg: %v", err)
	}
}
