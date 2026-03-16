# pk/run Subpackage Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development
> (if subagents available) or superpowers:executing-plans to implement this
> plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move task-authoring utilities from `pk` to `pk/run`, reducing
config-author autocomplete noise from ~35 to ~20 symbols.

**Architecture:** Extract shared implementations into `pk/internal/engine`
(context keys, exec, output, flags). Create `pk/run` as public API wrapping
engine. Update `pk` to use engine internally and unexport internal-only symbols.

**Tech Stack:** Go, no new dependencies.

**Spec:** `docs/superpowers/specs/2026-03-16-pk-run-subpackage-design.md`

---

## Chunk 1: Create pk/internal/engine

### Task 1: Create engine package — context keys and accessors

**Files:**

- Create: `pk/internal/engine/doc.go`
- Create: `pk/internal/engine/context.go`

This is the foundation. All context key types and their accessor/modifier
functions move here. For keys whose values are types defined in `pk` (like
`*Plan` and `*executionTracker`), the engine stores and retrieves `any`.

- [ ] **Step 1: Create `pk/internal/engine/doc.go`**

```go
// Package engine provides shared implementations for the pk task runner.
//
// This is an internal package. Use [github.com/fredrikaverpil/pocket/pk]
// for config authoring and [github.com/fredrikaverpil/pocket/pk/run] for
// task authoring.
package engine
```

- [ ] **Step 2: Create `pk/internal/engine/context.go`**

Move all context key types and their accessor/modifier functions from
`pk/context.go` into `engine/context.go`. Change package to `engine`.

Context keys to move (gathered from `pk/context.go`, `pk/task.go`, `pk/plan.go`,
`pk/tracker.go`, `pk/exec.go`, `pk/output.go`):

```
pathKey, forceRunKey, verboseKey, gitDiffKey, commitsCheckKey,
envKey, nameSuffixKey, autoExecKey, outputKey, noticePatternsKey,
planKey, trackerKey, taskFlagsKey, cliFlagsKey
```

For each key, move its accessor(s) and modifier(s). Export them all since both
`pk` and `pk/run` need access.

Key functions to move and export:

From `pk/context.go`:

- `PathFromContext` (already exported)
- `Verbose` (already exported)
- `ContextWithPath` (already exported)
- `contextWithVerbose` → export as `ContextWithVerbose`
- `contextWithGitDiffEnabled` → export as `ContextWithGitDiffEnabled`
- `gitDiffEnabledFromContext` → export as `GitDiffEnabledFromContext`
- `contextWithCommitsCheckEnabled` → export as `ContextWithCommitsCheckEnabled`
- `commitsCheckEnabledFromContext` → export as `CommitsCheckEnabledFromContext`
- `isAutoExec` → export as `IsAutoExec`
- `contextWithAutoExec` → export as `ContextWithAutoExec`
- `nameSuffixFromContext` → export as `NameSuffixFromContext`
- `contextWithNameSuffix` → export as `ContextWithNameSuffix`
- `forceRunFromContext` → export as `ForceRunFromContext`
- `withForceRun` → export as `WithForceRun`
- `ContextWithEnv` (already exported)
- `ContextWithoutEnv` (already exported)
- `EnvConfig` (already exported)
- `EnvConfigFromContext` (already exported)

From `pk/task.go`:

- `taskFlagsKey` type
- `withTaskFlags` → export as `WithTaskFlags`
- `taskFlagsFromContext` → export as `TaskFlagsFromContext`
- `cliFlagsKey` type
- `withCLIFlags` → export as `WithCLIFlags`
- `cliFlagsFromContext` → export as `CLIFlagsFromContext`

From `pk/plan.go`:

- `planKey` type
- Plan context: store as `any`. Export `SetPlan(ctx, any) context.Context` and
  `PlanFromContext(ctx) any`

From `pk/tracker.go`:

- `trackerKey` type
- Tracker context: store as `any`. Export `SetTracker(ctx, any) context.Context`
  and `TrackerFromContext(ctx) any`

