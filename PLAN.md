# PLAN

## Goals

Rewrite [Pocket](https://github.com/fredrikaverpil/pocket) (also available in
`~/code/public/pocket-v1`) with feature parity in this v2 git worktree folder.

The previous public API and architecture can be viewed in pocket-v1's:

- README.md
- ARCHITECTURE.md

It will be very important to quicky get to an end to end state, where we can
execute a task from the compositional configuration.

I want you to be my pair-programming buddy and help me gradually go through
every step carefully when planning and implementing.

Guidelines:

- Simplistic, easy-to-understand, ideomatic to Go, aims to use standard library
  for everything
- Clear API so that IDE code completion will help understanding how to construct
  config, tasks, tools. We can use different packages to signal the scope in
  which the user is currently configuring/building something.
- Core and internals are kept private/unexported. Avoid making internals
  available to users.
- Throughout the work, let's populate a new ARCHITECTURE.md file.
- Stop at inbetween each change, ask if I am happy with the implementation. Then
  git commit and move on.

## Phase 1: Read the config into a "plan"

- [x] Create a temporary "main.go" that runs (replaced by shim in Phase 5)
- [x] Define a new "hello-world" Task that takes option(s) like "name"
- [x] Define the Config struct
- [x] Define a Serial function, so that we can compose the config
- [x] Calculate the "plan"
- [x] Have plan sent to an "executor", which executes the composition

Questions to answer:

- [x] What is the composition actually, in technical terms. Is it a tree or a
      DAG? The action to show this could be called a "plan", but what is it,
      actually? Answer: It's a tree, not a DAG.
- [x] When the phase 1 is done, do we think we can generate a shim from the
      plan/tree/graph? Answer: Answer: Yes! The plan has everything needed:

  ```
  plan.Tasks // All task names
  plan.ModuleDirectories // Where to create shims
  plan.PathMappings // Which dirs each task runs in
  ```

  A shim in services/api/pok would:
  1. Parse CLI args: ./pok lint
  2. Look up "lint" in plan.Tasks
  3. Filter by current directory (services/api)
  4. Execute that specific task

  Gap: The current plan doesn't distinguish between:
  - CLI-invocable tasks (top-level, user wants ./pok lint)
  - Internal tasks (composition-only, like a helper task inside Serial)

## Phase 2: Shim generation

- [x] Created `internal/shim/` package with:
  - [x] `gomod.go` - reads Go version from go.mod using Go 1.25+
        `strings.SplitSeq`
  - [x] `checksums.go` - fetches SHA256 checksums from go.dev/dl API
  - [x] `shim.go` - main generation logic with `GenerateShims()` function
  - [x] `templates/` - pok.sh.tmpl (POSIX), pok.cmd.tmpl (Windows), pok.ps1.tmpl
        (PowerShell)
- [x] Added task visibility support to `pk/task.go`:
  - [x] `Hidden()` method to mark tasks hidden from CLI
  - [x] `IsHidden()` method for filtering in CLI
- [x] Integrated shim generation into `pk/config.go`:
  - [x] Called in `Config.Execute()` after planning, before execution
  - [x] Generates shims at root and all module directories from
        `plan.ModuleDirectories`
  - [x] Fails fast if generation errors
- [x] Shim features:
  - [x] Correct relative paths to `.pocket` based on directory depth
  - [x] Embedded Go version from `.pocket/go.mod`
  - [x] Embedded checksums for all platforms from go.dev/dl
  - [x] Auto-downloads Go if not found in PATH
  - [x] Verifies checksums during download
  - [x] Sets `TASK_SCOPE` environment variable for execution context

Questions answered:

- [x] Where is shim generation invoked? Answer: In `Config.Execute()` after
      creating the plan but before executing tasks. This ensures shims are
      always fresh and have access to `plan.ModuleDirectories`.

## Phase 3: Bootstrapper

