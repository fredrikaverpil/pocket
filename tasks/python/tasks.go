// Package python provides task bundles for Python projects.
package python

import (
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
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
	return pk.DetectByFile("pyproject.toml", "setup.py", "setup.cfg")
}

// Tasks returns Python-related tasks with auto-detection.
// If no tasks are provided, it defaults to running Typecheck and Test in parallel,
// then Format, then Lint (serial because format/lint modify files).
func Tasks(tasks ...pk.Runnable) pk.Runnable {
	if len(tasks) == 0 {
		return pk.WithOptions(
			pk.Serial(
				pk.Parallel(Typecheck, Test),
				Format,
				Lint,
			),
			pk.WithDetect(Detect()),
		)
	}

	return pk.WithOptions(
		pk.Serial(tasks...),
		pk.WithDetect(Detect()),
	)
}
