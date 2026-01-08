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

Edit `.pocket/config.go` to configure tasks.

**Basic configuration with auto-detection:**

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/fredrikaverpil/pocket/tasks/golang"
    "github.com/fredrikaverpil/pocket/tasks/python"
    "github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
    Run: pocket.Serial(
        pocket.P(golang.Tasks()).Detect(),    // auto-detects Go modules (go.mod)
        pocket.P(python.Tasks()).Detect(),    // auto-detects Python projects
        pocket.P(markdown.Tasks()).Detect(),  // formats markdown from root
    ),
}
```

**Path filtering with explicit paths:**

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        // Only run Go tasks in specific directories
        pocket.P(golang.Tasks()).In("proj1", "proj2"),

        // Auto-detect but exclude certain directories
        pocket.P(python.Tasks()).Detect().Except("vendor", `\.pocket`),

        // Run markdown tasks only at root
        pocket.P(markdown.Tasks()).In("."),
    ),
}
```

**Regex patterns for path matching:**

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        // Match all services subdirectories
        pocket.P(golang.Tasks()).Detect().In("services/.*"),

        // Exclude test directories
        pocket.P(python.Tasks()).Detect().Except(".*_test", "testdata"),
    ),
}
```

**Parallel execution:**

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        pocket.P(golang.Tasks()).Detect(),  // run first
        pocket.Parallel(                     // then these in parallel
            pocket.P(python.Tasks()).Detect(),
            pocket.P(markdown.Tasks()).Detect(),
        ),
    ),
}
```

### Custom Tasks

Add your own tasks in `.pocket/config.go`:

```go
import (
    "context"
    "fmt"

    "github.com/fredrikaverpil/pocket"
    "github.com/fredrikaverpil/pocket/tasks/golang"
)

var Config = pocket.Config{
    Run: pocket.Serial(
        pocket.P(golang.Tasks()).Detect(),
        deployTask,  // custom task runs at root only (no P() wrapper)
    ),
}

var deployTask = &pocket.Task{
    Name:  "deploy",
    Usage: "deploy to production",
    Action: func(ctx context.Context, args map[string]string) error {
        fmt.Println("Deploying...")
        // your logic here
        return nil
    },
}
```

Custom tasks appear in `./pok -h` and run as part of `./pok` (no args).

> [!NOTE]
>
> Tasks without a `P()` wrapper are only visible when running from the git root.
> Use `pocket.P(myTask).In("subdir")` to make a task visible in specific
> directories.

**Tasks with arguments:**

Tasks can declare arguments that users pass via `key=value` syntax:

```go
var greetTask = &pocket.Task{
    Name:  "greet",
    Usage: "print a greeting",
    Args: []pocket.ArgDef{
        {Name: "name", Usage: "who to greet", Default: "world"},
    },
    Action: func(ctx context.Context, args map[string]string) error {
        fmt.Printf("Hello, %s!\n", args["name"])
        return nil
    },
}
```

```bash
./pok greet              # Hello, world!
./pok greet name=Freddy  # Hello, Freddy!
./pok -h greet           # show task help with arguments
```

For multi-module projects, you can define context-specific tasks that only
appear when running the shim from that folder:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        rootTask,                              // visible at root only
        pocket.P(apiTask).In("services/api"),  // visible in services/api/
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

- Anything that can be executed: `Task`, `Serial()`, `Parallel()`, or `Paths`
- `Serial(...)` runs children in order
- `Parallel(...)` runs children concurrently
- `P(runnable)` wraps a runnable with path filtering

### Paths (Path Filtering)

- Wraps a Runnable with directory-based visibility
- `P(r).Detect()` - auto-detect directories (using `DefaultDetect()`)
- `P(r).In("dir1", "dir2")` - explicit directories (supports regex)
- `P(r).Except("vendor")` - exclude directories
- Tasks without `P()` wrapper are only visible at git root

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
