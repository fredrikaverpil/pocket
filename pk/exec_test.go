package pk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestContainsNotice(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		patterns []string
		want     bool
	}{
		{
			name:     "matches WARNING",
			output:   "some WARNING: something bad",
			patterns: DefaultNoticePatterns,
			want:     true,
		},
		{
			name:     "matches deprecation",
			output:   "DeprecationWarning: old API",
			patterns: DefaultNoticePatterns,
			want:     true,
		},
		{
			name:     "case insensitive",
			output:   "NOTICE: update available",
			patterns: DefaultNoticePatterns,
			want:     true,
		},
		{
			name:     "no match",
			output:   "all good, no issues",
			patterns: DefaultNoticePatterns,
			want:     false,
		},
		{
			name:     "custom patterns",
			output:   "FIXME: broken thing",
			patterns: []string{"fixme", "todo"},
			want:     true,
		},
		{
			name:     "empty output",
			output:   "",
			patterns: DefaultNoticePatterns,
			want:     false,
		},
		{
			name:     "empty patterns",
			output:   "WARNING: this should not match",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "nil patterns",
			output:   "WARNING: this should not match",
			patterns: nil,
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := containsNotice(tc.output, tc.patterns)
			if got != tc.want {
				t.Errorf("containsNotice(%q, %v) = %v, want %v", tc.output, tc.patterns, got, tc.want)
			}
		})
	}
}

func TestApplyEnvConfig(t *testing.T) {
	t.Run("NoChanges", func(t *testing.T) {
		environ := []string{"HOME=/home/user", "PATH=/usr/bin"}
		cfg := EnvConfig{}
		got := applyEnvConfig(environ, cfg)
		if !slices.Equal(got, environ) {
			t.Errorf("expected unchanged environ, got %v", got)
		}
	})

	t.Run("FilterPrefix", func(t *testing.T) {
		environ := []string{"HOME=/home/user", "VIRTUAL_ENV=/venv", "VIRTUAL_ENV_PROMPT=(venv)", "PATH=/usr/bin"}
		cfg := EnvConfig{Filter: []string{"VIRTUAL_ENV"}}
		got := applyEnvConfig(environ, cfg)
		want := []string{"HOME=/home/user", "PATH=/usr/bin"}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("SetReplacesExisting", func(t *testing.T) {
		environ := []string{"HOME=/home/user", "FOO=old"}
		cfg := EnvConfig{Set: map[string]string{"FOO": "new"}}
		got := applyEnvConfig(environ, cfg)

		// FOO=old should be removed and FOO=new should be appended.
		if slices.Contains(got, "FOO=old") {
			t.Error("old FOO should be removed")
		}
		if !slices.Contains(got, "FOO=new") {
			t.Error("new FOO should be present")
		}
		if !slices.Contains(got, "HOME=/home/user") {
			t.Error("HOME should be preserved")
		}
	})

	t.Run("SetAddsNew", func(t *testing.T) {
		environ := []string{"HOME=/home/user"}
		cfg := EnvConfig{Set: map[string]string{"NEW_VAR": "value"}}
		got := applyEnvConfig(environ, cfg)
		if !slices.Contains(got, "NEW_VAR=value") {
			t.Errorf("expected NEW_VAR=value, got %v", got)
		}
	})

	t.Run("FilterAndSet", func(t *testing.T) {
		environ := []string{"HOME=/home/user", "PYENV_ROOT=/pyenv", "PYENV_VERSION=3.9"}
		cfg := EnvConfig{
			Filter: []string{"PYENV_"},
			Set:    map[string]string{"PYENV_VERSION": "3.10"},
		}
		got := applyEnvConfig(environ, cfg)

		if slices.Contains(got, "PYENV_ROOT=/pyenv") {
			t.Error("PYENV_ROOT should be filtered out")
		}
		if slices.Contains(got, "PYENV_VERSION=3.9") {
			t.Error("old PYENV_VERSION should be filtered out")
		}
		if !slices.Contains(got, "PYENV_VERSION=3.10") {
			t.Error("new PYENV_VERSION should be present")
		}
		if !slices.Contains(got, "HOME=/home/user") {
			t.Error("HOME should be preserved")
		}
	})
}

func TestLookPathInEnv(t *testing.T) {
	t.Run("NameWithSeparator", func(t *testing.T) {
		name := "." + string(filepath.Separator) + "mybin"
		got := lookPathInEnv(name, nil)
		if got != name {
			t.Errorf("expected %q, got %q", name, got)
		}
	})

	t.Run("FoundInPATH", func(t *testing.T) {
		tmpDir := t.TempDir()
		binPath := filepath.Join(tmpDir, "mytool")
		if err := os.WriteFile(binPath, []byte("#!/bin/sh"), 0o755); err != nil {
			t.Fatal(err)
		}

		env := []string{"PATH=" + tmpDir}
		got := lookPathInEnv("mytool", env)
		if got != binPath {
			t.Errorf("expected %q, got %q", binPath, got)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		env := []string{"PATH=/nonexistent"}
		got := lookPathInEnv("nosuchbin", env)
		if got != "nosuchbin" {
			t.Errorf("expected original name, got %q", got)
		}
	})

	t.Run("EmptyPATH", func(t *testing.T) {
		env := []string{"HOME=/home/user"}
		got := lookPathInEnv("mybin", env)
		if got != "mybin" {
			t.Errorf("expected original name, got %q", got)
		}
	})
}

func TestPrependBinToPath(t *testing.T) {
	// Save and restore findGitRootFunc.
	origFunc := findGitRootFunc
	findGitRootFunc = func() string { return "/repo" }
	defer func() { findGitRootFunc = origFunc }()

	environ := []string{"HOME=/home/user", "PATH=/usr/bin:/usr/local/bin"}
	got := prependBinToPath(environ)

	var pathValue string
	for _, e := range got {
		if len(e) > 5 && e[:5] == "PATH=" {
			pathValue = e[5:]
			break
		}
	}

	binDir := filepath.Join("/repo", ".pocket", "bin")
	if pathValue == "" {
		t.Fatal("PATH not found in result")
	}
	if pathValue[:len(binDir)] != binDir {
		t.Errorf("expected PATH to start with %q, got %q", binDir, pathValue)
	}
}
