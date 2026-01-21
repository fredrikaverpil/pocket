package pk

import (
	"os"
	"path/filepath"
	"testing"
)

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
