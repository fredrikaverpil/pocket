// Package python provides task bundles for Python projects.
package python

import (
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// pythonVersionToRuff converts a Python version (e.g., "3.9") to ruff's format (e.g., "py39").
func pythonVersionToRuff(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return "py" + parts[0] + parts[1]
	}
	return "py" + strings.ReplaceAll(version, ".", "")
}

// Detect returns a DetectFunc that finds Python projects.
func Detect() pk.DetectFunc {
	return pk.DetectByFile("pyproject.toml", "uv.lock", "setup.py", "setup.cfg")
}

// Tasks returns all Python tasks (Format, Lint, Typecheck, Test).
// Use with WithVersion to specify the Python version.
//
// Example:
//
//	pk.WithOptions(
//	    python.Tasks(),
//	    python.WithVersion("3.9"),
//	    python.WithTestCoverage(),
//	    pk.WithDetect(python.Detect()),
//	)
func Tasks() pk.Runnable {
	return pk.Serial(uv.Install, Format, Lint, pk.Parallel(Typecheck, Test))
}
