package pk

// DefaultSkipDirs contains the default directories to skip during filesystem walking.
// Use this when extending the defaults: append(pk.DefaultSkipDirs, "custom").
//
// Note: Hidden directories (starting with ".") are skipped separately via
// IncludeHiddenDirs, so .venv, .cache, .git etc. don't need to be listed here.
var DefaultSkipDirs = []string{
	"vendor",       // Go, PHP, Ruby dependencies
	"node_modules", // Node.js dependencies
	"dist",         // Build output (JS/TS)
	"__pycache__",  // Python bytecode cache
	"venv",         // Python virtual environment (non-hidden variant)
}

// Config represents the Pocket configuration.
// It holds the task graph composition and manual tasks.
type Config struct {
	// Auto is the top-level runnable that composes all tasks.
	// This is typically created using Serial() to compose multiple tasks.
	// Tasks in Auto are executed on bare `./pok` invocation.
	Auto Runnable

	// Manual contains tasks that only run when explicitly invoked.
	// These tasks are not executed as part of Auto on bare `./pok`.
	// Example: deploy tasks, setup scripts, or tasks requiring specific flags.
	//
	// Manual: []pk.Runnable{
	//	    Deploy,
	//	    Hello.Manual(),
	//	}
	Manual []Runnable

	// SkipDirs specifies directory names to skip during filesystem walking.
	//
	// Behavior:
	//   - nil: uses DefaultSkipDirs (vendor, node_modules, dist, __pycache__, venv)
	//   - empty slice: skips nothing
	//   - non-empty: skips exactly these directories
	//
	// Example to extend defaults:
	//   SkipDirs: append(pk.DefaultSkipDirs, "testdata", "generated")
	SkipDirs []string

	// IncludeHiddenDirs controls whether hidden directories (starting with ".")
	// are included during filesystem walking.
	//
	// Default (false): hidden directories are skipped (.git, .cache, .venv, etc.)
	// Set to true to include hidden directories in the walk.
	//
	// Note: Even with IncludeHiddenDirs=true, you can still exclude specific
	// hidden directories via SkipDirs: []string{".cache", ".venv"}
	IncludeHiddenDirs bool
}