Also define a `WarningMarker` interface in engine for tracker access from `Exec`
(which can't import `pk` for `*executionTracker`):

```go
// WarningMarker is implemented by types that can record warnings.
// Used by Exec to mark warnings without importing the tracker's package.
type WarningMarker interface {
	MarkWarning()
}
```

`executionTracker` in `pk/tracker.go` already has a `markWarning()` method —
export it as `MarkWarning()` to satisfy this interface.

From `pk/output.go`:

- `outputKey` type
- `outputFromContext` → export as `OutputFromContext`
- Also export `SetOutput(ctx, *Output) context.Context` for use by
  `pk/composition.go` and `pk/cli.go` when setting output in context

From `pk/exec.go`:

- `noticePatternsKey` type
- `noticePatternsFromContext` → export as `NoticePatternsFromContext`

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/fredrik/code/public/pocket && go build ./pk/internal/engine/`
Expected: SUCCESS (engine has no pk imports at this point)

- [ ] **Step 4: Commit**

```
feat(pk): add pk/internal/engine with context keys and accessors
```

### Task 2: Move output implementation to engine

**Files:**

- Create: `pk/internal/engine/output.go`

- [ ] **Step 1: Move output types and functions from `pk/output.go`**

Move to `engine/output.go`, export all:

- `Output` struct (already exported)
- `StdOutput` func (already exported)
- `bufferedOutput` → export as `BufferedOutput`
- `newBufferedOutput` → export as `NewBufferedOutput`
- `BufferedOutput.Output` method
- `BufferedOutput.Flush` method
- `Printf` func (already exported)
- `Println` func (already exported)
- `Errorf` func (already exported)

Also add `SetOutput(ctx, *Output) context.Context` for setting output in context
(used by `pk/composition.go` and `pk/cli.go`).

These functions call `OutputFromContext` and `SetOutput` which are in
`engine/context.go` (same package), so no import issues.

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/fredrik/code/public/pocket && go build ./pk/internal/engine/`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```
feat(pk): move output implementation to pk/internal/engine
```

### Task 3: Move exec implementation to engine

**Files:**

- Create: `pk/internal/engine/exec.go`
- Create: `pk/internal/engine/exec_unix.go` (from `pk/exec_unix.go`,
  `//go:build unix`)
- Create: `pk/internal/engine/exec_other.go` (from `pk/exec_other.go`,
  `//go:build !unix`)

- [ ] **Step 1: Move exec implementation from `pk/exec.go`**

Move to `engine/exec.go`, export all:

- `Exec` func (already exported)
- `Do` func → keep in `pk` (returns `Runnable`, sealed interface)
- `doRunnable` → keep in `pk`
- `RegisterPATH` func (already exported)
- `DefaultNoticePatterns` var (already exported)
- `containsNotice` → export as `ContainsNotice`
- `lookPathInEnv` → export as `LookPathInEnv`
- `prependBinToPath` → export as `PrependBinToPath`
- `applyEnvConfig` → export as `ApplyEnvConfig`
- `initColorEnv` → export as `InitColorEnv`
- `colorForceEnvVars` → export as `ColorForceEnvVars`
- `colorEnvOnce` → export as `ColorEnvOnce`
- `colorEnvVars` → export as `ColorEnvVars`
- `isTerminal` → export as `IsTerminal`
- `waitDelay` → export as `WaitDelay`
- `extraPATHDirs`, `extraPATHDirsMu` → export

Also move the build-tagged files:

- `pk/exec_unix.go` → `pk/internal/engine/exec_unix.go` (contains
  `setGracefulShutdown` with `//go:build unix`)
- `pk/exec_other.go` → `pk/internal/engine/exec_other.go` (contains
  `setGracefulShutdown` no-op with `//go:build !unix`)

Export `setGracefulShutdown` → `SetGracefulShutdown` in both files.

The `Exec` function uses `engine.TrackerFromContext(ctx)` to get the tracker as
`any`, then type-asserts to `engine.WarningMarker` to call `MarkWarning()`. This
avoids importing `pk` for `*executionTracker`:

```go
if tracker := TrackerFromContext(ctx); tracker != nil {
    if wm, ok := tracker.(WarningMarker); ok {
        wm.MarkWarning()
    }
}
```

Engine imports `pk/repopath` for `FromGitRoot` and `FromBinDir`. This is a leaf
package — no cycles.

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/fredrik/code/public/pocket && go build ./pk/internal/engine/`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```
feat(pk): move exec implementation to pk/internal/engine
```

### Task 4: Move flag handling to engine

**Files:**

- Create: `pk/internal/engine/flags.go`

- [ ] **Step 1: Move flag functions from `pk/flags.go`**

Move to `engine/flags.go`, export all:

- `GetFlags` func (already exported, generic)
- `buildFlagSetFromStruct` → export as `BuildFlagSetFromStruct`
- `structToMap` → export as `StructToMap`
- `mapToStruct` → export as `MapToStruct`
- `diffStructs` → export as `DiffStructs`
- `flagError` type → export as `FlagError`

`GetFlags` calls `TaskFlagsFromContext` and `MapToStruct`, both now in engine —
same package, no issues.

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/fredrik/code/public/pocket && go build ./pk/internal/engine/`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```
feat(pk): move flag handling to pk/internal/engine
```

## Chunk 2: Create pk/run and update pk

### Task 5: Create pk/run package

**Files:**

- Create: `pk/run/doc.go`
- Create: `pk/run/run.go`

- [ ] **Step 1: Create `pk/run/doc.go`**

```go
// Package run provides task-authoring utilities for Pocket.
//
// Task and tool authors use this package for command execution, output,
// flag handling, and context accessors. Config authors typically only
// need [github.com/fredrikaverpil/pocket/pk].
//
// # Command Execution
//
// Use [Exec] to run external commands with automatic PATH setup,
// output buffering, and graceful shutdown:
//
//	run.Exec(ctx, "golangci-lint", "run", "./...")
//
// # Output
//
// Use [Printf], [Println], and [Errorf] for output that works correctly
// in parallel task execution:
//
//	run.Printf(ctx, "  running: %s\n", cmd)
//
// # Flags
//
// Use [GetFlags] to retrieve resolved flag values from context:
//
//	f := run.GetFlags[TestFlags](ctx)
//
// # Context
//
// Use [PathFromContext], [Verbose], and the ContextWith* functions to
// read and modify execution context:
//
//	if run.Verbose(ctx) {
//	    run.Printf(ctx, "  verbose output\n")
//	}
package run
```

- [ ] **Step 2: Create `pk/run/run.go`**

Thin public API wrapping engine. Each function delegates to engine:

```go
package run

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// Exec runs an external command with .pocket/bin prepended to PATH.
// See [engine.Exec] for full documentation.
func Exec(ctx context.Context, name string, args ...string) error {
	return engine.Exec(ctx, name, args...)
}

// Printf formats and writes to the context's stdout.
func Printf(ctx context.Context, format string, a ...any) {
	engine.Printf(ctx, format, a...)
}

// Println writes to the context's stdout, appending a newline.
func Println(ctx context.Context, a ...any) {
	engine.Println(ctx, a...)
}

// Errorf formats and writes to the context's stderr.
func Errorf(ctx context.Context, format string, a ...any) {
	engine.Errorf(ctx, format, a...)
}

// GetFlags retrieves the resolved flags for a task from context.
func GetFlags[T any](ctx context.Context) T {
	return engine.GetFlags[T](ctx)
}

// PathFromContext returns the current execution path from the context.
func PathFromContext(ctx context.Context) string {
	return engine.PathFromContext(ctx)
}

// Verbose returns whether verbose mode is enabled in the context.
func Verbose(ctx context.Context) bool {
	return engine.Verbose(ctx)
}

// ContextWithPath returns a new context with the given execution path.
func ContextWithPath(ctx context.Context, path string) context.Context {
	return engine.ContextWithPath(ctx, path)
}

// ContextWithEnv returns a new context that sets an environment variable
// for [Exec] calls. The keyValue must be in "KEY=value" format.
func ContextWithEnv(ctx context.Context, keyValue string) context.Context {
	return engine.ContextWithEnv(ctx, keyValue)
}

// ContextWithoutEnv returns a new context that filters out environment
// variables matching the given prefix from [Exec] calls.
func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	return engine.ContextWithoutEnv(ctx, prefix)
}

// EnvConfig holds environment variable overrides applied to [Exec] calls.
type EnvConfig = engine.EnvConfig

// EnvConfigFromContext returns a copy of the environment config from the context.
func EnvConfigFromContext(ctx context.Context) EnvConfig {
	return engine.EnvConfigFromContext(ctx)
}

// PlanFromContext returns the Plan from the context.
// Returns nil if no plan is set.
func PlanFromContext(ctx context.Context) *pk.Plan {
	v := engine.PlanFromContext(ctx)
	if v == nil {
		return nil
	}
	return v.(*pk.Plan)
}

// RegisterPATH registers a directory to be added to PATH for all [Exec] calls.
func RegisterPATH(dir string) {
	engine.RegisterPATH(dir)
}

// DefaultNoticePatterns are the substrings used to detect warning-like output.
var DefaultNoticePatterns = engine.DefaultNoticePatterns
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /Users/fredrik/code/public/pocket && go build ./pk/run/` Expected:
SUCCESS

- [ ] **Step 4: Commit**

```
feat(pk): add pk/run public API for task authoring
```

### Task 6: Update pk to use engine internally

**Files:**

- Modify: `pk/context.go` — replace implementations with engine calls
- Modify: `pk/output.go` — replace implementations with engine calls
- Modify: `pk/exec.go` — replace implementations with engine calls, keep `Do`
- Modify: `pk/flags.go` — replace implementations with engine calls
- Modify: `pk/task.go` — use engine context accessors
- Modify: `pk/plan.go` — use engine context accessors
- Modify: `pk/tracker.go` — use engine context accessors
- Modify: `pk/composition.go` — use engine output types
- Modify: `pk/cli.go` — use engine context modifiers
- Modify: `pk/builtins.go` — use engine functions
- Modify: `pk/doc.go` — update package documentation

This is the most complex task. Each file in `pk/` that defined functions now in
engine must be updated to delegate to or directly use engine.

- [ ] **Step 1: Update `pk/context.go`**

Replace all function bodies with engine delegations. Keep the exported functions
as thin wrappers for backward compatibility during migration, but they will be
removed in Task 8 (unexport phase).

For unexported functions used only within `pk`, call engine directly at call
sites instead.

```go
package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

// Re-export types that stay in pk's public API during migration.
// These will be removed when the symbols are dropped from pk.

func PathFromContext(ctx context.Context) string {
	return engine.PathFromContext(ctx)
}

func Verbose(ctx context.Context) bool {
	return engine.Verbose(ctx)
}

func ContextWithPath(ctx context.Context, path string) context.Context {
	return engine.ContextWithPath(ctx, path)
}

func ContextWithEnv(ctx context.Context, keyValue string) context.Context {
	return engine.ContextWithEnv(ctx, keyValue)
}

func ContextWithoutEnv(ctx context.Context, prefix string) context.Context {
	return engine.ContextWithoutEnv(ctx, prefix)
}

type EnvConfig = engine.EnvConfig

func EnvConfigFromContext(ctx context.Context) EnvConfig {
	return engine.EnvConfigFromContext(ctx)
}
```

For internal-only functions, use `engine.XXX` directly at call sites in
`pk/task.go`, `pk/cli.go`, etc.

- [ ] **Step 2: Update `pk/output.go`**

Replace with engine delegations:

```go
package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

type Output = engine.Output

func StdOutput() *Output {
	return engine.StdOutput()
}

func Printf(ctx context.Context, format string, a ...any) {
	engine.Printf(ctx, format, a...)
}

func Println(ctx context.Context, a ...any) {
	engine.Println(ctx, a...)
}

func Errorf(ctx context.Context, format string, a ...any) {
	engine.Errorf(ctx, format, a...)
}
```

- [ ] **Step 3: Update `pk/exec.go`**

Replace exec implementation with engine delegation. Keep `Do` and `doRunnable`
in this file:

```go
package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

func Exec(ctx context.Context, name string, args ...string) error {
	return engine.Exec(ctx, name, args...)
}

func RegisterPATH(dir string) {
	engine.RegisterPATH(dir)
}

var DefaultNoticePatterns = engine.DefaultNoticePatterns

// Do wraps a Go function as a [Runnable] for use in task composition.
func Do(fn func(ctx context.Context) error) Runnable {
	return &doRunnable{fn: fn}
}

type doRunnable struct {
	fn func(ctx context.Context) error
}

func (d *doRunnable) run(ctx context.Context) error {
	return d.fn(ctx)
}
```

- [ ] **Step 4: Update `pk/flags.go`**

Replace with engine delegations:

```go
package pk

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk/internal/engine"
)

func GetFlags[T any](ctx context.Context) T {
	return engine.GetFlags[T](ctx)
}
```

Keep internal functions as engine calls at their call sites.

- [ ] **Step 5: Update `pk/task.go`**

Replace all internal context accessor calls with engine equivalents:

- `withTaskFlags` → `engine.WithTaskFlags`
- `taskFlagsFromContext` → `engine.TaskFlagsFromContext`
- `withCLIFlags` → `engine.WithCLIFlags`
- `cliFlagsFromContext` → `engine.CLIFlagsFromContext`
- `nameSuffixFromContext` → `engine.NameSuffixFromContext`
- `forceRunFromContext` → `engine.ForceRunFromContext`
- `PlanFromContext` → typed wrapper using `engine.PlanFromContext`
- `Printf` → `engine.Printf`
- `flagError` → `engine.FlagError`

In `Task.execute()`, the `recover()` type assertion must change:

```go
// Before: if fe, ok := r.(flagError); ok {
// After:
if fe, ok := r.(engine.FlagError); ok {
```

Also update `buildFlagSet` to call `engine.BuildFlagSetFromStruct`.

- [ ] **Step 6: Update `pk/plan.go`**

Replace internal context accessor calls:

- `planKey` → use `engine.SetPlan` / `engine.PlanFromContext`
- Add unexported `planFromContext` that casts `engine.PlanFromContext` to
  `*Plan`

- [ ] **Step 7: Update `pk/tracker.go`**

Replace internal context accessor calls:

- `trackerKey` → use `engine.SetTracker` / `engine.TrackerFromContext`
- Add unexported typed wrappers
- Export `markWarning` → `MarkWarning` so `executionTracker` satisfies
  `engine.WarningMarker` interface (used by `engine.Exec`)

- [ ] **Step 8: Update `pk/composition.go`**

Replace output internals with engine equivalents:

- `outputFromContext` → `engine.OutputFromContext`
- `context.WithValue(gCtx, outputKey{}, ...)` → `engine.SetOutput(gCtx, ...)`
- `newBufferedOutput` → `engine.NewBufferedOutput`

- [ ] **Step 9: Update `pk/cli.go`**

Replace internal context modifier calls:

- `contextWithVerbose` → `engine.ContextWithVerbose`
- `contextWithGitDiffEnabled` → `engine.ContextWithGitDiffEnabled`
- `contextWithCommitsCheckEnabled` → `engine.ContextWithCommitsCheckEnabled`
- `withCLIFlags` → `engine.WithCLIFlags`
- `context.WithValue(ctx, outputKey{}, ...)` → `engine.SetOutput(ctx, ...)`
- `context.WithValue(ctx, planKey{}, ...)` → `engine.SetPlan(ctx, ...)`
- `outputFromContext` → `engine.OutputFromContext` (in `printTaskHelp`)
- `isTerminal` → `engine.IsTerminal` (in `printFinalStatus`)

- [ ] **Step 10: Update `pk/builtins.go`**

Replace function calls with engine equivalents. Builtins use:

- `Printf` → `engine.Printf`
- `Println` → `engine.Println`
- `Verbose` → `engine.Verbose`
- `Exec` → `engine.Exec`
- `GetFlags` → `engine.GetFlags`
- `ContextWithPath` → `engine.ContextWithPath`
- `ContextWithEnv` → `engine.ContextWithEnv`
- `PlanFromContext` → use unexported `planFromContext`
- `outputFromContext` → `engine.OutputFromContext` (in `printPlanJSON`)

- [ ] **Step 11: Update `pk/doc.go`**

```go
// Package pk is the config-authoring API for Pocket, a composable task
// runner framework.
//
// Use this package to define task composition trees in .pocket/config.go.
// For task implementation utilities (command execution, output, flags),
// see [github.com/fredrikaverpil/pocket/pk/run].
//
// # Configuration
//
// Define your task tree using [Config]:
//
//	var Config = &pk.Config{
//	    Auto: pk.Serial(
//	        Format,
//	        pk.Parallel(Lint, Test),
//	        Build,
//	    ),
//	}
//
// # Task Definition
//
// Define tasks as struct literals:
//
//	var Lint = &pk.Task{
//	    Name:  "lint",
//	    Usage: "run linters",
//	    Do: func(ctx context.Context) error {
//	        return run.Exec(ctx, "golangci-lint", "run")
//	    },
//	}
//
// # Composition
//
// Use [Serial] and [Parallel] to compose tasks into execution trees.
// Use [WithOptions] with [WithPath], [WithDetect], and [WithFlags] to
// control path filtering and flag overrides.
//
// # Plan Introspection
//
// Access the execution plan at runtime via [run.PlanFromContext] to
// generate CI workflows or custom tooling.
package pk
```

- [ ] **Step 12: Run pk tests only**

Run:
`cd /Users/fredrik/code/public/pocket && go test ./pk/... ./pk/internal/...`
Expected: ALL PASS

Note: Only `pk/` and `pk/internal/` tests should pass at this point. The full
repo build (`go build ./...`) will fail because `tasks/` and `tools/` still
reference removed `pk.Exec`, `pk.Printf`, etc. This is expected and will be
fixed in Tasks 8-9.

- [ ] **Step 14: Commit**

```
refactor(pk): delegate to pk/internal/engine for shared implementations
```

### Task 7: Unexport internal-only symbols from pk

**Files:**

- Modify: `pk/exec.go` — remove `Exec`, `RegisterPATH`, `DefaultNoticePatterns`
  re-exports
- Modify: `pk/output.go` — remove `Output`, `StdOutput`, `Printf`, `Println`,
  `Errorf` re-exports
- Modify: `pk/flags.go` — remove `GetFlags` re-export
- Modify: `pk/context.go` — remove moved symbols re-exports
- Modify: `pk/plan.go` — unexport `NewPlan`, `ExecuteTask`
- Modify: `pk/builtins.go` — unexport `ErrGitDiffUncommitted`,
  `ErrCommitsInvalid`

- [ ] **Step 1: Remove re-exported symbols from pk**

Delete the thin wrapper functions added in Task 6 for symbols that are now
exclusively in `pk/run`:

From `pk/context.go`, remove:

- `PathFromContext`, `Verbose`, `ContextWithPath`
- `ContextWithEnv`, `ContextWithoutEnv`
- `EnvConfig` type alias, `EnvConfigFromContext`

From `pk/output.go`, remove:

- `Output` type alias, `StdOutput`
- `Printf`, `Println`, `Errorf`

From `pk/exec.go`, remove:

- `Exec`, `RegisterPATH`, `DefaultNoticePatterns`

From `pk/flags.go`, remove:

- `GetFlags`

- [ ] **Step 2: Unexport `NewPlan` in `pk/plan.go`**

Rename `NewPlan` → `newPlan` (the unexported `newPlan` already exists, so rename
the exported one to something like `buildPlan` or merge them).

Update test files that call `NewPlan`:

- `pk/e2e_test.go` (~15 calls)
- `pk/plan_test.go` (if applicable)

Since tests are in package `pk`, they can access unexported names.

- [ ] **Step 3: Unexport `ExecuteTask` in `pk/cli.go`**

Rename `ExecuteTask` → `executeTaskByName` or similar.

Update test files that call `ExecuteTask`:

- `pk/e2e_test.go` (~12 calls)
- `pk/shim_test.go` (~3 calls)

- [ ] **Step 4: Unexport errors in `pk/builtins.go`**

Rename:

- `ErrGitDiffUncommitted` → `errGitDiffUncommitted`
- `ErrCommitsInvalid` → `errCommitsInvalid`

Update `pk/cli.go` (`printFinalStatus`) which references these errors.

- [ ] **Step 5: Verify pk compiles**

Run: `cd /Users/fredrik/code/public/pocket && go build ./pk/...` Expected:
SUCCESS

- [ ] **Step 6: Commit**

```
refactor(pk): unexport internal-only symbols
```

## Chunk 3: Migrate tasks/ and tools/

### Task 8: Migrate tasks/ packages

Each task package needs the same mechanical change: add `pk/run` import, replace
`pk.Exec` → `run.Exec`, `pk.Printf` → `run.Printf`, etc.

**This task should be parallelized with one subagent per package.**

**Migration pattern** (apply to each file):

1. Add import: `"github.com/fredrikaverpil/pocket/pk/run"`
2. Replace these symbols:
   - `pk.Exec` → `run.Exec`
   - `pk.Printf` → `run.Printf`
   - `pk.Println` → `run.Println`
   - `pk.Errorf` → `run.Errorf`
   - `pk.GetFlags` → `run.GetFlags`
   - `pk.PathFromContext` → `run.PathFromContext`
   - `pk.Verbose` → `run.Verbose`
   - `pk.ContextWithPath` → `run.ContextWithPath`
   - `pk.ContextWithEnv` → `run.ContextWithEnv`
   - `pk.ContextWithoutEnv` → `run.ContextWithoutEnv`
   - `pk.PlanFromContext` → `run.PlanFromContext`
   - `pk.RegisterPATH` → `run.RegisterPATH`
3. If file no longer uses any `pk` symbols, remove the `pk` import
4. Run `goimports` or `go build` to verify

**Packages and their affected files:**

- [ ] **tasks/golang** (7 files: test.go, lint.go, fix.go, format.go,
      vulncheck.go, release.go, pprof.go)
- [ ] **tasks/python** (4 files: test.go, format.go, lint.go, typecheck.go)
- [ ] **tasks/github** (1 file: workflows.go — uses `pk.PlanFromContext`)
- [ ] **tasks/markdown** (1 file: format.go)
- [ ] **tasks/lua** (1 file: format.go)
- [ ] **tasks/claude** (1 file: validate.go)
- [ ] **tasks/docs** (1 file: zensical.go)
- [ ] **tasks/treesitter** (2 files: parser.go, query.go)

- [ ] **Verify all tasks compile**

Run: `cd /Users/fredrik/code/public/pocket && go build ./tasks/...` Expected:
SUCCESS

- [ ] **Commit**

```
refactor(tasks): migrate to pk/run for task-authoring utilities
```

### Task 9: Migrate tools/ packages

Same mechanical migration as tasks/.

**Packages and their affected files:**

- [ ] **tools/uv** (uv.go — heaviest usage: Exec, PathFromContext, Verbose,
      Printf, ContextWithPath, ContextWithoutEnv, ContextWithEnv)
- [ ] **tools/bun** (bun.go — uses Exec)
- [ ] **tools/golang** (golang.go — uses Verbose, Printf)
- [ ] **tools/neovim** (neovim.go — uses RegisterPATH)

- [ ] **Verify all tools compile**

Run: `cd /Users/fredrik/code/public/pocket && go build ./tools/...` Expected:
SUCCESS

- [ ] **Commit**

```
refactor(tools): migrate to pk/run for task-authoring utilities
```

### Task 10: Full verification

- [ ] **Step 1: Run full build**

Run: `cd /Users/fredrik/code/public/pocket && go build ./...` Expected: SUCCESS

- [ ] **Step 2: Run all tests**

Run: `cd /Users/fredrik/code/public/pocket && go test ./...` Expected: ALL PASS

- [ ] **Step 3: Run pok**

Run: `cd /Users/fredrik/code/public/pocket && ./pok` Expected: SUCCESS (all
tasks pass)

- [ ] **Step 4: Commit if any fixes were needed**

## Chunk 4: Documentation updates

### Task 11: Update inline Go documentation

**Files:**

- Modify: `pk/task.go` — update `Task.Do` doc to reference `run.Exec`
- Modify: `pk/options.go` — update `WithOptions` doc example
- Modify: `pk/config.go` — review and update any references
- Review all files in `pk/` for stale `[Exec]`, `[Printf]` godoc links

- [ ] **Step 1: Update `pk/task.go` doc comments**

The `Task` struct doc and `Do` field doc should reference `run.Exec`:

```go
// Do is the task's executable function. Use [run.Exec] to run external
// commands and [run.Printf] for output. Mutually exclusive with Body.
Do func(context.Context) error
```

- [ ] **Step 2: Review and update all godoc cross-references in pk/**

Search for `[Exec]`, `[Printf]`, `[Println]`, `[Errorf]`, `[GetFlags]`,
`[PathFromContext]`, `[Verbose]`, `[ContextWithPath]`, `[ContextWithEnv]`,
`[ContextWithoutEnv]`, `[PlanFromContext]` in pk/ doc comments and update to
reference `run.XXX` or `[github.com/fredrikaverpil/pocket/pk/run.XXX]`.

- [ ] **Step 3: Commit**

```
docs(pk): update godoc references to pk/run
```

### Task 12: Update markdown documentation

**Files:**

- Modify: `README.md`
- Modify: `docs/guide.md`
- Modify: `docs/reference.md`

- [ ] **Step 1: Update `README.md`**

Update the quickstart task example to show `pk/run` imports:

```go
import (
    "context"
    "fmt"
    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/pk/run"
)

var Hello = &pk.Task{
    Name:  "hello",
    Usage: "say hello",
    Flags: HelloFlags{Name: "World"},
    Do: func(ctx context.Context) error {
        f := run.GetFlags[HelloFlags](ctx)
        fmt.Printf("Hello, %s, from Pocket!\n", f.Name)
        return nil
    },
}
```

Update any other code examples that reference `pk.Exec`, `pk.Printf`, etc.

- [ ] **Step 2: Update `docs/guide.md`**

Search for all `pk.Exec`, `pk.Printf`, `pk.GetFlags`, etc. references and update
to `run.Exec`, `run.Printf`, `run.GetFlags`. Add import examples showing both
`pk` and `pk/run`.

- [ ] **Step 3: Update `docs/reference.md`**

Major update:

- Add `pk/run` section documenting all moved symbols
- Update `pk` section to remove moved symbols
- Remove `ErrGitDiffUncommitted` / `ErrCommitsInvalid` examples
- Remove `NewPlan` / `ExecuteTask` documentation
- Update all code examples

- [ ] **Step 4: Commit**

```
docs: update documentation for pk/run package split
```

### Task 13: Update skills documentation

**Files:**

- Modify: `.claude/skills/adding-tasks/SKILL.md`
- Modify: `.claude/skills/adding-tasks/PATTERNS.md`
- Modify: `.claude/skills/adding-tools/SKILL.md`
- Modify: `.claude/skills/adding-tools/PATTERNS.md`
- Modify: `.claude/skills/pocket-engine/SKILL.md`
- Modify: `.claude/skills/pocket-engine/INTERNALS.md`

- [ ] **Step 1: Update adding-tasks skill**

Replace all `pk.Exec` → `run.Exec`, `pk.Printf` → `run.Printf`, etc. in both
SKILL.md and PATTERNS.md. Add `pk/run` import to examples.

- [ ] **Step 2: Update adding-tools skill**

Same mechanical updates in SKILL.md and PATTERNS.md.

- [ ] **Step 3: Update pocket-engine skill**

Update SKILL.md and INTERNALS.md to document `pk/internal/engine` as the shared
implementation layer. Update architecture descriptions and diagrams.

- [ ] **Step 4: Commit**

```
docs: update skills for pk/run package split
```

### Task 14: Final verification

- [ ] **Step 1: Run full build**

Run: `cd /Users/fredrik/code/public/pocket && go build ./...` Expected: SUCCESS

- [ ] **Step 2: Run all tests**

Run: `cd /Users/fredrik/code/public/pocket && go test ./...` Expected: ALL PASS

- [ ] **Step 3: Run pok with all checks**

Run: `cd /Users/fredrik/code/public/pocket && ./pok -v` Expected: SUCCESS with
all tasks passing

- [ ] **Step 4: Verify pk autocomplete surface**

Run: `cd /Users/fredrik/code/public/pocket && go doc ./pk/ | head -60`

Verify that `Exec`, `Printf`, `GetFlags`, `PathFromContext`, `Verbose`,
`ContextWithPath`, `ContextWithEnv`, `ContextWithoutEnv`,
`EnvConfigFromContext`, `PlanFromContext`, `RegisterPATH`,
`DefaultNoticePatterns`, `Output`, `StdOutput`, `NewPlan`, `ExecuteTask`,
`ErrGitDiffUncommitted`, `ErrCommitsInvalid` are NOT listed.

- [ ] **Step 5: Verify pk/run surface**

Run: `cd /Users/fredrik/code/public/pocket && go doc ./pk/run/`

Verify all task-authoring symbols are present and documented.
