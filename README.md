# pocket

An opinionated build/task system platform.

> [!NOTE]
>
> Pocket is written in Go, but you don't need Go installed to use it. The
> `./pok` shim (`pk.ps1` on Windows) automatically downloads Go to `.pocket/` if
> needed.

> [!TIP]
>
> If you don't agree with Pocket's opinonated tasks, fork it!\
> You can still leverage both tools and tasks from Pocket, but from your own
> fork; your own platform.

> [!WARNING]
>
> Under heavy development. Breaking changes will occur until the initial release
> happens.
>
> Feedback is welcome!

## Features

- **Cross-platform**: No Makefiles - works on Windows, macOS, and Linux
- **Task management**: Defines tasks like `go-test`, `python-lint`,
  `md-format`...
- **Tool management**: Downloads and caches tools in `.pocket/`
- **Simple invocation**: Just `./pok <task>` or `./pok -h` to list all tasks

## Quickstart

### Bootstrap your project with Pocket

This is the only part of Pocket which requires you to have Go installed (could
be changed in the future). Run the init command in your project root:

```bash
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

This creates:

- `.pocket/` - build module with config and tasks
- `./pok` - wrapper script (or `pok.cmd`/`pok.ps1` on Windows)

### Run tasks

```bash
./pok -h         # list all tasks
./pok            # run all tasks
./pok update     # update pocket to latest version
./pok generate   # regenerate "pok" shim(s)
```

> [!NOTE]
>
> Pocket comes preloaded with useful tasks and tools, but none enabled out of
> the box. You have to explicitly define them in `.pocket/config.go` as desired,
> or make your own. More on that below.

> [!TIP]
>
> **Shim customization:** If you don't like to type out "`./pok`", configure a
> different name in `.pocket/config.go`:
>
> ```go
> Shim: &pocket.ShimConfig{Name: "build"}  // creates ./build instead
> ```
>
> Or add a shell alias: `alias pok='./pok'` if you want to skip having to type
> out the `./` part

## Configuration

Edit `.pocket/config.go` to define your tasks.

### Defining Tasks

```go
import (
    "context"
    "fmt"

    "github.com/fredrikaverpil/pocket"
)

var Config = pocket.Config{
    Run: pocket.Serial(
        lintTask,
        testTask,
        buildTask,
    ),
}

var lintTask = &pocket.Task{
    Name:  "lint",
    Usage: "run linter",
    Action: func(ctx context.Context, opts *pocket.TaskOptions) error {
        cmd := pocket.Command(ctx, "golangci-lint", "run", "./...")
        return cmd.Run()
    },
}

var testTask = &pocket.Task{
    Name:  "test",
    Usage: "run tests",
    Action: func(ctx context.Context, opts *pocket.TaskOptions) error {
        cmd := pocket.Command(ctx, "go", "test", "./...")
        return cmd.Run()
    },
}

var buildTask = &pocket.Task{
    Name:  "build",
    Usage: "build the project",
    Action: func(ctx context.Context, opts *pocket.TaskOptions) error {
        fmt.Println("Building...")
        cmd := pocket.Command(ctx, "go", "build", "./...")
        return cmd.Run()
    },
}
```

Tasks appear in `./pok -h` and run as part of `./pok` (no args).

### Tasks with Arguments

Tasks can declare arguments that users pass via `key=value` syntax:

```go
var deployTask = &pocket.Task{
    Name:  "deploy",
    Usage: "deploy to environment",
    Args: []pocket.ArgDef{
        {Name: "env", Usage: "target environment", Default: "staging"},
    },
    Action: func(ctx context.Context, opts *pocket.TaskOptions) error {
        fmt.Printf("Deploying to %s...\n", opts.Args["env"])
        return nil
    },
}
```

```bash
./pok deploy              # Deploying to staging...
./pok deploy env=prod     # Deploying to prod...
./pok -h deploy           # show task help with arguments
```

### Tasks with Options

For reusable tasks that need configuration, create functions that accept
options:

```go
// Options for the lint task
type LintOptions struct {
    ConfigFile string
    Fix        bool
}

// LintTask returns a configured lint task
func LintTask(opts LintOptions) *pocket.Task {
    return &pocket.Task{
        Name:  "lint",
        Usage: "run linter",
        Action: func(ctx context.Context, _ *pocket.TaskOptions) error {
            cmdArgs := []string{"run"}
            if opts.ConfigFile != "" {
                cmdArgs = append(cmdArgs, "-c", opts.ConfigFile)
            }
            if opts.Fix {
                cmdArgs = append(cmdArgs, "--fix")
            }
            cmdArgs = append(cmdArgs, "./...")
            return pocket.Command(ctx, "golangci-lint", cmdArgs...).Run()
        },
    }
}

