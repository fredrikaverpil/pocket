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
// It holds the task graph composition and plan configuration.
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

	// Plan contains configuration for plan building and shim generation.
	//
	// Example:
	//   Plan: &pk.PlanConfig{
	//       Shims: pk.AllShimsConfig(),
	//   }
	Plan *PlanConfig
}

// PlanConfig contains configuration for plan building and shim generation.
type PlanConfig struct {
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

	// Shims controls which shim scripts are generated.
	//
	// Default (nil): generates only POSIX shim (pok)
	// Explicit: generates only the shims set to true
	//
	// Example to generate all shims:
	//   Shims: &pk.ShimConfig{Posix: true, Windows: true, PowerShell: true}
	Shims *ShimConfig
}

// ShimConfig controls which shim scripts are generated.
type ShimConfig struct {
	// Posix generates a POSIX shell script (pok).
	// This is the default if Shims is nil.
	Posix bool

	// Windows generates a Windows batch file (pok.cmd).
	Windows bool

	// PowerShell generates a PowerShell script (pok.ps1).
	PowerShell bool
}

// DefaultShimConfig returns the default shim configuration (POSIX only).
func DefaultShimConfig() *ShimConfig {
	return &ShimConfig{Posix: true}
}

// AllShimsConfig returns a shim configuration with all shims enabled.
func AllShimsConfig() *ShimConfig {
	return &ShimConfig{Posix: true, Windows: true, PowerShell: true}
}
