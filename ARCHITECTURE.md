# Architecture

This document describes pocket's internal architecture. For user-facing APIs and
usage, see [README.md](README.md).

## Overview

Pocket is built around a tree of composable **Runnables**. The user defines a
configuration with an execution tree, and pocket walks this tree to either
execute tasks or visualize the plan.

```
┌─────────────────────────────────────────────────────────────┐
│                        pocket.Config                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ AutoRun: Serial(                                    │   │
│  │   Paths(golang.Workflow()).DetectBy(golang.Detect())│   │
│  │   Paths(python.Workflow()).DetectBy(python.Detect())│   │
│  │ )                                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────┐                                       │
│  │ ManualRun: [...]│                                       │
│  └─────────────────┘                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Runnable Tree Walk                      │
│                                                             │
│   Serial ──► Paths ──► Serial ──► FuncDef ──► Serial       │
│                                       │                     │
│                                       ▼                     │
│                                   Install ──► fn            │
└─────────────────────────────────────────────────────────────┘
```

## Core Abstraction: Runnable

The `Runnable` interface is the foundation of pocket. It has two unexported
methods to prevent external implementations:

```go
type Runnable interface {
    run(ctx context.Context) error  // execute this runnable
    funcs() []*FuncDef              // collect all named functions
}
```

Five types implement Runnable:

```
                           Runnable
                              │
       ┌──────────┬───────────┼───────────┬──────────┐
       │          │           │           │          │
   FuncDef     serial     parallel   PathFilter  funcRunnable
       │                                              │
       └─ named, CLI-visible                          └─ internal wrapper
```

| Type           | Purpose                                       |
| -------------- | --------------------------------------------- |
| `FuncDef`      | Named function with implementation            |
| `serial`       | Sequential execution of children              |
| `parallel`     | Concurrent execution of children              |
| `PathFilter`   | Wraps a Runnable with directory-based filters |
| `funcRunnable` | Internal wrapper for plain functions          |

`funcRunnable` wraps plain `func(context.Context) error` functions passed to
Serial/Parallel, enabling them to participate in the tree without being named.

### FuncDef

A `FuncDef` represents a named function:

```go
type FuncDef struct {
    name   string                          // CLI command name
    usage  string                          // help text
    body   Runnable                        // implementation (plain function or composition)
    opts   any                             // default CLI options
    hidden bool                            // hide from help
}
```

A FuncDef wraps a `Runnable` body. Plain functions are automatically wrapped in an internal `funcRunnable` during creation via `Func()`. Calling the function walks its entire subtree.

### Composition

`Serial()` and `Parallel()` create group Runnables:

```
Serial(Install, lint)          Parallel(Lint, Test)
        │                              │
        ▼                              ▼
   ┌─────────┐                    ┌─────────┐
   │ Install │───►│ lint │        │  Lint   │
   └─────────┘    └──────┘        └────┬────┘
                                       │
                                  ┌────┴────┐
                                  │  Test   │
                                  └─────────┘
                               (concurrent)
```

Parallel execution buffers output per-goroutine and flushes sequentially after
all complete, preventing interleaved output.

### Error Handling

Errors propagate differently based on composition type:

| Type       | On Error                                              |
| ---------- | ----------------------------------------------------- |
| `Serial`   | Stop immediately, return the error                    |
| `Parallel` | Cancel context, wait for all goroutines, return first |

Serial fails fast - subsequent items don't run. Parallel uses `errgroup` which
cancels the shared context on first error, allowing goroutines to observe
cancellation and clean up.

## Execution Flow