- [x] Created `internal/scaffold/` package with:
  - [x] `templates/config.go.tmpl` - user-editable config (one-time, never
        overwritten)
  - [x] `templates/main.go.tmpl` - auto-generated entry point (always
        regenerated)
  - [x] `templates/gitignore.tmpl` - ignores `bin/` and `tools/`
  - [x] `scaffold.go` - `GenerateAll()` and `RegenerateMain()` functions
- [x] Created `cmd/pocket/main.go` bootstrapper CLI:
  - [x] `pocket init` command creates `.pocket/` directory
  - [x] Initializes Go module (`go mod init pocket`)
  - [x] Adds pocket dependency
        (`go get github.com/fredrikaverpil/pocket@latest`)
  - [x] Generates scaffold files via `scaffold.GenerateAll()`
  - [x] Runs `go mod tidy`
  - [x] Generates platform-specific shims (reuses `internal/shim`)
  - [x] Platform detection for shim types (POSIX on Unix, cmd/ps1 on Windows)
  - [x] `pocket help` displays usage

Usage (once Pocket v2 is published):

```
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

Still TODO:

- [ ] End-to-end test once `pk` package is published to GitHub

## Phase 4: Output and error handling

Completed:

- [x] Output abstraction (`pk/output.go`)
  - [x] `Output` struct with `Stdout`/`Stderr` writers
  - [x] `StdOutput()` returns standard output writers
  - [x] `bufferedOutput` captures per-goroutine output for parallel execution
  - [x] `lockedWriter` for thread-safe concurrent writes
- [x] Context-based output (`pk/context.go`)
  - [x] `WithOutput()`/`OutputFromContext()` for propagating output through context
- [x] Execution with graceful shutdown (`pk/exec.go`)
  - [x] `Exec()` uses context output instead of `os.Stdout/Stderr`
  - [x] `WaitDelay = 5s` constant for graceful shutdown
  - [x] Platform-specific graceful shutdown (SIGINT on Unix, no-op on Windows)
- [x] Buffered parallel output (`pk/composition.go`)
  - [x] Single task runs without buffering
  - [x] Multiple tasks: each gets `bufferedOutput`, flushes on completion
  - [x] `errgroup.WithContext` for cooperative cancellation (fail-fast)
  - [x] First-to-complete flushes first (no interleaving)
  - [x] Deduplication handled by `Task.run()` (single source of truth)
- [x] Signal handling (`pk/cli.go`)
  - [x] `signal.NotifyContext` catches SIGINT/SIGTERM
  - [x] Context cancellation propagates to all running tasks
- [x] Task deduplication (a task only runs once per invocation)
  - [x] Global dedup by task pointer identity (same `*Task` runs once)
  - [x] `WithForceRun()` option to bypass deduplication
  - [x] Thread-safe with `sync.Mutex`
  - [x] Unit tests in `pk/context_test.go`, `pk/task_test.go`,
        `pk/paths_test.go`
- [x] Dependency: `golang.org/x/sync` added for `errgroup`

## Phase 5: CLI argument parsing and help

Completed:

- [x] CLI argument parsing with flag package
- [x] `-v` flag for verbose mode (changed in Phase 6)
- [x] `--version` flag for version (added in Phase 6)
- [x] `-h` flag for help
- [x] `-json` flag for JSON output
- [x] Help text generation from plan (lists visible tasks)
- [x] Implemented `./pok plan` builtin command
  - [x] Text output with tree visualization
  - [x] JSON output with `-json` flag
  - [x] Shows composition tree (Serial/Parallel/PathFilter)
  - [x] Shows task paths and shim generation directories
- [x] Exported `Plan` and `PathInfo` as public API for introspection
- [x] CLI code organized in `pk/cli.go` (can access internal types)

Architecture decisions:

- `plan` is a **CLI builtin** (not a task) - it's a meta-operation for
  introspection
- CLI lives in `pk/cli.go` to access internal composition types
- `.pocket/go.mod` maintained for dogfooding with replace directive
- Helper functions (printHelp, printPlan, printTree) are unexported
- Plan exposed with clear documentation that composition types are internal

Additional cleanup:

- [x] Removed temporary `main.go` - now dogfooding with `./pok` shim

## Phase 6: Implement initial tools and tasks

See pocket-v1's tasks and tools. Let's now start implementing them but here in
pocket v2.

- [x] Discuss how we will store tools and packages in pocket. In registry/tasks,
      registry/tools packages (or just tasks, tools packages)? Compare with how
      this was done in pocket-v1.
      **Decision:** Root-level `tools/` and `tasks/` packages (e.g.,
      `tools/golangcilint/`, `tasks/golang/`).
- [x] First tool is implemented; I propose golangci-lint.
      **Done:** `tools/golangcilint/golangcilint.go` with `Install` task,
      versioned installation to `.pocket/tools/go/<pkg>/<version>/bin/`,
      symlinked to `.pocket/bin/`.
- [x] First task, which uses previous tool, is implemented; I propose a "golang"
      task would encapsulate several Go-specific tasks/tools. We can then
      dogfood ./pok go-lint for example.
      **Done:** `tasks/golang/` with `Lint` task, `Tasks()` bundle function.
- [x] Discussion, review; what went well, what works, what do we have to go back
      and fix/simplify?
      **Done:** Simplified execution API from `Do`+`Run`+`Exec` to just `Do`+`Exec`.
      Removed unused `RunCommand`, `RunCommandString`, `DetectByFile`.
      Created `REFERENCE.md` documenting public API.
- [ ] Implement the GitHub workflows task along with the ci matrix capability.
- [ ] Tools installations should be tested, so that we know each tool can
      install fine and symlink its binary.

Additional completed:

- [x] Added `-v` verbose flag (changed from version to verbose)
- [x] Added `--version` flag for version display
- [x] Verbose mode threads through context, affects `Exec()` and `InstallGo()`
- [x] Task naming: `DefineTask()` for Runnable-based tasks (avoids conflict with
      `Task` type)

## Phase 7: Auto-detection of modules

- [ ] Let's implement a way to autodetect e.g. `go.mod` files, and run the
      golang task(s) in each such module. This would replace the need for
      `WithIncludePath`.
- [ ] When auto-detecting, we might want to exclude both paths and tasks. See
      how we did this in pocket-v1, since I think that worked pretty well there.
      How can that fit into pocket v2?

## Phase 7: Mid-review

Where are we currently at? What works great, what works less great?

- [ ] From a DX perspective; is the API surface easy to understand?
- [ ] From a files/packages perspective; is the git project laid out well?
- [ ] From a Go ideomatic view; is the project following Go ideoms, leveraging
      std lib, easy to understand?

Specifically targeted changes:

- [ ] Can the `./pok --version` be generated from Git tag? (currently, no tag
      exists and that needs to work too)

## Phase 8: Documentation

- [ ] Add README.md; compare against pocket-v1 so we don't forget anything
      important
  - [ ] Inspect the pk package; this is our public API. Add documentation around
        all public symbols that are intended to be used by users.
- [ ] Add ARCHTECTURE.md ; this document is targeting contributors and
      maintainers, explaining how Pocket works.

## Phase "Wrapup" (last phase)

- [ ] Analyze Pocket
  - [ ] DX - do we have good developer experience?
  - [ ] Long-term maintainability, is the codebase simple and ideomatic to Go?
  - [ ] Compare with pocket-v1; which areas have been improved, which areas were
        done better/simpler in pocket-v1?
- [ ] Go through each go file and add an equivalent \_test.go file, for adding
      unit tests.
- [ ] Keep Windows in mind. We need to support Windows.
- [x] Cleanup:
  - [x] Renamed shim variables for clarity: `SHIM_DIR`, `POCKET_DIR`, `TASK_SCOPE`
  - [x] Fixed shim path resolution (works when invoked from any directory)
- [ ] Analyze pocket-v1; are we missing any features in this rewrite?
- [ ] Implement all tasks and tools from pocket v1