// Usage in config.go
var Config = pocket.Config{
    Run: pocket.Serial(
        LintTask(LintOptions{
            ConfigFile: pocket.FromGitRoot(".golangci.yml"),
            Fix:        true,
        }),
    ),
}
```

### Grouping Tasks with Options

For related tasks that share configuration, create a struct that implements
`Runnable` and accepts options:

```go
type MyOptions struct {
    ConfigFile string
    Verbose    bool
}

func MyTasks(opts MyOptions) pocket.Runnable {
    return &myTasks{
        opts:   opts,
        format: formatTask(opts),
        lint:   lintTask(opts),
    }
}

type myTasks struct {
    opts   MyOptions
    format *pocket.Task
    lint   *pocket.Task
}

func (m *myTasks) Run(ctx context.Context) error {
    return pocket.SerialDeps(ctx, m.format, m.lint)
}

func (m *myTasks) Tasks() []*pocket.Task {
    return []*pocket.Task{m.format, m.lint}
}

// Usage
var Config = pocket.Config{
    Run: MyTasks(MyOptions{ConfigFile: ".myconfig.yml", Verbose: true}),
}
```

Options flow from the group to individual tasks - each task function (like
`formatTask(opts)`) receives the options and uses them in its Action closure.

### Path Filtering (Multi-Directory Projects)

For monorepos or multi-module projects, use `Paths()` to control where tasks are
visible:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        rootTask,                                  // visible at root only
        pocket.Paths(apiTask).In("services/api"),  // visible in services/api/
        pocket.Paths(webTask).In("services/web"),  // visible in services/web/
    ),
}
```

Tasks are visible based on the current working directory:

- Without `Paths()` wrapper: only visible at git root
- With `Paths().In(...)`: visible in specified directories (supports regex)

```go
// Run in multiple directories
pocket.Paths(myTask).In("proj1", "proj2")

// Match patterns (regex)
pocket.Paths(myTask).In("services/.*")

// Exclude directories
pocket.Paths(myTask).In("services/.*").Except("services/legacy")
```

### Execution Order

Use `Serial()` and `Parallel()` to control execution:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        formatTask,           // run first
        pocket.Parallel(      // then these in parallel
            lintTask,
            testTask,
        ),
        buildTask,            // run last
    ),
}
```

### Multi-Module Projects and Context Awareness

Pocket automatically generates shims in each module directory, making tasks
context-aware. When you run `./pok` from a subdirectory, only tasks relevant to
that module are executed.

**Example project structure:**

```
myproject/
├── .pocket/           # build configuration
├── pok                # root shim
├── go.mod             # root Go module
├── services/
│   └── api/
│       ├── pok        # auto-generated shim for this module
│       └── go.mod     # separate Go module
└── libs/
    └── common/
        ├── pok        # auto-generated shim for this module
        └── go.mod     # separate Go module
```

**How it works:**

1. `./pok generate` creates shims in all detected module directories
1. Each shim knows its "context" (relative path from repo root)
1. Running `./pok` from a subdirectory filters tasks to that module only

```bash
# From repo root - runs tasks on ALL modules
./pok go-test

# From services/api/ - runs tasks only on that module
cd services/api
./pok go-test
```

This enables focused workflows in large monorepos while keeping a single
configuration in `.pocket/config.go`.

### Windows Support

When bootstrapping, pocket automatically detects your platform:

- **Unix/macOS/WSL**: Creates `./pok` (Posix bash script)
- **Windows**: Creates `pok.cmd` and `pok.ps1`

To add additional shim types after bootstrapping, update your
`.pocket/config.go`:

```go
var Config = pocket.Config{
    Shim: &pocket.ShimConfig{
        Posix:      true,   // ./pok (bash) - default
        Windows:    true,   // pok.cmd (requires Go in PATH)
        PowerShell: true,   // pok.ps1 (can auto-download Go)
    },
    // ... rest of config
}
```

After updating the config, run `./pok generate` to create the Windows shims.

**Shim types:**

| Shim          | File      | Go Auto-Download | Notes                        |
| ------------- | --------- | ---------------- | ---------------------------- |
| Posix         | `./pok`   | Yes              | Works with bash, Git Bash    |
| Windows (CMD) | `pok.cmd` | No               | Requires Go in PATH          |
| PowerShell    | `pok.ps1` | Yes              | Full-featured Windows option |

**Using the shims on Windows:**

```batch
rem CMD
pok.cmd go-test