```
./pok [args]
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│ RunConfig(cfg)                                              │
│   1. Collect all FuncDefs from AutoRun tree                 │
│   2. Build path mappings (func name → PathFilter)           │
│   3. Add ManualRun functions                                │
│   4. Add built-in tasks (plan, clean, generate, update)     │
│   5. Validate no duplicate names                            │
└──────────────────────────┬──────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ cliMain()                                                   │
│   1. Parse flags (-h, -v)                                   │
│   2. Detect cwd relative to git root                        │
│   3. Filter visible functions based on cwd + PathFilters    │
│   4. Dispatch to help or execution                          │
└──────────────────────────┬──────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ runWithContext()                                            │
│   1. Create execContext with mode, output, cwd              │
│   2. Initialize deduplication state (shared across tree)    │
│   3. Attach to context.Context                              │
│   4. Call root.run(ctx)                                     │
└──────────────────────────┬──────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Runnable.run(ctx)                                           │
│   FuncDef:   print header → execute fn or body.run()        │
│   serial:    for each child → run(ctx) sequentially         │
│   parallel:  for each child → run(ctx) concurrently         │
│   PathFilter: for each path → set path context → inner.run()│
└─────────────────────────────────────────────────────────────┘
```

### Function Visibility

Functions are visible based on the current working directory:

| Condition              | Visibility                                        |
| ---------------------- | ------------------------------------------------- |
| Has PathFilter wrapper | Visible if `paths.RunsIn(cwd)` returns true       |
| No PathFilter wrapper  | Only visible at git root (cwd == ".")             |
| Hidden flag set        | Never shown in help, but can be called explicitly |
| Built-in tasks         | Always visible                                    |

## Dual-Mode Execution

Pocket operates in two modes, controlled by `execMode` in `execContext`:

```
                       Runnable.run(ctx)
                              │
                  ┌───────────┴───────────┐
                  │                       │
             modeExecute             modeCollect
                  │                       │
          ┌───────┴───────┐       ┌───────┴───────┐
          │               │       │               │
    Run commands    Print    Walk tree only   Build plan
    Mutate files    output   No side effects  Discard output
```

### Execute Mode (default)

Normal execution. Commands run, files are modified, output is printed.

### Collect Mode (plan)

Used by `./pok plan`. Walks the tree without executing, building an
`ExecutionPlan` for visualization:

```go
type ExecutionPlan struct {
    steps []PlanStep  // hierarchical tree of steps
    stack []*PlanStep // nesting stack during collection
}

type PlanStep struct {
    Type     string      // "func", "serial", "parallel"
    Name     string      // function name
    Usage    string      // description
    Hidden   bool        // installation dependency
    Deduped  bool        // would be skipped (already ran)
    Children []PlanStep  // nested steps
}
```

## Path Filtering

`PathFilter` wraps a Runnable with directory-based filtering:

```go
type PathFilter struct {
    inner     Runnable          // wrapped runnable
    include   []*regexp.Regexp  // patterns to include
    exclude   []*regexp.Regexp  // patterns to exclude
    detect    func() []string   // detection function
    skipRules []skipRule        // per-function skip rules
}
```

### Resolution Flow

```
                  PathFilter
                      │
         ┌────────────┴────────────┐
         │                         │
     Resolve()                 ResolveFor(cwd)
         │                         │
         ▼                         ▼
   All matching dirs        Dirs matching cwd
         │                         │
         └──────────┬──────────────┘
                    ▼
              For each dir:
                    │
                    ▼
         ┌──────────────────────┐
         │ ctx = withPath(dir)  │
         │ inner.run(ctx)       │
         └──────────────────────┘
```

### Builder Pattern

PathFilter uses immutable builders that return new copies:

```go
Paths(golang.Workflow()).
    DetectBy(golang.Detect()).  // auto-detect directories
    In("services/.*").          // include pattern
    Except("vendor").           // exclude pattern
    Skip(golang.Test, "docs")   // skip function in path
```

## Deduplication

Functions that appear multiple times in the tree (e.g., shared Install
dependencies) run only once per execution.

### Strategy

Uses pointer identity as the unique key:

```go
key := reflect.ValueOf(runnable).Pointer()
```

### Shared State

`dedupState` is a thread-safe map shared across the entire execution:

```go
type dedupState struct {
    mu    sync.Mutex
    state map[uintptr]bool
}

func (d *dedupState) shouldRun(key uintptr) bool {
    d.mu.Lock()
    defer d.mu.Unlock()
    if d.state[key] {
        return false  // already ran
    }
    d.state[key] = true
    return true
}
```

