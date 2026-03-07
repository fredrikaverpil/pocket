package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

type tarGzFile struct {
	name    string
	content string
	mode    int64
}

func createTarGz(t *testing.T, path string, files []tarGzFile) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, file := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: file.name,
			Mode: file.mode,
			Size: int64(len(file.content)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(file.content)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveOutputName(t *testing.T) {
	tests := []struct {
		name        string
		fullPath    string
		baseName    string
		cfg         *extractConfig
		wantName    string
		wantExtract bool
	}{
		{
			name:        "no config extracts with full path",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{}},
			wantName:    "dir/file.txt",
			wantExtract: true,
		},
		{
			name:        "flatten returns base name",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{}, flatten: true},
			wantName:    "file.txt",
			wantExtract: true,
		},
		{
			name:        "rename map matches full path",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"dir/file.txt": "renamed.txt"}},
			wantName:    "renamed.txt",
			wantExtract: true,
		},
		{
			name:        "rename map matches base name",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"file.txt": "renamed.txt"}},
			wantName:    "renamed.txt",
			wantExtract: true,
		},
		{
			name:        "rename map skips unmatched file",
			fullPath:    "dir/other.txt",
			baseName:    "other.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"file.txt": "renamed.txt"}},
			wantName:    "",
			wantExtract: false,
		},
		{
			name:        "full path match takes precedence over base name",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"dir/file.txt": "full.txt", "file.txt": "base.txt"}},
			wantName:    "full.txt",
			wantExtract: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotExtract := resolveOutputName(tt.fullPath, tt.baseName, tt.cfg)
			if gotName != tt.wantName {
				t.Errorf("name: got %q, want %q", gotName, tt.wantName)
			}
			if gotExtract != tt.wantExtract {
				t.Errorf("extract: got %v, want %v", gotExtract, tt.wantExtract)
			}
		})
	}
}

func TestExtractTarGz(t *testing.T) {
	t.Run("extracts all files preserving structure", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.tar.gz")
		dest := t.TempDir()
		createTarGz(t, src, []tarGzFile{
			{name: "dir/hello.txt", content: "hello", mode: 0o644},
			{name: "dir/world.txt", content: "world", mode: 0o644},
		})

		// Act
		err := ExtractTarGz(src, dest)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "dir", "hello.txt"))
		if err != nil {
			t.Fatalf("read hello.txt: %v", err)
		}
		if string(got) != "hello" {
			t.Errorf("hello.txt content: got %q, want %q", string(got), "hello")
		}
		got, err = os.ReadFile(filepath.Join(dest, "dir", "world.txt"))
		if err != nil {
			t.Fatalf("read world.txt: %v", err)
		}
		if string(got) != "world" {
			t.Errorf("world.txt content: got %q, want %q", string(got), "world")
		}
	})

	t.Run("with flatten", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.tar.gz")
		dest := t.TempDir()
		createTarGz(t, src, []tarGzFile{
			{name: "deep/nested/file.txt", content: "flat", mode: 0o644},
		})

		// Act
		err := ExtractTarGz(src, dest, WithFlatten())

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "file.txt"))
		if err != nil {
			t.Fatalf("read file.txt: %v", err)
		}
		if string(got) != "flat" {
			t.Errorf("content: got %q, want %q", string(got), "flat")
		}
	})

	t.Run("with rename file", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.tar.gz")
		dest := t.TempDir()
		createTarGz(t, src, []tarGzFile{
			{name: "bin/tool-v1.2", content: "binary", mode: 0o755},
			{name: "README.md", content: "docs", mode: 0o644},
		})

		// Act
		err := ExtractTarGz(src, dest, WithRenameFile("bin/tool-v1.2", "tool"))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "tool"))
		if err != nil {
			t.Fatalf("read tool: %v", err)
		}
		if string(got) != "binary" {
			t.Errorf("content: got %q, want %q", string(got), "binary")
		}
		// README.md should NOT be extracted (rename map filters).
		if _, err := os.Stat(filepath.Join(dest, "README.md")); err == nil {
			t.Error("README.md should not be extracted when using WithRenameFile")
		}
	})

	t.Run("with extract file by base name", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.tar.gz")
		dest := t.TempDir()
		createTarGz(t, src, []tarGzFile{
			{name: "dir/wanted.txt", content: "yes", mode: 0o644},
			{name: "dir/unwanted.txt", content: "no", mode: 0o644},
		})

		// Act
		err := ExtractTarGz(src, dest, WithExtractFile("wanted.txt"))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "wanted.txt"))
		if err != nil {
			t.Fatalf("read wanted.txt: %v", err)
		}
		if string(got) != "yes" {
			t.Errorf("content: got %q, want %q", string(got), "yes")
		}
		if _, err := os.Stat(filepath.Join(dest, "unwanted.txt")); err == nil {
			t.Error("unwanted.txt should not be extracted")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.tar.gz")
		dest := t.TempDir()
		createTarGz(t, src, []tarGzFile{
			{name: "../../../etc/passwd", content: "malicious", mode: 0o644},
		})

		// Act
		err := ExtractTarGz(src, dest)

		// Assert
		if err == nil {
			t.Fatal("expected error for path traversal, got nil")
		}
		want := "invalid file path: ../../../etc/passwd"
		if err.Error() != want {
			t.Errorf("error message: got %q, want %q", err.Error(), want)
		}
	})
}

