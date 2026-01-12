package pocket

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

// Output holds stdout and stderr writers for task output.
// This is passed through the Runnable chain to direct output appropriately.
type Output struct {
	Stdout io.Writer
	Stderr io.Writer
}

// StdOutput returns an Output that writes to os.Stdout and os.Stderr.
func StdOutput() *Output {
	return &Output{Stdout: os.Stdout, Stderr: os.Stderr}
}

// Printf formats and prints to stdout.
func (o *Output) Printf(format string, a ...any) (int, error) {
	return fmt.Fprintf(o.Stdout, format, a...)
}

// Println prints to stdout with a newline.
func (o *Output) Println(a ...any) (int, error) {
	return fmt.Fprintln(o.Stdout, a...)
}

// bufferedOutput captures output to buffers for later printing.
// Parent writers are where the buffer flushes to (supports nested parallel).
type bufferedOutput struct {
	parent *Output
	mu     sync.Mutex
	stdout bytes.Buffer
	stderr bytes.Buffer
}

// newBufferedOutput creates a bufferedOutput that flushes to the given parent.
func newBufferedOutput(parent *Output) *bufferedOutput {
	return &bufferedOutput{parent: parent}
}

// Stdout returns a writer for stdout that is safe for concurrent use.
func (b *bufferedOutput) Stdout() io.Writer {
	return &lockedWriter{mu: &b.mu, w: &b.stdout}
}

// Stderr returns a writer for stderr that is safe for concurrent use.
func (b *bufferedOutput) Stderr() io.Writer {
	return &lockedWriter{mu: &b.mu, w: &b.stderr}
}

// Flush writes all buffered output to the parent output.
func (b *bufferedOutput) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, _ = io.Copy(b.parent.Stdout, &b.stdout)
	_, _ = io.Copy(b.parent.Stderr, &b.stderr)
}

// Output returns an Output that writes to the buffers.
func (b *bufferedOutput) Output() *Output {
	return &Output{
		Stdout: b.Stdout(),
		Stderr: b.Stderr(),
	}
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
