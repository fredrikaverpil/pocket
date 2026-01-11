package pocket

import "slices"

// executionState holds state shared across the entire execution tree.
// It is created once by the CLI and passed through all Runnables.
// Fields are immutable after creation.
type executionState struct {
	cwd     string        // where CLI was invoked (relative to git root)
	verbose bool          // verbose mode enabled
	dedup   *dedupTracker // tracks which tasks have run (thread-safe)
}

// newExecutionState creates an executionState for a new execution.
func newExecutionState(cwd string, verbose bool) *executionState {
	return &executionState{
		cwd:     cwd,
		verbose: verbose,
		dedup:   newDedupTracker(),
	}
}

// taskSetup holds configuration accumulated during Runnable tree traversal.
// It is modified by PathFilter and other wrappers before tasks execute.
type taskSetup struct {
	paths     map[string][]string          // task name -> resolved paths
	args      map[string]map[string]string // task name -> CLI args
	skipRules []skipRule                   // accumulated skip rules
}

// newTaskSetup creates an empty taskSetup.
func newTaskSetup() *taskSetup {
	return &taskSetup{
		paths: make(map[string][]string),
		args:  make(map[string]map[string]string),
	}
}

// withSkipRules returns a copy with additional skip rules appended.
func (ts *taskSetup) withSkipRules(rules []skipRule) *taskSetup {
	return &taskSetup{
		paths:     ts.paths, // shared, set up front
		args:      ts.args,  // shared, set up front
		skipRules: append(slices.Clone(ts.skipRules), rules...),
	}
}
