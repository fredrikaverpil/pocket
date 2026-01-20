# Pocket architecture

## Pok shim

The `./pok` shim executes `go run .pocket/main.go`.

### Shim Variables

| Variable | Purpose |
|----------|---------|
| `SHIM_DIR` | Directory where the shim script lives (resolved at runtime) |
| `POCKET_DIR` | Path to `.pocket` directory (computed from `SHIM_DIR`) |
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
- `pathMappings` stores both `includePaths` (for visibility) and `resolvedPaths` (for execution)
- `moduleDirectories` derived from `pathMappings` (single source of truth)

No double traversal - data is collected once and shared throughout execution.

## Output and Error Handling

### Output Abstraction

Output is propagated through context rather than using `os.Stdout/Stderr` directly:

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
ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
defer stop()
```

Signals propagate through context cancellation to all running tasks and external commands.
