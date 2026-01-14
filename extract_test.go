package pocket

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// createTestTarGz creates a test .tar.gz archive with the given files.
func createTestTarGz(t *testing.T, files map[string][]byte) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.tar.gz")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer tmpFile.Close()

	gw := gzip.NewWriter(tmpFile)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	return tmpFile.Name()
}

// createTestZip creates a test .zip archive with the given files.
func createTestZip(t *testing.T, files map[string][]byte) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.zip")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer tmpFile.Close()

	zw := zip.NewWriter(tmpFile)

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatalf("write zip content: %v", err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return tmpFile.Name()
}

func TestExtractTarGz(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		files     map[string][]byte
		opts      []ExtractOpt
		wantFiles map[string][]byte
	}{
		{
			name: "extract all files",
			files: map[string][]byte{
				"file1.txt": []byte("content1"),
				"file2.txt": []byte("content2"),
			},
			opts: nil,
			wantFiles: map[string][]byte{
				"file1.txt": []byte("content1"),
				"file2.txt": []byte("content2"),
			},
		},
		{
			name: "extract single file",
			files: map[string][]byte{
				"file1.txt": []byte("content1"),
				"file2.txt": []byte("content2"),
			},
			opts: []ExtractOpt{WithExtractFile("file1.txt")},
			wantFiles: map[string][]byte{
				"file1.txt": []byte("content1"),
			},
		},
		{
			name: "extract and rename",
			files: map[string][]byte{
				"tool-1.0.0/binary": []byte("binary content"),
			},
			opts: []ExtractOpt{WithRenameFile("tool-1.0.0/binary", "binary")},
			wantFiles: map[string][]byte{
				"binary": []byte("binary content"),
			},
		},
		{
			name: "extract by base name",
			files: map[string][]byte{
				"nested/dir/mybinary": []byte("nested binary"),
				"other/file.txt":      []byte("other"),
			},
			opts: []ExtractOpt{WithExtractFile("mybinary")},
			wantFiles: map[string][]byte{
				"mybinary": []byte("nested binary"),
			},
		},
		{
			name: "flatten directory structure",
			files: map[string][]byte{
				"dir1/file1.txt": []byte("content1"),
				"dir2/file2.txt": []byte("content2"),
			},
			opts: []ExtractOpt{WithFlatten()},
			wantFiles: map[string][]byte{
				"file1.txt": []byte("content1"),
				"file2.txt": []byte("content2"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			archive := createTestTarGz(t, tt.files)
			destDir := t.TempDir()

			if err := ExtractTarGz(archive, destDir, tt.opts...); err != nil {
				t.Fatalf("ExtractTarGz: %v", err)
			}

			for wantFile, wantContent := range tt.wantFiles {
				path := filepath.Join(destDir, wantFile)
				got, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("read %s: %v", wantFile, err)
					continue
				}
				if string(got) != string(wantContent) {
					t.Errorf("%s content = %q, want %q", wantFile, got, wantContent)
				}
			}
		})
	}
}

func TestExtractZip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		files     map[string][]byte
		opts      []ExtractOpt
		wantFiles map[string][]byte
	}{
		{
			name: "extract all files",
			files: map[string][]byte{
				"file1.txt": []byte("content1"),
				"file2.txt": []byte("content2"),
			},
			opts: nil,
			wantFiles: map[string][]byte{
				"file1.txt": []byte("content1"),
				"file2.txt": []byte("content2"),
			},
		},
		{
			name: "extract single file",
			files: map[string][]byte{
				"file1.txt": []byte("content1"),
				"file2.txt": []byte("content2"),
			},
			opts: []ExtractOpt{WithExtractFile("file1.txt")},
			wantFiles: map[string][]byte{
				"file1.txt": []byte("content1"),
			},
		},
		{
			name: "extract and rename",
			files: map[string][]byte{
				"tool-1.0.0/binary": []byte("binary content"),
			},
			opts: []ExtractOpt{WithRenameFile("tool-1.0.0/binary", "binary")},
			wantFiles: map[string][]byte{
				"binary": []byte("binary content"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			archive := createTestZip(t, tt.files)
			destDir := t.TempDir()

			if err := ExtractZip(archive, destDir, tt.opts...); err != nil {
				t.Fatalf("ExtractZip: %v", err)
			}

			for wantFile, wantContent := range tt.wantFiles {
				path := filepath.Join(destDir, wantFile)
				got, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("read %s: %v", wantFile, err)
					continue
				}
				if string(got) != string(wantContent) {
					t.Errorf("%s content = %q, want %q", wantFile, got, wantContent)
				}
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "source.txt")
	content := []byte("test content")

	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	destDir := t.TempDir()
	destPath := filepath.Join(destDir, "nested", "dest.txt")

	if err := CopyFile(srcPath, destPath); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}

	// Check permissions (should be executable).
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("stat dest: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Errorf("file should be executable, mode = %o", info.Mode())
	}
}
