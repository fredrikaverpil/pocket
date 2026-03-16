# pk/run Subpackage Design

## Goal

Optimize the `pk` package for config authors by moving task-authoring utilities
to a new `pk/run` subpackage. Config authors writing `.pocket/config.go` should
see only composition and configuration symbols (~20) when typing `pk.`, not the
full ~35 symbols that include execution, output, flag handling, and context
accessors they never use.

## Problem

The `pk` package serves two audiences with different needs:

1. **Config authors** compose task trees in `.pocket/config.go`. They need
   `Config`, `Serial`, `Parallel`, `WithOptions`, `WithFlags`, etc.
2. **Task/tool authors** write task implementations in `tasks/` and `tools/`.
   They need `Exec`, `Printf`, `GetFlags`, `PathFromContext`, `Verbose`, etc.

Today both audiences share one flat namespace. Config authors see task-authoring
symbols they never use, making the API feel overwhelming and requiring
documentation to sort out what belongs to whom.

## Design

### New package: `pk/run`

A new public package at `github.com/fredrikaverpil/pocket/pk/run` containing
task-authoring utilities. Task and tool authors import both `pk` (for types like
`Task`, `Runnable`) and `pk/run` (for execution utilities).

### New internal package: `pk/internal/engine`

A shared implementation package that holds context key types, exec
implementation, output handling, and flag machinery. Both `pk` and `pk/run`
import it. This breaks what would otherwise be a circular dependency between `pk`
and `pk/run`.

### `internal/scaffold` and `internal/shim` stay at `internal/`

These packages are used by both `pk` (in `cli.go` and `builtins.go`) and
`cmd/pocket/main.go`. Go's `internal` visibility rules require them to remain
at the module root level to be accessible to both.

## Package Structure After

```
pk/                          Config authoring + core engine
pk/run/                      Task/tool authoring utilities
pk/internal/engine/          Shared implementations
internal/scaffold/           Scaffold generation (unchanged location)
internal/shim/               Shim generation (unchanged location)
pk/platform/                 Unchanged
pk/repopath/                 Unchanged
pk/download/                 Unchanged
pk/conventionalcommits/      Unchanged
```

## What Moves to `pk/run`

| Symbol | Type | Notes |
|--------|------|-------|
| `Exec` | func | Command execution |
| `Printf` | func | Context-aware stdout |
| `Println` | func | Context-aware stdout |
| `Errorf` | func | Context-aware stderr |
| `GetFlags[T]` | func | Generic flag access from context |
| `PathFromContext` | func | Returns current execution path |
| `Verbose` | func | Returns verbose mode status |
| `ContextWithPath` | func | Sets execution path in context |
| `ContextWithEnv` | func | Sets env variable in context |
| `ContextWithoutEnv` | func | Filters env variable in context |
| `EnvConfig` | type | Environment variable overrides |
| `EnvConfigFromContext` | func | Returns env config from context |
| `PlanFromContext` | func | Returns `*pk.Plan` from context |
| `RegisterPATH` | func | Adds directory to PATH for Exec |
| `DefaultNoticePatterns` | var | Warning detection patterns |

### `Do` stays in `pk`

`Do` wraps a function as a `Runnable`. It **cannot** move to `pk/run` because
`Runnable` has an unexported `run()` method — a sealed interface. Only types
defined in `pk` can implement it. The `doRunnable` struct created by `Do` must
live in `pk`.

This is also conceptually correct: `Do` is a composition primitive (creates a
`Runnable` from a function, like `Serial` and `Parallel` create `Runnable`s from
other `Runnable`s). It belongs with the composition API, not the task-authoring
utilities.

## What Stays in `pk`

Config authoring, composition, and core engine symbols:

```
Config, PlanConfig, ShimConfig, AllShimsConfig
Task, Runnable, Option, DetectFunc
Serial, Parallel, Do
WithOptions, WithPath, WithSkipPath, WithSkipTask, WithFlags,
WithDetect, WithForceRun, WithNameSuffix, WithNoticePatterns
DetectByFile, DefaultSkipDirs
Plan, TaskInfo
RunMain
```

## What Gets Unexported

These symbols are only used internally within `pk` and should not be part of the
public API:

| Symbol | Reason |
|--------|--------|
| `NewPlan` | Only called by `RunMain` |
| `ExecuteTask` | Only called internally by CLI dispatch |
| `Output` | Internal output plumbing |
| `StdOutput` | Internal output plumbing |
| `ErrGitDiffUncommitted` | Only used by builtin task (update docs/reference.md) |
| `ErrCommitsInvalid` | Only used by builtin task |

