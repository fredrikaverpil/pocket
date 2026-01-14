package pocket

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		archiveFiles map[string][]byte
		format       string
		opts         func(destDir string) []DownloadOpt
		wantFiles    map[string][]byte
	}{
		{
			name: "download and extract tar.gz",
			archiveFiles: map[string][]byte{
				"binary": []byte("binary content"),
			},
			format: "tar.gz",
			opts: func(destDir string) []DownloadOpt {
				return []DownloadOpt{
					WithDestDir(destDir),
					WithFormat("tar.gz"),
				}
			},
			wantFiles: map[string][]byte{
				"binary": []byte("binary content"),
			},
		},
		{
			name: "download and extract zip with rename",
			archiveFiles: map[string][]byte{
				"tool-1.0.0/tool": []byte("tool binary"),
			},
			format: "zip",
			opts: func(destDir string) []DownloadOpt {
				return []DownloadOpt{
					WithDestDir(destDir),
					WithFormat("zip"),
					WithExtract(WithRenameFile("tool-1.0.0/tool", "tool")),
				}
			},
			wantFiles: map[string][]byte{
				"tool": []byte("tool binary"),
			},
		},
		{
			name: "skip if exists",
			archiveFiles: map[string][]byte{
				"binary": []byte("new content"),
			},
			format: "tar.gz",
			opts: func(destDir string) []DownloadOpt {
				// Create the file first
				existingFile := filepath.Join(destDir, "existing")
				_ = os.WriteFile(existingFile, []byte("existing content"), 0o644)
				return []DownloadOpt{
					WithDestDir(destDir),
					WithFormat("tar.gz"),
					WithSkipIfExists(existingFile),
				}
			},
			wantFiles: map[string][]byte{
				"existing": []byte("existing content"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test archive
			var archivePath string
			switch tt.format {
			case "tar.gz":
				archivePath = createTestTarGz(t, tt.archiveFiles)
			case "zip":
				archivePath = createTestZip(t, tt.archiveFiles)
			}

			// Read archive content for serving
			archiveContent, err := os.ReadFile(archivePath)
			if err != nil {
				t.Fatalf("read archive: %v", err)
			}

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(archiveContent)
			}))
			defer server.Close()

			destDir := t.TempDir()
			out := &Output{Stdout: io.Discard, Stderr: io.Discard}
			exec := NewExecution(out, false, ".")
			tc := exec.TaskContext(".")

			opts := tt.opts(destDir)
			err = Download(context.Background(), tc, server.URL+"/test.archive", opts...)
			if err != nil {
				t.Fatalf("Download: %v", err)
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

func TestDownload_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	out := &Output{Stdout: io.Discard, Stderr: io.Discard}
	exec := NewExecution(out, false, ".")
	tc := exec.TaskContext(".")

	err := Download(context.Background(), tc, server.URL+"/notfound",
		WithDestDir(t.TempDir()),
		WithFormat("tar.gz"),
	)
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestFromLocal(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{
		"tool": []byte("tool content"),
	}
	archivePath := createTestTarGz(t, files)
	destDir := t.TempDir()

	out := &Output{Stdout: io.Discard, Stderr: io.Discard}
	exec := NewExecution(out, false, ".")
	tc := exec.TaskContext(".")

	err := FromLocal(context.Background(), tc, archivePath,
		WithDestDir(destDir),
		WithFormat("tar.gz"),
	)
	if err != nil {
		t.Fatalf("FromLocal: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destDir, "tool"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(content) != "tool content" {
		t.Errorf("content = %q, want %q", content, "tool content")
	}
}
