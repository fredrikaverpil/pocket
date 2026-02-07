# Engine internals

## Plan building

### Data structures

```go
type Plan struct {
    tree              Runnable           // Original composition tree
    taskInstances     []taskInstance     // Flat list of all tasks
    taskIndex         map[string]*taskInstance  // Lookup by effective name
    pathMappings      map[string]pathInfo       // Task → execution paths
    moduleDirectories []string           // Dirs where shims are generated
    shimConfig        *ShimConfig
}

type taskInstance struct {
    task          *Task
    name          string               // Effective name (with suffix)
    contextValues []contextValue
    flags         map[string]any       // Pre-merged flag overrides
    isManual      bool
    resolvedPaths []string
}
```

### Algorithm

`NewPlan` builds the plan in a single pass:

1. **Walk filesystem** — `walkDirectories()` from git root, skipping
   `DefaultSkipDirs` (vendor, node_modules, etc.). Result is cached.

2. **Collect tasks** — `taskCollector` traverses the composition tree. For each
   `pathFilter` encountered, it pushes scope (include/exclude patterns, flags,
   context values, name suffix) onto a stack. For each `Task`, it creates a
   `taskInstance` with the accumulated scope.

3. **Resolve paths** — For each task instance, paths are resolved against the
   cached directory list:
   - Global excludes (`WithExcludePath`) filter candidates first
   - Detection function runs against filtered candidates (if present)
   - Include patterns filter by regex match (if no detection)
   - Task-specific excludes (`WithExcludeTask`) apply last
   - Default: `["."]` (root only)

4. **Build index** — Task instances indexed by effective name for O(1) lookup.

5. **Build flag sets** — Each task's `flagSet` is built from its `FlagDef` map.
   This happens lazily but is triggered during plan building for user tasks.

### Flag merge order

Flags resolve with increasing priority:

1. **Task defaults** — `FlagDef.Default` values
2. **Plan overrides** — `WithFlag()` values accumulated during tree traversal
3. **CLI arguments** — User-provided flags on the command line

Plan overrides are stored in `taskInstance.flags` and applied at execution time
before CLI parsing.

### Name suffixes

`WithNameSuffix("3.9")` appends to the task's effective name with a `:`
separator. Suffixes accumulate left-to-right through nesting. The effective name
determines deduplication scope — tasks with different suffixes run
independently.

Example: `py-test` with suffix `"3.9"` becomes `py-test:3.9`.

---

## Path resolution

Path resolution transforms user-declared patterns into concrete directories.

### Resolution order

```
All directories (from filesystem walk)
    ↓
Global excludes (WithExcludePath) — removes from ALL tasks in scope
    ↓
Detection OR include filter
    ↓
Task-specific excludes (WithExcludeTask) — removes for ONE task
    ↓
Resolved paths (stored in taskInstance)
```

### Scope refinement

Nested `WithOptions` calls are **cumulative**. An inner scope only sees
directories that passed the outer scope's filters. This means:

- Inner detection functions search within the outer scope, not the full tree
- Outer excludes cannot be overridden by inner includes
- Each nesting level can only narrow, never widen

### Pattern matching

Include and exclude patterns are **regular expressions**, not globs. Matched
against directory paths relative to git root. Regex compilation is cached per
pattern.

---

## Task execution

### Execution flow

```
task.run(ctx)
    │
    ├── Check manual status (skip if auto-exec and task is manual)
    ├── Build resolved flags (defaults + plan + CLI)
    ├── Check deduplication (tracker.markDone)
    ├── Check task-specific path exclusions
    ├── Print header (":: taskname @ path")
    └── Execute (Do function or Body runnable)
```

### Flag panic recovery

`GetFlag[T]` panics with a `flagError` if the flag is not found or the type
doesn't match. `task.execute()` recovers this panic and converts it to a
regular error. This provides a clean API without requiring error returns at
every flag access.

### Deduplication

```go
type taskID struct {
    Name string  // Effective name (includes suffix)
    Path string  // Execution path ("." for global tasks)
}
```

The `executionTracker` is thread-safe (mutex-protected). `markDone` returns
true if the task was already executed, signaling the caller to skip.

---

## Composition execution

### Serial

Executes runnables sequentially. Stops on first error. Context passes through
unchanged.

### Parallel

Executes runnables concurrently via `errgroup.WithContext`:

1. **Single item** — runs directly, no buffering overhead
2. **Multiple items** — each goroutine receives a context with `bufferedOutput`
3. On completion, each buffer flushes to parent output under `flushMu`
4. First error cancels the context, remaining goroutines exit cooperatively

### pathFilter

Wraps an inner runnable with path/option configuration:

1. Apply context values to context
2. Apply name suffix to context
3. For each resolved path:
   - Set path in context via `ContextWithPath`
   - Execute inner runnable

---

## Exec pipeline

### Command setup

```go
func Exec(ctx context.Context, name string, args ...string) error
```

