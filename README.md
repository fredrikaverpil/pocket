# pocket

A cross-platform build system. Write tasks, control execution order, and let
pocket handle tool installation.

> [!NOTE]
>
> Pocket is written in Go, but you don't need Go installed to use it. The
> `./pok` shim (`pok.ps1` on Windows) automatically downloads Go to `.pocket/`
> if needed.

> [!WARNING]
>
> Under heavy development. Breaking changes will occur until the initial
> release.

## Features

- **Cross-platform**: No Makefiles - works on Windows, macOS, and Linux
- **Task management**: Define tasks with `Serial()` and `Parallel()` execution
- **Tool management**: Downloads and caches tools in `.pocket/`
- **Simple invocation**: Just `./pok` or `./pok -h` to list tasks

See [how Pocket compares](docs/comparison.md) to Mage, Sage, Task, Dagger, and
other build tools.

### Todos

- [ ] Review config and tasks/tools API UX and improve it, make it simpler.
- [ ] Make as much parts of Pocket as possible non-exported, so we don't have to
  worry users starts using things we cannot refactor later.
  - [ ] Move as much as possible into an internal folder, that export parts of
    Pocket that is only intended for the internals powering Pocket.
- [ ] Diagrams/tables showing how Pocket works in different senses. I would like
  to explain;
  - [ ] Project config driven command execution
  - [ ] Path resolver behavior
  - [ ] What a "Runnable" is, and how it gets executed
    - [ ] Concurrency
    - [ ] Errors

## Quickstart

### Bootstrap

This is the only step that requires Go installed. Run in your project root:

```bash
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

This creates `.pocket/` (build module) and `./pok` (wrapper script).

### Your first task

Edit `.pocket/config.go`:

```go
package main

import "github.com/fredrikaverpil/pocket"

var Config = pocket.Config{
    // AutoRun: tasks that run on ./pok (no arguments).
    ManualRun: []pocket.Runnable{helloTask},
}

// helloAction is defined separately from the task constructor.
func helloAction(_ context.Context, tc *pocket.TaskContext) error {
    tc.Out.Println("Hello from pocket!")
    return nil
}

// helloTask uses NewTask to create a task with required fields.
var helloTask = pocket.NewTask("hello", "say hello", helloAction)
```

```bash
./pok -h      # list tasks
./pok hello   # run specific task
```

> [!NOTE]
>
> Use `ManualRun` for tasks that should only run when explicitly called (like
> `deploy`). See [Auto-run vs manual tasks](#auto-run-vs-manual-tasks).

> [!TIP]
>
> **Shim customization:** If you don't like `./pok`, configure a different name
> in `.pocket/config.go`:
>
> ```go
> Shim: &pocket.ShimConfig{Name: "build"}  // creates ./build instead
> ```
>
> Or add a shell alias: `alias pok='./pok'`

### Execution order

When using `AutoRun` instead of `ManualRun`, you can run all specified tasks
directly on invoking `./pok`.

Use `Serial()` and `Parallel()` to control how tasks run:

```go
var Config = pocket.Config{
    AutoRun: pocket.Serial(
        formatTask,          // first
        pocket.Parallel(     // then these in parallel
            lintTask,
            testTask,
        ),
        buildTask,           // last
    ),
}
```

### Tool management

Pocket downloads and caches tools in `.pocket/tools/`. Tools implement
`Runnable`, so they compose with `Serial`/`Parallel` just like tasks.

**Using a bundled tool:**

```go
import "github.com/fredrikaverpil/pocket/tools/golangcilint"

func lintAction(ctx context.Context, tc *pocket.TaskContext) error {
    return golangcilint.Tool.Exec(ctx, tc, "run", "./...")
}

var lintTask = pocket.NewTask("lint", "run linter", lintAction)
```

**Creating a custom tool:**

```go
// .pocket/tools.go
package main

import (
    "context"
    "github.com/fredrikaverpil/pocket"
)

const myToolVersion = "1.0.0"

var myTool = pocket.NewTool("mytool", myToolVersion, installMyTool)

func installMyTool(ctx context.Context, tc *pocket.TaskContext) error {
    // Option 1: Download binary from URL
    return pocket.DownloadBinary(ctx, tc, "https://example.com/mytool.tar.gz", pocket.DownloadOpts{
        DestDir: pocket.FromToolsDir("mytool", myToolVersion),
        Format:  "tar.gz",
        Symlink: true,
    })
    // Option 2: Go install
    // return pocket.GoInstall(ctx, tc, "example.com/mytool", myToolVersion)
}

// Use in a task action
func myAction(ctx context.Context, tc *pocket.TaskContext) error {
    return myTool.Exec(ctx, tc, "--flag", "arg")
}
```

Tools are installed on first use and cached for subsequent runs.

## Configuration

### Tasks with options

Tasks can accept options configurable both at project level and via CLI flags.
Define a struct for your options and use `WithOptions()`:

```go
// DeployOptions configures the deploy task.
type DeployOptions struct {
    Env    string `usage:"target environment"`
    DryRun bool   `usage:"print actions without executing"`
}