### Example

```
Serial(
    Format,              Format depends on: Install
    Lint,                Lint depends on:   Install
)

Execution:
  1. Format.run() → Install.run() → format()
  2. Lint.run()   → Install skipped (deduped) → lint()
```

## Context Threading

Execution state flows through `context.Context` via `execContext`:

```go
type execContext struct {
    mode    execMode            // execute or collect
    plan    *ExecutionPlan      // plan being built (collect mode)
    out     *Output             // stdout/stderr writers
    path    string              // current path (set by PathFilter)
    cwd     string              // where CLI was invoked
    verbose bool                // verbose mode
    dedup   *dedupState         // shared deduplication state
}
```

Options are stored directly in the `context.Context` keyed by their type, ensuring thread-safety during parallel execution.

### Helpers

Functions access context via helpers:

```go
pocket.Path(ctx)           // current path from PathFilter
pocket.Options[T](ctx)     // typed options for current function
pocket.Verbose(ctx)        // verbose flag
pocket.Printf(ctx, ...)    // output to stdout
```

## Tool Architecture

Tools are external binaries that pocket downloads and manages. The architecture
enables tasks to use tools without knowing installation details.

### Tool Structure

Each tool package exports:

```go
package mytool

const Name = "mytool"           // binary name for pocket.Exec
const Version = "1.0.0"         // pinned version

var Install = pocket.Func(      // hidden installer
    "install:mytool", "install mytool", install,
).Hidden()
```

### Installation Layout

```
.pocket/
├── tools/
│   └── mytool/
│       └── 1.0.0/              # version-specific directory
│           └── mytool          # downloaded binary
├── bin/
│   └── mytool -> ../tools/mytool/1.0.0/mytool  # symlink
```

### Dependency Composition

Tasks depend on tools via Serial composition:

```go
var Lint = pocket.Func("lint", "run linter", pocket.Serial(
    mytool.Install,  // ensure installed first
    lint,            // then run the task
))

func lint(ctx context.Context) error {
    return pocket.Exec(ctx, mytool.Name, "check", ".")
}
```

This pattern ensures:

1. Install runs before the task (Serial ordering)
2. Install runs only once per execution (deduplication)
3. The task can use the tool by name (PATH prepending)

## Command Execution

`pocket.Command()` creates commands with the tool binary path automatically
resolved:

```go
func Command(ctx context.Context, name string, args ...string) *exec.Cmd
```

### PATH Prepending

Commands get `.pocket/bin/` prepended to PATH:

```
Original PATH:  /usr/bin:/bin
Modified PATH:  .pocket/bin:/usr/bin:/bin
```

This enables calling tools by name without knowing their full path.

### Graceful Shutdown

Commands are configured for graceful termination:

```
Context cancelled
       │
       ▼
   SIGINT sent to process
       │
       ▼ (WaitDelay = 5s)
   SIGKILL if still running
```

### Execution Helpers

| Helper                         | Behavior                      |
| ------------------------------ | ----------------------------- |
| `Exec(ctx, name, args...)`     | Run in `Path(ctx)` directory  |
| `ExecIn(ctx, dir, name, args)` | Run in specific directory     |
| `Command(ctx, name, args...)`  | Create cmd for manual control |

All helpers are no-ops in collect mode (plan generation).

## Output Management

### Standard Output

```go
type Output struct {
    Stdout io.Writer
    Stderr io.Writer
}
```

### Parallel Buffering

Parallel execution prevents interleaved output:

```
┌─────────────────────────────────────────────────────┐
│ parallel.run(ctx)                                   │
│                                                     │
│   Goroutine 1          Goroutine 2                  │
│   ┌──────────┐         ┌──────────┐                 │
│   │ buffer 1 │         │ buffer 2 │                 │
│   └────┬─────┘         └────┬─────┘                 │
│        │                    │                       │
│        │ (wait for all)     │                       │
│        ▼                    ▼                       │
│   ┌──────────────────────────────┐                  │
│   │ flush buffer 1, then buffer 2│                  │
│   └──────────────────────────────┘                  │
└─────────────────────────────────────────────────────┘
```

