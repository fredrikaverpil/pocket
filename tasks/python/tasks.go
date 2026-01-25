// Package python provides task bundles for Python projects.
package python

import (
	"context"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

// versionKey is the context key for Python version.
type versionKey struct{}

// coverageKey is the context key for coverage enabled flag.
type coverageKey struct{}

// WithVersion sets the Python version for tasks in this scope.
// Use with pk.WithName to also suffix the task names.
//
// Example:
//
//	pk.WithOptions(
//	    python.Tasks(),
//	    pk.WithName("3.9"),
//	    python.WithVersion("3.9"),
//	    pk.WithDetect(python.Detect()),
//	)
func WithVersion(version string) pk.PathOption {
	return pk.WithContextValue(versionKey{}, version)
}

// WithCoverage enables coverage for the test task.
//
// Example:
//
//	pk.WithOptions(
//	    python.Test,
//	    pk.WithName("3.9"),
//	    python.WithVersion("3.9"),
//	    python.WithCoverage(),
//	)
func WithCoverage() pk.PathOption {
	return pk.WithContextValue(coverageKey{}, true)
}

// VersionFromContext returns the Python version from context.
// Returns empty string if not set.
func VersionFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(versionKey{}).(string); ok {
		return v
	}
	return ""
}

// coverageFromContext returns whether coverage is enabled.
func coverageFromContext(ctx context.Context) bool {
	if v, ok := ctx.Value(coverageKey{}).(bool); ok {
		return v
	}
	return false
}

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
// Use with WithVersion and pk.WithName to specify the Python version.
//
// Example:
//
//	pk.WithOptions(
//	    python.Tasks(),
//	    pk.WithName("3.9"),
//	    python.WithVersion("3.9"),
//	    pk.WithDetect(python.Detect()),
//	)
func Tasks() pk.Runnable {
	return pk.Serial(uv.Install, Format, Lint, Typecheck, Test)
}
