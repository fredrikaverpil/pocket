package pocket

import (
	"context"
	"testing"
)

func TestSerial_ExecutionMode(t *testing.T) {
	var executed []string

	fn1 := func(_ context.Context) error {
		executed = append(executed, "fn1")
		return nil
	}
	fn2 := func(_ context.Context) error {
		executed = append(executed, "fn2")
		return nil
	}

	// Create execution context
	out := StdOutput()
	ec := newExecContext(out, ".", false)
	ctx := withExecContext(context.Background(), ec)

	// Execute in serial - should work with plain functions
	Serial(ctx, fn1, fn2)

	if len(executed) != 2 {
		t.Errorf("expected 2 executions, got %d", len(executed))
	}
	if executed[0] != "fn1" || executed[1] != "fn2" {
		t.Errorf("wrong execution order: %v", executed)
	}
}

func TestParallel_ExecutionMode(t *testing.T) {
	executed := make(chan string, 2)

	fn1 := func(_ context.Context) error {
		executed <- "fn1"
		return nil
	}
	fn2 := func(_ context.Context) error {
		executed <- "fn2"
		return nil
	}

	// Create execution context
	out := StdOutput()
	ec := newExecContext(out, ".", false)
	ctx := withExecContext(context.Background(), ec)

	// Execute in parallel - should work with plain functions
	Parallel(ctx, fn1, fn2)

	close(executed)
	results := make([]string, 0, 2)
	for s := range executed {
		results = append(results, s)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 executions, got %d", len(results))
	}
}
