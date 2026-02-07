package pk

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSerial_RunsInOrder(t *testing.T) {
	var order []int

	s := Serial(
		Do(func(_ context.Context) error { order = append(order, 1); return nil }),
		Do(func(_ context.Context) error { order = append(order, 2); return nil }),
		Do(func(_ context.Context) error { order = append(order, 3); return nil }),
	)

	if err := s.run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", order)
	}
}

func TestSerial_StopsOnFirstError(t *testing.T) {
	var ran []int
	errBoom := errors.New("boom")

	s := Serial(
		Do(func(_ context.Context) error { ran = append(ran, 1); return nil }),
		Do(func(_ context.Context) error { ran = append(ran, 2); return errBoom }),
		Do(func(_ context.Context) error { ran = append(ran, 3); return nil }),
	)

	err := s.run(context.Background())
	if !errors.Is(err, errBoom) {
		t.Errorf("expected errBoom, got %v", err)
	}
	if len(ran) != 2 || ran[0] != 1 || ran[1] != 2 {
		t.Errorf("expected [1 2], got %v", ran)
	}
}

func TestSerial_Empty(t *testing.T) {
	s := Serial()
	if err := s.run(context.Background()); err != nil {
		t.Errorf("expected nil for empty serial, got %v", err)
	}
}

func TestParallel_RunsConcurrently(t *testing.T) {
	var count atomic.Int32

	p := Parallel(
		Do(func(_ context.Context) error {
			count.Add(1)
			time.Sleep(10 * time.Millisecond)
			return nil
		}),
		Do(func(_ context.Context) error {
			count.Add(1)
			time.Sleep(10 * time.Millisecond)
			return nil
		}),
		Do(func(_ context.Context) error {
			count.Add(1)
			time.Sleep(10 * time.Millisecond)
			return nil
		}),
	)

	ctx := context.Background()
	ctx = context.WithValue(ctx, outputKey{}, testOutput())

	if err := p.run(ctx); err != nil {
		t.Fatal(err)
	}

	if got := count.Load(); got != 3 {
		t.Errorf("expected 3 tasks to run, got %d", got)
	}
}

func TestParallel_ReturnsError(t *testing.T) {
	errBoom := errors.New("boom")
	var count atomic.Int32

	p := Parallel(
		Do(func(_ context.Context) error {
			count.Add(1)
			return nil
		}),
		Do(func(_ context.Context) error {
			count.Add(1)
			return errBoom
		}),
		Do(func(_ context.Context) error {
			// This may or may not run depending on goroutine scheduling,
			// but all are started.
			count.Add(1)
			return nil
		}),
	)

	ctx := context.Background()
	ctx = context.WithValue(ctx, outputKey{}, testOutput())

	err := p.run(ctx)
	if err == nil {
		t.Fatal("expected error from parallel")
	}
}

func TestParallel_Empty(t *testing.T) {
	p := Parallel()
	if err := p.run(context.Background()); err != nil {
		t.Errorf("expected nil for empty parallel, got %v", err)
	}
}

func TestParallel_SingleItemNoBuf(t *testing.T) {
	var ran bool
	p := Parallel(
		Do(func(_ context.Context) error {
			ran = true
			return nil
		}),
	)

	if err := p.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Error("expected single item to run")
	}
}

func TestParallel_OutputBuffering(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := &Output{Stdout: &stdout, Stderr: &stderr}

	// Two tasks that write to output.
	p := Parallel(
		Do(func(ctx context.Context) error {
			o := outputFromContext(ctx)
			for range 50 {
				_, _ = o.Stdout.Write([]byte("A"))
			}
			return nil
		}),
		Do(func(ctx context.Context) error {
			o := outputFromContext(ctx)
			for range 50 {
				_, _ = o.Stdout.Write([]byte("B"))
			}
			return nil
		}),
	)

	ctx := context.WithValue(context.Background(), outputKey{}, out)
	if err := p.run(ctx); err != nil {
		t.Fatal(err)
	}

	result := stdout.String()
	if len(result) != 100 {
		t.Fatalf("expected 100 chars, got %d", len(result))
	}

	// Output should be in contiguous blocks (not interleaved) because of buffering.
	// Each task's output is flushed atomically.
	trimA := strings.TrimLeft(result, "A")
	trimB := strings.TrimLeft(trimA, "B")
	trimA2 := strings.TrimLeft(result, "B")
	trimB2 := strings.TrimLeft(trimA2, "A")

	// Either all A's then all B's, or all B's then all A's.
	if trimB != "" && trimB2 != "" {
		t.Errorf("expected contiguous blocks, got interleaved output: %s", result)
	}
}

func TestParallel_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	p := Parallel(
		Do(func(_ context.Context) error {
			t.Error("should not run with cancelled context")
			return nil
		}),
	)

	err := p.run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// testOutput returns an Output that discards all output.
func testOutput() *Output {
	return &Output{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}
}
