package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// Output holds the stdout and stderr writers used by [Printf], [Println],
// [Errorf], and [Exec]. In parallel execution, each goroutine receives
// a buffered Output to prevent interleaved output.
type Output = engine.Output

// StdOutput returns an Output that writes to os.Stdout and os.Stderr.
func StdOutput() *Output {
	return engine.StdOutput()
}

// Printf formats and writes to the context's stdout.
// Use this instead of [fmt.Printf] to ensure correct output in parallel tasks.
func Printf(ctx context.Context, format string, a ...any) {
	engine.Printf(ctx, format, a...)
}

// Println writes to the context's stdout, appending a newline.
// Use this instead of [fmt.Println] to ensure correct output in parallel tasks.
func Println(ctx context.Context, a ...any) {
	engine.Println(ctx, a...)
}

// Errorf formats and writes to the context's stderr.
// Use this instead of [fmt.Fprintf](os.Stderr, ...) to ensure correct output in parallel tasks.
func Errorf(ctx context.Context, format string, a ...any) {
	engine.Errorf(ctx, format, a...)
}
