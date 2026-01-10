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
- **Simple invocation**: Just `./pok <task>` or `./pok -h` to list tasks

### Todos

- [x] Prevent accidental use of `defaults` in a task's `Action` function; must
      use the helper `GetOptions`. (Solved by separating action functions from
      task constructors - no closure capture possible.)
- [x] It is difficult to just read the source code today, and get a grasp on how
      to create tasks and task groups. Tasks and task groups could use `New`
      functions that would help users with creation? (Solved with `NewTask()`
      constructor and chainable methods.)
- [ ] `./pok clean`; to clean `./pocket/tools`, `./pocket/bin`
- [ ] Ability to define tasks which run on ./pok and tasks which must be run
      only explicitly with `./pok <taskname>`. Such "explicit tasks" should
      appear in their own section in `./pok -h`. Default behavior is to not
      auto-run.
- [ ] Make as much parts of Pocket as possible non-exported, so we don't have to
      worry users starts using things we cannot refactor later.
  - [ ] Move as much as possible into an internal folder, that export parts of
        Pocket that is only intended for the internals powering Pocket.

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

import (
    "context"

    "github.com/fredrikaverpil/pocket"
)

var Config = pocket.Config{
    Run: helloTask,
}

// helloAction is defined separately from the task constructor.
func helloAction(ctx context.Context, rc *pocket.RunContext) error {
    pocket.Println(ctx, "Hello from pocket!")
    return nil
}

// helloTask uses NewTask to create a task with required fields.
var helloTask = pocket.NewTask("hello", "say hello", helloAction)
```

```bash
./pok -h      # list tasks
./pok hello   # run specific task
./pok         # run all tasks
```

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

Use `Serial()` and `Parallel()` to control how tasks run:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
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

Pocket can download and cache tools in `.pocket/tools/`, so you don't need to
rely on CI to install them. Here's a task that uses golangci-lint:

```go
import "github.com/fredrikaverpil/pocket/tools/golangcilint"

func lintAction(ctx context.Context, rc *pocket.RunContext) error {
    return golangcilint.Run(ctx, "run", "./...")
}

var lintTask = pocket.NewTask("lint", "run linter", lintAction)
```

Tools are automatically downloaded on first use and cached for subsequent runs.

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
func deployAction(ctx context.Context, rc *pocket.RunContext) error {
    opts := pocket.GetOptions[DeployOptions](rc)  // defaults merged with CLI flags
    if opts.DryRun {
        pocket.Printf(ctx, "Would deploy to %s\n", opts.Env)
        return nil
    }
    pocket.Printf(ctx, "Deploying to %s...\n", opts.Env)
    return nil
}

// DeployTask returns a task that deploys to an environment.
func DeployTask() *pocket.Task {
    return pocket.NewTask("deploy", "deploy to environment", deployAction)
}

var Config = pocket.Config{
    Run: DeployTask().WithOptions(DeployOptions{Env: "staging"}),  // project defaults
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

For monorepos, use `Paths()` to control where `pok` shims are generated:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
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
    Run: pocket.Serial(
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
    Run: pocket.Serial(
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
    Run: pocket.Serial(
        golang.FormatTask().WithOptions(golang.FormatOptions{LintConfig: ".golangci.yml"}),
        golang.LintTask(),
        golang.TestTask().WithOptions(golang.TestOptions{SkipRace: true}),
    ),
}
```

### Auto-detection

For projects with multiple modules, use `AutoDetect()` to automatically find
directories containing relevant files:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        pocket.AutoDetect(golang.Tasks()),    // finds all go.mod directories
        pocket.AutoDetect(python.Tasks()),    // finds all pyproject.toml directories
    ),
}
```

This is opinionated: it runs the same tasks across all detected directories.
Combine with path filtering for more control:

```go
pocket.AutoDetect(golang.Tasks()).Except("vendor", "testdata")
```

### Skipping tasks

Use `Skip()` to exclude specific tasks from a task group. This works with any
task constructor that returns `*pocket.Task`, including bundled packages like
`golang.TestTask()`.

Skip a task everywhere (global skip):

```go
pocket.AutoDetect(golang.Tasks()).Skip(golang.TestTask())
```

Skip only in specific directories (path-specific skip):

```go
pocket.AutoDetect(golang.Tasks()).Skip(golang.TestTask(), "docs")
```

Skip in multiple directories:

```go
pocket.AutoDetect(golang.Tasks()).Skip(golang.TestTask(), "docs", "examples")
```

Skip multiple tasks by chaining:

```go
pocket.AutoDetect(golang.Tasks()).Skip(golang.TestTask()).Skip(golang.VulncheckTask())
```

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

### Task group creation

```go
// Create a task group (runs tasks in parallel by default)
group := pocket.NewTaskGroup(formatTask, lintTask, testTask)

// Configure execution order
group.RunWith(func(ctx context.Context) error {
    if err := pocket.Serial(formatTask, lintTask).Run(ctx); err != nil {
        return err
    }
    return testTask.Run(ctx)
})

// Configure auto-detection
group.DetectByFile("go.mod")           // detect by marker files
group.DetectByExtension(".py")         // detect by file extensions
group.DetectBy(func() []string { ... }) // custom detection
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
pocket.Printf(ctx, "Hello %s\n", name)  // writes to stdout, buffered for parallel tasks
pocket.Println(ctx, "Done!")            // writes to stdout with newline
pocket.Stdout(ctx)                      // io.Writer for stdout
pocket.Stderr(ctx)                      // io.Writer for stderr

// Execution
cmd := pocket.Command(ctx, "go", "build", "./...")  // PATH includes .pocket/bin/
cmd.Run()

// Detection (for Detectable interface)
pocket.DetectByFile("go.mod")       // dirs containing file
pocket.DetectByExtension(".lua")    // dirs containing extension

// Options
opts := pocket.GetOptions[MyOptions](rc)  // retrieve typed options in action
```

> [!NOTE]
>
> Use `pocket.Printf`/`pocket.Println` instead of `fmt.Printf`/`fmt.Println` in
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
