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
