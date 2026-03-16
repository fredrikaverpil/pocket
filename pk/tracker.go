package pk

import (
	"context"
	"sync"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// taskID uniquely identifies a task execution for deduplication.
type taskID struct {
	Name string
	Path string
}

// String implements fmt.Stringer for debugging and logging.
func (id taskID) String() string {
	return id.Name + "@" + id.Path
}

// executionTracker tracks which tasks have already executed.
// It is safe for concurrent use.
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
		return true
	}
	t.done[id] = true
	return false
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
	return engine.SetTracker(ctx, t)
}

// executionTrackerFromContext returns the execution tracker from the context.
// Returns nil if no tracker is set.
func executionTrackerFromContext(ctx context.Context) *executionTracker {
	v := engine.TrackerFromContext(ctx)
	if v == nil {
		return nil
	}
	return v.(*executionTracker)
}

// MarkWarning records that a warning was detected during execution.
// Satisfies the engine.WarningMarker interface.
func (t *executionTracker) MarkWarning() {
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
