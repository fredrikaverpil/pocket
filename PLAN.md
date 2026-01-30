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
  - [x] `WithOutput()`/`OutputFromContext()` for propagating output through
        context
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
  - [x] Output behavior:
    - `./pok <task>` - Always realtime streaming (no buffering)
    - `./pok` (full tree) - Tasks in `Parallel()` use buffering; tasks in
      `Serial()` stream in realtime
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
      this was done in pocket-v1. **Decision:** Root-level `tools/` and `tasks/`
      packages (e.g., `tools/golangcilint/`, `tasks/golang/`).
- [x] First tool is implemented; I propose golangci-lint. **Done:**
      `tools/golangcilint/golangcilint.go` with `Install` task, versioned
      installation to `.pocket/tools/go/<pkg>/<version>/bin/`, symlinked to
      `.pocket/bin/`.
- [x] First task, which uses previous tool, is implemented; I propose a "golang"
      task would encapsulate several Go-specific tasks/tools. We can then
      dogfood ./pok go-lint for example. **Done:** `tasks/golang/` with `Lint`
      task, `Tasks()` bundle function.
- [x] Discussion, review; what went well, what works, what do we have to go back
      and fix/simplify? **Done:** Simplified execution API from
      `Do`+`Run`+`Exec` to just `Do`+`Exec`. Removed unused `RunCommand`,
      `RunCommandString`, `DetectByFile`. Created `REFERENCE.md` documenting
      public API.
- [ ] Implement the GitHub workflows task along with the ci matrix capability.
- [ ] Tools installations should be tested, so that we know each tool can
      install fine and symlink its binary.

Additional completed:

- [x] Added `-v` verbose flag (changed from version to verbose)
- [x] Added `--version` flag for version display
- [x] Verbose mode threads through context, affects `Exec()` and `InstallGo()`
- [x] Task naming: `NewTask()` accepts `Runnable` body (use `Do()` to wrap
      functions)

## Phase 7: Auto-detection of modules

- [x] Let's implement a way to autodetect e.g. `go.mod` files, and run the
      golang task(s) in each such module. This would replace the need for
      `WithIncludePath`. **Done:** Implemented `DetectFunc`, `DetectByFile()` in
      `pk/detect.go`. Added `WithDetect()` path option. Updated
      `tasks/golang/tasks.go` to use `WithDetect(golang.Detect())`.
- [x] When auto-detecting, we might want to exclude both paths and tasks. See
      how we did this in pocket-v1, since I think that worked pretty well there.
      How can that fit into pocket v2? **Done:** Detection integrates with
      existing `WithExcludePath()`. Also added manual tasks via `Config.Manual`
      field and `(*Task).Manual()` method for tasks that only run when
      explicitly invoked (not on bare `./pok`).

## Phase 8: Builtin WithOptions, WithOverrides...

- [x] Why do we have both pk.WithFlag and xxx.WithXOption ? Do we need both? Or
      could we opt for only defining task flags, and then pass the desired flags
      in the `Config{Auto: {...}}` composition?
- [ ] We recently (in sha `bff236d410dcf9c33a2977ebb81992d2e5a7811d`) introduced
      parsing of pk.WithName so that we appended the value to the task's
      effective name. Is the entire name properly evaluated by e.g. the GitHub
      pocket-matrix workflow logic? Are we sure that task names are treated
      everywhere with this suffix applied?
- [ ] Unless something changed, tasks have both a "name" and an "id". Is this
      really necessary? Can we just opt for a name which we will treat as the
      id, and require that tasks must have unique names? I noticed the
      `./pok plan -json` only outputs the "name" (not the "id"). When would we
      _actually_ need to have a separate ID?
- [x] For GitHub pocket matrix workflow, can we change how this works, so that
      we instead generate the workflow file that will be executed? This makes it
      a lot easier to debug and write tests for the GitHub workflows. Right now,
      the more complex GHA matrix workflow generates jobs on the fly in GHA
      which is difficult to debug. A complex case is the GitHub Matrix in
      ~/code/public/creosote which today does not quite work as `./pok plan` and
      `./pok -h` looks right with task naming (several variants of py-test), but
      where the CI does not use the same task names (suffix missing, from
      pk.WithName): https://github.com/fredrikaverpil/neotest-golang/pull/541