// deployAction is the task implementation.
func deployAction(_ context.Context, tc *pocket.TaskContext) error {
    opts := pocket.GetOptions[DeployOptions](tc)  // defaults merged with CLI flags
    if opts.DryRun {
        tc.Out.Printf("Would deploy to %s\n", opts.Env)
        return nil
    }
    tc.Out.Printf("Deploying to %s...\n", opts.Env)
    return nil
}

// DeployTask returns a task that deploys to an environment.
func DeployTask() *pocket.Task {
    return pocket.NewTask("deploy", "deploy to environment", deployAction)
}

var Config = pocket.Config{
    // Deploy is a manual task - only runs with ./pok deploy
    ManualRun: []pocket.Runnable{
        DeployTask().WithOptions(DeployOptions{Env: "staging"}),
    },
}
```

CLI flags are derived from field names (`DryRun` → `-dry-run`). Supported types:
`string`, `int`, `bool`. Use `` `usage:"description"` `` for help text.

```bash
./pok deploy                        # uses project default (staging)
./pok deploy -env=prod -dry-run     # override at runtime
./pok deploy -h                     # show task-specific help
```

### Path filtering

For monorepos, use `Paths()` to control where tasks are visible:

```go
var Config = pocket.Config{
    AutoRun: pocket.Serial(
        rootTask,                                  // visible at git root only
        pocket.Paths(apiTask).In("services/api"),  // visible in services/api/
        pocket.Paths(webTask).In("services/web"),  // visible in services/web/
    ),
}
```

```go
// Multiple directories
pocket.Paths(myTask).In("proj1", "proj2")

// Regex patterns
pocket.Paths(myTask).In("services/.*")

// Exclude directories
pocket.Paths(myTask).In("services/.*").Except("services/legacy")
```

Running `./pok` from a subdirectory shows only tasks relevant to that directory.

You can also run multiple tasks in the same path:

```go
pocket.Paths(myTask1, myTask2, myTask3).In(pocket.FromGitRoot("docs"))
```

### Bundled task packages

Pocket includes ready-made task packages for common languages:

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/fredrikaverpil/pocket/tasks/golang"
    "github.com/fredrikaverpil/pocket/tasks/python"
    "github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
    AutoRun: pocket.Serial(
        golang.Tasks(),
        python.Tasks(),
        markdown.Tasks(),
    ),
}
```

Each package internally defines whether its tasks run in parallel or serial
(e.g., format before lint).

Task arguments can be overridden at runtime:

```bash
./pok go-test -skip-race        # skip race detection
./pok go-lint -lint-config=.golangci.yml
```

Or set project-level defaults using functional options:

```go
var Config = pocket.Config{
    AutoRun: pocket.Serial(
        golang.Tasks(
            golang.WithFormat(golang.FormatOptions{LintConfig: ".golangci.yml"}),
            golang.WithTest(golang.TestOptions{SkipRace: true}),
        ),
        python.Tasks(
            python.WithFormat(python.FormatOptions{RuffConfig: "ruff.toml"}),
        ),
    ),
}
```

Or construct individual tasks with options:

```go
var Config = pocket.Config{
    AutoRun: pocket.Serial(
        golang.FormatTask().WithOptions(golang.FormatOptions{LintConfig: ".golangci.yml"}),
        golang.LintTask(),
        golang.TestTask().WithOptions(golang.TestOptions{SkipRace: true}),
    ),
}
```

### Auto-detection

For projects with multiple modules, use `Paths().DetectBy()` with the package's
`Detect()` function to automatically find directories containing relevant files:

```go
var Config = pocket.Config{
    AutoRun: pocket.Serial(
        pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()),    // finds all go.mod directories
        pocket.Paths(python.Tasks()).DetectBy(python.Detect()),    // finds all pyproject.toml directories
    ),
}
```

This is opinionated: it runs the same tasks across all detected directories.
Combine with path filtering for more control:

```go
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).Except("vendor", "testdata")
```

### Skipping tasks

Use `Skip()` to exclude specific tasks from a task group. This works with any
task constructor that returns `*pocket.Task`, including bundled packages like
`golang.TestTask()`.

Skip a task everywhere (global skip):

```go
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).Skip(golang.TestTask())
```

Skip only in specific directories (path-specific skip):

```go
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).Skip(golang.TestTask(), "docs")
```

Skip in multiple directories:

```go
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).Skip(golang.TestTask(), "docs", "examples")
```

Skip multiple tasks by chaining:

```go
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).Skip(golang.TestTask()).Skip(golang.VulncheckTask())
```

### Auto-run vs manual tasks

Pocket separates tasks into two categories:

- **AutoRun**: Tasks that execute when running `./pok` without arguments
- **ManualRun**: Tasks that only run when explicitly invoked with
  `./pok <taskname>`

