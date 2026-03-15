package pk

import (
	"runtime"
	"testing"
)

func TestArchToX8664(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"amd64", "x86_64"},
		{"arm64", "aarch64"},
		{"riscv64", "riscv64"}, // passthrough
	}
	for _, tc := range tests {
		if got := ArchToX8664(tc.in); got != tc.want {
			t.Errorf("ArchToX8664(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestArchToX64(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"amd64", "x64"},
		{"arm64", "arm64"}, // unchanged
		{"riscv64", "riscv64"},
	}
	for _, tc := range tests {
		if got := ArchToX64(tc.in); got != tc.want {
			t.Errorf("ArchToX64(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestOSToTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"darwin", "Darwin"},
		{"linux", "Linux"},
		{"windows", "Windows"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := OSToTitle(tc.in); got != tc.want {
			t.Errorf("OSToTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHostOS(t *testing.T) {
	if got := HostOS(); got != runtime.GOOS {
		t.Errorf("HostOS() = %q, want %q", got, runtime.GOOS)
	}
}

func TestHostArch(t *testing.T) {
	if got := HostArch(); got != runtime.GOARCH {
		t.Errorf("HostArch() = %q, want %q", got, runtime.GOARCH)
	}
}

func TestBinaryName(t *testing.T) {
	got := BinaryName("mybin")
	if runtime.GOOS == "windows" {
		if got != "mybin.exe" {
			t.Errorf("expected mybin.exe on Windows, got %q", got)
		}
	} else {
		if got != "mybin" {
			t.Errorf("expected mybin on non-Windows, got %q", got)
		}
	}
}

func TestDefaultArchiveFormat(t *testing.T) {
	got := DefaultArchiveFormat()
	if runtime.GOOS == "windows" {
		if got != "zip" {
			t.Errorf("expected zip on Windows, got %q", got)
		}
	} else {
		if got != "tar.gz" {
			t.Errorf("expected tar.gz on non-Windows, got %q", got)
		}
	}
}
