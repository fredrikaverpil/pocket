package pocket

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Runnable is anything that can be executed as part of the build.
// Both individual tasks and task groups implement this interface.
type Runnable interface {
	// Run executes this runnable.
	Run(ctx context.Context) error

	// Tasks returns all tasks contained in this runnable (for CLI registration).
	Tasks() []*Task
}

// Detectable is an optional interface for Runnables that support auto-detection.
// When a Runnable implements this interface, P(r).Detect() will use the
// DefaultDetect function to find directories where the Runnable should run.
type Detectable interface {
	// DefaultDetect returns a function that detects directories where this
	// Runnable should run. The returned paths should be relative to git root.
	DefaultDetect() func() []string
}

// serial runs children in order, stopping on first error.
type serial struct {
	children []Runnable
}

// Serial creates a Runnable that executes children sequentially.
// Execution stops on first error.
func Serial(children ...Runnable) Runnable {
	return &serial{children: children}
}

func (s *serial) Run(ctx context.Context) error {
	for _, child := range s.children {
		if child == nil {
			continue
		}
		if err := child.Run(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *serial) Tasks() []*Task {
	var tasks []*Task
	for _, child := range s.children {
		if child != nil {
			tasks = append(tasks, child.Tasks()...)
		}
	}
	return tasks
}

// Children returns the child Runnables for tree traversal.
func (s *serial) Children() []Runnable {
	return s.children
}

// parallel runs children concurrently, waiting for all to complete.
type parallel struct {
	children []Runnable
}

// Parallel creates a Runnable that executes children concurrently.
// Waits for all children to complete, returns first error encountered.
func Parallel(children ...Runnable) Runnable {
	return &parallel{children: children}
}

func (p *parallel) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, child := range p.children {
		if child == nil {
			continue
		}
		g.Go(func() error {
			return child.Run(ctx)
		})
	}
	return g.Wait()
}

func (p *parallel) Tasks() []*Task {
	var tasks []*Task
	for _, child := range p.children {
		if child != nil {
			tasks = append(tasks, child.Tasks()...)
		}
	}
	return tasks
}

// Children returns the child Runnables for tree traversal.
func (p *parallel) Children() []Runnable {
	return p.children
}