```go
var Config = pocket.Config{
    // AutoRun: these execute on ./pok
    AutoRun: pocket.Serial(
        pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()),
        pocket.Paths(python.Tasks()).DetectBy(python.Detect()),
    ),
    // ManualRun: these require ./pok <taskname>
    ManualRun: []pocket.Runnable{
        deployTask,
        pocket.Paths(benchmarkTask).In("services/api"),
    },
}
```

The `./pok -h` output shows tasks in separate sections:

```
Tasks:
  go-format      format Go code
  go-lint        run linter

Manual Tasks:
  deploy         deploy to environment
  benchmark      run benchmarks

Builtin tasks:
  clean          remove tools and bin
  generate       regenerate files
```

Both `AutoRun` and `ManualRun` support the same wrappers (`Paths()`, `Serial()`,
`Parallel()`, etc.).

## Reference

### Task creation

```go
// Create a task with required fields (name, usage, action)
task := pocket.NewTask("my-task", "description", myAction)

// Add optional configuration via chaining
task.WithOptions(MyOptions{})  // CLI-configurable options
task.AsHidden()                // hide from CLI help
task.AsBuiltin()               // mark as built-in task
```

### Task composition

```go
// Run tasks in sequence
pocket.Serial(formatTask, lintTask, testTask)

// Run tasks in parallel
pocket.Parallel(lintTask, testTask)

// Mixed serial and parallel
pocket.Serial(formatTask, pocket.Parallel(lintTask, testTask))
```

### Path filtering and detection

```go
// Wrap with detection using a package's Detect() function
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect())

// Custom detection function
pocket.Paths(myTasks).DetectBy(func() []string {
    return pocket.DetectByFile("go.mod")
})

// Explicit directories
pocket.Paths(myTasks).In("proj1", "proj2")

// Combine detection with filtering
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).Except("vendor")
```

### Config structure

```go
var Config = pocket.Config{
    // AutoRun: tasks that run on ./pok (no arguments)
    AutoRun: pocket.Serial(
        pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()),
        pocket.Paths(python.Tasks()).DetectBy(python.Detect()),
    ),

    // ManualRun: tasks that only run with ./pok <taskname>
    ManualRun: []pocket.Runnable{
        deployTask,
        pocket.Paths(benchmarkTask).In("services/api"),
    },

    // Shim: configure generated wrapper scripts
    Shim: &pocket.ShimConfig{
        Name:       "pok",   // base name (default: "pok")
        Posix:      true,    // ./pok (bash)
        Windows:    true,    // pok.cmd
        PowerShell: true,    // pok.ps1
    },

    // SkipGitDiff: disable git diff check after running tasks
    SkipGitDiff: false,
}
```

### Convenience functions

```go
// Path helpers
pocket.GitRoot()              // git repository root
pocket.FromGitRoot("subdir")  // path relative to git root
pocket.FromPocketDir("file")  // path relative to .pocket/
pocket.FromBinDir("tool")     // path relative to .pocket/bin/
pocket.BinaryName("mytool")   // appends .exe on Windows

// Output (use these instead of fmt.Printf/Println in task actions)
tc.Out.Printf("Hello %s\n", name)  // writes to stdout, buffered for parallel tasks
tc.Out.Println("Done!")            // writes to stdout with newline
tc.Out.Stdout                      // io.Writer for stdout
tc.Out.Stderr                      // io.Writer for stderr

// Execution (in task actions, use tc.Command for proper output handling)
cmd := tc.Command(ctx, "go", "build", "./...")  // output wired to tc.Out
cmd.Run()

// Tools (auto-installed on first use, output wired to tc.Out)
golangcilint.Tool.Exec(ctx, tc, "run", "./...")

// Detection (for Detectable interface)
pocket.DetectByFile("go.mod")       // dirs containing file
pocket.DetectByExtension(".lua")    // dirs containing extension

// Options
opts := pocket.GetOptions[MyOptions](tc)  // retrieve typed options in action
```

> [!NOTE]
>
> Use `tc.Out.Printf`/`tc.Out.Println` instead of `fmt.Printf`/`fmt.Println` in
> task actions. This ensures output is properly buffered when tasks run in
> parallel, preventing interleaved output. Single tasks and serial execution
> still get real-time output.

### Windows support

Pocket auto-detects your platform during bootstrap:

- **Unix/macOS/WSL**: Creates `./pok` (bash script)
- **Windows**: Creates `pok.cmd` and `pok.ps1`

Add additional shim types in `.pocket/config.go`:

```go
var Config = pocket.Config{
    Shim: &pocket.ShimConfig{
        Posix:      true,   // ./pok
        Windows:    true,   // pok.cmd
        PowerShell: true,   // pok.ps1
    },
}
```

## Acknowledgements

- [einride/sage](https://github.com/einride/sage) - Inspiration for the task
  system and tool management approach
