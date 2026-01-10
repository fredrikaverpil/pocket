package pocket

// Config defines the configuration for a project using pocket.
type Config struct {
	// AutoRun defines the execution tree for ./pok (no arguments).
	// Use Serial() and Parallel() to control execution order.
	// All tasks in AutoRun execute when running ./pok without arguments.
	//
	// Example:
	//
	//	AutoRun: pocket.Serial(
	//	    pocket.AutoDetect(golang.Tasks()),
	//	    pocket.AutoDetect(python.Tasks()),
	//	),
	AutoRun Runnable

	// ManualRun registers additional tasks that only run when explicitly
	// invoked with ./pok <taskname>. These tasks appear in ./pok -h under
	// "Manual Tasks" and support the same wrappers as AutoRun (Paths, AutoDetect, etc.).
	//
	// Example:
	//
	//	ManualRun: []pocket.Runnable{
	//	    deployTask,
	//	    pocket.Paths(benchmarkTask).In("services/api"),
	//	},
	ManualRun []Runnable

	// Shim controls shim script generation.
	// By default, only Posix (./pok) is generated with name "pok".
	Shim *ShimConfig

	// SkipGitDiff disables the git diff check at the end of the "all" task.
	// By default, "all" fails if there are uncommitted changes after running all tasks.
	// Set to true to disable this check.
	SkipGitDiff bool
}

// ShimConfig controls shim script generation.
type ShimConfig struct {
	// Name is the base name of the generated shim scripts (without extension).
	// Default: "pok"
	Name string

	// Posix generates a bash script (./pok).
	// This is enabled by default if ShimConfig is nil.
	Posix bool

	// Windows generates a batch file (pok.cmd).
	// The batch file requires Go to be installed and in PATH.
	Windows bool

	// PowerShell generates a PowerShell script (pok.ps1).
	// The PowerShell script can auto-download Go if not found.
	PowerShell bool
}

// WithDefaults returns a copy of the config with default values applied.
func (c Config) WithDefaults() Config {
	// Default to Posix shim only if no Shim config is provided.
	if c.Shim == nil {
		c.Shim = &ShimConfig{Posix: true}
	}
	// Apply shim defaults.
	shim := *c.Shim
	if shim.Name == "" {
		shim.Name = "pok"
	}
	c.Shim = &shim

	return c
}