Each goroutine gets a `bufferedOutput` that captures writes. After all complete,
buffers are flushed sequentially to maintain deterministic output order.

## Shim Generation

The `./pok` script is a generated shim that bootstraps Go and runs pocket.

### Bootstrap Flow

```
./pok [args]
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│ Shim Script (pok.sh / pok.cmd / pok.ps1)                    │
│   1. Check if Go exists at .pocket/go/                      │
│   2. If not, download Go (with checksum verification)       │
│   3. Set PATH to include .pocket/go/bin                     │
│   4. Run: go run ./.pocket -- [args]                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ .pocket/main.go                                             │
│   package main                                              │
│   import "github.com/fredrikaverpil/pocket"                 │
│   func main() { pocket.RunConfig(Config) }                  │
└─────────────────────────────────────────────────────────────┘
```

### Multi-Module Shims

For monorepos, shims are generated in each module directory:

```
repo/
├── pok                    # root shim (context: ".")
├── .pocket/               # pocket configuration
├── services/
│   ├── api/
│   │   └── pok            # shim (context: "services/api")
│   └── web/
│       └── pok            # shim (context: "services/web")
```

Each shim sets `POK_CONTEXT` to its relative path, enabling directory-aware
function visibility:

```bash
# In services/api/pok:
export POK_CONTEXT="services/api"
go run ../../.pocket -- "$@"
```

### Shim Generation Process

```
shim.Generate(cfg)
        │
        ├─ Read Go version from .pocket/go.mod
        ├─ Fetch Go download checksums
        ├─ CollectModuleDirectories(cfg.AutoRun)
        │         │
        │         └─ Walk Runnable tree
        │            Find all PathFilters
        │            Call Resolve() on each
        │            Return unique directories
        │
        └─ For each directory + shim type:
              Generate from template
              Write executable script
```

## Scaffold Generation

`scaffold.GenerateAll()` creates and maintains the `.pocket/` directory:

```
scaffold.GenerateAll(cfg)
        │
        ├─ Create .pocket/ directory
        │
        ├─ config.go (if not exists)
        │     User-editable, never overwritten
        │
        ├─ .gitignore (if not exists)
        │     Ignores bin/, tools/, go/
        │
        ├─ main.go (always regenerated)
        │     Minimal: calls pocket.RunConfig(Config)
        │
        └─ shim.Generate(cfg)
              Generates ./pok at root and module directories
```

### File Ownership

| File                 | Ownership | Regenerated     |
| -------------------- | --------- | --------------- |
| `.pocket/main.go`    | pocket    | Always          |
| `.pocket/config.go`  | user      | Never           |
| `.pocket/.gitignore` | pocket    | Only if missing |
| `./pok`              | pocket    | Always          |

## Built-in Tasks

These tasks are always available:

| Task       | Purpose                                  |
| ---------- | ---------------------------------------- |
| `plan`     | Show execution tree                      |
| `clean`    | Remove `.pocket/tools` and `.pocket/bin` |
| `generate` | Regenerate shim scripts                  |
| `update`   | Update pocket and regenerate             |
| `git-diff` | Show git diff (CI helper)                |

## Key Design Patterns

### Unexported Interface Methods

The `Runnable` interface uses unexported methods (`run`, `funcs`) to prevent
external implementations, ensuring only pocket's types can be Runnables.

### Immutable Builders

`PathFilter.In()`, `Except()`, etc. return new copies, enabling fluent chaining:

```go
// Each call returns a new PathFilter
paths := Paths(fn).In("a").Except("b")
```

### Dual-Mode via Context

The same code tree handles both execution and planning by checking mode:

```go
func (f *FuncDef) run(ctx context.Context) error {
    ec := getExecContext(ctx)
    if ec.mode == modeCollect {
        // register in plan, walk body only
        return nil
    }
    // actually execute
    return f.fn(ctx)
}
```

### Pointer-Based Identity

Using `reflect.ValueOf(r).Pointer()` for deduplication enables shared references
in complex trees including across parallel branches.
