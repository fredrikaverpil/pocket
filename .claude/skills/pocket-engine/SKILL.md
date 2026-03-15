---
name: pocket-engine
description: >-
  Pocket core engine architecture and internals. Covers plan building, task
  execution, composition, context propagation, path resolution, deduplication,
  output buffering, exec pipeline, and shim generation. Use when modifying or
  understanding code in the pk/ package.
---

# Pocket engine architecture

Pocket is a composable task runner that separates **planning** from
**execution**. The plan phase does all computation (tree walking, path
resolution, flag merging) once. The execution phase is simple iteration.

## Design principles

**Single-pass planning.** The filesystem is walked once. The composition tree is
traversed once. Path patterns are resolved once. All results are cached in the
`Plan` struct and shared throughout execution.

**Composition preservation.** The original `Serial`/`Parallel` tree is kept
intact in the plan. Execution follows the same tree structure. This enables
introspection (`./pok plan -json`) and CI integration without separate
codepaths.

**Context layering.** Each nesting level can add context (path, flags, env,
name suffix). Values accumulate as composition nests. CLI flags override plan
flags, which override task defaults.

**Cooperative concurrency.** Parallel tasks use `errgroup.WithContext` for
fail-fast. Output is buffered per-goroutine and flushed atomically to prevent
interleaving. Signals propagate via context cancellation.

## File ownership

| File             | Responsibility                                      |
|------------------|-----------------------------------------------------|
| `cli.go`         | Entry point, flag parsing, dispatch                 |
| `config.go`      | Config struct, defaults, shim config                |
| `plan.go`        | Plan building, path resolution, task collection     |
| `task.go`        | Task definition, flag system, execution, dedup      |
| `composition.go` | Runnable interface, Serial, Parallel                |
| `options.go`     | WithOptions, pathFilter, detection, flag overrides  |
| `context.go`     | Context keys, path/verbose/env propagation          |
| `exec.go`        | Command execution, PATH, output buffering, TTY      |
| `exec_unix.go`   | Unix graceful shutdown (SIGINT)                     |
| `exec_other.go`  | Non-Unix shutdown fallback                          |
| `output.go`      | Output abstraction, buffered parallel output        |
| `tracker.go`     | Deduplication tracking, warning tracking            |
| `builtins.go`    | Built-in tasks (plan, shims, self-update, purge)    |
| `paths.go`       | Git root, directory walking, path helpers           |
| `platform.go`    | OS/arch detection, naming helpers                   |

## Subsystem overview

### Plan building

The plan is the bridge between user configuration and execution. It transforms
a composition tree + filesystem into a flat, pre-computed execution plan.

**Inputs:** `Config.Auto`, `Config.Manual`, filesystem walk, shim config.
**Outputs:** `Plan` struct with task instances, path mappings, task index.

See [INTERNALS.md](INTERNALS.md) for the plan building algorithm and path
resolution details.

### Composition model

The `Runnable` interface has a single unexported method `run(ctx) error`. Four
types implement it:

- **Task** — leaf node, runs a function
- **serial** — sequential execution, stops on first error
- **parallel** — concurrent execution with buffered output
- **pathFilter** — wraps a runnable, iterates over resolved paths

`Serial()` and `Parallel()` are the public constructors. `WithOptions()` wraps
a runnable in a `pathFilter` with the given options.

### Context propagation

Context carries execution state through the composition tree:

- **Path** (`PathFromContext`) — current directory, set by pathFilter
- **Verbose** (`Verbose`) — from `-v` CLI flag
- **Env** — per-task environment overrides (set/filter)
- **Name suffix** — accumulated from `WithNameSuffix`
- **Auto-exec** — whether manual tasks should be skipped
- **Output** — writer for stdout/stderr (swapped for buffered in parallel)

### Exec pipeline

`pk.Exec` runs external commands with:

1. `.pocket/bin` prepended to PATH (plus any registered dirs)
2. Working directory from `PathFromContext`
3. Color env vars forced when TTY detected
4. Stdin set to nil (prevents CI hangs)
5. Graceful shutdown: SIGINT on Unix, then SIGKILL after 5s
6. Output: streamed in verbose mode, buffered in normal mode (shown only on
   error or when warning patterns detected)

### Deduplication

Tasks are deduplicated by `(effectiveName, path)` tuple. A task at the same
path only runs once per invocation. Global tasks (`Global: true`) use a fixed
path of `"."`, so they run at most once total. `WithForceRun()` bypasses this.

### Parallel output buffering

When multiple tasks run in parallel, each goroutine gets a `bufferedOutput`
that captures all writes. On completion, the buffer is flushed to the parent
output under a mutex. This prevents interleaved output while preserving
first-to-complete ordering. Single-item parallel runs skip buffering entirely.

### Shim generation

Shims are generated at the repo root and at each unique include/detect path.
Three variants: POSIX (`pok`), Windows batch (`pok.cmd`), PowerShell
(`pok.ps1`). Each shim computes a relative path back to `.pocket` and sets
`TASK_SCOPE` for path-scoped task visibility.

See [INTERNALS.md](INTERNALS.md) for details on each subsystem.
