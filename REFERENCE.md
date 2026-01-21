# Pocket API Reference

Public API for building task configurations and tools.

## Composition

Functions for building the task tree in `.pocket/config.go`:

| Function                          | Description                                         |
| --------------------------------- | --------------------------------------------------- |
| `Serial(runnables ...Runnable)`   | Execute runnables sequentially, stop on first error |
| `Parallel(runnables ...Runnable)` | Execute runnables concurrently, wait for all        |
| `WithOptions(runnable, opts...)`  | Apply path filtering options to a runnable          |

### Path Options

Used with `WithOptions()`:

| Function                   | Description                                   |
| -------------------------- | --------------------------------------------- |
| `WithIncludePath(pattern)` | Only run in directories matching pattern      |
| `WithExcludePath(pattern)` | Skip directories matching pattern             |
| `WithDetect(fn)`           | Dynamically discover paths via detection func |
| `WithForceRun()`           | Bypass deduplication, always run              |

### Detection Functions

Used with `WithDetect()`:

| Function                     | Description                                   |
| ---------------------------- | --------------------------------------------- |
| `DetectByFile(filenames...)` | Find directories containing any of the files |

Custom detection functions can be created using the `DetectFunc` type:
`func(dirs []string, gitRoot string) []string`

## Task Construction

| Function                            | Description                                            |
| ----------------------------------- | ------------------------------------------------------ |
| `NewTask(name, usage, flags, body)` | Create task with Runnable body and optional flags      |
| `(*Task).Hidden()`                  | Return hidden copy of task (excluded from CLI help)    |
| `(*Task).Manual()`                  | Return manual copy (only runs when explicitly invoked) |
| `(*Task).IsManual()`                | Check if task is manual-only                           |

## Execution

Used inside task functions:

| Function                   | Description                                    |
| -------------------------- | ---------------------------------------------- |
| `Do(fn)`                   | Wrap `func(context.Context) error` as Runnable |
| `Exec(ctx, name, args...)` | Run command with `.pocket/bin` in PATH         |

## Tool Installation

| Function                  | Description                                  |
| ------------------------- | -------------------------------------------- |
| `InstallGo(pkg, version)` | Install Go package, symlink to `.pocket/bin` |

## Context Helpers

| Function               | Description                                       |
| ---------------------- | ------------------------------------------------- |
| `PathFromContext(ctx)` | Get current execution path (relative to git root) |
| `Verbose(ctx)`         | Check if verbose mode is enabled                  |

## Path Helpers

| Function                 | Description                            |
| ------------------------ | -------------------------------------- |
| `FromGitRoot(paths...)`  | Absolute path from git repository root |
| `FromPocketDir(elem...)` | Absolute path within `.pocket/`        |
| `FromToolsDir(elem...)`  | Absolute path within `.pocket/tools/`  |
| `FromBinDir(elem...)`    | Absolute path within `.pocket/bin/`    |

---

## Config Structure

The main configuration struct:

```go
type Config struct {
    Root   Runnable    // Tasks executed on bare ./pok
    Manual []Runnable  // Tasks only run when explicitly invoked
}
```

## Internal API

These functions are exported but intended for Pocket's internal use only (called
by generated `.pocket/main.go` or the executor):

| Function                 | Description                                    |
| ------------------------ | ---------------------------------------------- |
| `RunMain(cfg *Config)`   | CLI entry point, called from `.pocket/main.go` |
| `WithPath(ctx, path)`    | Set execution path in context                  |
| `WithVerbose(ctx, bool)` | Set verbose mode in context                    |
