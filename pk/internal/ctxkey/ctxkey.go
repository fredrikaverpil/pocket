// Package ctxkey defines context key types shared between pk and pk/run.
package ctxkey

type Path struct{}           // Current execution path.
type ForceRun struct{}       // Forcing task execution.
type Verbose struct{}        // Verbose mode.
type GitDiff struct{}        // Git diff enabled flag.
type CommitsCheck struct{}   // Commits check enabled flag.
type Env struct{}            // Environment variable overrides.
type NameSuffix struct{}     // Task name suffix.
type AutoExec struct{}       // Auto execution mode.
type TaskFlags struct{}      // Resolved task flag values.
type CLIFlags struct{}       // CLI-provided flag overrides.
type NoticePatterns struct{} // Custom notice patterns.
type Plan struct{}           // Execution plan.
type Tracker struct{}        // Execution tracker.
type Output struct{}         // Output writers.
