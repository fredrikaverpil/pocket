package pocket

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// outputKey is the context key for output writers.
const outputKey contextKey = 100

// output holds stdout and stderr writers for a task.
type output struct {
	stdout io.Writer
	stderr io.Writer
}

// bufferedOutput captures output to buffers for later printing.
type bufferedOutput struct {
	mu     sync.Mutex
	stdout bytes.Buffer
	stderr bytes.Buffer
}

// Stdout returns a writer for stdout that is safe for concurrent use.
func (b *bufferedOutput) Stdout() io.Writer {
	return &lockedWriter{mu: &b.mu, w: &b.stdout}
}

// Stderr returns a writer for stderr that is safe for concurrent use.
func (b *bufferedOutput) Stderr() io.Writer {
	return &lockedWriter{mu: &b.mu, w: &b.stderr}
}

// Flush writes all buffered output to os.Stdout and os.Stderr.
func (b *bufferedOutput) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, _ = io.Copy(os.Stdout, &b.stdout)
	_, _ = io.Copy(os.Stderr, &b.stderr)
}

// lockedWriter wraps a writer with a mutex for safe concurrent writes.
type lockedWriter struct {
	mu *sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// withOutput returns a context with the given output writers.
func withOutput(ctx context.Context, stdout, stderr io.Writer) context.Context {
	return context.WithValue(ctx, outputKey, &output{stdout: stdout, stderr: stderr})
}

// Stdout returns the stdout writer from context, or os.Stdout if not set.
func Stdout(ctx context.Context) io.Writer {
	if o, ok := ctx.Value(outputKey).(*output); ok {
		return o.stdout
	}
	return os.Stdout
}

// Stderr returns the stderr writer from context, or os.Stderr if not set.
func Stderr(ctx context.Context) io.Writer {
	if o, ok := ctx.Value(outputKey).(*output); ok {
		return o.stderr
	}
	return os.Stderr
}

// Printf formats and prints to stdout from context.
// Use this in task actions instead of fmt.Printf for proper output handling.
func Printf(ctx context.Context, format string, a ...any) (int, error) {
	return fmt.Fprintf(Stdout(ctx), format, a...)
}

// Println prints to stdout from context with a newline.
// Use this in task actions instead of fmt.Println for proper output handling.
func Println(ctx context.Context, a ...any) (int, error) {
	return fmt.Fprintln(Stdout(ctx), a...)
}