- [ ] Found during investigation: ⚠️ Minor inconsistency found planTask uses an
      IIFE pattern to capture flags:  
       var planTask = func() *Task {  
       fs := flag.NewFlagSet("plan", flag.ContinueOnError)  
       jsonFlag := fs.Bool("json", false, "output as JSON")  
       return NewTask("plan", ..., fs, Do(func(ctx context.Context) error {  
       if *jsonFlag { ... }  
       })).HideHeader()  
       }() While selfUpdateTask uses package-level vars:  
       var (  
       selfUpdateFlags = flag.NewFlagSet("self-update", flag.ContinueOnError)  
       selfUpdateForce = selfUpdateFlags.Bool("force", false, "...")  
       )  
       var selfUpdateTask = NewTask("self-update", ..., selfUpdateFlags, ...)
      The IIFE pattern is valid Go but inconsistent. Would you like me to align
      planTask with the selfUpdateTask pattern for consistency?
- [ ] The generated GHA matrix workflow contains a separate git diff job. This
      we can remove, as each Pocket task runs with `./pok -g` and the -g flag
      instructs Pocket to run the git diff task after the given task that runs.
      So right now, we don't need the "Check for uncommitted changes" job.
- [x] Is not all tasks' TaskInfo part of the plan, and doesn't the plan JSON get
      derived from the plan?  
       It seeks like you are manually registering what should be going into a
      special taskJSON map or something when the  
       JSON should be generated purely from data in the plan.

## Phase 9: Mid-review

Where are we currently at? What works great, what works less great?

- [ ] From a DX perspective; is the API surface easy to understand?
- [ ] From a files/packages perspective; is the git project laid out well?
- [ ] From a Go ideomatic view; is the project following Go ideoms, leveraging
      std lib, easy to understand?

Specifically targeted changes:

- [ ] Can the `./pok --version` be generated from Git tag? (currently, no tag
      exists and that needs to work too)

## Phase 10: Documentation

- [x] Add README.md; professional high-level overview with quickstart
- [x] Create detailed guides in `docs/`:
  - [x] `docs/tasks-and-tools.md`
  - [x] `docs/composition-and-paths.md`
- [x] Add ARCHTECTURE.md; targeting contributors, explaining internals
- [x] Add REFERENCE.md; technical API reference for `pk` package
- [x] Taskify built-in maintenance commands (`generate`, `update`)
- [x] Professionalize output with `pk.Printf`, `pk.Println`, `pk.Errorf`
- [x] Implement `git-diff` as a standard task in `tasks/git/diff.go`
- [x] Refactor version logic into `pk/version.go`
- [x] Unified and robust signal handling in `pk/cli.go`

## Phase 11: Fixups

- [x] Add configurable directory skip options to Config:
  - [x] `SkipDirs []string` - directories to skip during filesystem walk
  - [x] `DefaultSkipDirs` - sensible defaults (vendor, node_modules, dist,
        \_\_pycache\_\_, venv)
  - [x] `IncludeHiddenDirs bool` - opt-in to include hidden directories
  - [x] Tests in `pk/filesystem_test.go`
- [x] Configurable shim generation in Config:
  - [x] `Shims *ShimConfig` - controls which shims are generated
  - [x] `DefaultShimConfig()` - POSIX only (default when nil)
  - [x] `AllShimsConfig()` - all three shims enabled
  - [x] Scaffold template shows explicit config with all shims enabled
  - [x] Plan stores and exposes ShimConfig for generate task
- [x] Tools like golangci-lint and stylua comes with config files. Let's discuss
      a way to have a task use these, or fall back to e.g. a repo root config
      file. You can see what we did in pocket-v1 for this. (Already handled)
- [x] Pocket-v1 had a github-workflows task which copied very common github
      workflows into place. Let's add that back in.
  - [x] Created `tasks/github/` package with `Workflows` task
  - [x] Embedded templates for pocket, pr, release, stale workflows
  - [x] Skip flags for selective generation, platforms flag for customization
  - [x] Silent by default, verbose with `-v`