Note: `ErrGitDiffUncommitted` is currently referenced in `docs/reference.md` as
a public API example. Since the project is pre-v1, unexporting it is acceptable.
The docs example should be updated or removed.

## Dependency Architecture

```
pk/internal/engine       (imports pk/repopath, stdlib, golang.org/x)
      ^            ^
      |            |
     pk         pk/run ---> pk (for types: Runnable, Plan, Task)
```

- **`pk/internal/engine`**: Contains context key types, context
  accessors/modifiers, exec implementation, output implementation, flag
  handling, env config, and notice pattern detection. Imports `pk/repopath`
  (a leaf package with no pk-tree imports), stdlib, and `golang.org/x`
  packages (`golang.org/x/term` for TTY detection, `golang.org/x/sync` is
  NOT needed here — `errgroup` is only used in `pk/composition.go`).
  Does NOT import `pk` or `pk/run`.
- **`pk`**: Imports `pk/internal/engine` for internal use in builtins and core
  engine (`Task.run`, `pathFilter.run`, `parallel.run`). The `parallel.run`
  method in `composition.go` uses engine's output types (`outputFromContext`,
  `bufferedOutput`) to set up buffered output per goroutine.
  Does NOT import `pk/run`.
- **`pk/run`**: Imports `pk/internal/engine` for implementations and `pk` for
  types (`Runnable`, `Plan`). Exported functions are thin wrappers that add
  type assertions where needed (e.g., `PlanFromContext` casts engine's `any`
  return to `*pk.Plan`).

No circular dependencies.

## What Goes in `pk/internal/engine`

### Context keys and accessors

All context key types and their accessor/modifier functions:

- `pathKey{}` — execution path
- `outputKey{}` — output writers
- `verboseKey{}` — verbose mode
- `envKey{}` — environment overrides
- `forceRunKey{}` — force task execution
- `nameSuffixKey{}` — task name suffix
- `autoExecKey{}` — auto execution mode
- `gitDiffKey{}` — git diff enabled flag
- `commitsCheckKey{}` — commits check enabled flag
- `taskFlagsKey{}` — resolved task flag values
- `cliFlagsKey{}` — CLI-provided flag overrides
- `noticePatternsKey{}` — custom notice patterns
- `planKey{}` — execution plan (stored as `any`, typed accessors in `pk` and `pk/run`)
- `trackerKey{}` — execution tracker (stored as `any`, typed accessor in `pk` only)

For context values whose types are defined in `pk` (like `*Plan` and
`*executionTracker`), the engine stores and retrieves them as `any`. The typed
wrappers live in the packages that own the types:

