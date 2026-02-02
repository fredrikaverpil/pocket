package pk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWalkDirectories_SkipDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"src",
		"vendor",
		"node_modules",
		"testdata",
		".hidden",
		"src/nested",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		skipDirs []string
		want     []string
	}{
		{
			name:     "skip vendor and node_modules",
			skipDirs: []string{"vendor", "node_modules"},
			want:     []string{".", "src", "src/nested", "testdata"},
		},
		{
			name:     "skip only testdata",
			skipDirs: []string{"testdata"},
			want:     []string{".", "node_modules", "src", "src/nested", "vendor"},
		},
		{
			name:     "skip nothing",
			skipDirs: []string{},
			want:     []string{".", "node_modules", "src", "src/nested", "testdata", "vendor"},
		},
		{
			name:     "skip multiple custom dirs",
			skipDirs: []string{"vendor", "testdata"},
			want:     []string{".", "node_modules", "src", "src/nested"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := walkDirectories(tmpDir, tc.skipDirs, false)
			if err != nil {
				t.Fatal(err)
			}

			slices.Sort(got)
			slices.Sort(tc.want)

			if !slices.Equal(got, tc.want) {
				t.Errorf("walkDirectories() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWalkDirectories_HiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories including hidden ones
	dirs := []string{".git", ".hidden", "visible", ".hidden/nested"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("hidden dirs skipped by default", func(t *testing.T) {
		got, err := walkDirectories(tmpDir, []string{}, false)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{".", "visible"}
		slices.Sort(got)

		if !slices.Equal(got, want) {
			t.Errorf("walkDirectories() = %v, want %v", got, want)
		}
	})

	t.Run("hidden dirs included when includeHidden is true", func(t *testing.T) {
		got, err := walkDirectories(tmpDir, []string{}, true)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{".", ".git", ".hidden", ".hidden/nested", "visible"}
		slices.Sort(got)

		if !slices.Equal(got, want) {
			t.Errorf("walkDirectories() = %v, want %v", got, want)
		}
	})

	t.Run("hidden dirs can be skipped via skipDirs when included", func(t *testing.T) {
		got, err := walkDirectories(tmpDir, []string{".git"}, true)
		if err != nil {
			t.Fatal(err)
		}

		want := []string{".", ".hidden", ".hidden/nested", "visible"}
		slices.Sort(got)

		if !slices.Equal(got, want) {
			t.Errorf("walkDirectories() = %v, want %v", got, want)
		}
	})
}
