package pocket

import (
	"runtime"
	"strings"
	"unicode"
)

// OS name constants matching runtime.GOOS values.
const (
	Darwin  = "darwin"
	Linux   = "linux"
	Windows = "windows"
)

// Architecture constants in various naming conventions.
const (
	// Go-style architecture names (matching runtime.GOARCH).
	AMD64 = "amd64"
	ARM64 = "arm64"

	// Alternative naming conventions used by various tools.
	X8664   = "x86_64"
	AARCH64 = "aarch64"
	X64     = "x64"
)

// HostOS returns the current operating system (runtime.GOOS).
func HostOS() string {
	return runtime.GOOS
}

// HostArch returns the current architecture (runtime.GOARCH).
func HostArch() string {
	return runtime.GOARCH
}

// ArchToX8664 converts Go-style architecture names to x86_64/aarch64 naming.
//
//	amd64 -> x86_64
//	arm64 -> aarch64
//
// Other values are returned unchanged.
func ArchToX8664(arch string) string {
	switch arch {
	case AMD64:
		return X8664
	case ARM64:
		return AARCH64
	default:
		return arch
	}
}

// ArchToAMD64 converts x86_64/aarch64 naming to Go-style architecture names.
//
//	x86_64 -> amd64
//	aarch64 -> arm64
//
// Other values are returned unchanged.
func ArchToAMD64(arch string) string {
	switch arch {
	case X8664:
		return AMD64
	case AARCH64:
		return ARM64
	default:
		return arch
	}
}

// ArchToX64 converts Go-style architecture names to x64/arm64 naming.
//
//	amd64 -> x64
//	arm64 -> arm64 (unchanged)
//
// Other values are returned unchanged.
func ArchToX64(arch string) string {
	if arch == AMD64 {
		return X64
	}
	return arch
}

// OSToTitle converts an OS name to title case.
//
//	darwin -> Darwin
//	linux -> Linux
//	windows -> Windows
func OSToTitle(os string) string {
	if os == "" {
		return ""
	}
	return strings.ToUpper(os[:1]) + os[1:]
}

// OSToUpper converts an OS name to uppercase.
//
//	darwin -> DARWIN
//	linux -> LINUX
func OSToUpper(os string) string {
	return strings.ToUpper(os)
}

// toInitialCap converts a string to initial capital (first letter uppercase).
func toInitialCap(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// DefaultArchiveFormat returns the typical archive format for the current OS.
// Returns "zip" on Windows, "tar.gz" on other platforms.
func DefaultArchiveFormat() string {
	if runtime.GOOS == Windows {
		return "zip"
	}
	return "tar.gz"
}

// DefaultArchiveFormatFor returns the typical archive format for the given OS.
// Returns "zip" for Windows, "tar.gz" for other platforms.
func DefaultArchiveFormatFor(os string) string {
	if os == Windows {
		return "zip"
	}
	return "tar.gz"
}
