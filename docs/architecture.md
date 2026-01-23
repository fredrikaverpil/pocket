# Pocket architecture

## Pok shim

The `./pok` shim executes `go run -C .pocket .` (runs `.pocket/main.go` from the
`.pocket` directory).

### Shim Variables

| Variable     | Purpose                                                           |
| ------------ | ----------------------------------------------------------------- |
| `SHIM_DIR`   | Directory where the shim script lives (resolved at runtime)       |
| `POCKET_DIR` | Path to `.pocket` directory (computed from `SHIM_DIR`)            |
| `TASK_SCOPE` | Task visibility and execution scope (e.g., ".", "pk", "internal") |

### Shim Generation

Shims are generated at:

1. **Root** (`.`) - always, shows all tasks
2. **Include paths** - each `WithIncludePath("dir")` gets a shim

Shims are NOT generated at every resolved subdirectory. For example, if
`WithIncludePath("internal")` resolves to `internal/`, `internal/shim/`, and
`internal/scaffold/`, only `internal/pok` is generated (not shims in every
subdirectory).

### Path-Scoped Task Visibility

When running from a subdirectory shim (e.g., `./pk/pok`):

- `TASK_SCOPE` is set to the directory (e.g., "pk")
- Only tasks configured for that path are visible in help
- Task execution is scoped to that specific path

When running from root (`./pok`):

- `TASK_SCOPE` is "."
- All tasks are visible
- Tasks execute in all their configured paths

## Plan

The plan is built once by walking the composition tree and filesystem:

- Tree walked once → extracts tasks and path mappings
- Filesystem walked once → cached directory list for path resolution
- `pathMappings` stores both `includePaths` (for visibility) and `resolvedPaths`
  (for execution)
- `moduleDirectories` derived from `pathMappings` (single source of truth)

No double traversal - data is collected once and shared throughout execution.

## Auto-Detection

Auto-detection dynamically discovers directories based on marker files or
patterns, eliminating the need to manually specify paths with `WithIncludePath`.

### Detection Functions

```go
// DetectFunc receives pre-walked directories and git root
type DetectFunc func(dirs []string, gitRoot string) []string

// Built-in detection function
pk.DetectByFile("go.mod", "go.sum")  // Directories with go.mod or go.sum
```

### How Detection Works

1. Filesystem is walked once during plan building (cached in `allDirs`)
2. Detection function filters current scope candidates (using `os.Stat()`
   checks)
3. Inner detection functions resolve against their parent scope, allowing for
   cumulative refining.

### Scoping and Refining

Task composition in Pocket is **refining**. Nested `pk.WithOptions` calls
accumulate their constraints:

- **Cumulative Intersection**: An inner `WithDetect` or `WithIncludePath` only
  sees directories that passed the outer scope's filters.
- **Inherited Exclusions**: If an outer scope excludes a path, no task inside
  that scope can execute there, even if an inner detection function finds it.

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithDetect(golang.Detect()), // Run in go.mod directories
    pk.WithExcludePath("vendor"),   // Global exclusion for this scope
)
```

### Task-Specific Scoping

You can apply constraints to specific tasks within a bundle without refactoring
the tree. All path patterns are interpreted as **regular expressions**.

- **`WithExcludePath(patterns...)`**: Directories matching any of the patterns
  will be excluded for ALL tasks in the current scope.
- **`WithExcludeTask(task, patterns...)`**: If tasks are provided, the exclusion
  only applies to them for the specified patterns.
- **`WithSkipTask(tasks...)`**: Completely removes specific tasks from the
  current scope.
- **`WithFlag(task, name, value)`**: Sets a default flag value for a specific
  task.

Tasks can be specified either by their string name or by the task object itself
(e.g., `golang.Lint`). Using the task object is recommended for type safety and
IDE support.

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithExcludePath("vendor"),            // Global: exclude vendor
    pk.WithExcludeTask(golang.Test, "foo/"), // Targeted: only golang.Test skips foo/
    pk.WithFlag(golang.Lint, "fix", false),  // Disable auto-fix for this scope
)
```

### golang.Tasks() Example

The `tasks/golang` package provides a default task bundle:

```go
func Tasks(tasks ...pk.Runnable) pk.Runnable {
    if len(tasks) == 0 {
        return pk.Serial(
            Fix,
            Format,
            Lint,
            pk.Parallel(Test, Vulncheck),
        )
    }
    return pk.Serial(tasks...)
}
```

Note that `Tasks()` does not include auto-detection. Wrap it with `WithDetect`
to run in all Go module directories:

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithDetect(golang.Detect()),
)
```

## Manual Tasks

Manual tasks only run when explicitly invoked (e.g., `./pok hello`), not on bare
`./pok` execution. This is useful for:

- Setup/initialization tasks
- Deployment tasks requiring confirmation
- Tasks with mandatory flags

### Config.Manual

```go
var Config = &pk.Config{
    Auto: pk.Serial(golang.Tasks()),  // Runs on bare ./pok

    Manual: []pk.Runnable{
        Hello.Manual(),  // Only runs via ./pok hello
        Deploy,          // Only runs via ./pok deploy
    },
}
```

### Task.Manual()

The `Manual()` method returns a copy of the task marked as manual:

```go
var Hello = pk.NewTask("hello", "greet user", flags, pk.Do(fn))
// Hello.Manual() → manual copy, won't run on bare ./pok
```

### Help Output

Manual tasks appear in a separate section:

```
Tasks:
  go-lint       run golangci-lint

Manual tasks (explicit invocation only):
  hello         print a greeting message
```

## Output and Error Handling

### Output Abstraction

Output is propagated through context rather than using `os.Stdout/Stderr`
directly:

```go
type Output struct {
    Stdout io.Writer
    Stderr io.Writer
}

ctx = WithOutput(ctx, StdOutput())
out := OutputFromContext(ctx)  // Returns StdOutput() if unset
```

### Buffered Parallel Output

When tasks run in parallel, each gets a `bufferedOutput` that captures output:

- Single task → runs directly without buffering
- Multiple tasks → each gets buffered output, flushes atomically on completion
- First-to-complete flushes first (no output interleaving)

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Task A    │     │   Task B    │     │   Task C    │
│  (buffer)   │     │  (buffer)   │     │  (buffer)   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │ flushMu.Lock()    │ flushMu.Lock()    │ flushMu.Lock()
       ▼                   ▼                   ▼
┌────────────────────────────────────────────────────┐
│                   Parent Output                     │
│               (os.Stdout/Stderr)                    │
└────────────────────────────────────────────────────┘
```

### Cooperative Cancellation

Uses `errgroup.WithContext` for fail-fast behavior:

- When one task fails, context is cancelled
- Other goroutines check `ctx.Done()` and exit early
- External commands receive SIGINT, then SIGKILL after `WaitDelay` (5s)

### Signal Handling

CLI sets up signal handling at entry:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

Signals propagate through context cancellation to all running tasks and external
commands.

## Platform Specifics

Pocket uses **Go build tags** to handle platform-specific behavior while
maintaining a clean, cross-platform core.

### Graceful Shutdown

Graceful shutdown requires sending different signals depending on the operating
system:

- **Unix (`pk/exec_unix.go`)**: Uses `syscall.SIGINT` to allow processes to
  clean up before termination.
- **Other (`pk/exec_other.go`)**: Defaults to immediate termination as standard
  interrupt signals are not available or handled differently.

### Terminal Detection

Terminal detection is consolidated in `pk/exec.go` and works across all
platforms (including Windows) using `golang.org/x/term`. This allows Pocket to
automatically enable colored output and other TTY-dependent features when
running in a real terminal.
