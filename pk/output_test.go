package pk

import (
	"bytes"
	"context"
	"testing"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

func TestBufferedOutput_Flush(t *testing.T) {
	var parentStdout, parentStderr bytes.Buffer
	parent := &engine.Output{Stdout: &parentStdout, Stderr: &parentStderr}

	buf := engine.NewBufferedOutput(parent)
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
	parent := &engine.Output{Stdout: &parentStdout, Stderr: &parentStderr}

	buf := engine.NewBufferedOutput(parent)
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
		out := &engine.Output{Stdout: &buf, Stderr: &buf}
		ctx := engine.SetOutput(context.Background(), out)

		got := engine.OutputFromContext(ctx)
		if got != out {
			t.Error("expected to get the same output back")
		}
	})

	t.Run("ReturnsNilWhenNotSet", func(t *testing.T) {
		got := engine.OutputFromContext(context.Background())
		if got != nil {
			t.Error("expected nil when no output set in context")
		}
	})
}

func TestPrintf(t *testing.T) {
	var buf bytes.Buffer
	out := &engine.Output{Stdout: &buf, Stderr: &bytes.Buffer{}}
	ctx := engine.SetOutput(context.Background(), out)

	engine.Printf(ctx, "hello %s", "world")

	if got := buf.String(); got != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got)
	}
}

func TestErrorf(t *testing.T) {
	var buf bytes.Buffer
	out := &engine.Output{Stdout: &bytes.Buffer{}, Stderr: &buf}
	ctx := engine.SetOutput(context.Background(), out)

	engine.Errorf(ctx, "error: %d", 42)

	if got := buf.String(); got != "error: 42" {
		t.Errorf("expected %q, got %q", "error: 42", got)
	}
}

func TestPrintln(t *testing.T) {
	var buf bytes.Buffer
	out := &engine.Output{Stdout: &buf, Stderr: &bytes.Buffer{}}
	ctx := engine.SetOutput(context.Background(), out)

	engine.Println(ctx, "hello")

	if got := buf.String(); got != "hello\n" {
		t.Errorf("expected %q, got %q", "hello\n", got)
	}
}
