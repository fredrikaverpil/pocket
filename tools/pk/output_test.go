package pk

import (
	"bytes"
	"context"
	"testing"
)

func TestBufferedOutput_Flush(t *testing.T) {
	var parentStdout, parentStderr bytes.Buffer
	parent := &Output{Stdout: &parentStdout, Stderr: &parentStderr}

	buf := newBufferedOutput(parent)
	out := buf.Output()

	_, _ = out.Stdout.Write([]byte("hello stdout"))
	_, _ = out.Stderr.Write([]byte("hello stderr"))

	buf.Flush()

	if got := parentStdout.String(); got != "hello stdout" {
		t.Errorf("stdout: expected %q, got %q", "hello stdout", got)
	}
	if got := parentStderr.String(); got != "hello stderr" {
		t.Errorf("stderr: expected %q, got %q", "hello stderr", got)
	}
}

func TestBufferedOutput_FlushEmpty(t *testing.T) {
	var parentStdout, parentStderr bytes.Buffer
	parent := &Output{Stdout: &parentStdout, Stderr: &parentStderr}

	buf := newBufferedOutput(parent)
	buf.Flush() // Should be a no-op.

	if parentStdout.Len() != 0 {
		t.Error("expected no stdout output from empty flush")
	}
	if parentStderr.Len() != 0 {
		t.Error("expected no stderr output from empty flush")
	}
}

func TestOutputFromContext(t *testing.T) {
	t.Run("ReturnsSetOutput", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{Stdout: &buf, Stderr: &buf}
		ctx := context.WithValue(context.Background(), outputKey{}, out)

		got := outputFromContext(ctx)
		if got != out {
			t.Error("expected to get the same output back")
		}
	})

	t.Run("FallsBackToStdOutput", func(t *testing.T) {
		got := outputFromContext(context.Background())
		if got == nil {
			t.Fatal("expected non-nil fallback output")
		}
		// StdOutput should have os.Stdout and os.Stderr.
		if got.Stdout == nil || got.Stderr == nil {
			t.Error("expected non-nil Stdout and Stderr")
		}
	})
}

func TestPrintf(t *testing.T) {
	var buf bytes.Buffer
	out := &Output{Stdout: &buf, Stderr: &bytes.Buffer{}}
	ctx := context.WithValue(context.Background(), outputKey{}, out)

	Printf(ctx, "hello %s", "world")

	if got := buf.String(); got != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got)
	}
}

func TestErrorf(t *testing.T) {
	var buf bytes.Buffer
	out := &Output{Stdout: &bytes.Buffer{}, Stderr: &buf}
	ctx := context.WithValue(context.Background(), outputKey{}, out)

	Errorf(ctx, "error: %d", 42)

	if got := buf.String(); got != "error: 42" {
		t.Errorf("expected %q, got %q", "error: 42", got)
	}
}

func TestPrintln(t *testing.T) {
	var buf bytes.Buffer
	out := &Output{Stdout: &buf, Stderr: &bytes.Buffer{}}
	ctx := context.WithValue(context.Background(), outputKey{}, out)

	Println(ctx, "hello")

	if got := buf.String(); got != "hello\n" {
		t.Errorf("expected %q, got %q", "hello\n", got)
	}
}
