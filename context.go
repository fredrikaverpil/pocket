package pocket

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sync"
)

// execContext holds runtime state for function execution.
// Stored in context.Context and accessed via helper functions.
type execContext struct {
	mode       execMode            // execution mode (execute or collect)
	plan       *ExecutionPlan      // plan being collected (only in modeCollect)
	configPlan *ConfigPlan         // the full config plan (for tasks that need it)
	out        *Output             // where to write output
	path       string              // current path for this invocation
	cwd        string              // where CLI was invoked (relative to git root)
	verbose    bool                // verbose mode enabled
	dedup      *dedupState         // shared deduplication state (thread-safe)
	skipRules  map[string][]string // task name -> paths to skip in (empty = skip everywhere)
}

// dedupState tracks executed runnables for deduplication.
// Shared across parallel executions with thread-safe access.
type dedupState struct {
	mu       sync.Mutex
	executed map[uintptr]bool
}

// newDedupState creates a new deduplication state.
func newDedupState() *dedupState {
	return &dedupState{
		executed: make(map[uintptr]bool),
	}
}

// shouldRun checks if a runnable should run (not already executed).
// Marks it as executed if it should run. Thread-safe.
func (d *dedupState) shouldRun(key uintptr) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.executed[key] {
		return false
	}
	d.executed[key] = true
	return true
}

// contextKey is the type for context keys to avoid collisions.
type contextKey int

const (
	execContextKey contextKey = iota
)

// newExecContext creates a new execution context.
func newExecContext(out *Output, cwd string, verbose bool, configPlan *ConfigPlan) *execContext {
	return &execContext{
		mode:       modeExecute, // explicit for clarity (default is execute)
		configPlan: configPlan,
		out:        out,
		cwd:        cwd,
		verbose:    verbose,
		dedup:      newDedupState(),
	}
}

// withExecContext returns a new context with the execContext attached.
func withExecContext(ctx context.Context, ec *execContext) context.Context {
	return context.WithValue(ctx, execContextKey, ec)
}

// getExecContext retrieves the execContext from the context.
// Panics if not present (programming error).
func getExecContext(ctx context.Context) *execContext {
	ec, ok := ctx.Value(execContextKey).(*execContext)
	if !ok {
		panic("pocket: execContext not found in context - are you calling from outside pocket.Func?")
	}
	return ec
}

// withPath returns a context with the path set for the current invocation.
func withPath(ctx context.Context, path string) context.Context {
	ec := getExecContext(ctx)
	// Create a shallow copy with the new path
	newEC := *ec
	newEC.path = path
	return withExecContext(ctx, &newEC)
}

// withOptions stores options for a function in the context.
// It normalizes to the struct type if a pointer is provided.
// Panics if the same options type is already in the context (nested functions
// cannot share the same options type, as the inner would shadow the outer).
func withOptions(ctx context.Context, opts any) context.Context {
	t := reflect.TypeOf(opts)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Detect shadowing: nested functions cannot use the same options type
	if ctx.Value(t) != nil {
		panic(
			fmt.Sprintf(
				"pocket: options type %s already in context; nested functions cannot share the same options type",
				t,
			),
		)
	}

	if reflect.TypeOf(opts).Kind() == reflect.Pointer {
		return context.WithValue(ctx, t, reflect.ValueOf(opts).Elem().Interface())
	}
	return context.WithValue(ctx, t, opts)
}

// Options retrieves typed options from the context.
// It handles both struct and pointer types for T, always looking up the base struct type.
func Options[T any](ctx context.Context) T {
	var zero T
	t := reflect.TypeOf(zero)
	isPtr := t.Kind() == reflect.Pointer
	if isPtr {
		t = t.Elem()
	}

	val := ctx.Value(t)
	if val == nil {
		return zero
	}

	if isPtr {
		// If T is *Struct, return a pointer to the stored Struct
		rv := reflect.New(t)
		rv.Elem().Set(reflect.ValueOf(val))
		return rv.Interface().(T)
	}

	return val.(T)
}

