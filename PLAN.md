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

- [x] Create a temporary "main.go" that runs (will be replaced later by the
      shim)
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
  - [x] `gomod.go` - reads Go version from go.mod using Go 1.25+ `strings.SplitSeq`
  - [x] `checksums.go` - fetches SHA256 checksums from go.dev/dl API
  - [x] `shim.go` - main generation logic with `GenerateShims()` function
  - [x] `templates/` - pok.sh.tmpl (POSIX), pok.cmd.tmpl (Windows), pok.ps1.tmpl (PowerShell)
- [x] Added task visibility support to `pk/task.go`:
  - [x] `Hidden()` method to mark tasks hidden from CLI
  - [x] `IsHidden()` method for filtering in CLI
- [x] Integrated shim generation into `pk/config.go`:
  - [x] Called in `Config.Execute()` after planning, before execution
  - [x] Generates shims at root and all module directories from `plan.ModuleDirectories`
  - [x] Fails fast if generation errors
- [x] Shim features:
  - [x] Correct relative paths to `.pocket` based on directory depth
  - [x] Embedded Go version from `.pocket/go.mod`
  - [x] Embedded checksums for all platforms from go.dev/dl
  - [x] Auto-downloads Go if not found in PATH
  - [x] Verifies checksums during download
  - [x] Sets `POK_CONTEXT` environment variable for execution context

Questions answered:

- [x] Where is shim generation invoked? Answer: In `Config.Execute()` after creating the plan but before executing tasks. This ensures shims are always fresh and have access to `plan.ModuleDirectories`.

## Phase 3: Output and error handling

Deferred from Phase 2 implementation plan:
- [ ] Output abstraction (`pk/output.go`)
- [ ] Execution context (`pk/exec.go`)
- [ ] Buffered parallel output
  - [ ] When a single task runs, run it without buffered output?
  - [ ] Waitgroups with errors?
  - [ ] Collect errors?
  - [ ] Fail fast; signal that something failed, abort other go-routines
  - [ ] First task to complete outputs the results

Completed:
- [x] Implement pk.Parallel (basic version)
- [x] Task deduplication (a task only runs once per invocation)
  - [x] Global dedup by task pointer identity (same `*Task` runs once)
  - [x] `WithForceRun()` option to bypass deduplication
  - [x] Thread-safe with `sync.Mutex`
  - [x] Unit tests in `pk/context_test.go`, `pk/task_test.go`, `pk/paths_test.go`

## Phase 4: CLI argument parsing and help

Completed:
- [x] CLI argument parsing with flag package
- [x] `-v` flag for version
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
- `plan` is a **CLI builtin** (not a task) - it's a meta-operation for introspection
- CLI lives in `pk/cli.go` to access internal composition types
- `.pocket/go.mod` maintained for dogfooding with replace directive
- Helper functions (printHelp, printPlan, printTree) are unexported
- Plan exposed with clear documentation that composition types are internal

Still TODO:
- [ ] CLI-invocable tasks (run specific tasks like `./pok lint`)
- [ ] Task filtering (run tasks by name from CLI)

## Phase 5: Tasks and tools package structures

- [ ] Discuss how we will store tools and packages in pocket. In registry/tasks,
      registry/tools packages (or just tasks, tools packages)?
- [ ] Tools installations should be tested.

## Phase 6:

TBD...

## Phase TBD (last phase)

- [ ] Go through each go file and add an equivalent \_test.go file, for adding
      unit tests.
- [ ] Keep Windows in mind. We need to support Windows.
