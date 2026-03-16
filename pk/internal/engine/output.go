package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
)

// Output holds the stdout and stderr writers used by Printf, Println,
// Errorf, and Exec. In parallel execution, each goroutine receives
// a buffered Output to prevent interleaved output.
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

// BufferedOutput captures output per-goroutine for parallel execution.
// Flushes to parent Output on completion.
type BufferedOutput struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	parent *Output
}

// NewBufferedOutput creates a new buffered output that will flush to parent.
func NewBufferedOutput(parent *Output) *BufferedOutput {
	return &BufferedOutput{
		stdout: new(bytes.Buffer),
		stderr: new(bytes.Buffer),
		parent: parent,
	}
}

// Output returns an Output that writes to the internal buffers.
func (b *BufferedOutput) Output() *Output {
	return &Output{
		Stdout: b.stdout,
		Stderr: b.stderr,
	}
}

// Flush writes all buffered content to the parent output.
// This should be called with external synchronization when used in parallel.
func (b *BufferedOutput) Flush() {
	if b.stdout.Len() > 0 {
		_, _ = b.parent.Stdout.Write(b.stdout.Bytes())
	}
	if b.stderr.Len() > 0 {
		_, _ = b.parent.Stderr.Write(b.stderr.Bytes())
	}
}

// Printf formats and writes to the context's stdout.
func Printf(ctx context.Context, format string, a ...any) {
	out := outputFromContext(ctx)
	fmt.Fprintf(out.Stdout, format, a...)
}

// Println writes to the context's stdout, appending a newline.
func Println(ctx context.Context, a ...any) {
	out := outputFromContext(ctx)
	fmt.Fprintln(out.Stdout, a...)
}

// Errorf formats and writes to the context's stderr.
func Errorf(ctx context.Context, format string, a ...any) {
	out := outputFromContext(ctx)
	fmt.Fprintf(out.Stderr, format, a...)
}

// outputFromContext returns the Output from the context.
// Returns StdOutput() if no output is set.
func outputFromContext(ctx context.Context) *Output {
	if out := OutputFromContext(ctx); out != nil {
		return out
	}
	return StdOutput()
}