type zipFile struct {
	name    string
	content string
}

func createZip(t *testing.T, path string, files []zipFile) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, file := range files {
		w, err := zw.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(file.content)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExtractZip(t *testing.T) {
	t.Run("extracts all files preserving structure", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.zip")
		dest := t.TempDir()
		createZip(t, src, []zipFile{
			{name: "dir/hello.txt", content: "hello"},
			{name: "dir/world.txt", content: "world"},
		})

		// Act
		err := ExtractZip(src, dest)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "dir", "hello.txt"))
		if err != nil {
			t.Fatalf("read hello.txt: %v", err)
		}
		if string(got) != "hello" {
			t.Errorf("hello.txt content: got %q, want %q", string(got), "hello")
		}
		got, err = os.ReadFile(filepath.Join(dest, "dir", "world.txt"))
		if err != nil {
			t.Fatalf("read world.txt: %v", err)
		}
		if string(got) != "world" {
			t.Errorf("world.txt content: got %q, want %q", string(got), "world")
		}
	})

	t.Run("with flatten", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.zip")
		dest := t.TempDir()
		createZip(t, src, []zipFile{
			{name: "deep/nested/file.txt", content: "flat"},
		})

		// Act
		err := ExtractZip(src, dest, WithFlatten())

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "file.txt"))
		if err != nil {
			t.Fatalf("read file.txt: %v", err)
		}
		if string(got) != "flat" {
			t.Errorf("content: got %q, want %q", string(got), "flat")
		}
	})

	t.Run("with rename file", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.zip")
		dest := t.TempDir()
		createZip(t, src, []zipFile{
			{name: "bin/tool-v1.2", content: "binary"},
			{name: "README.md", content: "docs"},
		})

		// Act
		err := ExtractZip(src, dest, WithRenameFile("bin/tool-v1.2", "tool"))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "tool"))
		if err != nil {
			t.Fatalf("read tool: %v", err)
		}
		if string(got) != "binary" {
			t.Errorf("content: got %q, want %q", string(got), "binary")
		}
		if _, err := os.Stat(filepath.Join(dest, "README.md")); err == nil {
			t.Error("README.md should not be extracted when using WithRenameFile")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		// Arrange
		src := filepath.Join(t.TempDir(), "test.zip")
		dest := t.TempDir()

		// Create zip with raw header containing traversal path.
		f, err := os.Create(src)
		if err != nil {
			t.Fatal(err)
		}
		zw := zip.NewWriter(f)
		header := &zip.FileHeader{Name: "../../../etc/passwd"}
		header.Method = zip.Store
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("malicious")); err != nil {
			t.Fatal(err)
		}
		zw.Close()
		f.Close()

		// Act
		err = ExtractZip(src, dest)

		// Assert
		if err == nil {
			t.Fatal("expected error for path traversal, got nil")
		}
		want := "invalid file path: ../../../etc/passwd"
		if err.Error() != want {
			t.Errorf("error message: got %q, want %q", err.Error(), want)
		}
	})
}