1. Build environment: start with `os.Environ()`, apply context env config
   (filter then set), prepend `.pocket/bin` to PATH, force color vars if TTY
2. Resolve binary: `lookPathInEnv` searches the modified PATH
3. Create `exec.Cmd` with working directory from `PathFromContext`
4. Set stdin to nil (prevents interactive prompts hanging CI)
5. Apply graceful shutdown handler (platform-specific)

### Output modes

**Verbose (`-v`):** stdout and stderr stream directly to context output.

**Normal:** stdout and stderr are captured in buffers. On success, output is
discarded unless warning patterns are detected. On error, the full buffered
output is printed to stderr.

### Warning detection

`DefaultNoticePatterns`: `"warn"`, `"deprecat"`, `"notice"`, `"caution"`,
`"error"`. Case-insensitive substring matching on stderr lines. Custom patterns
can be set via `WithNoticePatterns`. When a warning is detected in normal mode,
output is shown even on success.

### Graceful shutdown

- **Unix** (`exec_unix.go`): sends `SIGINT` via `cmd.Cancel`, allowing the
  process to clean up. `WaitDelay` of 5 seconds before forced kill.
- **Non-Unix** (`exec_other.go`): no-op cancel function; relies on default
  `os.Process.Kill` behavior.

### PATH management

`.pocket/bin` is always first in PATH. Additional directories can be registered
via `RegisterPATH(dir)` (thread-safe, mutex-protected). The modified PATH is
constructed per `Exec` call from the current environment plus registered dirs.

---

## Output system

### Layering

```
Context carries Output (stdout + stderr writers)
    │
    ├── Default: os.Stdout / os.Stderr
    ├── Parallel: replaced with bufferedOutput per goroutine
    └── Printf/Println/Errorf: write to context's Output
```

### Buffered output for parallel execution

```go
type bufferedOutput struct {
    stdout  *bytes.Buffer
    stderr  *bytes.Buffer
    parent  *Output       // Where to flush on completion
    flushMu *sync.Mutex   // Shared across sibling goroutines
}
```

The flush mutex is shared among all goroutines in a single `Parallel` call.
This ensures atomic output blocks — one task's output is never interleaved with
another's. The order is determined by completion time.

---

## Shim generation

### Where shims are generated

1. **Root** (`.`) — always
2. **Include paths** — each unique path from `WithIncludePath`
3. **Detected paths** — each path from detection functions

Only the declared/detected path gets a shim, not every resolved subdirectory.
Example: `WithIncludePath("internal")` generates `internal/pok`, even if
`internal/` contains multiple subdirectories.

### Shim variants

| Variant     | File        | Platform          |
|-------------|-------------|-------------------|
| POSIX       | `pok`       | Linux, macOS      |
| Batch       | `pok.cmd`   | Windows (cmd.exe) |
| PowerShell  | `pok.ps1`   | Windows (pwsh)    |

Configured via `ShimConfig` / `AllShimsConfig()`.

### Shim variables

Each shim sets:

- `SHIM_DIR` — directory where the shim lives (resolved at runtime)
- `POCKET_DIR` — relative path from shim to `.pocket` directory
- `TASK_SCOPE` — the directory path for scoped task visibility

### Task visibility scoping

When `TASK_SCOPE` is set (by a subdirectory shim), only tasks whose resolved
paths include that scope appear in help and are eligible for execution. Root
shims set `TASK_SCOPE="."`, making all tasks visible.

---

## CLI dispatch

### Entry point

`RunMain(cfg)` → `run(cfg)`:

1. Parse global flags (`-v`, `-g`, `-h`, `--version`)
2. Build `Plan` from config + filesystem
3. Dispatch:
   - No args → `executeAll` (run full Auto tree + shims)
   - Builtin name → run builtin
   - Task name → `executeTask` (run single task with its paths)
   - Unknown → error with suggestions

### Builtin tasks

| Name          | Purpose                              | Hidden |
|---------------|--------------------------------------|--------|
| `plan`        | Show execution plan (text or JSON)   | No     |
| `shims`       | Regenerate shims                     | No     |
| `self-update` | Update Pocket, regenerate files      | No     |
| `purge`       | Remove tools, bin, venvs directories | No     |
| `git-diff`    | Check for uncommitted changes        | Yes    |

Builtins are checked before user tasks during name lookup.

---

## Concurrency safety

| Resource           | Protection    | Location       |
|--------------------|---------------|----------------|
| `executionTracker` | `sync.Mutex`  | `tracker.go`   |
| `extraPATHDirs`    | `sync.Mutex`  | `exec.go`      |
| Regex cache        | `sync.RWMutex`| `paths.go`     |
| Parallel output    | `sync.Mutex`  | `output.go`    |
| `gitRoot` cache    | `sync.Once`   | `paths.go`     |

No shared mutable state in goroutines beyond these synchronized structures.
Each parallel goroutine operates on its own buffered output and receives an
immutable context snapshot.