- `pk` has unexported `planFromContext(ctx) *Plan` (casts engine's `any`)
- `pk/run` has exported `PlanFromContext(ctx) *pk.Plan` (same cast)
- `pk` has unexported `executionTrackerFromContext(ctx) *executionTracker`
  (tracker is internal, never exposed via `pk/run`)

### Exec implementation

Command execution, PATH prepending (`prependBinToPath`), env config application
(`applyEnvConfig`), command resolution (`lookPathInEnv`), color env detection
(`initColorEnv`, `colorForceEnvVars`), notice pattern matching
(`containsNotice`), and graceful shutdown (`setGracefulShutdown`). Imports
`pk/repopath` for `FromGitRoot` and `FromBinDir`.

### Output implementation

`Output` struct (stdout/stderr writers), `StdOutput` constructor,
`Printf`/`Println`/`Errorf` functions, `bufferedOutput` type with `Flush`
logic.

### Flag handling

`GetFlags` implementation (generic — package-level functions support generics),
`buildFlagSetFromStruct`, `structToMap`, `mapToStruct`, `diffStructs`.

### Env config

`EnvConfig` type, context accessors for env (`ContextWithEnv`,
`ContextWithoutEnv`, `EnvConfigFromContext`).

### Notice patterns

`DefaultNoticePatterns` variable, `noticePatternsFromContext` accessor.

## Code Examples

### Config author (unchanged)

```go
package main

import (
    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/tasks/golang"
    "github.com/fredrikaverpil/pocket/tasks/github"
)

var Config = &pk.Config{
    Auto: pk.Parallel(
        golang.Tasks(),
        pk.WithOptions(
            github.Tasks(),
            pk.WithFlags(github.WorkflowFlags{
                Platforms: []github.Platform{github.Ubuntu, github.MacOS},
            }),
        ),
    ),
    Plan: &pk.PlanConfig{
        Shims: pk.AllShimsConfig(),
    },
}
```

### Task author (before)

```go
package golang

import (
    "github.com/fredrikaverpil/pocket/pk"
)

var Test = &pk.Task{
    Name:  "go-test",
    Usage: "run Go tests",
    Flags: TestFlags{Race: true},
    Do: func(ctx context.Context) error {
        f := pk.GetFlags[TestFlags](ctx)
        if pk.Verbose(ctx) {
            pk.Printf(ctx, "  running tests\n")
        }
        return pk.Exec(ctx, "go", "test", "./...")
    },
}
```

### Task author (after)

```go
package golang

import (
    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/pk/run"
)

var Test = &pk.Task{
    Name:  "go-test",
    Usage: "run Go tests",
    Flags: TestFlags{Race: true},
    Do: func(ctx context.Context) error {
        f := run.GetFlags[TestFlags](ctx)
        if run.Verbose(ctx) {
            run.Printf(ctx, "  running tests\n")
        }
        return run.Exec(ctx, "go", "test", "./...")
    },
}
```

## Migration Impact

### Config files (`.pocket/config.go`)

**No changes.** Config authors only use symbols that stay in `pk`.

### Task packages (`tasks/*`)

Mechanical migration: add `pk/run` import, change `pk.Exec` to `run.Exec`,
`pk.Printf` to `run.Printf`, etc. No logic changes.

Affected packages: `tasks/golang`, `tasks/python`, `tasks/github`,
`tasks/markdown`, `tasks/lua`, `tasks/claude`, `tasks/docs`,
`tasks/treesitter`.

### Tool packages (`tools/*`)

Same mechanical migration for any tool that uses `pk.Exec`, `pk.Do`, etc.
Note: `pk.Do` stays in `pk`, so only `Exec`/`Printf`/etc. references change.

Affected packages: `tools/bun`, `tools/golang`, `tools/uv`, `tools/prettier`,
`tools/mdformat`, `tools/neovim`, and others that call `pk.Exec`.

### Breaking change

Yes. The project is pre-v1, so this is expected and acceptable.

## Documentation Updates

All documentation must be reviewed and updated to reflect the new structure:

### Inline Go docs

- `pk/doc.go`: Update package doc to describe `pk` as the config-authoring
  package. Reference `pk/run` for task-authoring utilities.
- `pk/run/doc.go`: New package doc describing `pk/run` as the task-authoring
  package with examples.
- `pk/internal/engine/doc.go`: Internal package doc.
- All moved functions: Update godoc comments, especially cross-references
  (e.g., `[Exec]` links, `[Printf]` references).
- All functions remaining in `pk` that reference moved symbols in their docs
  (e.g., `Task.Do` doc mentioning `Exec` should reference `run.Exec`).

### Markdown docs

- `README.md`: Update quickstart example, code examples, and any references
  to `pk.Exec`, `pk.Printf`, etc.
- `docs/guide.md`: Update all code examples and import paths.
- `docs/reference.md`: Update API reference to reflect the split. Add `pk/run`
  section. Remove `ErrGitDiffUncommitted` / `ErrCommitsInvalid` examples.

### Skills

- `.claude/skills/adding-tasks/`: Update patterns and examples to use `pk/run`.
- `.claude/skills/adding-tools/`: Update patterns and examples to use `pk/run`.
- `.claude/skills/pocket-engine/`: Update internals documentation to reflect
  `pk/internal/engine`.

## Implementation Strategy

### Parallelizable with subagents

The migration of `tasks/*` and `tools/*` packages is mechanical and
independent per package. Each package can be migrated by a separate subagent
in parallel:

- One subagent per `tasks/*` package (8 packages)
- One subagent per `tools/*` package (~13 packages)

### Execution order

1. Create `pk/internal/engine` — extract shared implementations from `pk`
2. Create `pk/run` — public API wrapping engine, importing `pk` for types
3. Update `pk` — use engine internally, unexport symbols, update godoc
4. Migrate `tasks/*` packages (parallel subagents)
5. Migrate `tools/*` packages (parallel subagents)
6. Update all documentation (inline godoc + markdown + skills)
7. Run `./pok` to verify everything passes
