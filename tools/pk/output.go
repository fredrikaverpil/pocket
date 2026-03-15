package pk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
)

// outputKey is the context key for output writers.
// Used for stdout/stderr redirection during task execution.
type outputKey struct{}

// Output holds stdout and stderr writers for task output.
type Output struct {
	Stdout io.Writer
	Stderr io.Writer
}

// StdOutput returns an Output that writes to os.Stdout and os.Stderr.
func StdOutput() *Output {
	return &Output{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// bufferedOutput captures output per-goroutine for parallel execution.
// Flushes to parent Output on completion.
type bufferedOutput struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	parent *Output
}

// newBufferedOutput creates a new buffered output that will flush to parent.
func newBufferedOutput(parent *Output) *bufferedOutput {
	return &bufferedOutput{
		stdout: new(bytes.Buffer),
		stderr: new(bytes.Buffer),
		parent: parent,
	}
}

// Output returns an Output that writes to the internal buffers.
func (b *bufferedOutput) Output() *Output {
	return &Output{
		Stdout: b.stdout,
		Stderr: b.stderr,
	}
}

// Flush writes all buffered content to the parent output.
// This should be called with external synchronization when used in parallel.
func (b *bufferedOutput) Flush() {
	if b.stdout.Len() > 0 {
		_, _ = b.parent.Stdout.Write(b.stdout.Bytes())
	}
	if b.stderr.Len() > 0 {
		_, _ = b.parent.Stderr.Write(b.stderr.Bytes())
	}
}

// outputFromContext returns the Output from the context.
// Returns StdOutput() if no output is set.
func outputFromContext(ctx context.Context) *Output {
	if out, ok := ctx.Value(outputKey{}).(*Output); ok {
		return out
	}
	return StdOutput()
}

// Printf formats and prints to the output in the context.
func Printf(ctx context.Context, format string, a ...any) {
	out := outputFromContext(ctx)
	fmt.Fprintf(out.Stdout, format, a...)
}

// Println prints to the output in the context.
func Println(ctx context.Context, a ...any) {
	out := outputFromContext(ctx)
	fmt.Fprintln(out.Stdout, a...)
}

// Errorf formats and prints to the error output in the context.
func Errorf(ctx context.Context, format string, a ...any) {
	out := outputFromContext(ctx)
	fmt.Fprintf(out.Stderr, format, a...)
}
