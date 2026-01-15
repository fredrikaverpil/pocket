package pocket

import "testing"

func TestArchToX8664(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{AMD64, X8664},
		{ARM64, AARCH64},
		{"386", "386"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ArchToX8664(tt.input); got != tt.want {
			t.Errorf("ArchToX8664(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestArchToAMD64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{X8664, AMD64},
		{AARCH64, ARM64},
		{"386", "386"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ArchToAMD64(tt.input); got != tt.want {
			t.Errorf("ArchToAMD64(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestArchToX64(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{AMD64, X64},
		{ARM64, ARM64},
		{"386", "386"},
	}
	for _, tt := range tests {
		if got := ArchToX64(tt.input); got != tt.want {
			t.Errorf("ArchToX64(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOSToTitle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{Darwin, "Darwin"},
		{Linux, "Linux"},
		{Windows, "Windows"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := OSToTitle(tt.input); got != tt.want {
			t.Errorf("OSToTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOSToUpper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{Darwin, "DARWIN"},
		{Linux, "LINUX"},
		{Windows, "WINDOWS"},
	}
	for _, tt := range tests {
		if got := OSToUpper(tt.input); got != tt.want {
			t.Errorf("OSToUpper(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func Test_toInitialCap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"darwin", "Darwin"},
		{"macOS", "MacOS"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := toInitialCap(tt.input); got != tt.want {
			t.Errorf("toInitialCap(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefaultArchiveFormatFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		os   string
		want string
	}{
		{Windows, "zip"},
		{Darwin, "tar.gz"},
		{Linux, "tar.gz"},
	}
	for _, tt := range tests {
		if got := DefaultArchiveFormatFor(tt.os); got != tt.want {
			t.Errorf("DefaultArchiveFormatFor(%q) = %q, want %q", tt.os, got, tt.want)
		}
	}
}
