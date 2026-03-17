package run

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// Output holds the stdout and stderr writers used by [Printf], [Println],
// [Errorf], and [Exec]. In parallel execution, each goroutine receives
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

// IsTerminal returns true if the given file is a terminal.
func IsTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// Printf formats and writes to the context's stdout.
func Printf(ctx context.Context, format string, a ...any) {
	out := outputOrStd(ctx)
	fmt.Fprintf(out.Stdout, format, a...)
}

// Println writes to the context's stdout, appending a newline.
func Println(ctx context.Context, a ...any) {
	out := outputOrStd(ctx)
	fmt.Fprintln(out.Stdout, a...)
}

// Errorf formats and writes to the context's stderr.
func Errorf(ctx context.Context, format string, a ...any) {
	out := outputOrStd(ctx)
	fmt.Fprintf(out.Stderr, format, a...)
}