- [x] Pocket-v1 also had a much more complex matrix-based github workflow which
      hooked into the plan and generated concurrent workflow jobs. Let's analyze
      how we can get that back into pocket v2!
- [ ] Verify that all documentation is up to date. We need to mention all public
      API methods and configurable parts.
- [ ] We had a false example golang.WithRace() in the documentation that doesn't
      exist. Should we:  
       1. Remove the false example - Keep golang using the -race flag as-is  
       2. Implement golang.WithTestRace() - Add the option for consistency with
      python
- [x] Task deduplication uses `taskID` struct as map key (not string
      concatenation). Dedup key is `(effectiveName, path)` where effectiveName
      includes any suffix from `pk.WithName`. Global tasks dedupe by base name
      only. Comprehensive tests exist in `pk/task_test.go`.
- [ ] We have introduced a number of ways to pass options to tasks. We have
      pk.WithOptions which is the main wrapper you need to use when passing such
      options. We have then formalized that we have golang.Tasks() or
      python.Tasks() for example, if not invoking an individual task (like
      python.Test or golang.Test). Generic options are introduced as pk-scoped,
      like pk.WithName. But then the package-level can provide their own
      options, like python.WithVersion. This notion is important to get across
      as a pattern in our documentation and also make sure we do this
      consistently throughout the existing codebase today. I also think that the
      task-level option functions must be named like
      <package>.With<Task><Option>, like golang.WithRace is not okay but
      golang.WithTestRace is good. This way we can type in our IDEs and get
      completion for all task options by e.g. starting to write golang.WithTest
      (which would show all options for the test task).
- [ ] I have a feeling that the reference.md documentation might be out of sync.
      We might have exported/public symbols we haven't added in here. Both for
      the pk.XXX API and for the Pocket-bundled tasks.
- [ ] We are in documentation sometimes distinguishing "end-users" and
      "task/tool authors". These are for the most part going to be the same
      person. The difference is really that one person might build tasks/tools
      and then reuse them in multiple projects. We should make that more clear
      in the different markdown documentation files we have.
- [ ] We keep documentation on reference, architecture, user guides. Is there
      overlap and/or risk of updating one but not the other and then cause a
      drift where the documentation is not aligned? Can we consolidate the
      documentation better, and here I'm actually thinking alot about LLMs
      finding one place where documentation needs updating and doesn't see the
      other file which also needs updating.
- [x] The Matrix configuration for github workflows using the matrix-pocket
      workflow has weird UX. Is this well documented somewhere? **Resolved:**
      `github.Tasks()` now includes the matrix task internally. Users configure
      it via `pk.WithFlag` and `pk.WithContextValue` in `pk.WithOptions`,
      eliminating the separate `Manual` entry.
- [ ] I'm worried we have implemented too many ways to identify/classify and
      reference a given task in Pocket core (pk package). Like for example, we
      have Task, taskID, taskInstance and we sometimes refer to the task name
      (string type). All of which are used to decide whether to do thinks to
      task(s). Is there some way we can consolidate our code around
      identifying/classifying tasks in one way? Or is this not a problem? This
      all revolves around me wanting to make sure there is one "canonical" way
      to identify, classify and refer to tasks.
- [ ] Add zensical as tool, with flags -serve and -build.

## Phase 12: Wrapup & Polish

- [ ] Analyze Pocket
  - [ ] DX - do we have good developer experience?
  - [ ] Long-term maintainability, is the codebase simple and ideomatic to Go?
  - [ ] Compare with pocket-v1; which areas have been improved, which areas were
        done better/simpler in pocket-v1?
- [ ] Tag the first release (v0.1.0) and verify version reporting
- [ ] End-to-end test of the bootstrapper (`pocket init`)
- [ ] Unit test coverage:
  - [ ] `pk/builtins.go`
  - [ ] `pk/install.go`
- [ ] Implementation of remaining tasks/tools from v1:
  - [ ] GitHub Workflows + Matrix generation
  - [ ] Pre-commit hook integration
