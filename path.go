// Package pocket provides core utilities for the pocket build system.
package pocket

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	// DirName is the name of the pocket directory.
	DirName = ".pocket"
	// ToolsDirName is the name of the tools subdirectory.
	ToolsDirName = "tools"
	// BinDirName is the name of the bin subdirectory (for symlinks).
	BinDirName = "bin"
)

var (
	gitRootOnce sync.Once
	gitRoot     string
)

// GitRoot returns the root directory of the git repository.
func GitRoot() string {
	gitRootOnce.Do(func() {
		var err error
		gitRoot, err = findGitRoot()
		if err != nil {
			panic("pocket: unable to find git root: " + err.Error())
		}
	})
	return gitRoot
}

func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// FromGitRoot returns a path relative to the git root.
func FromGitRoot(elem ...string) string {
	return filepath.Join(append([]string{GitRoot()}, elem...)...)
}

// FromPocketDir returns a path relative to the .pocket directory.
func FromPocketDir(elem ...string) string {
	return FromGitRoot(append([]string{DirName}, elem...)...)
}

// FromToolsDir returns a path relative to the .pocket/tools directory.
func FromToolsDir(elem ...string) string {
	return FromPocketDir(append([]string{ToolsDirName}, elem...)...)
}

// FromBinDir returns a path relative to the .pocket/bin directory.
// If no elements are provided, returns the bin directory itself.
func FromBinDir(elem ...string) string {
	return FromPocketDir(append([]string{BinDirName}, elem...)...)
}

// BinaryName returns the binary name with the correct extension for the current OS.
// On Windows, it appends ".exe" to the name.
func BinaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
