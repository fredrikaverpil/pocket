package pk

import (
	"context"
	"sync"

	"github.com/fredrikaverpil/pocket/pk/pcontext"
)

// trackerKey is the context key for the execution tracker.
// Used internally for task deduplication.
type trackerKey struct{}

// taskID uniquely identifies a task execution for deduplication.
// Used directly as a map key (Go structs with comparable fields are valid map keys).
type taskID struct {
	Name string // Effective name (may include suffix like "py-test:3.9")
	Path string // Execution path relative to git root
}

// String implements fmt.Stringer for debugging and logging.
func (id taskID) String() string {
	return id.Name + "@" + id.Path
}

// executionTracker tracks which tasks have already executed.
// It is safe for concurrent use. Deduplication is by taskID (effective name + path) -
// the same task can run multiple times if configured for different paths or suffixes,
// but will only run once per unique combination.
type executionTracker struct {
	mu          sync.Mutex
	done        map[taskID]bool
	hadWarnings bool
}

// newExecutionTracker creates a new execution tracker.
func newExecutionTracker() *executionTracker {
	return &executionTracker{
		done: make(map[taskID]bool),
	}
}

// markDone records that a task has executed.
// Returns true if it was already done (should skip), false if first time.
func (t *executionTracker) markDone(id taskID) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done[id] {
		return true // already done, should skip
	}
	t.done[id] = true
	return false // first time
}

// executedTaskPath represents a task that was executed at a specific path.
type executedTaskPath struct {
	TaskName string
	Path     string
}

// executed returns all task+path combinations that have been executed.
func (t *executionTracker) executed() []executedTaskPath {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]executedTaskPath, 0, len(t.done))
	for id := range t.done {
		result = append(result, executedTaskPath{
			TaskName: id.Name,
			Path:     id.Path,
		})
	}
	return result
}

// withExecutionTracker returns a new context with the given tracker set.
func withExecutionTracker(ctx context.Context, t *executionTracker) context.Context {
	return context.WithValue(ctx, trackerKey{}, t)
}

// executionTrackerFromContext returns the execution tracker from the context.
// Returns nil if no tracker is set.
func executionTrackerFromContext(ctx context.Context) *executionTracker {
	if t, ok := ctx.Value(trackerKey{}).(*executionTracker); ok {
		return t
	}
	return nil
}

// withForceRun returns a new context with forceRun set to true.
func withForceRun(c context.Context) context.Context {
	return context.WithValue(c, pcontext.ForceRunKey, true)
}

// forceRunFromContext returns whether forceRun is set in the context.
func forceRunFromContext(c context.Context) bool {
	if v, ok := c.Value(pcontext.ForceRunKey).(bool); ok {
		return v
	}
	return false
}

// markWarning records that a warning was detected during execution.
func (t *executionTracker) markWarning() {
	t.mu.Lock()
	t.hadWarnings = true
	t.mu.Unlock()
}

// warnings returns true if any warnings were detected.
func (t *executionTracker) warnings() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.hadWarnings
}