rem PowerShell
.\pok.ps1 go-test
```

### Built-in Task Packages (Optional)

Pocket includes opinionated task packages for common languages. These use
auto-detection to find project directories:

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/fredrikaverpil/pocket/tasks/golang"
    "github.com/fredrikaverpil/pocket/tasks/python"
    "github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
    Run: pocket.Serial(
        pocket.AutoDetect(golang.Tasks()),    // finds go.mod files
        pocket.AutoDetect(python.Tasks()),    // finds pyproject.toml
        pocket.AutoDetect(markdown.Tasks()),  // formats markdown from root
    ),
}
```

Available packages:

- `tasks/golang` - Go formatting (gofumpt, goimports), linting (golangci-lint),
  testing, vulncheck
- `tasks/python` - Python formatting and linting (ruff), type checking (mypy)
- `tasks/markdown` - Markdown formatting (mdformat)
- `tasks/lua` - Lua formatting (stylua)

#### Configuring Built-in Tasks

Each package accepts an optional `Options` struct to customize behavior:

```go
// Use custom golangci-lint config
pocket.AutoDetect(golang.Tasks(golang.Options{
    LintConfig: ".golangci.yml",
}))

// Use custom ruff config
pocket.AutoDetect(python.Tasks(python.Options{
    RuffConfig: "pyproject.toml",
}))
```

Without options, sensible defaults are used (e.g., race detection enabled,
pocket's bundled configs).

> [!TIP]
>
> You can combine options with path filtering:
>
> ```go
> pocket.AutoDetect(golang.Tasks(golang.Options{
>     LintConfig: ".golangci.yml",
> })).Except("vendor", "testdata")
> ```

## Terminology

Pocket has three levels of configuration:

```
Config (project)
  └── Runnable (execution tree: Serial, Parallel, Paths wrappers)
        └── Task (executable unit of work)
```

### Config ([`config.go`](config.go))

- Project-level configuration
- Defines the execution tree via `Run`, shim settings, and options
- Lives in [`.pocket/config.go`](.pocket/config.go)

### Runnable

- Anything that can be executed: `Task`, `Serial()`, `Parallel()`, or
  `PathFilter`
- `Serial(...)` runs children in order
- `Parallel(...)` runs children concurrently
- `Paths(runnable)` wraps a runnable with path filtering

### PathFilter (Path Filtering)

- Wraps a Runnable with directory-based visibility
- `Paths(r).In("dir1", "dir2")` - visible in specified directories (supports
  regex)
- `Paths(r).Except("vendor")` - exclude directories from visibility
- Tasks without `Paths()` wrapper are only visible at git root

### Task

- Executable unit of work: `go-format`, `go-lint`, `py-typecheck`...
- Has `Name`, `Usage`, optional `Args`, and `Action` function
- Individual tasks: `golang.FormatTask()`, `golang.LintTask()`, etc.

## Convenience Functions

Pocket provides helpers for writing custom tasks:

**Path helpers:**

```go
pocket.GitRoot()              // returns git repository root path
pocket.FromGitRoot("subdir")  // joins paths relative to git root
pocket.FromPocketDir("file")  // joins paths relative to .pocket/
pocket.FromBinDir("tool")     // joins paths relative to .pocket/bin/
pocket.BinaryName("mytool")   // appends .exe on Windows
```

**Execution helpers:**

```go
// Creates exec.Cmd with PATH including .pocket/bin/
cmd := pocket.Command(ctx, "go", "build", "./...")
cmd.Dir = pocket.FromGitRoot("subdir")
cmd.Run()
```

**Detection helpers (for Auto mode):**

```go
pocket.DetectByFile("go.mod")        // finds dirs with go.mod
pocket.DetectByExtension(".lua")     // finds dirs with .lua files
```

**Task orchestration:**

```go
// Run tasks in parallel
pocket.Deps(ctx, formatTask, lintTask)

// Run tasks sequentially
pocket.SerialDeps(ctx, formatTask, lintTask, testTask)

// Check verbose mode
if pocket.IsVerbose(ctx) {
    args = append(args, "-v")
}
```

## Acknowledgements

- [einride/sage](https://github.com/einride/sage) - Inspiration for the task
  system and tool management approach