// Exec runs an external command with output directed to the current context.
// The command runs in the current path directory.
func Exec(ctx context.Context, name string, args ...string) error {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return nil
	}
	cmd := newCommand(ctx, name, args...)
	cmd.Stdout = ec.out.Stdout
	cmd.Stderr = ec.out.Stderr
	if ec.path != "" {
		cmd.Dir = FromGitRoot(ec.path)
	} else {
		cmd.Dir = GitRoot()
	}
	return cmd.Run()
}

// ExecIn runs an external command in a specific directory.
func ExecIn(ctx context.Context, dir, name string, args ...string) error {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return nil
	}
	cmd := newCommand(ctx, name, args...)
	cmd.Stdout = ec.out.Stdout
	cmd.Stderr = ec.out.Stderr
	cmd.Dir = dir
	return cmd.Run()
}

// Printf writes formatted output to stdout.
func Printf(ctx context.Context, format string, args ...any) {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return
	}
	fmt.Fprintf(ec.out.Stdout, format, args...)
}

// Println writes a line to stdout.
func Println(ctx context.Context, args ...any) {
	ec := getExecContext(ctx)
	if ec.mode == modeCollect {
		return
	}
	fmt.Fprintln(ec.out.Stdout, args...)
}

// printTaskHeader writes the task execution header to output.
func printTaskHeader(ctx context.Context, name string) {
	ec := getExecContext(ctx)
	if ec.path != "" && ec.path != "." {
		fmt.Fprintf(ec.out.Stdout, ":: %s [%s]\n", name, ec.path)
	} else {
		fmt.Fprintf(ec.out.Stdout, ":: %s\n", name)
	}
}

// Path returns the current execution path (relative to git root).
func Path(ctx context.Context) string {
	ec := getExecContext(ctx)
	if ec.path == "" {
		return "."
	}
	return ec.path
}

// Verbose returns whether verbose mode is enabled.
func Verbose(ctx context.Context) bool {
	return getExecContext(ctx).verbose
}

// CWD returns where the CLI was invoked (relative to git root).
func CWD(ctx context.Context) string {
	return getExecContext(ctx).cwd
}

// GetOutput returns the current output writers.
func GetOutput(ctx context.Context) *Output {
	return getExecContext(ctx).out
}

// TestContext creates a context suitable for testing.
func TestContext(out *Output) context.Context {
	ec := newExecContext(out, ".", false, nil)
	return withExecContext(context.Background(), ec)
}

// GetConfigPlan returns the ConfigPlan from context.
// Returns nil if not set (e.g., in collect mode or testing).
func GetConfigPlan(ctx context.Context) *ConfigPlan {
	return getExecContext(ctx).configPlan
}

// shouldSkipTask checks if a task should be skipped based on skip rules.
// Returns true if the task should be skipped for the current path.
func (ec *execContext) shouldSkipTask(taskName string) bool {
	if ec.skipRules == nil {
		return false
	}
	paths, exists := ec.skipRules[taskName]
	if !exists {
		return false
	}

	// Empty paths means skip everywhere
	if len(paths) == 0 {
		return true
	}

	// Check if current path matches any skip pattern
	currentPath := ec.path
	if currentPath == "" {
		currentPath = "."
	}
	for _, pattern := range paths {
		if matchSkipPath(currentPath, pattern) {
			return true
		}
	}
	return false
}

// matchSkipPath checks if the current path matches the skip pattern.
// Supports exact matches and regex patterns.
func matchSkipPath(currentPath, pattern string) bool {
	// Clean paths for comparison
	currentPath = filepath.Clean(currentPath)
	pattern = filepath.Clean(pattern)

	// Try exact match first
	if currentPath == pattern {
		return true
	}

	// Try as regex pattern
	re, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		// Invalid regex, treat as literal
		return false
	}

	return re.MatchString(currentPath)
}
