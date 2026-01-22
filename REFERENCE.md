# Pocket API Reference

This document provides a technical reference for the public API in the
`github.com/fredrikaverpil/pocket/pk` package.

## Configuration

### `type Config`

The main entry point for configuring Pocket.

```go
type Config struct {
    Auto   Runnable    // Tasks executed on bare ./pok
    Manual []Runnable  // Tasks only run when explicitly invoked
}
```

## Composition

### `Serial(runnables ...Runnable) Runnable`

Executes runnables sequentially. Stops on the first error.

### `Parallel(runnables ...Runnable) Runnable`

Executes runnables concurrently. Waits for all to complete. Buffers output to
prevent interleaving.

### `WithOptions(runnable Runnable, opts ...Option) Runnable`

Applies path filtering and execution options to a runnable.

## Path Options

Used with `WithOptions()`:

| Function                   | Description                                            |
| :------------------------- | :----------------------------------------------------- |
| `WithIncludePath(pattern)` | Run only in directories matching the glob pattern.     |
| `WithExcludePath(pattern)` | Skip directories matching the glob pattern.            |
| `WithDetect(DetectFunc)`   | Dynamically discover paths using a detection function. |
| `WithForceRun()`           | Bypass task deduplication for the wrapped runnable.    |

## Detection Functions

Used with `WithDetect()`:

| Function                     | Description                                             |
| :--------------------------- | :------------------------------------------------------ |
| `DetectByFile(filenames...)` | Find directories containing any of the specified files. |

## Task Construction

### `NewTask(name, usage string, flags *flag.FlagSet, body Runnable) *Task`

Creates a named task.

### `(*Task) Hidden() *Task`

Returns a copy of the task that is excluded from CLI help output.

### `(*Task) Manual() *Task`

Returns a copy of the task that only runs when explicitly invoked by name.

## Execution Helpers

### `Do(func(context.Context) error) Runnable`

Wraps a Go function as a `Runnable`.

### `Exec(ctx context.Context, name string, args ...string) error`

Executes an external command. Captures output and respects context cancellation.
Adds `.pocket/bin` to `PATH`.

### `Printf(ctx context.Context, format string, a ...any)`

Formats and prints to the output in the context (correctly handles standard and
buffered output).

### `Println(ctx context.Context, a ...any)`

Prints to the output in the context.

### `Errorf(ctx context.Context, format string, a ...any)`

Formats and prints to the error output in the context.

## Tool Installation

### `InstallGo(pkg, version string) Runnable`

Installs a Go package using `go install`. The binary is placed in
`.pocket/tools` and symlinked to `.pocket/bin`.

## Context Accessors

| Function                 | Description                                                       |
| :----------------------- | :---------------------------------------------------------------- |
| `PathFromContext(ctx)`   | Returns the current execution path relative to the git root.      |
| `Verbose(ctx)`           | Returns `true` if the `-v` flag was provided.                     |
| `OutputFromContext(ctx)` | Returns the `Output` struct (Stdout/Stderr) for the current task. |

## Path Helpers

| Function                         | Description                                                   |
| :------------------------------- | :------------------------------------------------------------ |
| `FromGitRoot(elems ...string)`   | Returns an absolute path relative to the git repository root. |
| `FromPocketDir(elems ...string)` | Returns an absolute path relative to `.pocket/`.              |
| `FromBinDir(elems ...string)`    | Returns an absolute path relative to `.pocket/bin/`.          |
| `FromToolsDir(elems ...string)`  | Returns an absolute path relative to `.pocket/tools/`.        |
