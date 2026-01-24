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

// CoverageMarker is a marker type for WithCoverage option.
type CoverageMarker struct{}

// WithCoverage enables coverage for the test task when used with WithPython.
func WithCoverage() CoverageMarker {
	return CoverageMarker{}
}

// WithPython creates Python tasks for a specific version.
// If no tasks are specified, all tasks (Format, Lint, Typecheck, Test) are included.
// Use WithCoverage() to enable coverage for the test task.
//
// Examples:
//
//	// All tasks with Python 3.9
//	python.WithPython("3.9")
//
//	// Specific tasks with Python 3.9
//	python.WithPython("3.9", python.Format, python.Lint)
//
//	// With coverage for test
//	python.WithPython("3.9", python.WithCoverage(), python.Format, python.Test)
func WithPython(version string, items ...any) pk.Runnable {
	coverage := false
	var tasks []*pk.Task

	for _, item := range items {
		switch v := item.(type) {
		case CoverageMarker:
			coverage = true
		case *pk.Task:
			tasks = append(tasks, v)
		}
	}

	// If no tasks specified, include all
	if len(tasks) == 0 {
		tasks = []*pk.Task{Format, Lint, Typecheck, Test}
	}

	var versioned []pk.Runnable
	versioned = append(versioned, uv.Install)

	for _, t := range tasks {
		switch t {
		case Format:
			versioned = append(versioned, formatWith(version))
		case Lint:
			versioned = append(versioned, lintWith(version))
		case Typecheck:
			versioned = append(versioned, typecheckWith(version))
		case Test:
			if coverage {
				versioned = append(versioned, testWithCoverage(version))
			} else {
				versioned = append(versioned, testWith(version))
			}
		default:
			// Unknown task, include as-is
			versioned = append(versioned, t)
		}
	}

	return pk.Serial(versioned...)
}
