// Package ctxkey defines context key types shared between pk and pk/run.
package ctxkey

type (
	Path           struct{} // Current execution path.
	ForceRun       struct{} // Forcing task execution.
	Verbose        struct{} // Verbose mode.
	Serial         struct{} // Force serial execution of parallel tasks.
	GitDiff        struct{} // Git diff enabled flag.
	CommitsCheck   struct{} // Commits check enabled flag.
	Env            struct{} // Environment variable overrides.
	NameSuffix     struct{} // Task name suffix.
	AutoExec       struct{} // Auto execution mode.
	TaskFlags      struct{} // Resolved task flag values.
	CLIFlags       struct{} // CLI-provided flag overrides.
	NoticePatterns struct{} // Custom notice patterns.
	Plan           struct{} // Execution plan.
	Tracker        struct{} // Execution tracker.
	Output         struct{} // Output writers.
)
